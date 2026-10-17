package cliutil

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"
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
	recorder := DefaultEditorLeaseRecorder
	if recorder == nil || strings.TrimSpace(record.SessionID) == "" {
		return func(string) {}, false
	}
	if err := recorder.Create(ctx, record); err != nil {
		log.Printf("agent launch editor lease not recorded session=%s: %v", record.SessionID, err)
		return func(string) {}, false
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
					if err := recorder.Heartbeat(context.Background(), record.SessionID); err != nil {
						log.Printf("agent launch editor lease heartbeat failed session=%s: %v", record.SessionID, err)
					}
				}
			}
		}()
	}
	var once bool
	return func(reason string) {
		if once {
			return
		}
		once = true
		close(done)
		if err := recorder.Stop(context.Background(), record.SessionID, reason); err != nil {
			log.Printf("agent launch editor lease not stopped session=%s: %v", record.SessionID, err)
		}
	}, true
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
