package lark

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/test"
)

func TestAlertProvider_Validate(t *testing.T) {
	if err := (&AlertProvider{WebhookURL: ""}).Validate(); err == nil {
		t.Error("provider shouldn't have been valid when WebhookURL is empty")
	}
	if err := (&AlertProvider{WebhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/abc"}).Validate(); err != nil {
		t.Errorf("provider should have been valid, got: %v", err)
	}
}

func TestAlertProvider_ValidateWithOverride(t *testing.T) {
	scenarios := []struct {
		Name        string
		Provider    AlertProvider
		ExpectError bool
	}{
		{
			Name: "override-with-empty-group",
			Provider: AlertProvider{
				WebhookURL: "https://example.com",
				Overrides:  []Override{{Group: "", WebhookURL: "https://example.com/other"}},
			},
			ExpectError: true,
		},
		{
			Name: "override-with-empty-webhook-url",
			Provider: AlertProvider{
				WebhookURL: "https://example.com",
				Overrides:  []Override{{Group: "g1", WebhookURL: ""}},
			},
			ExpectError: true,
		},
		{
			Name: "override-with-duplicate-group",
			Provider: AlertProvider{
				WebhookURL: "https://example.com",
				Overrides: []Override{
					{Group: "g1", WebhookURL: "https://example.com/a"},
					{Group: "g1", WebhookURL: "https://example.com/b"},
				},
			},
			ExpectError: true,
		},
		{
			Name: "valid-override",
			Provider: AlertProvider{
				WebhookURL: "https://example.com",
				Overrides:  []Override{{Group: "g1", WebhookURL: "https://example.com/a"}},
			},
			ExpectError: false,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			err := scenario.Provider.Validate()
			if scenario.ExpectError && err == nil {
				t.Error("expected error, got none")
			}
			if !scenario.ExpectError && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}

func TestAlertProvider_Send(t *testing.T) {
	defer client.InjectHTTPClient(nil)
	description := "description-1"
	scenarios := []struct {
		Name             string
		Provider         AlertProvider
		Alert            alert.Alert
		Resolved         bool
		MockRoundTripper test.MockRoundTripper
		ExpectedError    bool
	}{
		{
			Name:     "triggered",
			Provider: AlertProvider{WebhookURL: "https://example.com"},
			Alert:    alert.Alert{Description: &description, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			ExpectedError: false,
		},
		{
			Name:     "triggered-error",
			Provider: AlertProvider{WebhookURL: "https://example.com"},
			Alert:    alert.Alert{Description: &description, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: http.NoBody}
			}),
			ExpectedError: true,
		},
		{
			Name:     "resolved",
			Provider: AlertProvider{WebhookURL: "https://example.com"},
			Alert:    alert.Alert{Description: &description, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: true,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			ExpectedError: false,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			client.InjectHTTPClient(&http.Client{Transport: scenario.MockRoundTripper})
			err := scenario.Provider.Send(
				&endpoint.Endpoint{Name: "endpoint-name", URL: "https://example.com", Group: "core"},
				&scenario.Alert,
				&endpoint.Result{
					ConditionResults: []*endpoint.ConditionResult{
						{Condition: "[CONNECTED] == true", Success: scenario.Resolved},
						{Condition: "[STATUS] == 200", Success: scenario.Resolved},
					},
				},
				scenario.Resolved,
			)
			if scenario.ExpectedError && err == nil {
				t.Error("expected error, got none")
			}
			if !scenario.ExpectedError && err != nil {
				t.Error("expected no error, got", err.Error())
			}
		})
	}
}

// TestAlertProvider_buildRequestBody verifies the body is valid JSON and contains expected fields.
// We don't assert byte-exact because the payload embeds time.Now().
func TestAlertProvider_buildRequestBody(t *testing.T) {
	description := `quotes " and backslashes \ that would break fmt.Sprintf JSON`
	scenarios := []struct {
		Name             string
		Resolved         bool
		ExpectedTemplate string
	}{
		{Name: "triggered", Resolved: false, ExpectedTemplate: "red"},
		{Name: "resolved", Resolved: true, ExpectedTemplate: "green"},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			provider := &AlertProvider{WebhookURL: "https://example.com"}
			body := provider.buildRequestBody(
				&endpoint.Endpoint{Name: "endpoint-name", URL: "https://example.com", Group: "core"},
				&alert.Alert{Description: &description, SuccessThreshold: 5, FailureThreshold: 3},
				&endpoint.Result{
					ConditionResults: []*endpoint.ConditionResult{
						{Condition: "[CONNECTED] == true", Success: scenario.Resolved},
						{Condition: "[STATUS] == 200", Success: scenario.Resolved},
					},
				},
				scenario.Resolved,
			)
			var parsed Body
			if err := json.Unmarshal(body, &parsed); err != nil {
				t.Fatalf("expected valid JSON, got error: %v\nbody: %s", err, string(body))
			}
			if parsed.MsgType != "interactive" {
				t.Errorf("expected msg_type=interactive, got %q", parsed.MsgType)
			}
			if parsed.Card.Header.Template != scenario.ExpectedTemplate {
				t.Errorf("expected template=%q, got %q", scenario.ExpectedTemplate, parsed.Card.Header.Template)
			}
			if len(parsed.Card.Elements) != 4 {
				t.Fatalf("expected 4 elements, got %d", len(parsed.Card.Elements))
			}
			if !strings.Contains(parsed.Card.Elements[0].Text.Content, description) {
				t.Errorf("expected message element to include the raw description (including quotes/backslashes), got %q", parsed.Card.Elements[0].Text.Content)
			}
		})
	}
}

func TestAlertProvider_GetDefaultAlert(t *testing.T) {
	if (&AlertProvider{DefaultAlert: &alert.Alert{}}).GetDefaultAlert() == nil {
		t.Error("expected default alert to be not nil")
	}
	if (&AlertProvider{DefaultAlert: nil}).GetDefaultAlert() != nil {
		t.Error("expected default alert to be nil")
	}
}

func TestAlertProvider_getWebhookURLForGroup(t *testing.T) {
	provider := &AlertProvider{
		WebhookURL: "https://default.example.com",
		Overrides: []Override{
			{Group: "core", WebhookURL: "https://core.example.com"},
			{Group: "edge", WebhookURL: "https://edge.example.com"},
		},
	}
	if got := provider.getWebhookURLForGroup("core"); got != "https://core.example.com" {
		t.Errorf("expected core override, got %q", got)
	}
	if got := provider.getWebhookURLForGroup("unknown"); got != "https://default.example.com" {
		t.Errorf("expected default URL for unknown group, got %q", got)
	}
}

func TestAlertProvider_ValidateOverrides(t *testing.T) {
	provider := &AlertProvider{
		WebhookURL: "https://default.example.com",
		Overrides:  []Override{{Group: "core", WebhookURL: "https://core.example.com"}},
	}
	if err := provider.ValidateOverrides("core", &alert.Alert{}); err != nil {
		t.Errorf("expected no error for known group, got: %v", err)
	}
	if err := provider.ValidateOverrides("unknown", &alert.Alert{}); err != nil {
		t.Errorf("expected no error for unknown group (falls back to default), got: %v", err)
	}
	emptyProvider := &AlertProvider{}
	if err := emptyProvider.ValidateOverrides("any", &alert.Alert{}); err == nil {
		t.Error("expected error when no webhook-url is set")
	}
}
