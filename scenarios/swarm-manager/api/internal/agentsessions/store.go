package agentsessions

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"

	"swarm-manager/internal/storage"

	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/operatordecision"
	corestorage "github.com/vrooli/api-core/storage"
)

var ErrNotFound = errors.New("agent session not found")

const (
	sessionFileName    = "session.json"
	messagesFileName   = "messages.jsonl"
	artifactsFileName  = "artifacts.jsonl"
	attachmentsDirName = "attachments"
	proposalsDirName   = "proposals"
)

const legacySourceMigrationMarker = ".legacy-source-migration-v1"

// MigrateLegacySourceData copies pre-storage-migration session data out of the
// scenario source tree before a data-root-backed store is opened. It never
// deletes or overwrites source data: removal is a separate, verified cleanup
// step. A marker makes successful migrations idempotent.
func MigrateLegacySourceData(scenarioRoot, dataRoot string) error {
	legacyRoot := filepath.Join(scenarioRoot, "agent-sessions")
	targetRoot := filepath.Join(dataRoot, "agent-sessions")
	if filepath.Clean(legacyRoot) == filepath.Clean(targetRoot) {
		return nil
	}
	if _, err := os.Stat(legacyRoot); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return fmt.Errorf("inspect legacy agent sessions: %w", err)
	}
	marker := filepath.Join(targetRoot, legacySourceMigrationMarker)
	if _, err := os.Stat(marker); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect session migration marker: %w", err)
	}
	if entries, err := os.ReadDir(targetRoot); err == nil && len(entries) > 0 {
		return fmt.Errorf("refusing to merge legacy sessions into non-empty data root %q", targetRoot)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect session data root: %w", err)
	}
	if err := copyDirectory(legacyRoot, targetRoot); err != nil {
		return fmt.Errorf("copy legacy agent sessions: %w", err)
	}
	if err := os.WriteFile(marker, []byte("legacy source sessions copied; source retained for verified cleanup\n"), 0o600); err != nil {
		return fmt.Errorf("write session migration marker: %w", err)
	}
	return nil
}

func copyDirectory(source, target string) error {
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(source, path)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return fmt.Errorf("unsafe legacy session path %q", path)
		}
		destination := filepath.Join(target, rel)
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("legacy session data contains symlink %q", path)
		}
		if entry.IsDir() {
			return os.MkdirAll(destination, 0o750)
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("legacy session data contains non-regular file %q", path)
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(destination, contents, 0o600)
	})
}

type Store interface {
	CreateSession(session Session) error
	SaveSession(session Session) error
	DeleteSession(sessionID string) error
	LoadSession(sessionID string) (Session, error)
	ListSessions(filters ListFilters) ([]Session, error)
	AppendMessage(sessionID string, message Message) error
	SaveProposal(sessionID string, proposal Proposal) error
	AppendArtifacts(sessionID string, artifacts []Artifact) error
	ListArtifacts(sessionID string) ([]Artifact, error)
	ListArtifactsByEntity(artifactType ArtifactType, entityRef string) ([]Artifact, error)
	SaveAttachment(sessionID string, attachment Attachment, reader io.Reader) error
	AttachmentPath(sessionID string, attachmentID string) (string, Attachment, error)
}

// ProposalDecisionPageStore is the owner-scoped scan and atomic receipt seam
// used by the proposal publisher. It is optional on Store so unrelated session
// test stores and adapters do not need a new generic pagination API.
type ProposalDecisionPageStore interface {
	ListProposalDecisionPage(after ProposalDecisionCursor, limit int) (ProposalDecisionPage, error)
	UpdateProposalDecisionPublication(sessionID, proposalID, expectedEventID string, next ProposalDecisionPublication) (bool, error)
}

// LegacyProposalDecisionGenerationStore atomically preserves a legacy marker
// while freezing one explicit new decision request for unchanged current terms.
type LegacyProposalDecisionGenerationStore interface {
	BeginLegacyProposalDecisionGeneration(sessionID, proposalID, expectedTermsSHA256 string, next ProposalDecisionPublication) (bool, error)
}

type ProposalDecisionCursor struct {
	SessionID  string
	ProposalID string
}

type ProposalDecisionCandidate struct {
	SessionID string
	Proposal  Proposal
}

