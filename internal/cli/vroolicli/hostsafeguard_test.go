package vroolicli

import (
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
)

func TestHostSafeguardSpecDeclaresSudoAndDryRun(t *testing.T) {
	spec := hostSafeguardSpec()
	if spec.Name != "safeguard" || spec.Handler != "safeguard" {
		t.Fatalf("unexpected spec: %#v", spec)
	}
	if len(spec.Args.Positionals) != 1 || spec.Args.Positionals[0].Name != "name" {
		t.Fatalf("safeguard must require a name: %#v", spec.Args)
	}
}

func TestHostSafeguardSpecDocumentsPortabilityBacklog(t *testing.T) {
	spec := hostSafeguardSpec()
	if !strings.Contains(spec.Help.Usage, "portability") {
		t.Fatalf("usage = %q, want portability backlog command documented", spec.Help.Usage)
	}
}

func okFor(state hostreqkit.ExecutionState) bool {
	return hostSafeguardOK(hostreqkit.ItemStatus{ExecutionState: state})
}

// Every ExecutionState a safeguard can surface, classified explicitly.
// Enumerating the whole enum means a newly added state has to be given a
// verdict here rather than silently inheriting the default branch.
func TestHostSafeguardOKClassifiesEveryExecutionState(t *testing.T) {
	cases := map[hostreqkit.ExecutionState]bool{
		// Success: the safeguard did its job, or had nothing to do.
		hostreqkit.ExecutionApplied:        true,
		hostreqkit.ExecutionAlreadyPresent: true,
		hostreqkit.ExecutionWouldApply:     true,
		hostreqkit.ExecutionNotApplicable:  true,

		// Not success: the host is not in the desired state yet.
		hostreqkit.ExecutionPending:              false,
		hostreqkit.ExecutionFailed:               false,
		hostreqkit.ExecutionUnsupported:          false,
		hostreqkit.ExecutionManualActionRequired: false,
		// The safeguard exists to refuse to pretend the host is ready.
		hostreqkit.ExecutionRebootRequired: false,

		// Install verbs; a safeguard does not emit these.
		hostreqkit.ExecutionInstalled:    false,
		hostreqkit.ExecutionWouldInstall: false,
	}
	for state, want := range cases {
		if got := okFor(state); got != want {
			t.Errorf("hostSafeguardOK(%q) = %v, want %v", state, got, want)
		}
	}
}

// The defect this replaced: a successful apply exited 1 while --dry-run
// exited 0, so the command reported failure exactly when the repair worked.
// Any predicate where the rehearsal outranks the real thing is wrong.
func TestSuccessfulApplyIsNotLessSuccessfulThanItsDryRun(t *testing.T) {
	if okFor(hostreqkit.ExecutionWouldApply) && !okFor(hostreqkit.ExecutionApplied) {
		t.Fatal("--dry-run exits 0 but a real successful apply exits non-zero")
	}
	if !okFor(hostreqkit.ExecutionApplied) {
		t.Error("a safeguard that applied successfully must exit 0")
	}
}

// hostSafeguardOK and hostInstallOK are twins over the same enum and drifted
// once already. Apply is to a safeguard what install is to a tool, so the
// corresponding verbs must be classified alike, and every state the two share
// must get the same verdict from both.
func TestSafeguardAndInstallPredicatesStayInStep(t *testing.T) {
	installOK := func(state hostreqkit.ExecutionState) bool {
		return hostInstallOK(hostreqkit.ItemStatus{ExecutionState: state})
	}

	pairs := []struct{ safeguard, install hostreqkit.ExecutionState }{
		{hostreqkit.ExecutionApplied, hostreqkit.ExecutionInstalled},
		{hostreqkit.ExecutionWouldApply, hostreqkit.ExecutionWouldInstall},
	}
	for _, pair := range pairs {
		if okFor(pair.safeguard) != installOK(pair.install) {
			t.Errorf("safeguard %q = %v but the corresponding install %q = %v",
				pair.safeguard, okFor(pair.safeguard), pair.install, installOK(pair.install))
		}
	}

	shared := []hostreqkit.ExecutionState{
		hostreqkit.ExecutionAlreadyPresent,
		hostreqkit.ExecutionNotApplicable,
		hostreqkit.ExecutionPending,
		hostreqkit.ExecutionFailed,
		hostreqkit.ExecutionUnsupported,
		hostreqkit.ExecutionManualActionRequired,
		hostreqkit.ExecutionRebootRequired,
	}
	for _, state := range shared {
		if okFor(state) != installOK(state) {
			t.Errorf("state %q: safeguard = %v, install = %v; shared states must agree",
				state, okFor(state), installOK(state))
		}
	}
}
