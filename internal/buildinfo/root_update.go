package buildinfo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/coreactivation"
	"github.com/vrooli/vrooli/internal/hostreqkit"
)

// EnrolledRootUpdate applies only to the canonical installed root executable,
// including an equivalent physical parent alias. Other CLI installs retain
// their existing atomic path. Missing/corrupt enrolled state is never disabled.
func EnrolledRootUpdate(destination string) (bool, error) {
	if runtime.GOOS != "linux" {
		return false, nil
	}
	home, canonical, err := canonicalRootTarget(destination)
	if err != nil || !canonical {
		return false, err
	}
	store, err := coreactivation.NewStore(home)
	if err != nil {
		return false, err
	}
	policy, exists, err := store.Enrollment()
	if err != nil {
		return false, err
	}
	return exists && policy.Enabled, nil
}

// RetainSuppressedRootBuild keeps a failed source build from causing repeated
// automatic compilation/activation, while leaving the prior CLI usable.
func RetainSuppressedRootBuild(destination, build string) (bool, error) {
	enrolled, err := EnrolledRootUpdate(destination)
	if err != nil || !enrolled {
		return false, err
	}
	home, _, err := canonicalRootTarget(destination)
	if err != nil {
		return false, err
	}
	store, err := coreactivation.NewStore(home)
	if err != nil {
		return false, err
	}
	r, exists, err := store.Current()
	if err != nil || !exists {
		return false, err
	}
	for _, failed := range r.FailedBuilds {
		if failed == build {
			_, _ = fmt.Fprintf(os.Stderr, "Root update build %s is suppressed after a failed trial; retaining the prior CLI.\n", build)
			return true, nil
		}
	}
	return false, nil
}

type rootUpdateOutput struct {
	data     bytes.Buffer
	overflow bool
}

func (o *rootUpdateOutput) Write(p []byte) (int, error) {
	n := len(p)
	remaining := 8192 - o.data.Len()
	if len(p) > remaining {
		o.overflow = true
		p = p[:remaining]
	}
	_, _ = o.data.Write(p)
	return n, nil
}

// rootUpdateReceipt is the coordinator's bounded answer. A request refused
// before admission has no ID; newer coordinators then state their reason.
type rootUpdateReceipt struct {
	ID           string               `json:"id"`
	Phase        coreactivation.Phase `json:"phase"`
	FailureClass string               `json:"failure_class"`
	Reason       string               `json:"reason,omitempty"`
}

// RequestRootUpdate hands off to the independent installed coordinator. Its
// caller must release the root install lock first: admission uses operation ->
// install order and requalifies the frozen prior. The native worker survives a
// cancelled caller; this function neither resubmits nor reports acceptance as
// verification. A request that never reached admission is explained from the
// coordinator's stated reason and a local comparison of the two images.
func RequestRootUpdate(ctx context.Context, candidate, source string) error {
	if ctx == nil {
		return fmt.Errorf("root update context is required")
	}
	home, err := homeDirFn()
	if err != nil {
		return err
	}
	worker, err := coreactivation.InstalledWatchdogPath(home)
	if err != nil {
		return err
	}
	candidate, err = filepath.Abs(candidate)
	if err != nil {
		return err
	}
	// The coordinator waits up to eight minutes for its admitted worker; give
	// it longer so this caller never kills it between admission and receipt.
	ctx, cancel := context.WithTimeout(ctx, 12*time.Minute)
	defer cancel()
	name, args := hostreqkit.InvokingUserCommand(worker, "--submit-enrolled-core-update", candidate, source)
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.WaitDelay = time.Second
	var output rootUpdateOutput
	var diagnostics rootUpdateTail
	cmd.Stdout = &output
	cmd.Stderr = &diagnostics
	runErr := cmd.Run()
	if output.overflow {
		return fmt.Errorf("root update receipt exceeded its bound; inspect `vrooli runtime update status --json`%s", diagnostics.suffix())
	}
	if len(bytes.TrimSpace(output.data.Bytes())) == 0 {
		return diagnoseRootRefusalFn(home, candidate, source, "", diagnostics.suffix())
	}
	var receipt rootUpdateReceipt
	decoder := json.NewDecoder(bytes.NewReader(output.data.Bytes()))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&receipt); err != nil {
		return fmt.Errorf("root update outcome is unresolved: the coordinator receipt is malformed; inspect `vrooli runtime update status --json`%s", diagnostics.suffix())
	}
	if decoder.Decode(new(any)) != io.EOF {
		return fmt.Errorf("root update returned an ambiguous receipt; inspect `vrooli runtime update status --json`%s", diagnostics.suffix())
	}
	if receipt.ID == "" {
		reason := strings.TrimSpace(receipt.Reason)
		if reason == "" {
			reason = "the installed coordinator refused the request before admission"
		}
		return diagnoseRootRefusalFn(home, candidate, source, reason, "")
	}
	if runErr != nil || receipt.Phase != coreactivation.Verified {
		return fmt.Errorf("root update %s did not verify (phase=%s, failure=%s); inspect `vrooli runtime update status --json`%s", receipt.ID, receipt.Phase, receipt.FailureClass, diagnostics.suffix())
	}
	return nil
}