// ProposalDecisionPageIssue describes one proposal record that could not be
// trusted for reconciliation. The file remains in place; the owner can advance
// over it while making the unresolved record visible to operators.
type ProposalDecisionPageIssue struct {
	SessionID    string
	ProposalFile string
	Reason       string
}

type ProposalDecisionPage struct {
	Candidates []ProposalDecisionCandidate
	Issues     []ProposalDecisionPageIssue
	Next       ProposalDecisionCursor
	HasMore    bool
}

// ContextStore can derive a request-scoped store from the active routed
// storage lease. The Store interface intentionally remains context-free for
// the many small file operations; services select this view once per request.
type ContextStore interface {
	ForContext(context.Context) (Store, error)
}

type ListFilters struct {
	Kind       Kind
	Status     Status
	ActiveOnly bool
	Limit      int
}

type FileStore struct {
	root          string
	kindValidator func(Kind) bool
	// mu is shared by pointer with every ForContext view of this store, so
	// concurrent requests against the same routed root still serialize.
	mu       *sync.Mutex
	roots    *filerouting.RoutedRoots
	writeCtx context.Context
}

func NewFileStore(root string) *FileStore {
	return &FileStore{root: filepath.Join(root, "agent-sessions"), mu: &sync.Mutex{}}
}

// NewRoutedFileStore selects the data-class root for each request. This is the
// store used by the production server; tests that construct NewFileStore keep
// the small, deterministic primary-root implementation.
//
// The returned store deliberately has no root of its own: only ForContext can
// supply one. Calling a data method on it directly is a programming error, not
// an empty result — see errUnroutedStore.
func NewRoutedFileStore(roots *filerouting.RoutedRoots) *FileStore {
	return &FileStore{roots: roots, mu: &sync.Mutex{}}
}

// ErrUnrouted reports a routed store used outside a request scope. Without the
// guard, filepath.Join("", id) resolves against the process working directory
// and every read reports ErrNotFound while every list reports zero sessions —
// a silent, total data outage that looks like an empty store.
var ErrUnrouted = errors.New("agent session store used without a routed request scope; call ForContext first")

func (s *FileStore) errUnroutedStore() error {
	if s.roots != nil && s.root == "" {
		return ErrUnrouted
	}
	return nil
}

func (s *FileStore) ForContext(ctx context.Context) (Store, error) {
	if s.roots == nil {
		return s, nil
	}
	root, err := s.roots.Pick(ctx, corestorage.ClassData)
	if err != nil {
		return nil, fmt.Errorf("resolve routed agent-session data root: %w", err)
	}
	scoped := &FileStore{
		root:          filepath.Join(root, "agent-sessions"),
		kindValidator: s.kindValidator,
		mu:            s.mu,
		roots:         s.roots,
		writeCtx:      ctx,
	}
	return scoped, nil
}

// SetSessionKindValidator installs the registry-owned session-kind predicate.
// It is intentionally optional on Store so small test stores and compatibility
// callers can retain the legacy validation behavior.
func (s *FileStore) SetSessionKindValidator(validator func(Kind) bool) {
	s.kindValidator = validator
}

func (s *FileStore) recordWrite() {
	if s.roots != nil && s.writeCtx != nil {
		s.roots.RecordWrite(s.writeCtx)
	}
}

