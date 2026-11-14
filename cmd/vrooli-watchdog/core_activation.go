package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	platform "github.com/vrooli/platform-go"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/agentscope"
	runtimeapp "github.com/vrooli/vrooli/internal/app/runtime"
	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/coreactivation"
	supervisor "github.com/vrooli/vrooli/internal/safeguards/runtime-supervisor"
)

// operatorCallerEvidence is replaced in tests.
var operatorCallerEvidence = agentscope.CallerSessionEvidence

// These fixed modes are separate from pressure reporting. A report-only run
// never executes root activation, enrollment, reconciliation or restoration.
func runCoreActivationCommand(args []string, stdout, stderr io.Writer) (bool, int) {
	if len(args) == 0 {
		return false, 0
	}
	mode := args[0]
	if mode != "--core-update-capability" && mode != "--core-update-execute" && mode != "--reconcile-core-updates" && mode != "--submit-core-update" && mode != "--submit-enrolled-core-update" && mode != "--retry-core-update" && mode != "--enroll-core-updates" && mode != "--disable-core-updates" {
		return false, 0
	}
	if mode == "--core-update-capability" && len(args) == 1 {
		homeDigest, err := buildinfo.RuntimeHomeSnapshotDigest()
		stateDigest := runtimeapp.NativePersistentStateContract()
		if err != nil || len(stateDigest) != 64 {
			_, _ = fmt.Fprintln(stderr, "root update capability is not stamped")
			return true, 1
		}
		err = json.NewEncoder(stdout).Encode(coreactivation.WorkerIdentity{Protocol: 1, BuildIdentity: buildVersion, RuntimeHomeContractSHA256: homeDigest, PersistentStateContractSHA256: stateDigest})
		if err != nil {
			return true, 1
		}
		return true, 0
	}
	if mode == "--core-update-capability" || (mode == "--core-update-execute" || mode == "--submit-core-update" || mode == "--retry-core-update" || mode == "--submit-enrolled-core-update") && len(args) != 3 || (mode == "--reconcile-core-updates" || mode == "--enroll-core-updates" || mode == "--disable-core-updates") && len(args) != 1 {
		_, _ = fmt.Fprintln(stderr, "invalid fixed root update invocation")
		return true, 2
	}
	if mode == "--enroll-core-updates" || mode == "--disable-core-updates" || mode == "--retry-core-update" {
		// Every caller of these operator decisions reaches this process, so the
		// agent-session backstop lives here as well as in the CLI verbs.
		if evidence := operatorCallerEvidence(); evidence != "" {
			_, _ = fmt.Fprintf(stderr, "%s is an operator decision and cannot run from a coding-agent session (%s); ask the operator to run it from their own terminal\n", mode, evidence)
			return true, 1
		}
	}
	if err := buildinfo.PinCompiledRuntimeHome(); err != nil {
		_, _ = fmt.Fprintln(stderr, "root update worker requires compiled runtime-home authority")
		return true, 1
	}
	home, err := config.HomeDir()
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "root update user home unavailable")
		return true, 1
	}
	state, err := buildinfo.CompiledRuntimeHomeEntryPath(home, repocontract.HomeKeyState)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "root update state authority unavailable")
		return true, 1
	}
	directory := filepath.Join(state, "core-activation")
	if mode == "--core-update-execute" && args[2] != directory {
		_, _ = fmt.Fprintln(stderr, "root update worker store differs from its compiled authority")
		return true, 1
	}
	store, err := coreactivation.NewStoreAt(home, directory)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "root update state binding refused")
		return true, 1
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if mode == "--enroll-core-updates" || mode == "--disable-core-updates" {
		ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
		defer cancel()
		enrollment, err := runtimeapp.ConfigureRootUpdates(ctx, home, mode == "--enroll-core-updates")
		if err != nil {
			_, _ = fmt.Fprintln(stderr, "root update enrollment was not changed:", err)
			return true, 1
		}
		if err := json.NewEncoder(stdout).Encode(enrollment); err != nil {
			return true, 1
		}
		return true, 0
	}
	if mode == "--submit-core-update" || mode == "--retry-core-update" || mode == "--submit-enrolled-core-update" {
		ctx, cancel := context.WithTimeout(ctx, 8*time.Minute)
		defer cancel()
		record, submitErr := runtimeapp.SubmitRootUpdate(ctx, home, runtimeapp.RootUpdateRequest{Source: args[2], Candidate: args[1], ExplicitRetry: mode == "--retry-core-update", RequireEnrollment: mode == "--submit-enrolled-core-update"})
		if record.SchemaVersion == 1 && record.ID != "" {
			record, err = runtimeapp.WaitRootUpdate(ctx, home, record.ID)
		} else {
			err = submitErr
		}
		if record.ID != "" {
			_ = json.NewEncoder(stdout).Encode(struct {
				ID           string               `json:"id"`
				Phase        coreactivation.Phase `json:"phase"`
				FailureClass string               `json:"failure_class"`
			}{record.ID, record.Phase, record.FailureClass})
		} else if err != nil {
			// Refused before admission: no operation exists. State the
			// coordinator's own reason so installers need not guess.
			_ = json.NewEncoder(stdout).Encode(struct {
				ID           string `json:"id"`
				Phase        string `json:"phase"`
				FailureClass string `json:"failure_class"`
				Reason       string `json:"reason"`
			}{"", "", "admission_refused", boundedRefusalReason(err.Error())})
		}
		if err != nil {
			_, _ = fmt.Fprintln(stderr, "root update did not verify:", err)
			return true, 1
		}
		return true, 0
	}
	engine := coreactivation.Engine{Store: store, Effects: supervisor.ActivationEffects{Store: store}}
	var record coreactivation.Record
	if mode == "--core-update-execute" {
		r, exists, readErr := store.Current()
		if readErr != nil || !exists || r.ID != args[1] {
			_, _ = fmt.Fprintln(stderr, "root update worker request is not recorded")
			return true, 1
		}
		ref, resolveErr := (coreactivation.NativeWorker{}).ResolveSelf(ctx, r)
		if resolveErr != nil {
			_, _ = fmt.Fprintln(stderr, "root update worker native identity refused")
			return true, 1
		}
		record, err = engine.RunWorker(ctx, r.ID, ref)
	} else {
		ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
		defer cancel()
		r, exists, readErr := store.Current()
		if readErr != nil {
			err = readErr
		} else if !exists {
			return true, 0
		} else {
			// Do not contend with a healthy bounded worker before its deadline.
			if !r.Terminal() && r.Fence != coreactivation.Revoked && time.Now().Before(time.Unix(0, r.DeadlineUnixNano)) {
				if r.Fence == coreactivation.Ready {
					return true, 0
				}
				if r.WorkerRef != nil {
					observation, inspectErr := platform.InspectOwnedOperation(ctx, *r.WorkerRef)
					if inspectErr != nil {
						_, _ = fmt.Fprintln(stderr, "root worker liveness is unresolved")
						return true, 1
					}
					if observation.RootAlive && observation.Populated && observation.State == platform.OperationRunning {
						return true, 0
					}
				}
			}
			record, _, err = engine.Recover(ctx)
		}
	}
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "root update operation remains unresolved:", err)
		return true, 1
	}
	if err := json.NewEncoder(stdout).Encode(struct {
		ID           string               `json:"id"`
		Phase        coreactivation.Phase `json:"phase"`
		FailureClass string               `json:"failure_class"`
	}{record.ID, record.Phase, record.FailureClass}); err != nil {
		return true, 1
	}
	return true, 0
}

// boundedRefusalReason keeps a refusal receipt within the installer's 8 KiB
// receipt bound, cutting at a line so explanation entries stay whole.
func boundedRefusalReason(reason string) string {
	const limit = 6 << 10
	if len(reason) <= limit {
		return reason
	}
	cut := strings.LastIndex(reason[:limit], "\n")
	if cut <= 0 {
		cut = limit
	}
	return reason[:cut] + "\n  - (further changes omitted; run `vrooli version --state-manifest` on both images to compare them in full)"
}
