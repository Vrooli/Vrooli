package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/storage"
	platform "github.com/vrooli/platform-go"
	"test-genie/internal/orchestrator/phases"
)

const seedConsumerTTL = 60 * time.Second
const seedSweepInterval = 5 * time.Second
const seedRegistryMaxBytes = 8 << 20

// Reserve space for four in-flight bounded error excerpts (including JSON's
// worst sixfold escaping), cleanup timestamps and 1024 pruning markers.
const seedRegistryAdmissionBytes = 7 << 20
const seedPrivateErrorMaxBytes = 4 << 10

func seedOutcomeExpired(retiredAt, now time.Time) bool {
	return !retiredAt.IsZero() && now.Sub(retiredAt) > seedSessionRetention
}

func validSeedScenario(name string) bool {
	return name != "" && name == strings.TrimSpace(name) && filepath.Base(name) == name && !strings.ContainsAny(name, "/\\:@\x00\r\n") && name != "." && name != ".."
}

var errSeedRegistryCapacity = errors.New("seed ownership journal byte capacity is exhausted")

// Successful registry publication resolves only this provenance, never an
// unrelated checkpoint or cleanup refusal stored in seedOwnerErr.
type seedRegistryPublicationError struct{ error }

func (e *seedRegistryPublicationError) Unwrap() error { return e.error }

type seedPendingReservationError struct {
	runID string
	error
}

func (e *seedPendingReservationError) Unwrap() error { return e.error }

const seedPendingCheckpointState = "checkpoint-only-v1"

// No target, allocation or native dispatch may precede transfer of this exact
// effect-free reservation into Sessions. One pending descriptor is sufficient:
// the registry mutex serializes reservation and refusal blocks later admission.
type seedPendingReservation struct {
	State  string         `json:"state"`
	Record seedRecordDisk `json:"record"`
}

var reserveSeedCheckpoint = phases.InitializeSeedCheckpoint
var retireSeedCheckpointReservation = phases.RetireSeedCheckpointReservation

type seedSessionStore struct {
	dir       string
	root      *os.Root
	release   func()
	validated bool
	pending   *seedPendingReservation
}
type seedRegistryDisk struct {
	Version            int                     `json:"version"`
	Sessions           []seedRecordDisk        `json:"sessions"`
	PendingReservation *seedPendingReservation `json:"pending_reservation,omitempty"`
}
type seedRecordDisk struct {
	Token            string    `json:"token"`
	Scenario         string    `json:"scenario"`
	RequestID        string    `json:"request_id"`
	ScenarioPath     string    `json:"scenario_path"`
	RunID            string    `json:"run_id"`
	Created          time.Time `json:"created"`
	RetiredAt        time.Time `json:"retired_at,omitempty"`
	ConsumerDeadline time.Time `json:"consumer_deadline,omitempty"`
	ConsumerLost     bool      `json:"consumer_lost,omitempty"`
	PrepareError     string    `json:"prepare_error,omitempty"`
	Pruning          bool      `json:"pruning,omitempty"`
	Retain           bool      `json:"retain,omitempty"`
}

func defaultSeedOwnerDir() (string, error) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{})
	if err != nil {
		return "", err
	}
	namespace, err := storage.ScenarioNamespace("test-genie")
	if err != nil {
		return "", err
	}
	return resolver.Path(storage.Options{ScenarioID: namespace}, storage.ClassState, "seed-sessions")
}