func (s *FileStore) CreateSession(session Session) error {
	if err := s.errUnroutedStore(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	session = snapshotOnly(session)
	if err := session.ValidateWith(s.kindValidator); err != nil {
		return err
	}
	dir := s.sessionDir(session.ID)
	if _, err := os.Stat(filepath.Join(dir, sessionFileName)); err == nil {
		return fmt.Errorf("%w: %s", ErrValidation, "session already exists")
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(filepath.Join(dir, proposalsDirName), 0o750); err != nil {
		return err
	}
	s.recordWrite()
	return storage.WriteJSONAtomic(filepath.Join(dir, sessionFileName), session)
}

func (s *FileStore) SaveSession(session Session) error {
	if err := s.errUnroutedStore(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveSessionLocked(session)
}

func (s *FileStore) DeleteSession(sessionID string) error {
	if err := s.errUnroutedStore(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	safeID, err := safeSessionID(sessionID)
	if err != nil {
		return err
	}
	if _, err := s.loadSessionLocked(safeID); err != nil {
		return err
	}
	s.recordWrite()
	return os.RemoveAll(s.sessionDir(safeID))
}

func (s *FileStore) LoadSession(sessionID string) (Session, error) {
	if err := s.errUnroutedStore(); err != nil {
		return Session{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadSessionLocked(sessionID)
}

func (s *FileStore) ListSessions(filters ListFilters) ([]Session, error) {
	if err := s.errUnroutedStore(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := os.ReadDir(s.root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Session{}, nil
		}
		return nil, err
	}

	sessions := make([]Session, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		session, err := s.loadSessionLocked(entry.Name())
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				continue
			}
			// A persisted legacy/corrupt session must not make the session index
			// (or callers such as proposal gates) unavailable. Mutating that
			// record still goes through LoadSession and remains deliberately
			// rejected until it is migrated or repaired.
			slog.Warn("agent sessions: skipping unreadable session during list", "session_id", entry.Name(), "error", err)
			continue
		}
		if !matchesListFilters(session, filters) {
			continue
		}
		sessions = append(sessions, session)
	}

	sort.Slice(sessions, func(i, j int) bool {
		if sessions[i].UpdatedAt == sessions[j].UpdatedAt {
			return sessions[i].ID > sessions[j].ID
		}
		return sessions[i].UpdatedAt > sessions[j].UpdatedAt
	})
	if filters.Limit > 0 && len(sessions) > filters.Limit {
		sessions = sessions[:filters.Limit]
	}
	return sessions, nil
}

func (s *FileStore) AppendMessage(sessionID string, message Message) error {
	if err := s.errUnroutedStore(); err != nil {
		return err
	}
	if err := message.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	session, err := s.loadSessionLocked(sessionID)
	if err != nil {
		return err
	}
	for _, existing := range session.Messages {
		if existing.ID != message.ID {
			continue
		}
		if reflect.DeepEqual(existing, message) {
			return nil
		}
		return validationError("message id already exists with different content")
	}
	if err := appendJSONL(filepath.Join(s.sessionDir(session.ID), messagesFileName), message); err != nil {
		return err
	}
	s.recordWrite()
	session.UpdatedAt = message.CreatedAt
	return s.saveSessionLocked(session)
}

func (s *FileStore) SaveProposal(sessionID string, proposal Proposal) error {
	if err := s.errUnroutedStore(); err != nil {
		return err
	}
	if err := proposal.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	session, err := s.loadSessionLocked(sessionID)
	if err != nil {
		return err
	}
	proposal.ID, err = safeProposalID(proposal.ID)
	if err != nil {
		return err
	}
	path := filepath.Join(s.sessionDir(session.ID), proposalsDirName, proposal.ID+".json")
	var existing Proposal
	exists, err := storage.ReadJSON(path, &existing)
	if err != nil {
		return err
	}
	if exists {
		switch {
		case existing.DecisionPublication != nil && proposal.DecisionPublication == nil:
			proposal.DecisionPublication = existing.DecisionPublication
		case existing.DecisionPublication != nil && proposal.DecisionPublication != nil:
			if existing.DecisionPublication.State == ProposalDecisionPublicationLegacyUnresolved {
				if proposal.DecisionPublication.State != ProposalDecisionPublicationLegacyUnresolved || proposal.DecisionPublication.EventID != "" || proposal.DecisionPublication.Event.Source != "" {
					return validationError("legacy decision publication state is immutable")
				}
				proposal.DecisionPublication = existing.DecisionPublication
				break
			}
			if existing.DecisionPublication.EventID != proposal.DecisionPublication.EventID || operatordecision.EventID(existing.DecisionPublication.Event) != existing.DecisionPublication.EventID || operatordecision.EventID(proposal.DecisionPublication.Event) != existing.DecisionPublication.EventID {
				return validationError("decision publication snapshot is immutable")
			}
			// A status/decision write may have read before a publication receipt
			// landed. Keep the newest owner receipt instead of replacing it with
			// the caller's stale copy.
			proposal.DecisionPublication = existing.DecisionPublication
		case existing.DecisionPublication == nil && proposal.DecisionPublication != nil:
			return validationError("decision publication snapshot cannot be added to an existing proposal")
		}
		if existing.DecisionPublication != nil && existing.DecisionPublication.DecisionTermsSHA256 != "" {
			termsDigest, digestErr := proposalDecisionTermsSHA256(proposal)
			if digestErr != nil {
				return digestErr
			}
			if termsDigest != existing.DecisionPublication.DecisionTermsSHA256 {
				return validationError("proposal facts bound to the active decision request are immutable")
			}
		}
	}
	if existing.PreviousDecisionPublication != nil {
		if proposal.PreviousDecisionPublication == nil {
			proposal.PreviousDecisionPublication = existing.PreviousDecisionPublication
		} else if !reflect.DeepEqual(proposal.PreviousDecisionPublication, existing.PreviousDecisionPublication) {
			return validationError("previous decision publication history is immutable")
		}
	} else if proposal.PreviousDecisionPublication != nil {
		return validationError("previous decision publication history can only be added by the legacy generation owner operation")
	}
	if err := storage.WriteJSONAtomic(path, proposal); err != nil {
		return err
	}
	s.recordWrite()
	session.UpdatedAt = proposal.UpdatedAt
	if proposal.Status == ProposalStatusReady {
		session.Status = StatusProposalReady
	}
	return s.saveSessionLocked(session)
}

func (s *FileStore) ListProposalDecisionPage(after ProposalDecisionCursor, limit int) (ProposalDecisionPage, error) {
	if err := s.errUnroutedStore(); err != nil {
		return ProposalDecisionPage{}, err
	}
	if limit <= 0 {
		limit = proposalDecisionPageSize
	} else if limit > proposalDecisionPageSize {
		limit = proposalDecisionPageSize
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := os.ReadDir(s.root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ProposalDecisionPage{}, nil
		}
		return ProposalDecisionPage{}, err
	}
	page := ProposalDecisionPage{Candidates: make([]ProposalDecisionCandidate, 0, limit)}
	scanned := 0
	for _, sessionEntry := range entries { // os.ReadDir returns lexical order.
		if !sessionEntry.IsDir() || sessionEntry.Name() < after.SessionID {
			continue
		}
		if _, idErr := safeSessionID(sessionEntry.Name()); idErr != nil {
			continue
		}
		proposalEntries, readErr := os.ReadDir(filepath.Join(s.root, sessionEntry.Name(), proposalsDirName))
		if readErr != nil && !errors.Is(readErr, os.ErrNotExist) {
			return ProposalDecisionPage{}, readErr
		}
		for _, proposalEntry := range proposalEntries {
			if proposalEntry.IsDir() || filepath.Ext(proposalEntry.Name()) != ".json" {
				continue
			}
			proposalID := strings.TrimSuffix(proposalEntry.Name(), ".json")
			if sessionEntry.Name() == after.SessionID && proposalID <= after.ProposalID {
				continue
			}
			if scanned == limit {
				page.HasMore = true
				return page, nil
			}
			scanned++
			page.Next = ProposalDecisionCursor{SessionID: sessionEntry.Name(), ProposalID: proposalID}
			if _, idErr := safeProposalID(proposalID); idErr != nil {
				page.Issues = append(page.Issues, ProposalDecisionPageIssue{SessionID: sessionEntry.Name(), ProposalFile: proposalEntry.Name(), Reason: "invalid_proposal_id"})
				continue
			}
			var proposal Proposal
			exists, readErr := storage.ReadJSON(filepath.Join(s.root, sessionEntry.Name(), proposalsDirName, proposalEntry.Name()), &proposal)
			if readErr != nil {
				page.Issues = append(page.Issues, ProposalDecisionPageIssue{SessionID: sessionEntry.Name(), ProposalFile: proposalEntry.Name(), Reason: "proposal_unreadable"})
				continue
			}
			if !exists {
				page.Issues = append(page.Issues, ProposalDecisionPageIssue{SessionID: sessionEntry.Name(), ProposalFile: proposalEntry.Name(), Reason: "proposal_disappeared_during_read"})
				continue
			}
			if proposal.ID != proposalID {
				page.Issues = append(page.Issues, ProposalDecisionPageIssue{SessionID: sessionEntry.Name(), ProposalFile: proposalEntry.Name(), Reason: "proposal_id_path_mismatch"})
				continue
			}
			candidate := ProposalDecisionCandidate{SessionID: sessionEntry.Name(), Proposal: proposal}
			page.Candidates = append(page.Candidates, candidate)
		}
	}
	return page, nil
}

// BeginLegacyProposalDecisionGeneration atomically preserves the unresolved
// historical marker and installs a new immutable event only while the exact
// current proposal terms still match the caller's digest.
func (s *FileStore) BeginLegacyProposalDecisionGeneration(sessionID, proposalID, expectedTermsSHA256 string, next ProposalDecisionPublication) (bool, error) {
	if err := s.errUnroutedStore(); err != nil {
		return false, err
	}
	safeSession, err := safeSessionID(sessionID)
	if err != nil {
		return false, err
	}
	safeProposal, err := safeProposalID(proposalID)
	if err != nil {
		return false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.sessionDir(safeSession), proposalsDirName, safeProposal+".json")
	var proposal Proposal
	exists, err := storage.ReadJSON(path, &proposal)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}
	if proposal.ID != safeProposal {
		return false, validationError("proposal file id does not match its path")
	}
	if !eligibleProposalDecision(proposal) || len(proposal.Decisions) != 0 || proposal.PreviousDecisionPublication != nil || proposal.DecisionPublication == nil || proposal.DecisionPublication.State != ProposalDecisionPublicationLegacyUnresolved {
		return false, nil
	}
	currentDigest, err := proposalDecisionTermsSHA256(proposal)
	if err != nil {
		return false, err
	}
	if !strings.EqualFold(strings.TrimSpace(expectedTermsSHA256), currentDigest) {
		return false, nil
	}
	termsDigest, err := proposalDecisionTermsSHA256(proposal)
	if err != nil {
		return false, err
	}
	if !strings.EqualFold(strings.TrimSpace(next.DecisionTermsSHA256), termsDigest) {
		return false, validationError("new legacy decision generation terms binding does not match the current proposal")
	}
	correlation, correlationOK := next.Event.Payload["correlation"].(map[string]any)
	if next.Version != 1 || next.State != ProposalDecisionPublicationPending || next.EventID == "" ||
		next.Event.Source != ProposalDecisionSource || next.Event.EventType != ProposalDecisionEventType || operatordecision.EventID(next.Event) != next.EventID || !correlationOK ||
		correlation["session_id"] != safeSession || correlation["proposal_id"] != safeProposal || correlation["subject_ref"] != safeSession+"/"+safeProposal || correlation["subject_kind"] != AttemptSubjectProposal {
		return false, validationError("new legacy decision generation is incomplete")
	}
	if err := proposal.Validate(); err != nil {
		return false, err
	}
	prior := *proposal.DecisionPublication
	proposal.PreviousDecisionPublication = &prior
	proposal.DecisionPublication = &next
	if err := proposal.Validate(); err != nil {
		return false, err
	}
	if err := storage.WriteJSONAtomic(path, proposal); err != nil {
		return false, err
	}
	s.recordWrite()
	return true, nil
}

// UpdateProposalDecisionPublication changes only publication metadata while
// holding the same owner mutex as proposal decisions. An empty expected ID
// means mark a legacy eligible row unresolved; otherwise the exact frozen
// event generation must still be current.
func (s *FileStore) UpdateProposalDecisionPublication(sessionID, proposalID, expectedEventID string, next ProposalDecisionPublication) (bool, error) {
	if err := s.errUnroutedStore(); err != nil {
		return false, err
	}
	safeSession, err := safeSessionID(sessionID)
	if err != nil {
		return false, err
	}
	safeProposal, err := safeProposalID(proposalID)
	if err != nil {
		return false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.sessionDir(safeSession), proposalsDirName, safeProposal+".json")
	var proposal Proposal
	exists, err := storage.ReadJSON(path, &proposal)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}
	if proposal.ID != safeProposal {
		return false, validationError("proposal file id does not match its path")
	}
	if expectedEventID == "" {
		if !eligibleProposalDecision(proposal) {
			return false, nil
		}
		if proposal.DecisionPublication != nil {
			return false, nil
		}
		if next.State != ProposalDecisionPublicationLegacyUnresolved {
			return false, validationError("missing snapshot can only be marked legacy unresolved")
		}
	} else if proposal.DecisionPublication == nil || proposal.DecisionPublication.EventID != expectedEventID || proposal.DecisionPublication.State == ProposalDecisionPublicationLegacyUnresolved {
		return false, nil
	} else {
		current := proposal.DecisionPublication
		if next.EventID != current.EventID || operatordecision.EventID(next.Event) != current.EventID || operatordecision.EventID(current.Event) != current.EventID {
			return false, validationError("decision publication event snapshot is immutable")
		}
		if next.DecisionTermsSHA256 != current.DecisionTermsSHA256 {
			return false, validationError("decision publication terms binding is immutable")
		}
		// Attempts and failures may only be recorded for a proposal that is
		// still pending human review. A successful publish receipt may land
		// after a concurrent decision; updating this nested metadata preserves
		// both that decision and the transport result.
		if next.State != ProposalDecisionPublicationPublished && !eligibleProposalDecision(proposal) {
			return false, nil
		}
		if current.State == ProposalDecisionPublicationPublished && next.State != ProposalDecisionPublicationPublished {
			return false, nil
		}
		if next.Attempts < current.Attempts {
			return false, nil
		}
	}
	proposal.DecisionPublication = &next
	if err := storage.WriteJSONAtomic(path, proposal); err != nil {
		return false, err
	}
	s.recordWrite()
	return true, nil
}

func (s *FileStore) ListArtifacts(sessionID string) ([]Artifact, error) {
	if err := s.errUnroutedStore(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := s.loadSessionLocked(sessionID); err != nil {
		return nil, err
	}
	return readJSONL[Artifact](filepath.Join(s.sessionDir(sessionID), artifactsFileName))
}

// AppendArtifacts preserves non-receipt review artifacts in the session's
// own durable record. These are domain handoff material, not transport
// evidence, and must remain available after the shared receipt migration.
func (s *FileStore) AppendArtifacts(sessionID string, artifacts []Artifact) error {
	if err := s.errUnroutedStore(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.loadSessionLocked(sessionID); err != nil {
		return err
	}
	path := filepath.Join(s.sessionDir(sessionID), artifactsFileName)
	for _, artifact := range artifacts {
		s.recordWrite()
		if err := appendJSONL(path, artifact); err != nil {
			return err
		}
	}
	return nil
}

func (s *FileStore) ListArtifactsByEntity(artifactType ArtifactType, entityRef string) ([]Artifact, error) {
	if err := s.errUnroutedStore(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := os.ReadDir(s.root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Artifact{}, nil
		}
		return nil, err
	}
	var matches []Artifact
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		artifacts, err := readJSONL[Artifact](filepath.Join(s.root, entry.Name(), artifactsFileName))
		if err != nil {
			return nil, err
		}
		for _, artifact := range artifacts {
			if artifact.ArtifactType == artifactType && strings.TrimSpace(artifact.EntityRef) == strings.TrimSpace(entityRef) {
				matches = append(matches, artifact)
			}
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].CreatedAt == matches[j].CreatedAt {
			return matches[i].ID > matches[j].ID
		}
		return matches[i].CreatedAt > matches[j].CreatedAt
	})
	return matches, nil
}

func (s *FileStore) SaveAttachment(sessionID string, attachment Attachment, reader io.Reader) error {
	if err := s.errUnroutedStore(); err != nil {
		return err
	}
	if err := attachment.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	session, err := s.loadSessionLocked(sessionID)
	if err != nil {
		return err
	}
	for _, existing := range session.Attachments {
		if existing.ID == attachment.ID {
			return validationError("attachment already exists")
		}
	}
	dir := filepath.Join(s.sessionDir(session.ID), attachmentsDirName, attachment.ID)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	s.recordWrite()
	path := filepath.Join(dir, sanitizeAttachmentFilename(attachment.Filename))
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	if _, err := io.Copy(file, reader); err != nil {
		if closeErr := file.Close(); closeErr != nil {
			slog.Debug("agentsessions: close attachment file failed", "err", closeErr)
		}
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	session.Attachments = append(session.Attachments, attachment)
	session.UpdatedAt = attachment.CreatedAt
	return s.saveSessionLocked(session)
}

func (s *FileStore) AttachmentPath(sessionID string, attachmentID string) (string, Attachment, error) {
	if err := s.errUnroutedStore(); err != nil {
		return "", Attachment{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	session, err := s.loadSessionLocked(sessionID)
	if err != nil {
		return "", Attachment{}, err
	}
	safeAttachmentID, err := safeAttachmentID(attachmentID)
	if err != nil {
		return "", Attachment{}, err
	}
	for _, attachment := range session.Attachments {
		if attachment.ID != safeAttachmentID {
			continue
		}
		path := filepath.Join(s.sessionDir(session.ID), attachmentsDirName, attachment.ID, sanitizeAttachmentFilename(attachment.Filename))
		return path, attachment, nil
	}
	return "", Attachment{}, fmt.Errorf("%w: %s", ErrNotFound, safeAttachmentID)
}

func (s *FileStore) saveSessionLocked(session Session) error {
	session = snapshotOnly(session)
	if err := session.ValidateWith(s.kindValidator); err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(s.sessionDir(session.ID), sessionFileName)); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%w: %s", ErrNotFound, session.ID)
		}
		return err
	}
	s.recordWrite()
	return storage.WriteJSONAtomic(filepath.Join(s.sessionDir(session.ID), sessionFileName), session)
}

func (s *FileStore) loadSessionLocked(sessionID string) (Session, error) {
	trimmed, err := safeSessionID(sessionID)
	if err != nil {
		return Session{}, err
	}
	var session Session
	exists, err := storage.ReadJSON(filepath.Join(s.sessionDir(trimmed), sessionFileName), &session)
	if err != nil {
		return Session{}, err
	}
	if !exists {
		return Session{}, fmt.Errorf("%w: %s", ErrNotFound, trimmed)
	}
	messages, err := readJSONL[Message](filepath.Join(s.sessionDir(trimmed), messagesFileName))
	if err != nil {
		return Session{}, err
	}
	proposals, err := s.readProposalsLocked(trimmed)
	if err != nil {
		return Session{}, err
	}
	artifacts, err := readJSONL[Artifact](filepath.Join(s.sessionDir(trimmed), artifactsFileName))
	if err != nil {
		return Session{}, err
	}
	session.Messages = messages
	session.Proposals = proposals
	session.Artifacts = artifacts
	if err := session.ValidateWith(s.kindValidator); err != nil {
		return Session{}, err
	}
	return session, nil
}

func (s *FileStore) readProposalsLocked(sessionID string) ([]Proposal, error) {
	dir := filepath.Join(s.sessionDir(sessionID), proposalsDirName)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Proposal{}, nil
		}
		return nil, err
	}
	proposals := make([]Proposal, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		var proposal Proposal
		exists, err := storage.ReadJSON(filepath.Join(dir, entry.Name()), &proposal)
		if err != nil {
			return nil, err
		}
		if !exists {
			continue
		}
		proposals = append(proposals, proposal)
	}
	sort.Slice(proposals, func(i, j int) bool {
		if proposals[i].UpdatedAt == proposals[j].UpdatedAt {
			return proposals[i].ID > proposals[j].ID
		}
		return proposals[i].UpdatedAt > proposals[j].UpdatedAt
	})
	return proposals, nil
}

