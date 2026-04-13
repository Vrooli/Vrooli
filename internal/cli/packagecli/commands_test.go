package packagecli

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseRefreshRequestCapturesTargetAndNoRestart(t *testing.T) {
	req, err := ParseRefreshRequest([]string{"api-core", "alpha", "--no-restart"})
	if err != nil {
		t.Fatalf("ParseRefreshRequest: %v", err)
	}
	if req.Name != "api-core" || req.Target != "alpha" || !req.NoRestart {
		t.Fatalf("req = %+v", req)
	}
}

func TestParseValidateRequestDefaultsToAll(t *testing.T) {
	req, err := ParseValidateRequest(nil)
	if err != nil {
		t.Fatalf("ParseValidateRequest: %v", err)
	}
	if !req.All || req.Name != "" {
		t.Fatalf("req = %+v", req)
	}
}

func TestParseAuditRequestRejectsUnknownFlag(t *testing.T) {
	if _, err := ParseAuditRequest([]string{"--bogus"}); err == nil || !strings.Contains(err.Error(), "unknown option for package audit") {
		t.Fatalf("ParseAuditRequest error = %v", err)
	}
}

func TestRenderCommandHelpIncludesRefresh(t *testing.T) {
	var stdout bytes.Buffer
	RenderCommandHelp(&stdout)
	if !strings.Contains(stdout.String(), "refresh") || !strings.Contains(stdout.String(), "propagate to affected consumers") {
		t.Fatalf("help = %q", stdout.String())
	}
}
