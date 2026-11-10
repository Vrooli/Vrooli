package cliutil

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Every coding-agent session is visible while it lives: the launcher records
// an editor lease (who, where, which scope, which claims) and the runtime
// registry expires it only on proof of death. cli-core cannot import the
// registry (244 modules replace cli-core), so the recorder is a seam the
// launcher binaries register; without one a launch is invisible and says so.
// The lease never blocks a launch.

// EditorLeaseRecord is what the launcher knows about its session.
type EditorLeaseRecord struct {
	SessionID         string
	Harness           string
	Agent             string
	PID               int
	WorkingDir        string
	Scope             string
	ContainmentMethod string
	Claims            []string
}

// ClaimHolder names a live session whose claim overlaps a requested path.
type ClaimHolder struct {
	SessionID  string
	Agent      string
	PID        int
	WorkingDir string
	HeldPath   string
	Overlap    string
	Age        time.Duration
}

// EditorLeaseRecorder writes the session's lease to the runtime registry.
// Heartbeat renews it on the spawn branch; the exec branch relies on the
// registry's proof-of-death expiry because the launcher stops existing.
type EditorLeaseRecorder interface {
	Create(ctx context.Context, record EditorLeaseRecord) error
	Heartbeat(ctx context.Context, sessionID string) error
	Stop(ctx context.Context, sessionID, reason string) error
	// Overlaps lists live holders whose claims overlap the requested paths.
	Overlaps(ctx context.Context, paths []string) ([]ClaimHolder, error)
}

// ErrEditorLeaseBindingUnavailable permits legacy visibility when this platform
// cannot observe an exact process incarnation. It never permits a proof.
var ErrEditorLeaseBindingUnavailable = errors.New("editor lease process incarnation unavailable")

// EditorLeaseBindingRecorder is the optional incarnation-fenced recording seam.
// Older recorders remain usable, but never supply containment provenance.
type EditorLeaseBindingRecorder interface {
	BeginBinding(context.Context, EditorLeaseRecord) (string, error)
	RecordContainment(context.Context, string, string, EditorLeaseContainment) error
	HeartbeatBinding(context.Context, string, string) error
	StopBinding(context.Context, string, string, string) error
}

// EditorLeaseContainment is an observation of one native attempt, not its plan.
// Successful descendant containment requires Session.BeforeExec and its exact
// process observation. Missing proof is unavailable, never inferred success.
type EditorLeaseContainment struct {
	Session     ContainedSession
	Failure     string
	Unavailable bool
}

// DefaultEditorLeaseRecorder is the registered recorder; nil means invisible.
var DefaultEditorLeaseRecorder EditorLeaseRecorder

// RegisterEditorLeaseRecorder installs the registry implementation.
func RegisterEditorLeaseRecorder(recorder EditorLeaseRecorder) { DefaultEditorLeaseRecorder = recorder }

// EditorLeaseHeartbeat is the spawn-branch renewal cadence; the registry TTL
// is three times it so one missed beat never looks like death.
const EditorLeaseHeartbeat = 20 * time.Second

// EditorLeaseTTL is the deadline the launcher asks for.
const EditorLeaseTTL = 3 * EditorLeaseHeartbeat

// recordEditorLease writes the lease and returns the stop function. It never
// fails the launch: an unavailable registry is logged and the session runs
// unrecorded, the same fail-open the attribution path uses.
func recordEditorLease(ctx context.Context, record EditorLeaseRecord, heartbeat bool) (stop func(reason string), recorded bool) {
	stop, recorded, _, _ = recordEditorLeaseBinding(ctx, record, heartbeat)
	return
}

