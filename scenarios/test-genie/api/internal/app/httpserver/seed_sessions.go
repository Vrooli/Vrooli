package httpserver

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"test-genie/internal/orchestrator/phases"
	"test-genie/internal/orchestrator/workspace"
)

const seedSessionLimit = 1024
const activeSeedSessionLimit = 4
const seedSessionRetention = time.Hour
const maxSeedSourcePathBytes = 4096

// One registry owns both the legacy CLI and generated inter-scenario surface.
// A reservation precedes the first effect. Retired outcomes remain available
// for lost-response retries; active ownership is never evicted for capacity.
type seedSession struct {
	Scenario         string
	Created          time.Time
	Session          *phases.PlaybooksSeedSession
	requestID        string
	scenarioPath     string
	ready            chan struct{}
	mu               sync.Mutex
	prepareErr       error
	retiredAt        time.Time
	runID            string
	consumerDeadline time.Time
	consumerLost     bool
	cleaning         bool
	pruning          bool
	retainedBacking  bool
}

func (s *Server) applySeedSession(ctx context.Context, name, scenarioPath, requestID string, retain bool) (string, *seedSession, error) {
	if s.scenarios == nil {
		return "", nil, connect.NewError(connect.CodeUnavailable, errors.New("scenario directory service unavailable"))
	}
	if len(name) > 128 || len(requestID) > 200 || len(scenarioPath) > maxSeedSourcePathBytes || strings.ContainsAny(name+requestID+scenarioPath, "\x00\r\n") {
		return "", nil, connect.NewError(connect.CodeInvalidArgument, errors.New("seed identity and checkout path must satisfy bounded input limits"))
	}
	name = strings.TrimSpace(name)
	if !validSeedScenario(name) {
		return "", nil, connect.NewError(connect.CodeInvalidArgument, errors.New("seed target must be a local live scenario slug"))
	}
	root := strings.TrimSpace(s.scenarios.ScenarioRoot())
	if root == "" {
		return "", nil, connect.NewError(connect.CodeUnavailable, errors.New("scenarios root is not configured"))
	}
	if scenarioPath == "" {
		scenarioPath = filepath.Join(root, name)
	}
	var err error
	scenarioPath, err = filepath.Abs(scenarioPath)
	if err != nil {
		return "", nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if len(scenarioPath) > maxSeedSourcePathBytes {
		return "", nil, connect.NewError(connect.CodeInvalidArgument, errors.New("canonical seed checkout path exceeds its input bound"))
	}
	if requestID == "" {
		requestID = uuid.NewString()
	}

	s.seedSessionsMu.Lock()
	if s.seedOwnerErr != nil || (s.seedStore != nil && (!s.seedStore.validated || s.seedStore.pending != nil)) {
		s.seedSessionsMu.Unlock()
		return "", nil, connect.NewError(connect.CodeUnavailable, errors.New("seed ownership journal is unavailable"))
	}
	if s.seedSessions == nil {
		s.seedSessions = make(map[string]*seedSession)
	}
	if s.seedSessionsByScenario == nil {
		s.seedSessionsByScenario = make(map[string]string)
	}
	for token, prior := range s.seedSessions {
		prior.mu.Lock()
		expired := seedOutcomeExpired(prior.retiredAt, time.Now())
		same := prior.Scenario == name && prior.requestID == requestID
		samePath := prior.scenarioPath == scenarioPath
		ready := prior.ready
		prior.mu.Unlock()
		if expired {
			continue
		}
		if same {
			s.seedSessionsMu.Unlock()
			if !samePath {
				return "", nil, connect.NewError(connect.CodeAlreadyExists, errors.New("seed request is bound to another source checkout"))
			}
			if ready != nil {
				select {
				case <-ready:
				case <-ctx.Done():
					return "", nil, ctx.Err()
				}
			}
			prior.mu.Lock()
			err := prior.prepareErr
			prior.mu.Unlock()
			if err == nil {
				_, err = s.observeSeedSession(name, token)
			}
			return token, prior, err
		}
	}
	if s.seedSessionsByScenario[name] != "" {
		s.seedSessionsMu.Unlock()
		return "", nil, connect.NewError(connect.CodeAlreadyExists, errors.New("seed session already active for scenario"))
	}
	if len(s.seedSessionsByScenario) >= activeSeedSessionLimit {
		s.seedSessionsMu.Unlock()
		return "", nil, connect.NewError(connect.CodeResourceExhausted, errors.New("active seed session capacity is full"))
	}
	if len(s.seedSessions) >= seedSessionLimit {
		s.seedSessionsMu.Unlock()
		return "", nil, connect.NewError(connect.CodeResourceExhausted, errors.New("seed session registry is full"))
	}
	token := uuid.NewString()
	record := &seedSession{Scenario: name, Created: time.Now().UTC(), requestID: requestID, scenarioPath: scenarioPath, runID: uuid.NewString(), ready: make(chan struct{})}
	if err := s.beginSeedReservationLocked(token, record); err != nil {
		s.seedSessionsMu.Unlock()
		if errors.Is(err, errSeedRegistryCapacity) {
			return "", nil, err
		}
		return "", nil, connect.NewError(connect.CodeUnavailable, errors.New("seed cleanup checkpoint reservation is unresolved"))
	}
	pending := (*seedPendingReservation)(nil)
	if s.seedStore != nil {
		pending = s.seedStore.pending
		s.seedStore.pending = nil
	}
	s.seedSessions[token] = record
	s.seedSessionsByScenario[name] = token
	if err := s.persistSeedSessionsWithinBoundLocked(seedRegistryAdmissionBytes); err != nil {
		delete(s.seedSessions, token)
		delete(s.seedSessionsByScenario, name)
		if s.seedStore != nil {
			s.seedStore.pending = pending
		}
		_ = s.failSeedReservationLocked(err)
		s.seedSessionsMu.Unlock()
		return "", nil, connect.NewError(connect.CodeUnavailable, errors.New("seed reservation transfer could not be verified"))
	}
	s.seedSessionsMu.Unlock()

	// The request owns setup. A lost response can reattach to this reservation;
	// no caller is authorized to blindly start a second setup for the same key.
	var prepared *phases.PlaybooksSeedSession
	ws, err := workspace.New(root, name)
	if err == nil {
		var apiURL string
		apiURL, err = resolveSeedTargetAPI(ctx, name, scenarioPath)
		if err != nil {
			err = connect.NewError(connect.CodeUnavailable, err)
		}
		if err == nil {
			ws.SetRuntimeURLs("", apiURL)
			env := ws.Environment()
			env.ScenarioDir = scenarioPath
			seedCtx := phases.WithSeedPersistence(ctx, s.seedCheckpointPath(record.runID), record.runID)
			prepared, err = applyPlaybooksSeed(seedCtx, env, nil, retain)
			if err == nil && prepared == nil {
				err = errors.New("seed owner returned no session")
			}
		}
	}
	record.mu.Lock()
	record.Session = prepared
	if prepared != nil {
		prepared.CleanupRef = token
	}
	if err != nil {
		record.prepareErr = err
		record.consumerLost = true
	} else {
		record.consumerDeadline = time.Now().Add(seedConsumerTTL)
	}
	close(record.ready)
	record.mu.Unlock()
	s.seedSessionsMu.Lock()
	record.mu.Lock()
	if err != nil && (record.Session == nil || record.Session.CleanupComplete()) {
		record.retiredAt = time.Now().UTC()
	}
	record.mu.Unlock()
	if persistErr := s.persistSeedSessionsLocked(); persistErr != nil {
		if !errors.Is(persistErr, errSeedRegistryCapacity) {
			s.recordSeedRegistryFailureLocked(persistErr)
		}
		err = errors.Join(err, persistErr)
	} else {
		s.releaseDurablyRetiredSeedSlotsLocked()
	}
	s.seedSessionsMu.Unlock()
	if err != nil {
		return token, record, fmt.Errorf("prepare seed session: %w", err)
	}
	return token, record, nil
}

func (s *Server) findSeedSession(name, token string) (*seedSession, error) {
	if strings.TrimSpace(name) == "" || strings.TrimSpace(token) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario and session token are required"))
	}
	s.seedSessionsMu.Lock()
	if s.seedStore != nil && !s.seedStore.validated {
		s.seedSessionsMu.Unlock()
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("seed ownership journal is unavailable"))
	}
	record := s.seedSessions[token]
	s.seedSessionsMu.Unlock()
	if record == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("seed session is not retained by this owner"))
	}
	if record.Scenario != name {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("seed session does not match scenario"))
	}
	if record.ready != nil {
		select {
		case <-record.ready:
		default:
			return nil, connect.NewError(connect.CodeUnavailable, errors.New("seed session is preparing"))
		}
	}
	record.mu.Lock()
	defer record.mu.Unlock()
	if record.Session == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("seed session preparation failed"))
	}
	return record, nil
}

func (s *Server) clearSeedSession(ctx context.Context, name, token string) (*seedSession, error) {
	record, err := s.findSeedSession(name, token)
	if err != nil {
		return nil, err
	}
	record.mu.Lock()
	err = record.Session.Cleanup(ctx)
	retired := record.Session.CleanupComplete()
	record.mu.Unlock()
	s.seedSessionsMu.Lock()
	record.mu.Lock()
	if retired && record.retiredAt.IsZero() {
		record.retiredAt = time.Now().UTC()
	}
	if !retired {
		record.consumerLost = true
	}
	record.mu.Unlock()
	persistErr := s.persistSeedSessionsLocked()
	if persistErr != nil {
		if !errors.Is(persistErr, errSeedRegistryCapacity) {
			s.recordSeedRegistryFailureLocked(persistErr)
		}
		err = errors.Join(err, persistErr)
	} else {
		s.releaseDurablyRetiredSeedSlotsLocked()
	}
	s.seedSessionsMu.Unlock()
	return record, err
}