// initializeSeedOwner acquires one process-lifetime journal lock before loading
// any cleanup authority. A replacement only retires sessions; it never restores
// an interrupted validation as healthy or dispatches its seed script again.
func (s *Server) initializeSeedOwner(dir string) error {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	info, err := os.Lstat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode().Perm()&0077 != 0 {
		return errors.New("seed owner directory must be private and must not be a symlink")
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return err
	}
	lock, err := platform.OpenLockFile(filepath.Join(dir, "owner.lock"), 0600)
	if err != nil {
		root.Close()
		return err
	}
	lockCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	unlock, err := platform.LockFileContext(lockCtx, lock)
	if err != nil {
		root.Close()
		lock.Close()
		return fmt.Errorf("seed owner is already held or unavailable: %w", err)
	}
	if err = platform.VerifyLockFile(lock, filepath.Join(dir, "owner.lock")); err != nil {
		unlock()
		lock.Close()
		root.Close()
		return err
	}
	store := &seedSessionStore{dir: dir, root: root, release: func() { unlock(); lock.Close(); root.Close() }}
	s.seedStore = store
	if s.seedSessions == nil {
		s.seedSessions = map[string]*seedSession{}
	}
	if s.seedSessionsByScenario == nil {
		s.seedSessionsByScenario = map[string]string{}
	}
	file, err := root.Open("registry.json")
	if errors.Is(err, os.ErrNotExist) {
		entries, readErr := os.ReadDir(dir)
		if readErr != nil {
			return readErr
		}
		for _, entry := range entries {
			if entry.Name() != "owner.lock" {
				return errors.New("seed registry is missing while retained owner state exists")
			}
		}
		// Persist the empty owner before any reservation or effect. Absence is
		// accepted only for this effect-free initial namespace, never as proof
		// that a retained allocation or native process does not exist.
		return s.finishSeedOwnerInitialization()
	}
	if err != nil {
		return err
	}
	defer file.Close()
	info, err = file.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() > seedRegistryMaxBytes {
		return errors.New("seed registry is not bounded private state")
	}
	var disk seedRegistryDisk
	decoder := json.NewDecoder(io.LimitReader(file, seedRegistryMaxBytes))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&disk); err != nil {
		return err
	}
	if err = decoder.Decode(new(any)); err != io.EOF {
		return errors.New("seed registry must contain exactly one complete JSON document")
	}
	// Validate the whole document before reconstructing any cleanup capability
	// or exposing pending state. Bearer uniqueness alone does not bind a request.
	if err := validateSeedRegistryIdentity(disk, time.Now()); err != nil {
		return err
	}
	store.pending = disk.PendingReservation
	for _, row := range disk.Sessions {
		if s.seedSessions[row.Token] != nil {
			return errors.New("duplicate seed session token")
		}
		record := &seedSession{Scenario: row.Scenario, Created: row.Created, requestID: row.RequestID, scenarioPath: row.ScenarioPath, runID: row.RunID, ready: make(chan struct{}), retiredAt: row.RetiredAt, consumerDeadline: row.ConsumerDeadline, consumerLost: row.ConsumerLost}
		record.pruning, record.retainedBacking = row.Pruning, row.Retain
		record.prepareErr = seedRetainedError(row.PrepareError)
		if row.Pruning {
			if row.RetiredAt.IsZero() {
				return errors.New("active seed record cannot prune its ownership checkpoint")
			}
			close(record.ready)
			s.seedSessions[row.Token] = record
			continue
		}
		session, err := phases.RecoverPlaybooksSeed(s.seedCheckpointPath(record.runID))
		if err != nil {
			return fmt.Errorf("recover retained seed owner: %w", err)
		}
		record.Session = session
		if record.retiredAt.IsZero() {
			record.consumerLost = true
			record.prepareErr = errors.Join(record.prepareErr, errors.New("seed owner was replaced; validation cannot resume"))
			if session == nil {
				record.retiredAt = time.Now().UTC()
			} else {
				if s.seedSessionsByScenario[row.Scenario] != "" {
					return errors.New("duplicate active seed scenario owner")
				}
				s.seedSessionsByScenario[row.Scenario] = row.Token
			}
		}
		close(record.ready)
		s.seedSessions[row.Token] = record
	}
	if len(s.seedSessionsByScenario) > activeSeedSessionLimit || (store.pending != nil && len(s.seedSessionsByScenario) >= activeSeedSessionLimit) {
		return errors.New("seed ownership exceeds active reservation capacity")
	}
	if store.pending != nil {
		s.seedOwnerErr = &seedPendingReservationError{store.pending.Record.RunID, errors.New("seed checkpoint reservation was interrupted; exact retirement is pending")}
	}
	return s.finishSeedOwnerInitialization()
}