func (s *FileStore) sessionDir(sessionID string) string {
	return filepath.Join(s.root, strings.TrimSpace(sessionID))
}

func safeSessionID(sessionID string) (string, error) {
	trimmed := strings.TrimSpace(sessionID)
	if trimmed == "" {
		return "", validationError("session_id is required")
	}
	if !strings.HasPrefix(trimmed, "sess_") {
		return "", validationError("session_id must start with sess_")
	}
	if trimmed == "." || trimmed == ".." || strings.Contains(trimmed, "/") || strings.Contains(trimmed, "\\") {
		return "", validationError("session_id is invalid")
	}
	if filepath.Clean(trimmed) != trimmed {
		return "", validationError("session_id is invalid")
	}
	return trimmed, nil
}

func snapshotOnly(session Session) Session {
	session.Messages = nil
	session.Proposals = nil
	session.Artifacts = nil
	return session
}

func safeAttachmentID(attachmentID string) (string, error) {
	trimmed := strings.TrimSpace(attachmentID)
	if trimmed == "" {
		return "", validationError("attachment_id is required")
	}
	if !strings.HasPrefix(trimmed, "att_") {
		return "", validationError("attachment_id must start with att_")
	}
	if trimmed == "." || trimmed == ".." || strings.Contains(trimmed, "/") || strings.Contains(trimmed, "\\") {
		return "", validationError("attachment_id is invalid")
	}
	if filepath.Clean(trimmed) != trimmed {
		return "", validationError("attachment_id is invalid")
	}
	return trimmed, nil
}

