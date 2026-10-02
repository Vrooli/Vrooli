package sidecar

import (
	"strings"
	"testing"
	"time"

	"github.com/vrooli/browser-automation-studio/automation/driver"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basdomain "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/domain"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestRecoveryAdminSecretUsesConfiguredValueOrGeneratesOne(t *testing.T) {
	t.Setenv(driver.PlaywrightDriverAdminSecretEnv, "configured-secret")
	configured, err := recoveryAdminSecret()
	if err != nil {
		t.Fatalf("configured recoveryAdminSecret: %v", err)
	}
	if configured != "configured-secret" {
		t.Fatalf("configured secret = %q", configured)
	}

	t.Setenv(driver.PlaywrightDriverAdminSecretEnv, "")
	generated, err := recoveryAdminSecret()
	if err != nil {
		t.Fatalf("generated recoveryAdminSecret: %v", err)
	}
	if len(generated) < 32 || strings.ContainsAny(generated, " \t\r\n") {
		t.Fatalf("generated secret is not a usable high-entropy token: %q", generated)
	}
}

func TestRecoveryActionsProjectCanonicalTimelineEntries(t *testing.T) {
	at := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	selector, value := "#submit", "hello"
	entries := []*bastimeline.TimelineEntry{
		{Id: "entry-1", SequenceNum: 1, Timestamp: timestamppb.New(at), Action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_INPUT, Params: &basactions.ActionDefinition_Input{Input: &basactions.InputParams{Selector: selector, Value: value}}}, Telemetry: &basdomain.ActionTelemetry{Url: "https://example.test"}},
	}
	actions, currentURL := recoveryActionsFromTimelineEntries(entries)
	if len(actions) != 1 || actions[0].Type != "input" || actions[0].Selector != selector || actions[0].Value != value || !actions[0].Timestamp.Equal(at) || currentURL != "https://example.test" {
		t.Fatalf("timeline projection = %#v, currentURL=%q", actions, currentURL)
	}
}
