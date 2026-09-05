package onboardinghandoff

import (
	"testing"

	"github.com/vrooli/vrooli/internal/hostpresentation"
)

func TestDecidePolicyTable(t *testing.T) {
	tests := []struct {
		name       string
		cap        hostpresentation.Capability
		mode       Mode
		stdin      bool
		wantAction string
		wantErr    bool
	}{
		{name: "auto local", cap: hostpresentation.Capability{Kind: hostpresentation.KindLocalGraphical, Reachable: true, Reason: "local"}, mode: ModeAuto, wantAction: "browser"},
		{name: "auto wsl", cap: hostpresentation.Capability{Kind: hostpresentation.KindWSLGraphical, Reachable: true, Reason: "wsl"}, mode: ModeAuto, wantAction: "browser"},
		{name: "auto rdp", cap: hostpresentation.Capability{Kind: hostpresentation.KindRemoteDesktop, Reachable: true, Reason: "rdp"}, mode: ModeAuto, wantAction: "browser"},
		{name: "auto forwarded", cap: hostpresentation.Capability{Kind: hostpresentation.KindForwardedGraphical, Reachable: true, Reason: "forwarded"}, mode: ModeAuto, wantAction: "url"},
		{name: "auto remote shell", cap: hostpresentation.Capability{Kind: hostpresentation.KindRemoteShell, Reason: "ssh"}, mode: ModeAuto, wantAction: "url"},
		{name: "auto headless", cap: hostpresentation.Capability{Kind: hostpresentation.KindHeadless, Reason: "headless"}, mode: ModeAuto, wantAction: "url"},
		{name: "auto unknown", cap: hostpresentation.Capability{Kind: hostpresentation.KindUnknown, Reason: "unknown"}, mode: ModeAuto, wantAction: "url"},
		{name: "auto degraded", cap: hostpresentation.Capability{Kind: hostpresentation.KindLocalGraphical, Reachable: true, Degraded: true}, mode: ModeAuto, wantAction: "url"},
		{name: "explicit browser", cap: hostpresentation.Capability{Kind: hostpresentation.KindHeadless}, mode: ModeBrowser, wantAction: "browser"},
		{name: "explicit cli", cap: hostpresentation.Capability{Kind: hostpresentation.KindHeadless}, mode: ModeCLI, stdin: true, wantAction: "cli"},
		{name: "explicit cli without terminal", cap: hostpresentation.Capability{Kind: hostpresentation.KindHeadless}, mode: ModeCLI, wantErr: true},
		{name: "explicit url", cap: hostpresentation.Capability{Kind: hostpresentation.KindLocalGraphical}, mode: ModeURL, wantAction: "url"},
		{name: "explicit none", cap: hostpresentation.Capability{Kind: hostpresentation.KindLocalGraphical}, mode: ModeNone, wantAction: "none"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Decide(tt.cap, tt.mode, tt.stdin)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, want error=%t", err, tt.wantErr)
			}
			if err == nil && got.Action != tt.wantAction {
				t.Fatalf("decision = %#v, want action %q", got, tt.wantAction)
			}
			if err == nil && got.Action != "none" && got.ResumeCommand == "" {
				t.Fatal("non-none decision must carry a resume command")
			}
		})
	}
}

func TestParseModeRejectsUnknownValues(t *testing.T) {
	if _, err := ParseMode("bogus"); err == nil {
		t.Fatal("expected invalid onboarding mode")
	}
}