func validSeedRecordIdentity(row seedRecordDisk) bool {
	return uuid.Validate(row.Token) == nil && uuid.Validate(row.RunID) == nil && validSeedScenario(row.Scenario) && filepath.IsAbs(row.ScenarioPath) && !row.Created.IsZero()
}

func validateSeedRegistryIdentity(disk seedRegistryDisk, now time.Time) error {
	if disk.Version != 1 || len(disk.Sessions) > seedSessionLimit {
		return errors.New("seed registry version or capacity is invalid")
	}
	if pending := disk.PendingReservation; pending != nil {
		row := pending.Record
		if pending.State != seedPendingCheckpointState || !validPendingSeedRecord(row) || len(disk.Sessions) >= seedSessionLimit || !row.RetiredAt.IsZero() || row.Pruning || row.ConsumerLost || !row.ConsumerDeadline.IsZero() || row.Retain || len(row.PrepareError) > seedPrivateErrorMaxBytes {
			return errors.New("seed pending reservation state or ownership identity is invalid")
		}
	}
	type binding struct{ scenario, request string }
	tokens, runs := map[string]bool{}, map[string]bool{}
	requests := map[binding]bool{}
	active := map[string]bool{}
	check := func(row seedRecordDisk) error {
		if !validSeedRecordIdentity(row) || (row.Pruning && row.RetiredAt.IsZero()) {
			return errors.New("seed registry has invalid ownership identity or pruning state")
		}
		if tokens[row.Token] || runs[row.RunID] {
			return errors.New("seed registry duplicates a token or run owner")
		}
		tokens[row.Token], runs[row.RunID] = true, true
		// Apply replaces empty requests before lookup; empty legacy identities
		// and expired retired history are not live request bindings.
		if row.RequestID != "" && !seedOutcomeExpired(row.RetiredAt, now) {
			key := binding{row.Scenario, row.RequestID}
			if requests[key] {
				return errors.New("seed registry duplicates a live scenario/request binding")
			}
			requests[key] = true
		}
		if row.RetiredAt.IsZero() {
			if active[row.Scenario] {
				return errors.New("seed registry duplicates an active scenario owner")
			}
			active[row.Scenario] = true
		}
		return nil
	}
	for _, row := range disk.Sessions {
		if err := check(row); err != nil {
			return err
		}
	}
	if disk.PendingReservation != nil {
		if err := check(disk.PendingReservation.Record); err != nil {
			return err
		}
	}
	if len(active) > activeSeedSessionLimit {
		return errors.New("seed registry exceeds active reservation capacity")
	}
	return nil
}

func validPendingSeedRecord(row seedRecordDisk) bool {
	runID, runErr := uuid.Parse(row.RunID)
	token, tokenErr := uuid.Parse(row.Token)
	return validSeedRecordIdentity(row) && runErr == nil && tokenErr == nil && runID != uuid.Nil && token != uuid.Nil && runID.String() == row.RunID && token.String() == row.Token && len(row.Scenario) <= 128 && len(row.RequestID) > 0 && len(row.RequestID) <= 200 && len(row.ScenarioPath) <= maxSeedSourcePathBytes && filepath.Clean(row.ScenarioPath) == row.ScenarioPath && !strings.ContainsAny(row.Scenario+row.RequestID+row.ScenarioPath, "\x00\r\n") && !strings.ContainsAny(row.Scenario, "/\\:@") && row.Scenario != "." && row.Scenario != ".."
}

func boundedSeedOwnerError(message string) string {
	message = strings.ToValidUTF8(message, "\uFFFD")
	if len(message) <= seedPrivateErrorMaxBytes {
		return message
	}
	const suffix = " [truncated]"
	end := seedPrivateErrorMaxBytes - len(suffix)
	for !utf8.ValidString(message[:end]) {
		end--
	}
	return message[:end] + suffix
}

