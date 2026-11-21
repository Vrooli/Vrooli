package phases

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/storage"
	platform "github.com/vrooli/platform-go"
	routingpb "github.com/vrooli/vrooli/packages/proto/gen/go/dev-routing/v1/routing"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/dev-routing/v1/routing/routing_v1connect"
	"test-genie/internal/orchestrator/phases/isolation"
	"test-genie/internal/orchestrator/phases/seeds"
)

type seedPersistenceKey struct{}
type seedPersistence struct{ Path, RunID string }

const seedCheckpointMaxBytes = 4 << 20
const seedCheckpointErrorMaxBytes = 4 << 10

var errSeedCheckpointCapacity = errors.New("seed cleanup checkpoint exceeds its bounded authority capacity")

// WithSeedPersistence binds one pre-reserved private session journal. It is
// selected by the Seed Service, never by an external caller or seed script.
func WithSeedPersistence(ctx context.Context, path, runID string) context.Context {
	return context.WithValue(ctx, seedPersistenceKey{}, seedPersistence{path, runID})
}

// seedCheckpoint contains cleanup authority, so it only lives in the private
// owner journal. Public SeedEvidence deliberately has none of these locations.
type seedCheckpoint struct {
	Version          int                      `json:"version"`
	RunID            string                   `json:"run_id"`
	TargetURL        string                   `json:"target_url,omitempty"`
	RoutingAttempted bool                     `json:"routing_attempted"`
	Installed        bool                     `json:"installed"`
	Cleared          bool                     `json:"cleared"`
	CleanupComplete  bool                     `json:"cleanup_complete"`
	Retain           bool                     `json:"retain"`
	Allocations      []isolation.Allocation   `json:"allocations,omitempty"`
	Operation        *seeds.OperationRecord   `json:"operation,omitempty"`
	Stats            *routingpb.LeaseStats    `json:"stats,omitempty"`
	SetupError       string                   `json:"setup_error,omitempty"`
	CleanupError     string                   `json:"cleanup_error,omitempty"`
	HeartbeatError   string                   `json:"heartbeat_error,omitempty"`
	ClearError       string                   `json:"clear_error,omitempty"`
	SeedState        map[string]any           `json:"seed_state,omitempty"`
	Env              map[string]string        `json:"env,omitempty"`
	Resources        []isolation.ResourceInfo `json:"resources,omitempty"`
	mu               sync.Mutex
	path             string
}

func newSeedCheckpoint(ctx context.Context) *seedCheckpoint {
	p, _ := ctx.Value(seedPersistenceKey{}).(seedPersistence)
	if p.RunID == "" {
		p.RunID = uuid.NewString()
	}
	return &seedCheckpoint{Version: 1, RunID: p.RunID, path: p.Path, Cleared: true}
}