func recordEditorLeaseBinding(ctx context.Context, record EditorLeaseRecord, heartbeat bool) (stop func(string), recorded bool, observe func(EditorLeaseContainment), fenced bool) {
	recorder := DefaultEditorLeaseRecorder
	noop := func(EditorLeaseContainment) {}
	if recorder == nil || strings.TrimSpace(record.SessionID) == "" {
		return func(string) {}, false, noop, false
	}
	bindingRecorder, bound := recorder.(EditorLeaseBindingRecorder)
	bindingID := ""
	var err error
	if bound {
		bindingID, err = bindingRecorder.BeginBinding(ctx, record)
		if errors.Is(err, ErrEditorLeaseBindingUnavailable) {
			bound = false
			err = recorder.Create(ctx, record)
		}
		if bound && err == nil && bindingID == "" {
			err = fmt.Errorf("binding recorder returned no binding ID")
		}
	} else {
		err = recorder.Create(ctx, record)
	}
	if err != nil {
		log.Printf("agent launch editor lease not recorded session=%s: %v", record.SessionID, err)
		return func(string) {}, false, noop, false
	}
	observe = noop
	if bound {
		observe = func(evidence EditorLeaseContainment) {
			if err := bindingRecorder.RecordContainment(context.Background(), record.SessionID, bindingID, evidence); err != nil {
				log.Printf("agent launch containment not recorded session=%s: %v", record.SessionID, err)
			}
		}
	}
	done := make(chan struct{})
	if heartbeat {
		go func() {
			ticker := time.NewTicker(EditorLeaseHeartbeat)
			defer ticker.Stop()
			for {
				select {
				case <-done:
					return
				case <-ticker.C:
					var err error
					if bound {
						err = bindingRecorder.HeartbeatBinding(context.Background(), record.SessionID, bindingID)
					} else {
						err = recorder.Heartbeat(context.Background(), record.SessionID)
					}
					if err != nil {
						log.Printf("agent launch editor lease heartbeat failed session=%s: %v", record.SessionID, err)
					}
				}
			}
		}()
	}
	var once sync.Once
	return func(reason string) {
		once.Do(func() {
			close(done)
			var err error
			if bound {
				err = bindingRecorder.StopBinding(context.Background(), record.SessionID, bindingID, reason)
			} else {
				err = recorder.Stop(context.Background(), record.SessionID, reason)
			}
			if err != nil {
				log.Printf("agent launch editor lease not stopped session=%s: %v", record.SessionID, err)
			}
		})
	}, true, observe, bound
}

// reportClaimOverlaps prints the holders of overlapping claims and returns
// them. Claims are advisory in this release: the launch continues.
func reportClaimOverlaps(ctx context.Context, request AgentLaunchRequest) []ClaimHolder {
	recorder := DefaultEditorLeaseRecorder
	if recorder == nil || len(request.Claims) == 0 {
		return nil
	}
	holders, err := recorder.Overlaps(ctx, request.Claims)
	if err != nil {
		log.Printf("agent launch claim check unavailable: %v", err)
		return nil
	}
	out := request.Stderr
	if out == nil {
		out = os.Stderr
	}
	for _, holder := range holders {
		_, _ = out.Write([]byte("claim overlap: " + holder.HeldPath + " is held by " + holder.Agent + " session " + holder.SessionID + " (pid " + strconv.Itoa(holder.PID) + ", " + holder.Age.Round(time.Second).String() + " old, in " + holder.WorkingDir + "); continuing, claims are advisory\n"))
	}
	return holders
}

// AgentSessionBindingProofVersion versions the control-plane proof shared with
// Agent Manager. A proof describes a verified host binding, not a credential.
const AgentSessionBindingProofVersion = 2

// AgentSessionBindingProof distinguishes the attached run's immutable harness
// anchor from the actual host lease key. They can differ for existing launches.
type AgentSessionBindingProof struct {
	ProofVersion        int            `json:"proof_version"`
	RunID               string         `json:"run_id"`
	HarnessSessionID    string         `json:"harness_session_id"`
	LeaseSessionID      string         `json:"lease_session_id"`
	WebConsoleSessionID string         `json:"web_console_session_id"`
	HarnessKind         string         `json:"harness_kind"`
	OwnerKind           string         `json:"owner_kind"`
	Binding             ProcessBinding `json:"binding"`
}