// Caller holds seedSessionsMu. A durable checkpoint-only plan precedes the
// first filesystem effect; no target work is authorized by this phase.
func (s *Server) beginSeedReservationLocked(token string, record *seedSession) error {
	if s.seedStore == nil {
		return reserveSeedCheckpoint("", record.runID)
	}
	if s.seedStore.pending != nil {
		return errors.New("seed checkpoint reservation retirement is pending")
	}
	if _, err := s.seedStore.root.Lstat(record.runID + ".json"); !errors.Is(err, os.ErrNotExist) {
		return errors.New("seed checkpoint reservation path is not absent")
	}
	s.seedStore.pending = &seedPendingReservation{State: seedPendingCheckpointState, Record: seedRecordDisk{Token: token, Scenario: record.Scenario, RequestID: record.requestID, ScenarioPath: record.scenarioPath, RunID: record.runID, Created: record.Created}}
	if err := s.persistSeedSessionsWithinBoundLocked(seedRegistryAdmissionBytes); err != nil {
		if errors.Is(err, errSeedRegistryCapacity) {
			// Encoding refused before publication, checkpoint creation or effects.
			s.seedStore.pending = nil
			return err
		}
		return s.failSeedReservationLocked(err)
	}
	if err := reserveSeedCheckpoint(s.seedCheckpointPath(record.runID), record.runID); err != nil {
		return s.failSeedReservationLocked(err)
	}
	return nil
}

func (s *Server) failSeedReservationLocked(cause error) error {
	if s.seedStore == nil || s.seedStore.pending == nil {
		if s.seedOwnerErr == nil {
			s.seedOwnerErr = cause
		}
		return cause
	}
	pending := s.seedStore.pending
	if pending.Record.PrepareError == "" {
		pending.Record.PrepareError = boundedSeedOwnerError(cause.Error())
	}
	_, registryOnly := s.seedOwnerErr.(*seedRegistryPublicationError)
	refusal, reservationOnly := s.seedOwnerErr.(*seedPendingReservationError)
	if s.seedOwnerErr == nil || registryOnly || (reservationOnly && refusal.runID == pending.Record.RunID) {
		s.seedOwnerErr = &seedPendingReservationError{pending.Record.RunID, errors.New(pending.Record.PrepareError)}
	}
	// Failure to record this excerpt does not permit work: the previously
	// acknowledged checkpoint-only plan still owns exactly the same retirement.
	return errors.Join(cause, s.persistSeedSessionsLocked())
}

func (s *Server) reconcileSeedReservationLocked() error {
	pending := s.seedStore.pending
	if pending == nil {
		return nil
	}
	row := pending.Record
	if err := retireSeedCheckpointReservation(s.seedCheckpointPath(row.RunID), row.RunID); err != nil {
		return s.failSeedReservationLocked(err)
	}
	message := row.PrepareError
	if message == "" {
		message = "seed checkpoint reservation was interrupted; validation cannot resume"
	}
	// Reuse the retained failed/pruning outcome: its effect-free checkpoint has
	// been exactly retired, and same-key retries must never restart preparation.
	record := &seedSession{Scenario: row.Scenario, Created: row.Created, requestID: row.RequestID, scenarioPath: row.ScenarioPath, runID: row.RunID, ready: make(chan struct{}), prepareErr: errors.New(message), retiredAt: time.Now().UTC(), pruning: true}
	close(record.ready)
	s.seedSessions[row.Token] = record
	s.seedStore.pending = nil
	if err := s.persistSeedSessionsLocked(); err != nil {
		delete(s.seedSessions, row.Token)
		s.seedStore.pending = pending
		return s.failSeedReservationLocked(err)
	}
	if refusal, ok := s.seedOwnerErr.(*seedPendingReservationError); ok && refusal.runID == row.RunID {
		s.seedOwnerErr = nil
	}
	return nil
}

func (s *Server) recordSeedRegistryFailureLocked(err error) {
	if err == nil || errors.Is(err, errSeedRegistryCapacity) {
		return
	}
	if _, registryOnly := s.seedOwnerErr.(*seedRegistryPublicationError); s.seedOwnerErr == nil || registryOnly {
		s.seedOwnerErr = err
	}
}