// InitializeSeedCheckpoint commits an empty, effect-free checkpoint before
// publishing the registry reservation. Missing journals after that point are
// corruption, never evidence that allocation did not happen.
func InitializeSeedCheckpoint(path, runID string) error {
	if path == "" {
		return nil
	}
	if uuid.Validate(runID) != nil || filepath.Base(path) != runID+".json" {
		return errors.New("invalid seed checkpoint reservation identity")
	}
	if _, err := os.Lstat(path); err == nil {
		return errors.New("seed checkpoint reservation already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return (&seedCheckpoint{Version: 1, RunID: runID, path: path, Cleared: true}).save()
}

// RetireSeedCheckpointReservation is restricted to a journaled checkpoint-only
// plan that never authorized allocation or dispatch. It refuses even a valid
// checkpoint containing additional state; this is not general orphan cleanup.
func RetireSeedCheckpointReservation(path, runID string) error {
	if uuid.Validate(runID) != nil || filepath.Base(path) != runID+".json" {
		return errors.New("invalid seed checkpoint reservation identity")
	}
	parent, err := os.Lstat(filepath.Dir(path))
	if err != nil || !parent.IsDir() || parent.Mode().Perm()&0077 != 0 {
		return errors.New("seed checkpoint reservation parent is not a private directory")
	}
	expected, err := json.Marshal(&seedCheckpoint{Version: 1, RunID: runID, Cleared: true})
	if err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if err == nil {
		if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
			return errors.New("seed checkpoint reservation is not a private regular file")
		}
		file, openErr := os.Open(path)
		if openErr != nil {
			return openErr
		}
		defer file.Close()
		if err := platform.VerifyLockFile(file, path); err != nil {
			return err
		}
		data, readErr := io.ReadAll(io.LimitReader(file, int64(len(expected)+1)))
		if readErr != nil {
			return readErr
		}
		if !bytes.Equal(data, expected) {
			return errors.New("seed checkpoint reservation differs from its exact effect-free plan")
		}
		if err := platform.VerifyLockFile(file, path); err != nil {
			return err
		}
		if err := os.Remove(path); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	// Absence without a successful directory sync is not a retirement ack.
	dir, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

func (s *PlaybooksSeedSession) BackingRetained() bool { return s.recovery != nil && s.recovery.Retain }

func (c *seedCheckpoint) save() error {
	if c.path == "" {
		return nil
	}
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	if len(data) > seedCheckpointMaxBytes {
		return errSeedCheckpointCapacity
	}
	if err = storage.WriteFileAtomic(c.path, data, 0600); err != nil {
		return fmt.Errorf("persist seed cleanup ownership: %w", err)
	}
	dir, err := os.Open(filepath.Dir(c.path))
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

func (s *PlaybooksSeedSession) checkpoint() error {
	if s.recovery == nil {
		return nil
	}
	c := s.recovery
	c.mu.Lock()
	defer c.mu.Unlock()
	c.RunID, c.Env, c.Resources, c.SeedState = s.RunID, s.Env, s.Resources, s.SeedState
	c.CleanupComplete = s.cleanupComplete
	c.SetupError = errorString(s.setupErr)
	c.CleanupError = errorString(s.cleanupErr)
	if state := s.routing.state; state != nil {
		state.mu.Lock()
		c.Installed, c.Cleared, c.Stats = state.installed, state.cleared, state.stats
		c.HeartbeatError, c.ClearError = errorString(state.heartbeatErr), errorString(state.clearErr)
		state.mu.Unlock()
	}
	return c.save()
}
func errorString(err error) string {
	if err == nil {
		return ""
	}
	value := err.Error()
	if len(value) > seedCheckpointErrorMaxBytes {
		const suffix = " [truncated]"
		return value[:seedCheckpointErrorMaxBytes-len(suffix)] + suffix
	}
	return value
}
func retainedError(text string) error {
	if text == "" {
		return nil
	}
	return errors.New(text)
}

// RecoverPlaybooksSeed reconstructs cleanup only. Interrupted validation is
// failed, never resumed or promoted to a healthy seed lease after owner loss.
func RecoverPlaybooksSeed(path string) (*PlaybooksSeedSession, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() > seedCheckpointMaxBytes {
		return nil, errors.New("seed checkpoint is not a bounded private file")
	}
	c := &seedCheckpoint{path: path}
	decoder := json.NewDecoder(io.LimitReader(file, seedCheckpointMaxBytes))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(c); err != nil {
		return nil, err
	}
	if err = decoder.Decode(new(any)); err != io.EOF {
		return nil, errors.New("seed checkpoint must contain exactly one complete JSON document")
	}
	if c.Version != 1 || uuid.Validate(c.RunID) != nil || filepath.Base(path) != c.RunID+".json" {
		return nil, errors.New("seed checkpoint identity is invalid")
	}
	for _, a := range c.Allocations {
		if a.RunID != c.RunID {
			return nil, errors.New("seed allocation belongs to another owner")
		}
		if a.Kind == "sqlite" && a.Root != filepath.Join(filepath.Dir(path), "backing", "test-genie-playbooks-"+c.RunID) {
			return nil, errors.New("seed sqlite allocation escapes its private owner root")
		}
		if a.Kind == "sqlite" && a.StagingRoot != "" && a.StagingRoot != filepath.Join(filepath.Dir(path), "backing", ".staging", c.RunID) {
			return nil, errors.New("seed sqlite staging escapes its private preparation namespace")
		}
	}
	if operation := c.Operation; operation != nil {
		attempt, err := uuid.Parse(operation.AttemptID)
		operationRoot := filepath.Join(filepath.Dir(path), "gates", c.RunID)
		if err != nil || attempt == uuid.Nil || attempt.String() != operation.AttemptID || operation.Gate == nil || operation.Gate.ReadyPath != filepath.Join(operationRoot, operation.AttemptID+".ready") {
			return nil, errors.New("seed native operation gate escapes its private owner identity")
		}
		for value, suffix := range map[string]string{operation.WorkerPath: ".worker", operation.InputPath: ".input.json", operation.GrantPath: ".grant.json", operation.LogPath: ".log"} {
			if value != filepath.Join(operationRoot, operation.AttemptID+suffix) {
				return nil, errors.New("seed worker file escapes its private operation identity")
			}
		}
		if operation.DispatchCommitted && (operation.Intent.ValidateIdentity() != nil || operation.Intent.Owner != "test-genie-seed" || operation.Intent.Kind != "" || operation.Intent.AttemptID != operation.AttemptID) {
			return nil, errors.New("seed native dispatch identity is invalid")
		}
		if operation.Ref != nil && (operation.Ref.ValidateIdentity() != nil || operation.Ref.Intent != operation.Intent || !operation.DispatchCommitted) {
			return nil, errors.New("seed native ref does not match dispatch")
		}
		if operation.Released && !operation.Quiescent {
			return nil, errors.New("seed native release has no quiescence evidence")
		}
	}
	state := &routedLeaseState{installed: c.Installed, cleared: c.Cleared, stats: c.Stats, lastAck: time.Now(), heartbeatErr: retainedError(c.HeartbeatError), clearErr: retainedError(c.ClearError)}
	routing := routedLease{leaseID: c.RunID, state: state}
	if c.RoutingAttempted {
		parsed, err := url.Parse(c.TargetURL)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil {
			return nil, errors.New("seed checkpoint target identity is invalid")
		}
		routing.client = routingconnect.NewRoutingServiceClient(&http.Client{Timeout: 15 * time.Second}, c.TargetURL)
	} else {
		state.cleared = true
	}
	result := &isolation.Result{RunID: c.RunID, Env: c.Env, Resources: c.Resources}
	result.Cleanup = func(ctx context.Context) error {
		if c.Retain {
			return nil
		}
		for i, a := range c.Allocations {
			if a.Retired {
				continue
			}
			if err := isolation.RetireAllocation(ctx, a); err != nil {
				return err
			}
			c.Allocations[i].Retired = true
			if err := c.save(); err != nil {
				return err
			}
		}
		return nil
	}
	session := newSeedSession(result, routing, nil)
	session.recovery = c
	session.SeedState = c.SeedState
	session.cleanupComplete = c.CleanupComplete
	session.cleanupErr = retainedError(c.CleanupError)
	session.setupErr = retainedError(c.SetupError)
	if !c.CleanupComplete {
		session.setupErr = errors.Join(session.setupErr, errors.New("seed owner was replaced; validation cannot resume"))
	}
	return session, nil
}
