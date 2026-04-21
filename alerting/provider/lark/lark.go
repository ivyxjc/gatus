package lark

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
)

var (
	ErrWebhookURLNotSet          = errors.New("webhook-url not set")
	ErrDuplicateGroupOverride    = errors.New("duplicate group override")
	ErrOverrideGroupRequired     = errors.New("override group must not be empty")
	ErrOverrideWebhookURLMissing = errors.New("override webhook-url must not be empty")
)

// AlertProvider is the configuration necessary for sending an alert using Lark (Feishu) interactive cards.
// See https://open.feishu.cn/document/uAjLw4CM/ukzMukzMukzM/feishu-cards/card-json-v1-structure
type AlertProvider struct {
	WebhookURL string `yaml:"webhook-url"`

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

// Validate the provider's configuration
func (provider *AlertProvider) Validate() error {
	registeredGroups := make(map[string]bool)
	for _, override := range provider.Overrides {
		if override.Group == "" {
			return ErrOverrideGroupRequired
		}
		if len(override.WebhookURL) == 0 {
			return ErrOverrideWebhookURLMissing
		}
		if registeredGroups[override.Group] {
			return ErrDuplicateGroupOverride
		}
		registeredGroups[override.Group] = true
	}
	if len(provider.WebhookURL) == 0 {
		return ErrWebhookURLNotSet
	}
	return nil
}

// Send an alert using the provider
func (provider *AlertProvider) Send(ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) error {
	body := provider.buildRequestBody(ep, alert, result, resolved)
	request, err := http.NewRequest(http.MethodPost, provider.getWebhookURLForGroup(ep.Group), bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.GetHTTPClient(nil).Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode > 399 {
		responseBody, _ := io.ReadAll(response.Body)
		return fmt.Errorf("call to provider alert returned status code %d: %s", response.StatusCode, string(responseBody))
	}
	return nil
}

// Body is the top-level payload of a Lark interactive card message
type Body struct {
	MsgType string `json:"msg_type"`
	Card    Card   `json:"card"`
}

type Card struct {
	Config   CardConfig    `json:"config"`
	Header   CardHeader    `json:"header"`
	Elements []CardElement `json:"elements"`
}

type CardConfig struct {
	WideScreenMode bool `json:"wide_screen_mode"`
	EnableForward  bool `json:"enable_forward"`
}

type CardHeader struct {
	Title    CardText `json:"title"`
	Template string   `json:"template"`
}

type CardElement struct {
	Tag  string   `json:"tag"`
	Text CardText `json:"text"`
}

type CardText struct {
	Content string `json:"content"`
	Tag     string `json:"tag"`
}

// buildRequestBody builds the request body for the provider
func (provider *AlertProvider) buildRequestBody(ep *endpoint.Endpoint, a *alert.Alert, result *endpoint.Result, resolved bool) []byte {
	template := "red"
	message := fmt.Sprintf("An alert for **%s** has been triggered due to having failed %d time(s) in a row", ep.DisplayName(), a.FailureThreshold)
	if resolved {
		template = "green"
		message = fmt.Sprintf("An alert for **%s** has been resolved after passing successfully %d time(s) in a row", ep.DisplayName(), a.SuccessThreshold)
	}
	if description := a.GetDescription(); len(description) > 0 {
		message += "\n> " + description
	}
	var results string
	for _, cr := range result.ConditionResults {
		prefix := "❌"
		if cr.Success {
			prefix = "✅"
		}
		results += fmt.Sprintf("%s %s\n", prefix, cr.Condition)
	}
	body := Body{
		MsgType: "interactive",
		Card: Card{
			Config: CardConfig{WideScreenMode: true, EnableForward: true},
			Header: CardHeader{
				Title:    CardText{Content: ep.DisplayName(), Tag: "plain_text"},
				Template: template,
			},
			Elements: []CardElement{
				{Tag: "div", Text: CardText{Content: message, Tag: "lark_md"}},
				{Tag: "div", Text: CardText{Content: "**URL:** " + ep.URL, Tag: "lark_md"}},
				{Tag: "div", Text: CardText{Content: "**Time:** " + time.Now().Format(time.RFC3339), Tag: "lark_md"}},
				{Tag: "div", Text: CardText{Content: results, Tag: "lark_md"}},
			},
		},
	}
	bodyAsJSON, _ := json.Marshal(body)
	return bodyAsJSON
}

// getWebhookURLForGroup returns the appropriate Webhook URL integration to use for the given group
func (provider *AlertProvider) getWebhookURLForGroup(group string) string {
	for _, override := range provider.Overrides {
		if group == override.Group {
			return override.WebhookURL
		}
	}
	return provider.WebhookURL
}

// GetDefaultAlert returns the provider's default alert configuration
func (provider *AlertProvider) GetDefaultAlert() *alert.Alert {
	return provider.DefaultAlert
}

// ValidateOverrides validates the alert's provider override and, if present, the group override
func (provider *AlertProvider) ValidateOverrides(group string, alert *alert.Alert) error {
	if len(provider.getWebhookURLForGroup(group)) == 0 {
		return ErrWebhookURLNotSet
	}
	return nil
}