func (s *Server) finishSeedOwnerInitialization() error {
	s.seedStore.validated = true
	if err := s.persistSeedSessionsLocked(); err != nil {
		s.seedStore.validated = false
		return err
	}
	return nil
}
func seedRetainedError(message string) error {
	if message == "" {
		return nil
	}
	return errors.New(message)
}
func (s *Server) seedCheckpointPath(runID string) string {
	if s.seedStore == nil {
		return ""
	}
	return filepath.Join(s.seedStore.dir, runID+".json")
}

// Caller holds seedSessionsMu. No cleanup callback takes this lock while
// holding record.mu; journal publication precedes releasing a scenario slot.
func (s *Server) persistSeedSessionsLocked() error {
	return s.persistSeedSessionsWithinBoundLocked(seedRegistryMaxBytes)
}

func (s *Server) persistSeedSessionsWithinBoundLocked(limit int) (resultErr error) {
	if s.seedStore == nil {
		return nil
	}
	if !s.seedStore.validated {
		return errors.New("seed ownership journal was not completely initialized")
	}
	defer func() {
		if resultErr != nil && !errors.Is(resultErr, errSeedRegistryCapacity) {
			resultErr = &seedRegistryPublicationError{resultErr}
		}
	}()
	disk := seedRegistryDisk{Version: 1, PendingReservation: s.seedStore.pending}
	for token, record := range s.seedSessions {
		record.mu.Lock()
		row := seedRecordDisk{Token: token, Scenario: record.Scenario, RequestID: record.requestID, ScenarioPath: record.scenarioPath, RunID: record.runID, Created: record.Created, RetiredAt: record.retiredAt, ConsumerDeadline: record.consumerDeadline, ConsumerLost: record.consumerLost}
		row.Pruning = record.pruning
		row.Retain = record.retainedBacking || (record.Session != nil && record.Session.BackingRetained())
		if record.prepareErr != nil {
			row.PrepareError = boundedSeedOwnerError(record.prepareErr.Error())
		}
		record.mu.Unlock()
		disk.Sessions = append(disk.Sessions, row)
	}
	data, err := json.Marshal(disk)
	if err != nil {
		return err
	}
	if len(data) > limit || len(data) > seedRegistryMaxBytes {
		return connect.NewError(connect.CodeResourceExhausted, errSeedRegistryCapacity)
	}
	if err = storage.WriteFileAtomicInRoot(s.seedStore.root, "registry.json", data, 0600); err != nil {
		return err
	}
	dir, err := s.seedStore.root.Open(".")
	if err != nil {
		return err
	}
	defer dir.Close()
	if err := dir.Sync(); err != nil {
		return err
	}
	if _, registryOnly := s.seedOwnerErr.(*seedRegistryPublicationError); registryOnly {
		s.seedOwnerErr = nil
	}
	return nil
}