func safeProposalID(proposalID string) (string, error) {
	trimmed := strings.TrimSpace(proposalID)
	if trimmed == "" || trimmed == "." || trimmed == ".." || strings.Contains(trimmed, "/") || strings.Contains(trimmed, "\\") || filepath.Clean(trimmed) != trimmed {
		return "", validationError("proposal_id is invalid")
	}
	return trimmed, nil
}

func eligibleProposalDecision(proposal Proposal) bool {
	return proposal.Kind == ProposalMutationList && proposal.Status == ProposalStatusReady && !proposal.NeedsRevision && proposal.Target != nil
}

func sanitizeAttachmentFilename(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	replacer := strings.NewReplacer("..", "_", "/", "_", "\\", "_", "\x00", "_")
	name = replacer.Replace(name)
	if name == "" || name == "." {
		return "unnamed"
	}
	return name
}

func matchesListFilters(session Session, filters ListFilters) bool {
	if filters.Kind != "" && session.Kind != filters.Kind {
		return false
	}
	if filters.Status != "" && session.Status != filters.Status {
		return false
	}
	if filters.ActiveOnly && !isActiveSessionStatus(session.Status) {
		return false
	}
	return true
}

func isActiveSessionStatus(status Status) bool {
	switch status {
	case StatusDraft, StatusStarting, StatusRunning, StatusWaitingForUser, StatusProposalReady, StatusApplying:
		return true
	default:
		return false
	}
}

