package cliutil

import (
	"strings"
	"testing"
)

const refusalCommand = `vrooli agent authorize "$VROOLI_AGENT_SESSION" --policy agent-management --target agent-manager:task/6041722d --reason "<why>"`

// A refused agent must see the command that asks the operator to approve the
// action. Agent Manager's 403 body carries it as `request_authority`; the
// terminal output of every scenario CLI is APIError.Error().
func TestAPIErrorPrintsRequestAuthorityWhenOnlyTheBodyFieldCarriesIt(t *testing.T) {
	body := `{"error":"task cancellation requires exact agent.manage authority","request_authority":` + quoteJSON(refusalCommand) + `}`
	err := ParseAPIError(403, []byte(body))
	if err.RequestAuthority != refusalCommand {
		t.Fatalf("request_authority was dropped: %+v", err)
	}
	if got := err.Error(); !strings.Contains(got, "api error (403): task cancellation requires exact agent.manage authority") || !strings.Contains(got, refusalCommand) {
		t.Fatalf("terminal text hides the request command: %q", got)
	}
	if !err.IsStructured() || !strings.Contains(err.FormatConcise(), refusalCommand) {
		t.Fatalf("structured rendering hides the request command: %q", err.FormatConcise())
	}
}

func TestAPIErrorDoesNotRepeatARequestCommandTheMessageAlreadyCarries(t *testing.T) {
	message := "run identity cannot perform lifecycle operation without an approved grant; ask the operator with: " + refusalCommand
	body := `{"error":` + quoteJSON(message) + `,"request_authority":` + quoteJSON(refusalCommand) + `}`
	got := ParseAPIError(403, []byte(body)).Error()
	if strings.Count(got, refusalCommand) != 1 {
		t.Fatalf("command printed %d times, want once: %q", strings.Count(got, refusalCommand), got)
	}
}

func TestAPIErrorIgnoresMalformedRequestAuthority(t *testing.T) {
	for _, body := range []string{`{"error":"nope","request_authority":42}`, `{"error":"nope","request_authority":null}`, `{"error":"nope","request_authority":{"cmd":"x"}}`, `{"error":"nope"}`} {
		err := ParseAPIError(403, []byte(body))
		if err.RequestAuthority != "" || err.Error() != "api error (403): nope" {
			t.Fatalf("%s: unexpected rendering %q (%+v)", body, err.Error(), err)
		}
	}
}

func quoteJSON(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`)
	return `"` + replacer.Replace(value) + `"`
}
