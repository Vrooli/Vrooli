package graphicalsession

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func requirement(config map[string]any) hostreqspec.ResolvedRequirement {
	return hostreqspec.ResolvedRequirement{Name: "graphical_session", Kind: hostreqspec.KindSafeguard, Config: config}
}

func TestInspectGraphicalSession(t *testing.T) {
	original := collectFacts
	t.Cleanup(func() { collectFacts = original })
	t.Run("present", func(t *testing.T) {
		collectFacts = func() hostinventory.Snapshot {
			return hostinventory.Snapshot{SessionType: "aqua", DisplayAttached: true, ActiveSessionUser: "alice"}
		}
		status := NewHandler(hostreqkit.SafeguardManifest{Name: "graphical_session"}).Inspect(hostreqkit.Host{OS: "darwin"}, requirement(nil))
		if !status.Applied || status.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
			t.Fatalf("status = %+v", status)
		}
	})
	t.Run("configured_but_not_active", func(t *testing.T) {
		collectFacts = func() hostinventory.Snapshot {
			return hostinventory.Snapshot{AutoLoginUser: "alice", DisplayAttached: true}
		}
		status := NewHandler(hostreqkit.SafeguardManifest{Name: "graphical_session"}).Inspect(hostreqkit.Host{OS: "darwin"}, requirement(nil))
		if status.Applied || status.ExecutionState == hostreqkit.ExecutionAlreadyPresent {
			t.Fatalf("status = %+v", status)
		}
		if len(status.Notes) < 3 || !strings.Contains(status.Notes[0], "automatic login is configured") {
			t.Fatalf("status = %+v", status)
		}
	})
	t.Run("absent_without_opt_in", func(t *testing.T) {
		collectFacts = func() hostinventory.Snapshot { return hostinventory.Snapshot{} }
		status := NewHandler(hostreqkit.SafeguardManifest{Name: "graphical_session"}).Inspect(hostreqkit.Host{OS: "linux"}, requirement(nil))
		if status.Applied || len(status.Notes) < 2 {
			t.Fatalf("status = %+v", status)
		}
	})
	t.Run("not_applicable", func(t *testing.T) {
		status := NewHandler(hostreqkit.SafeguardManifest{Name: "graphical_session"}).Inspect(hostreqkit.Host{OS: "freebsd"}, requirement(nil))
		if status.SupportClass != hostreqkit.SupportNotApplicable || status.ExecutionState != hostreqkit.ExecutionNotApplicable {
			t.Fatalf("status = %+v", status)
		}
	})
}

func TestApplyDoesNotMutateWithoutOptIn(t *testing.T) {
	status := NewHandler(hostreqkit.SafeguardManifest{Name: "graphical_session"}).Inspect(hostreqkit.Host{OS: "darwin"}, requirement(map[string]any{}))
	called := false
	original := runPrivileged
	runPrivileged = func(string, string, []string, hostreqkit.EnsureOptions) error { called = true; return nil }
	t.Cleanup(func() { runPrivileged = original })
	status, _ = NewHandler(hostreqkit.SafeguardManifest{Name: "graphical_session"}).Apply(hostreqkit.Host{OS: "darwin"}, status, hostreqkit.EnsureOptions{})
	if called || status.ExecutionState != hostreqkit.ExecutionManualActionRequired {
		t.Fatalf("status=%+v called=%v", status, called)
	}
}

func TestApplyDoesNotSilentlyIgnoreAuthorizedLaunchAgentWithoutProgram(t *testing.T) {
	status := hostreqkit.ItemStatus{Name: "graphical_session", Kind: hostreqspec.KindSafeguard, ExecutionState: hostreqkit.ExecutionPending, Config: map[string]any{
		"allow_enable_automatic_login": true, "allow_install_gui_launch_agent": true, "desktop_user": "alice",
	}}
	called := false
	original := runPrivileged
	runPrivileged = func(string, string, []string, hostreqkit.EnsureOptions) error { called = true; return nil }
	t.Cleanup(func() { runPrivileged = original })
	result, err := NewHandler(hostreqkit.SafeguardManifest{Name: "graphical_session"}).Apply(hostreqkit.Host{OS: "darwin"}, status, hostreqkit.EnsureOptions{})
	if err != nil || called || result.ExecutionState != hostreqkit.ExecutionManualActionRequired {
		t.Fatalf("status=%+v err=%v called=%v", result, err, called)
	}
}

func TestApplyMacOSUsesInjectedPasswordThroughStdin(t *testing.T) {
	originalFacts := collectFacts
	collectFacts = func() hostinventory.Snapshot {
		return hostinventory.Snapshot{ProbeStatuses: map[string]string{"macos_auto_login": "enabled"}}
	}
	t.Cleanup(func() { collectFacts = originalFacts })
	status := hostreqkit.ItemStatus{Name: "graphical_session", Kind: hostreqspec.KindSafeguard, ExecutionState: hostreqkit.ExecutionPending, Config: map[string]any{
		"allow_enable_automatic_login": true, "desktop_user": "alice",
	}}
	originalEnv, hadEnv := os.LookupEnv(macOSAutoLoginPasswordEnv)
	if err := os.Setenv(macOSAutoLoginPasswordEnv, "not-a-real-password"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if hadEnv {
			_ = os.Setenv(macOSAutoLoginPasswordEnv, originalEnv)
		} else {
			_ = os.Unsetenv(macOSAutoLoginPasswordEnv)
		}
	})

	var gotCommand, gotInput string
	var gotArgs []string
	originalStdin := runPrivilegedWithStdin
	originalCommand := runPrivileged
	runPrivilegedWithStdin = func(_ string, command, input string, args []string) ([]byte, error) {
		gotCommand, gotInput, gotArgs = command, input, append([]string(nil), args...)
		return nil, nil
	}
	runPrivileged = func(string, string, []string, hostreqkit.EnsureOptions) error {
		t.Fatal("passworded macOS autologin must use stdin, not argv")
		return nil
	}
	t.Cleanup(func() {
		runPrivilegedWithStdin = originalStdin
		runPrivileged = originalCommand
	})

	result, err := NewHandler(hostreqkit.SafeguardManifest{Name: "graphical_session"}).Apply(hostreqkit.Host{OS: "darwin"}, status, hostreqkit.EnsureOptions{})
	if err != nil || result.ExecutionState != hostreqkit.ExecutionApplied {
		t.Fatalf("status=%+v err=%v", result, err)
	}
	if gotCommand != "/usr/sbin/sysadminctl" || gotInput != "not-a-real-password\n" {
		t.Fatalf("command=%q input=%q", gotCommand, gotInput)
	}
	if !reflect.DeepEqual(gotArgs, []string{"-autologin", "set", "-userName", "alice", "-password", "-"}) {
		t.Fatalf("args=%v", gotArgs)
	}
}