// hasStoppableRun distinguishes an operator-visible active session from an
// Agent Manager execution that can still accept StopRun. Proposal review and
// application states retain an active UI presence after their attributed run
// has completed, so stopping them would turn a harmless delete into a 409.
func hasStoppableRun(status Status) bool {
	switch status {
	case StatusStarting, StatusRunning, StatusWaitingForUser:
		return true
	default:
		return false
	}
}

func appendJSONL(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Write(append(data, '\n')); err != nil {
		return err
	}
	return file.Sync()
}

func writeJSONLAtomic[T any](path string, values []T) error {
	parentDir := filepath.Dir(path)
	if err := os.MkdirAll(parentDir, 0o750); err != nil {
		return err
	}
	tempFile, err := os.CreateTemp(parentDir, "tmp-*.jsonl")
	if err != nil {
		return err
	}
	tempName := tempFile.Name()
	defer func() {
		if rmErr := os.Remove(tempName); rmErr != nil && !os.IsNotExist(rmErr) {
			slog.Debug("agentsessions: remove temp jsonl failed", "err", rmErr, "path", tempName)
		}
	}()
	for _, value := range values {
		data, err := json.Marshal(value)
		if err != nil {
			if closeErr := tempFile.Close(); closeErr != nil {
				slog.Debug("agentsessions: close temp jsonl failed", "err", closeErr)
			}
			return err
		}
		if _, err := tempFile.Write(append(data, '\n')); err != nil {
			if closeErr := tempFile.Close(); closeErr != nil {
				slog.Debug("agentsessions: close temp jsonl failed", "err", closeErr)
			}
			return err
		}
	}
	if err := tempFile.Sync(); err != nil {
		if closeErr := tempFile.Close(); closeErr != nil {
			slog.Debug("agentsessions: close temp jsonl failed", "err", closeErr)
		}
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tempName, 0o600); err != nil {
		return err
	}
	return os.Rename(tempName, path)
}

func readJSONL[T any](path string) ([]T, error) {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []T{}, nil
		}
		return nil, err
	}
	defer file.Close()

	var values []T
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var value T
		if err := json.Unmarshal([]byte(line), &value); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if values == nil {
		return []T{}, nil
	}
	return values, nil
}