func (s *Server) startSeedMaintenance(ctx context.Context) {
	s.seedMaintenance.Add(1)
	go func() {
		defer s.seedMaintenance.Done()
		ticker := time.NewTicker(seedSweepInterval)
		defer ticker.Stop()
		s.sweepSeedSessions(ctx, time.Now())
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				s.sweepSeedSessions(ctx, now)
			}
		}
	}()
}
func (s *Server) sweepSeedSessions(ctx context.Context, now time.Time) {
	type cleanup struct {
		name, token string
		record      *seedSession
	}
	var work []cleanup
	var pruned []*seedSession
	s.seedSessionsMu.Lock()
	if s.seedStore != nil && !s.seedStore.validated {
		s.seedSessionsMu.Unlock()
		return
	}
	if s.seedStore != nil && s.seedStore.pending != nil {
		_ = s.reconcileSeedReservationLocked()
	}
	for token, record := range s.seedSessions {
		record.mu.Lock()
		// A retired timestamp is pending until the journal publication succeeds.
		// Keep its handle while the exact reservation still awaits release.
		if seedOutcomeExpired(record.retiredAt, now) && s.seedSessionsByScenario[record.Scenario] != token && !record.cleaning {
			record.pruning = true
			record.mu.Unlock()
			pruned = append(pruned, record)
			continue
		}
		ready := false
		if record.ready != nil {
			select {
			case <-record.ready:
				ready = true
			default:
			}
		}
		if ready && record.retiredAt.IsZero() && record.Session != nil && !record.cleaning && (record.consumerLost || (!record.consumerDeadline.IsZero() && !now.Before(record.consumerDeadline))) {
			record.consumerLost = true
			if record.prepareErr == nil {
				record.prepareErr = errors.New("seed consumer lease expired or owner was replaced")
			}
			record.cleaning = true
			record.Session.FailSeed(errors.New("seed consumer lease expired or owner was replaced"))
			work = append(work, cleanup{record.Scenario, token, record})
		}
		record.mu.Unlock()
	}
	persistErr := s.persistSeedSessionsLocked()
	if persistErr == nil {
		s.releaseDurablyRetiredSeedSlotsLocked()
	}
	if persistErr != nil && !errors.Is(persistErr, errSeedRegistryCapacity) {
		s.recordSeedRegistryFailureLocked(persistErr)
	}
	s.seedSessionsMu.Unlock()
	if persistErr == nil && s.seedStore != nil {
		for _, record := range pruned {
			retained := record.retainedBacking || (record.Session != nil && record.Session.BackingRetained())
			// Publication no longer references these retired private artifacts.
			// A late native submit still sees its readiness path absent.
			if !retained {
				if err := s.seedStore.root.Remove(record.runID + ".json"); err != nil && !errors.Is(err, os.ErrNotExist) {
					continue
				}
				if err := s.seedStore.root.RemoveAll(filepath.Join("gates", record.runID)); err != nil {
					continue
				}
			}
			s.seedSessionsMu.Lock()
			for token, current := range s.seedSessions {
				if current == record {
					delete(s.seedSessions, token)
					break
				}
			}
			if err := s.persistSeedSessionsLocked(); err != nil && !errors.Is(err, errSeedRegistryCapacity) {
				s.recordSeedRegistryFailureLocked(err)
			}
			s.seedSessionsMu.Unlock()
		}
	}
	for _, item := range work {
		s.seedMaintenance.Add(1)
		go func(item cleanup) {
			defer s.seedMaintenance.Done()
			_, _ = s.clearSeedSession(ctx, item.name, item.token)
			item.record.mu.Lock()
			item.record.cleaning = false
			item.record.mu.Unlock()
		}(item)
	}
}

// Caller holds seedSessionsMu, immediately after successful registry publication.
// A late retirement can release only its own reservation, never a replacement.
func (s *Server) releaseDurablyRetiredSeedSlotsLocked() {
	for name, token := range s.seedSessionsByScenario {
		record := s.seedSessions[token]
		if record == nil {
			continue
		}
		record.mu.Lock()
		retired := !record.retiredAt.IsZero()
		record.mu.Unlock()
		if retired {
			delete(s.seedSessionsByScenario, name)
		}
	}
}

func (s *Server) observeSeedSession(name, token string) (*seedSession, error) {
	record, err := s.findSeedSession(name, token)
	if err != nil {
		return nil, err
	}
	s.seedSessionsMu.Lock()
	record.mu.Lock()
	if record.retiredAt.IsZero() && !record.consumerLost {
		if !record.consumerDeadline.IsZero() && !time.Now().Before(record.consumerDeadline) {
			record.consumerLost = true
			if record.prepareErr == nil {
				record.prepareErr = errors.New("seed consumer lease expired")
			}
			record.Session.FailSeed(errors.New("seed consumer lease expired"))
		} else {
			record.consumerDeadline = time.Now().Add(seedConsumerTTL)
		}
	}
	record.mu.Unlock()
	err = s.persistSeedSessionsLocked()
	if err != nil && !errors.Is(err, errSeedRegistryCapacity) {
		s.recordSeedRegistryFailureLocked(err)
	}
	s.seedSessionsMu.Unlock()
	return record, err
}