// rootUpdateTail keeps the last 2 KiB of the coordinator's diagnostics, its
// own error text, for reports that have no structured reason.
type rootUpdateTail struct{ data []byte }

func (t *rootUpdateTail) Write(p []byte) (int, error) {
	t.data = append(t.data, p...)
	if len(t.data) > 2048 {
		t.data = append([]byte(nil), t.data[len(t.data)-2048:]...)
	}
	return len(p), nil
}

func (t *rootUpdateTail) suffix() string {
	text := strings.TrimSpace(string(t.data))
	if text == "" {
		return ""
	}
	return "\ncoordinator: " + text
}

var diagnoseRootRefusalFn = diagnoseRootRefusal

// rootRebaselineRemedy is addressed to whoever ran the install, which is
// often an agent: crossing a stored-format change is the operator's decision.
func rootRebaselineRemedy(source string) string {
	return "The serving CLI cannot be relied on to read state this candidate writes, so it cannot activate with automatic rollback.\n" +
		"Operator decision required: the operator reviews the changes above and runs `make rebaseline` in " + source + " from their own terminal. " +
		"Agents must not run it; report this refusal to the operator."
}

// diagnoseRootRefusal explains a request that produced no verified
// operation. It compares the two images itself, so a state-contract refusal
// names its owners even when an older coordinator states no reason. Without
// a stated reason and without a contract difference the outcome is reported
// as unresolved, never as a refusal: the coordinator may have been
// interrupted after admission.
func diagnoseRootRefusal(home, candidate, source, reason, diagnostics string) error {
	probe, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	unresolved := fmt.Errorf("root update outcome is unresolved: the installed coordinator returned no receipt; inspect `vrooli runtime update status --json`%s", diagnostics)
	refused := func() error {
		if reason == "" {
			return unresolved
		}
		return fmt.Errorf("root update refused: %s", reason)
	}
	bin, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin)
	if err != nil {
		return refused()
	}
	serving := filepath.Join(bin, "vrooli")
	servingIdentity, servingErr := coreactivation.ProbeRootIdentity(probe, serving)
	candidateIdentity, candidateErr := coreactivation.ProbeRootIdentity(probe, candidate)
	if servingErr != nil || candidateErr != nil {
		return refused()
	}
	qualification := coreactivation.QualifyRollbackIdentities(servingIdentity, candidateIdentity)
	if qualification == nil {
		return refused()
	}
	if !errors.Is(qualification, coreactivation.ErrStateContractChanged) {
		// Admission cannot accept this pair, so the coordinator refused it.
		return fmt.Errorf("root update refused: %w", coreactivation.ExplainRollbackRefusal(probe, qualification, serving, candidate))
	}
	changes := reason
	if !strings.Contains(reason, "\n  - ") {
		changes = coreactivation.ExplainRollbackRefusal(probe, qualification, serving, candidate).Error()
	}
	return fmt.Errorf("root update refused: %s\n%s", changes, rootRebaselineRemedy(source))
}