func TestApplyMacOSFailsClosedWhenPostconditionIsNotEnabled(t *testing.T) {
	originalFacts := collectFacts
	collectFacts = func() hostinventory.Snapshot {
		return hostinventory.Snapshot{ProbeStatuses: map[string]string{"macos_auto_login": "disabled"}}
	}
	t.Cleanup(func() { collectFacts = originalFacts })
	originalEnv, hadEnv := os.LookupEnv(macOSAutoLoginPasswordEnv)
	_ = os.Setenv(macOSAutoLoginPasswordEnv, "not-a-real-password")
	t.Cleanup(func() {
		if hadEnv {
			_ = os.Setenv(macOSAutoLoginPasswordEnv, originalEnv)
		} else {
			_ = os.Unsetenv(macOSAutoLoginPasswordEnv)
		}
	})
	original := runPrivilegedWithStdin
	runPrivilegedWithStdin = func(string, string, string, []string) ([]byte, error) { return nil, nil }
	t.Cleanup(func() { runPrivilegedWithStdin = original })

	result, err := NewHandler(hostreqkit.SafeguardManifest{Name: "graphical_session"}).Apply(hostreqkit.Host{OS: "darwin"}, hostreqkit.ItemStatus{
		Name: "graphical_session", Kind: hostreqspec.KindSafeguard, ExecutionState: hostreqkit.ExecutionPending,
		Config: map[string]any{"allow_enable_automatic_login": true, "desktop_user": "alice"},
	}, hostreqkit.EnsureOptions{})
	if err != nil || result.ExecutionState != hostreqkit.ExecutionFailed || result.Applied {
		t.Fatalf("status=%+v err=%v", result, err)
	}
	if !strings.Contains(strings.Join(result.Notes, " "), "reports automatic login") {
		t.Fatalf("notes=%v", result.Notes)
	}
}

func TestApplyMacOSDoesNotClaimDefaultsWriteEnabledAutomaticLogin(t *testing.T) {
	status := hostreqkit.ItemStatus{Name: "graphical_session", Kind: hostreqspec.KindSafeguard, ExecutionState: hostreqkit.ExecutionPending, Config: map[string]any{
		"allow_enable_automatic_login": true, "desktop_user": "alice",
	}, Evidence: map[string]any{"macos_auto_login_status": "disabled"}}
	originalEnv, hadEnv := os.LookupEnv(macOSAutoLoginPasswordEnv)
	_ = os.Unsetenv(macOSAutoLoginPasswordEnv)
	t.Cleanup(func() {
		if hadEnv {
			_ = os.Setenv(macOSAutoLoginPasswordEnv, originalEnv)
		}
	})

	called := false
	original := runPrivileged
	runPrivileged = func(string, string, []string, hostreqkit.EnsureOptions) error { called = true; return nil }
	t.Cleanup(func() { runPrivileged = original })

	result, err := NewHandler(hostreqkit.SafeguardManifest{Name: "graphical_session"}).Apply(hostreqkit.Host{OS: "darwin"}, status, hostreqkit.EnsureOptions{})
	if err != nil || called || result.ExecutionState != hostreqkit.ExecutionManualActionRequired {
		t.Fatalf("status=%+v err=%v called=%v", result, err, called)
	}
}

func TestApplyMacOSDoesNotMutateWhenAutomaticLoginStatusIsUnknown(t *testing.T) {
	status := hostreqkit.ItemStatus{Name: "graphical_session", Kind: hostreqspec.KindSafeguard, ExecutionState: hostreqkit.ExecutionPending, Config: map[string]any{
		"allow_enable_automatic_login": true, "desktop_user": "alice",
	}, Evidence: map[string]any{"macos_auto_login_status": "unknown"}}
	originalEnv, hadEnv := os.LookupEnv(macOSAutoLoginPasswordEnv)
	_ = os.Unsetenv(macOSAutoLoginPasswordEnv)
	t.Cleanup(func() {
		if hadEnv {
			_ = os.Setenv(macOSAutoLoginPasswordEnv, originalEnv)
		}
	})

	called := false
	original := runPrivileged
	runPrivileged = func(string, string, []string, hostreqkit.EnsureOptions) error { called = true; return nil }
	t.Cleanup(func() { runPrivileged = original })

	result, err := NewHandler(hostreqkit.SafeguardManifest{Name: "graphical_session"}).Apply(hostreqkit.Host{OS: "darwin"}, status, hostreqkit.EnsureOptions{})
	if err != nil || called || result.ExecutionState != hostreqkit.ExecutionManualActionRequired {
		t.Fatalf("status=%+v err=%v called=%v", result, err, called)
	}
}
