package lark

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/core"
)

// AlertProvider is the configuration necessary for sending an alert using Lark
type AlertProvider struct {
	WebhookURL string `yaml:"webhook-url"` // Lark webhook URL
	// DefaultAlert is the default alert configuration to use for endpoints with an alert of the appropriate type
	DefaultAlert *alert.Alert `yaml:"default-alert,omitempty"`
	// Overrides is a list of Override that may be prioritized over the default configuration
	Overrides []Override `yaml:"overrides,omitempty"`
}

// Override is a case under which the default integration is overridden
type Override struct {
	Group      string `yaml:"group"`
	WebhookURL string `yaml:"webhook-url"`
}

// IsValid returns whether the provider's configuration is valid
func (provider *AlertProvider) IsValid() bool {
	registeredGroups := make(map[string]bool)
	if provider.Overrides != nil {
		for _, override := range provider.Overrides {
			if isAlreadyRegistered := registeredGroups[override.Group]; isAlreadyRegistered || override.Group == "" || len(override.WebhookURL) == 0 {
				return false
			}
			registeredGroups[override.Group] = true
		}
	}
	return len(provider.WebhookURL) > 0
}

// Send an alert using the provider
func (provider *AlertProvider) Send(endpoint *core.Endpoint, alert *alert.Alert, result *core.Result, resolved bool) error {
	buffer := bytes.NewBuffer([]byte(provider.buildRequestBody(endpoint, alert, result, resolved)))
	request, err := http.NewRequest(http.MethodPost, provider.getWebhookURLForGroup(endpoint.Group), buffer)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.GetHTTPClient(nil).Do(request)
	if err != nil {
		return err
	}
	if response.StatusCode > 399 {
		body, _ := io.ReadAll(response.Body)
		return fmt.Errorf("call to provider alert returned status code %d: %s", response.StatusCode, string(body))
	}
	return err
}

// buildRequestBody builds the request body for the provider
func (provider *AlertProvider) buildRequestBody(endpoint *core.Endpoint, alert *alert.Alert, result *core.Result, resolved bool) string {
	var color, results string
	if resolved {
		color = "green"
	} else {
		color = "red"
	}
	for _, conditionResult := range result.ConditionResults {
		var prefix string
		if conditionResult.Success {
			prefix = "✅"
		} else {
			prefix = "❌"
		}
		results += fmt.Sprintf("%s %s\\n", prefix, conditionResult.Condition)
	}
	//var description string
	//if alertDescription := alert.GetDescription(); len(alertDescription) > 0 {
	//	description = ":\\n> " + alertDescription
	//}
	return fmt.Sprintf(`{
    "msg_type": "interactive",
    "card": {
        "config": {
            "wide_screen_mode": true,
            "enable_forward": true
        },
        "header": {
            "title": {
                "content": "%s",
                "tag": "plain_text"
            },
            "template": "%s"
        },
        "elements": [
            {
                "tag": "div",
                "text": {
                    "content": "**%s**",
                    "tag": "lark_md"
                }
            },
			{
                "tag": "div",
                "text": {
                    "content": "%s",
                    "tag": "lark_md"
                }
            },
			{
                "tag": "div",
                "text": {
                    "content": "**Time: %s**",
                    "tag": "lark_md"
                }
            },
            {
                "tag": "div",
                "text": {
                    "content": "%s",
                    "tag": "lark_md"
                }
            }
        ]
    }
}`, endpoint.Name, color, endpoint.URL, endpoint.Group, time.Now().Format(time.RFC3339), results)
}

// getWebhookURLForGroup returns the appropriate Webhook URL integration to for a given group
func (provider *AlertProvider) getWebhookURLForGroup(group string) string {
	if provider.Overrides != nil {
		for _, override := range provider.Overrides {
			if group == override.Group {
				return override.WebhookURL
			}
		}
	}
	return provider.WebhookURL
}

// GetDefaultAlert returns the provider's default alert configuration
func (provider AlertProvider) GetDefaultAlert() *alert.Alert {
	return provider.DefaultAlert
}
