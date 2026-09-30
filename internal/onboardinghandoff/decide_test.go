package onboardinghandoff

import (
	"testing"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

func TestDecidePolicyTable(t *testing.T) {
	tests := []struct {
		name       string
		cap        hostinventory.Capability
		mode       Mode
		stdin      bool
		wantAction string
		wantErr    bool
	}{
		{name: "auto local", cap: hostinventory.Capability{Kind: hostinventory.KindLocalGraphical, Reachable: true, Reason: "local"}, mode: ModeAuto, wantAction: "browser"},
		{name: "auto wsl", cap: hostinventory.Capability{Kind: hostinventory.KindWSLGraphical, Reachable: true, Reason: "wsl"}, mode: ModeAuto, wantAction: "browser"},
		{name: "auto rdp", cap: hostinventory.Capability{Kind: hostinventory.KindRemoteDesktop, Reachable: true, Reason: "rdp"}, mode: ModeAuto, wantAction: "browser"},
		{name: "auto forwarded", cap: hostinventory.Capability{Kind: hostinventory.KindForwardedGraphical, Reachable: true, Reason: "forwarded"}, mode: ModeAuto, wantAction: "url"},
		{name: "auto remote shell", cap: hostinventory.Capability{Kind: hostinventory.KindRemoteShell, Reason: "ssh"}, mode: ModeAuto, wantAction: "url"},
		{name: "auto headless", cap: hostinventory.Capability{Kind: hostinventory.KindHeadless, Reason: "headless"}, mode: ModeAuto, wantAction: "url"},
		{name: "auto unknown", cap: hostinventory.Capability{Kind: hostinventory.KindUnknown, Reason: "unknown"}, mode: ModeAuto, wantAction: "url"},
		{name: "auto degraded", cap: hostinventory.Capability{Kind: hostinventory.KindLocalGraphical, Reachable: true, Degraded: true}, mode: ModeAuto, wantAction: "url"},
		{name: "explicit browser", cap: hostinventory.Capability{Kind: hostinventory.KindHeadless}, mode: ModeBrowser, wantAction: "browser"},
		{name: "explicit cli", cap: hostinventory.Capability{Kind: hostinventory.KindHeadless}, mode: ModeCLI, stdin: true, wantAction: "cli"},
		{name: "explicit cli without terminal", cap: hostinventory.Capability{Kind: hostinventory.KindHeadless}, mode: ModeCLI, wantErr: true},
		{name: "explicit url", cap: hostinventory.Capability{Kind: hostinventory.KindLocalGraphical}, mode: ModeURL, wantAction: "url"},
		{name: "explicit none", cap: hostinventory.Capability{Kind: hostinventory.KindLocalGraphical}, mode: ModeNone, wantAction: "none"},
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
