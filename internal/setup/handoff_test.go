//nolint:goconst // test data deliberately reuses stable executable fixtures.
package setup

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostpresentation"
	"github.com/vrooli/vrooli/internal/onboardinghandoff"
	"github.com/vrooli/vrooli/internal/testenv"
)

func prepareHandoffFixture(t *testing.T, capability hostpresentation.Capability) (*setupService, string, string) {
	t.Helper()
	root, home := t.TempDir(), t.TempDir()
	writeOnboardingScenarioFixture(t, root)
	svc := stubSetupDeps(t)
	svc.deps.detectPresentation = func(context.Context) hostpresentation.Capability { return capability }
	svc.deps.osExecutable = func() (string, error) { return "/bin/true", nil }
	svc.deps.onboardingPortCommandRunner = func(context.Context, string, ...string) ([]byte, error) {
		return []byte("41234\n"), nil
	}
	return svc, root, home
}

func withFakeOnboardingStart(t *testing.T) *int {
	t.Helper()
	previous := startOnboardingScenarioFn
	calls := 0
	startOnboardingScenarioFn = func(string, string) error {
		calls++
		return nil
	}
	t.Cleanup(func() { startOnboardingScenarioFn = previous })
	return &calls
}

func TestOnboardingHandoffRemoteShellDoesNotOpenBrowser(t *testing.T) {
	svc, root, home := prepareHandoffFixture(t, hostpresentation.Capability{
		Kind:      hostpresentation.KindRemoteShell,
		Reason:    "SSH session, no local display",
		Reachable: false,
	})
	withFakeOnboardingStart(t)
	opened := false
	svc.deps.openOnboardingURL = func(string) error { opened = true; return nil }

	stdout := &bytes.Buffer{}
	result, err := svc.runOnboardingHandoff(root, home, Options{}, stdout, io.Discard)
	if err != nil {
		t.Fatalf("runOnboardingHandoff: %v", err)
	}
	if opened {
		t.Fatal("remote-shell handoff opened a browser")
	}
	if result.Decision != "url" || result.Opened {
		t.Fatalf("result = %+v, want URL handoff without opening", result)
	}
	if result.ResumeCommand == "" || !strings.Contains(stdout.String(), result.ResumeCommand) {
		t.Fatalf("stdout = %q, want resumable CLI command", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Presentation: remote-shell") {
		t.Fatalf("stdout = %q, want presentation classification", stdout.String())
	}
}

func TestOnboardingHandoffLocalGraphicalUsesBrowserInAutoMode(t *testing.T) {
	svc, root, home := prepareHandoffFixture(t, hostpresentation.Capability{
		Kind:      hostpresentation.KindLocalGraphical,
		Reason:    "local graphical session",
		Reachable: true,
	})
	withFakeOnboardingStart(t)
	opened := ""
	svc.deps.openOnboardingURL = func(url string) error {
		opened = url
		return nil
	}

	result, err := svc.runOnboardingHandoff(root, home, Options{}, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("runOnboardingHandoff: %v", err)
	}
	if result.Decision != "browser" || !result.Opened || opened != "http://127.0.0.1:41234" {
		t.Fatalf("result = %+v, opened = %q", result, opened)
	}
}

func TestOnboardingHandoffBrowserFailureFallsBackToURL(t *testing.T) {
	svc, root, home := prepareHandoffFixture(t, hostpresentation.Capability{
		Kind:      hostpresentation.KindLocalGraphical,
		Reason:    "local graphical session",
		Reachable: true,
	})
	withFakeOnboardingStart(t)
	svc.deps.openOnboardingURL = func(string) error { return context.DeadlineExceeded }
	stdout := &bytes.Buffer{}

	result, err := svc.runOnboardingHandoff(root, home, Options{}, stdout, io.Discard)
	if err != nil {
		t.Fatalf("runOnboardingHandoff: %v", err)
	}
	if result.Decision != "url" || result.Opened || result.URL != "http://127.0.0.1:41234" {
		t.Fatalf("result = %+v", result)
	}
	if !strings.Contains(stdout.String(), "Or forward the port") {
		t.Fatalf("stdout = %q, want forwarding guidance", stdout.String())
	}
}

func TestOnboardingHandoffEveryPresentationKindHasResumePath(t *testing.T) {
	kinds := []hostpresentation.Kind{
		hostpresentation.KindLocalGraphical,
		hostpresentation.KindWSLGraphical,
		hostpresentation.KindForwardedGraphical,
		hostpresentation.KindRemoteDesktop,
		hostpresentation.KindRemoteShell,
		hostpresentation.KindHeadless,
		hostpresentation.KindUnknown,
	}
	for _, kind := range kinds {
		t.Run(string(kind), func(t *testing.T) {
			svc, root, home := prepareHandoffFixture(t, hostpresentation.Capability{Kind: kind, Reason: "test capability"})
			withFakeOnboardingStart(t)
			result, err := svc.runOnboardingHandoff(root, home, Options{Onboarding: onboardinghandoff.ModeURL}, io.Discard, io.Discard)
			if err != nil {
				t.Fatalf("runOnboardingHandoff: %v", err)
			}
			if result.ResumeCommand == "" || result.PresentationKind != string(kind) {
				t.Fatalf("result = %+v, want presentation and resume path", result)
			}
		})
	}
}

func TestStartOnboardingScenarioDoesNotUseOperatorSessionWhenUnprivileged(t *testing.T) {
	previousUID := invokingUIDFn
	previousOperatorLaunch := launchOnboardingAsOperatorFn
	t.Cleanup(func() {
		invokingUIDFn = previousUID
		launchOnboardingAsOperatorFn = previousOperatorLaunch
	})
	invokingUIDFn = func() int { return 1000 }
	operatorLaunches := 0
	launchOnboardingAsOperatorFn = func(string, string) error {
		operatorLaunches++
		return nil
	}
	testenv.SetSudoUser(t, "operator")

	if err := startOnboardingScenario(t.TempDir(), "/bin/true"); err != nil {
		t.Fatalf("startOnboardingScenario: %v", err)
	}
	if operatorLaunches != 0 {
		t.Fatalf("operator-session launches = %d, want 0", operatorLaunches)
	}
}
