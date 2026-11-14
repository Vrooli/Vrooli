package resourcecli

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/packages/harness-lifecycle"
)

func TestParseUpdateRequestSelectsOneAction(t *testing.T) {
	cases := map[string]struct {
		args []string
		want string
	}{
		"update":   {[]string{"codex"}, agentinstall.ActionUpdate},
		"install":  {[]string{"codex", "--action", "install"}, agentinstall.ActionInstall},
		"check":    {[]string{"codex", "--check"}, CheckAction},
		"use":      {[]string{"codex", "--action", "use-managed-copy"}, agentinstall.ActionUseManagedCopy},
		"link":     {[]string{"codex", "--action", "link-managed-copy", "--json"}, agentinstall.ActionLinkManagedCopy},
		"rollback": {[]string{"codex", "--action", "rollback"}, agentinstall.ActionRollback},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			req, err := ParseUpdateRequest(tc.args)
			if err != nil || req.Name != "codex" || req.Action != tc.want {
				t.Fatalf("req=%+v err=%v", req, err)
			}
		})
	}
	for name, args := range map[string][]string{
		"no name":        {},
		"unknown action": {"codex", "--action", "repair"},
		"both":           {"codex", "--check", "--action", "rollback"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseUpdateRequest(args); err == nil {
				t.Fatalf("%v must be refused", args)
			}
		})
	}
}

func TestParseUpdateCheckIsAlwaysReadOnly(t *testing.T) {
	req, err := ParseUpdateCheckRequest([]string{"grok", "--json"})
	if err != nil || req.Action != CheckAction || req.Name != "grok" {
		t.Fatalf("req=%+v err=%v", req, err)
	}
	if _, err := ParseUpdateCheckRequest([]string{"grok", "--action", "rollback"}); err == nil {
		t.Fatal("the read-only verb must not accept an action")
	}
}

// Web Console reads this envelope, including from a failed action.
func TestHarnessUpdateJSONCarriesTheReportOnFailure(t *testing.T) {
	var out bytes.Buffer
	result := agentinstall.HarnessActionResult{Action: agentinstall.ActionUpdate, Message: "refused", Report: agentinstall.HarnessReport{Name: "codex", State: agentinstall.StateCheckFailed}}
	if err := WriteHarnessUpdate(&out, cliout.FormatJSON, result); err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		Success bool `json:"success"`
		Update  struct {
			Message string `json:"message"`
			Report  struct {
				State string `json:"state"`
			} `json:"report"`
		} `json:"harness_update"`
	}
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatalf("%v: %s", err, out.String())
	}
	if envelope.Success || envelope.Update.Message != "refused" || envelope.Update.Report.State != agentinstall.StateCheckFailed {
		t.Fatalf("envelope = %+v", envelope)
	}
}
