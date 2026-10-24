package experimentation

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"slices"

	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	platform "github.com/vrooli/platform-go"
	"landing-page-business-suite-api/internal/presentation"
)

var (
	ErrPresentationInvalid      = errors.New("invalid presentation reference")
	ErrPresentationNotFound     = errors.New("presentation revision not found")
	ErrPresentationConflict     = errors.New("presentation generation conflict")
	ErrPresentationCorrupt      = errors.New("presentation storage integrity failure")
	ErrPresentationUnqualified  = errors.New("presentation publication evidence is unqualified")
	presentationRevisionPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)
)

const maxPresentationBytes = 8 << 20

// PresentationPublicationVerifier qualifies owner evidence and referenced asset
// bytes, not merely the syntax of references. A nil verifier refuses publication.
// It must be read-only: publication retries must not repeat external effects.
// The second document is the currently active published revision (nil when no
// revision is active); verifiers may inherit qualification for content that is
// unchanged from it, since that content already passed publication once.
type PresentationPublicationVerifier func(context.Context, presentation.Document, *presentation.Document) error

// PresentationState is the only mutable pointer. Generation is the editor's
// compare-and-swap token; revision IDs are hashes of immutable document bytes.
type PresentationState struct {
	Generation         uint64   `json:"generation"`
	DraftRevision      string   `json:"draft_revision,omitempty"`
	ActiveRevision     string   `json:"active_revision,omitempty"`
	PublishedRevisions []string `json:"published_revisions"`
}

type PresentationRevision struct {
	Revision string                `json:"revision"`
	Document presentation.Document `json:"document"`
}

// SetPresentationStorage configures the existing ConfigStore owner. Roots are
// selected per operation, so test-mode requests never reuse a live cached page.
func (cs *ConfigStore) SetPresentationStorage(roots *filerouting.RoutedRoots, verifier PresentationPublicationVerifier) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.presentationRoots, cs.presentationVerifier = roots, verifier
}

func (cs *ConfigStore) GetPresentationState(ctx context.Context, variant string) (*PresentationState, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	root, err := cs.openPresentationRoot(ctx, variant, false)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	return readPresentationState(root, variant)
}

func (cs *ConfigStore) GetPresentationRevision(ctx context.Context, variant, revision string) (*PresentationRevision, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	root, err := cs.openPresentationRoot(ctx, variant, false)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	return readPresentationRevision(root, variant, revision)
}

// GetPublishedPresentation pins one active revision before reading its document.
// Draft documents are available only through the separate administrative read.
func (cs *ConfigStore) GetPublishedPresentation(ctx context.Context, variant string) (*PresentationRevision, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	root, err := cs.openPresentationRoot(ctx, variant, false)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	state, err := readPresentationState(root, variant)
	if err != nil {
		return nil, err
	}
	if state.ActiveRevision == "" {
		return nil, ErrPresentationNotFound
	}
	return readPresentationRevision(root, variant, state.ActiveRevision)
}

func (cs *ConfigStore) SavePresentationDraft(ctx context.Context, variant string, document presentation.Document, expectedGeneration uint64) (*PresentationState, error) {
	// Round-trip first so neither a caller nor an injected verifier can mutate the
	// document while its immutable bytes are being selected for publication.
	data, err := json.Marshal(document)
	if err != nil {
		return nil, fmt.Errorf("encode presentation: %w", err)
	}
	if len(data) > maxPresentationBytes {
		return nil, fmt.Errorf("presentation exceeds %d bytes", maxPresentationBytes)
	}
	// Decode into a zero value: decoding into the caller's shallow struct copy
	// would reuse its slices/maps and race with another reader of that document.
	var snapshot presentation.Document
	if err := decodePresentation(data, &snapshot); err != nil {
		return nil, err
	}
	if err := presentation.Validate(snapshot); err != nil {
		return nil, err
	}
	digest := sha256.Sum256(data)
	revision := hex.EncodeToString(digest[:])
	cs.mu.Lock()
	defer cs.mu.Unlock()
	root, err := cs.openPresentationRoot(ctx, variant, true)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	unlock, err := cs.lockPresentationMutation(ctx, root, variant)
	if err != nil {
		return nil, err
	}
	defer unlock()
	state, err := readPresentationState(root, variant)
	if err != nil {
		return nil, err
	}
	if err := checkPresentationGeneration(state, expectedGeneration); err != nil {
		return nil, err
	}
	if prior, err := readPresentationRevision(root, variant, revision); err == nil {
		if prior.Revision != revision {
			return nil, ErrPresentationCorrupt
		}
	} else if errors.Is(err, ErrPresentationNotFound) {
		if err := storage.WriteFileAtomicInRoot(root, presentationRevisionPath(variant, revision), data, 0600); err != nil {
			return nil, fmt.Errorf("save immutable presentation: %w", err)
		}
		cs.presentationRoots.RecordWrite(ctx)
	} else {
		return nil, err
	}
	state.DraftRevision = revision
	return cs.writePresentationState(ctx, root, variant, state)
}

func (cs *ConfigStore) PublishPresentation(ctx context.Context, variant, revision string, expectedGeneration uint64) (*PresentationState, error) {
	return cs.activatePresentation(ctx, variant, revision, expectedGeneration, false)
}

func (cs *ConfigStore) RollbackPresentation(ctx context.Context, variant, revision string, expectedGeneration uint64) (*PresentationState, error) {
	return cs.activatePresentation(ctx, variant, revision, expectedGeneration, true)
}

func (cs *ConfigStore) activatePresentation(ctx context.Context, variant, revision string, expectedGeneration uint64, rollback bool) (*PresentationState, error) {
	initial, err := cs.GetPresentationState(ctx, variant)
	if err != nil {
		return nil, err
	}
	if err := checkPresentationGeneration(initial, expectedGeneration); err != nil {
		return nil, err
	}
	if rollback && !slices.Contains(initial.PublishedRevisions, revision) {
		return nil, ErrPresentationNotFound
	}
	if !rollback && initial.DraftRevision != revision {
		return nil, fmt.Errorf("%w: requested revision is not the current draft", ErrPresentationConflict)
	}
	// Never hold the ConfigStore lock across external owner verification. After
	// verification, reread and compare the generation before committing the head.
	cs.mu.RLock()
	verifier := cs.presentationVerifier
	cs.mu.RUnlock()
	if verifier == nil {
		return nil, ErrPresentationUnqualified
	}
	loaded, err := cs.GetPresentationRevision(ctx, variant, revision)
	if err != nil {
		return nil, err
	}
	if err := presentation.Validate(loaded.Document); err != nil {
		return nil, err
	}
	// The active published revision is the trust anchor for carry-forward:
	// content identical to it already passed this exact gate once.
	var active *presentation.Document
	if initial.ActiveRevision != "" && initial.ActiveRevision != revision {
		current, err := cs.GetPresentationRevision(ctx, variant, initial.ActiveRevision)
		if err == nil {
			active = &current.Document
		}
	}
	if err := verifier(ctx, loaded.Document, active); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPresentationUnqualified, err)
	}
	cs.mu.Lock()
	defer cs.mu.Unlock()
	root, err := cs.openPresentationRoot(ctx, variant, false)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	unlock, err := cs.lockPresentationMutation(ctx, root, variant)
	if err != nil {
		return nil, err
	}
	defer unlock()
	state, err := readPresentationState(root, variant)
	if err != nil {
		return nil, err
	}
	if err := checkPresentationGeneration(state, expectedGeneration); err != nil {
		return nil, err
	}
	if rollback {
		if !slices.Contains(state.PublishedRevisions, revision) {
			return nil, ErrPresentationNotFound
		}
	} else if state.DraftRevision != revision {
		return nil, fmt.Errorf("%w: requested revision is not the current draft", ErrPresentationConflict)
	}
	// Integrity is checked again after verification. Mutation of stored immutable
	// bytes must not be hidden by a still-valid old in-memory document.
	if _, err := readPresentationRevision(root, variant, revision); err != nil {
		return nil, err
	}
	state.ActiveRevision = revision
	if !slices.Contains(state.PublishedRevisions, revision) {
		state.PublishedRevisions = append(state.PublishedRevisions, revision)
	}
	return cs.writePresentationState(ctx, root, variant, state)
}

func (cs *ConfigStore) openPresentationRoot(ctx context.Context, variant string, create bool) (*os.Root, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !variantSlugPattern.MatchString(variant) {
		return nil, fmt.Errorf("%w: invalid variant slug", ErrPresentationInvalid)
	}
	if cs.presentationRoots == nil {
		return nil, ErrPresentationNotFound
	}
	dir, err := cs.presentationRoots.PickRequired(ctx, storage.ClassConfig)
	if err != nil {
		return nil, err
	}
	if create {
		if err := storage.EnsureDirectory(dir, 0700); err != nil {
			return nil, err
		}
	}
	root, err := os.OpenRoot(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrPresentationNotFound
	}
	return root, err
}

// The native lock coordinates independent ConfigStore instances and processes.
// Its inode is never replaced or removed. Readers need no lock: head.json is
// atomically replaced and every referenced revision is immutable. A competing
// writer receives a retryable conflict instead of waiting under ConfigStore.mu.
func (cs *ConfigStore) lockPresentationMutation(ctx context.Context, root *os.Root, variant string) (func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	dir := filepath.Join("presentations", variant)
	if err := root.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	file, err := root.OpenFile(filepath.Join(dir, ".writer.lock"), os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return nil, err
	}
	cs.presentationRoots.RecordWrite(ctx)
	unlock, err := platform.LockFile(file, true)
	if err != nil {
		_ = file.Close()
		if errors.Is(err, platform.ErrLockUnavailable) {
			return nil, fmt.Errorf("%w: another editor is committing", ErrPresentationConflict)
		}
		return nil, fmt.Errorf("lock presentation writer: %w", err)
	}
	return func() { unlock(); _ = file.Close() }, nil
}

func readPresentationState(root *os.Root, variant string) (*PresentationState, error) {
	state := &PresentationState{PublishedRevisions: []string{}}
	data, err := readPresentationFile(root, filepath.Join("presentations", variant, "head.json"))
	if errors.Is(err, ErrPresentationNotFound) {
		return state, nil
	}
	if err != nil {
		return nil, err
	}
	if err := decodePresentation(data, state); err != nil {
		return nil, fmt.Errorf("%w: invalid head: %v", ErrPresentationCorrupt, err)
	}
	if state.Generation == 0 {
		return nil, fmt.Errorf("%w: zero stored generation", ErrPresentationCorrupt)
	}
	for _, revision := range append(append([]string{}, state.PublishedRevisions...), state.DraftRevision, state.ActiveRevision) {
		if revision != "" && !presentationRevisionPattern.MatchString(revision) {
			return nil, fmt.Errorf("%w: invalid revision pointer", ErrPresentationCorrupt)
		}
	}
	if state.ActiveRevision != "" && !slices.Contains(state.PublishedRevisions, state.ActiveRevision) {
		return nil, fmt.Errorf("%w: active revision was never published", ErrPresentationCorrupt)
	}
	return state, nil
}

func readPresentationRevision(root *os.Root, variant, revision string) (*PresentationRevision, error) {
	if !presentationRevisionPattern.MatchString(revision) {
		return nil, fmt.Errorf("%w: revision must be a SHA-256 digest", ErrPresentationInvalid)
	}
	data, err := readPresentationFile(root, presentationRevisionPath(variant, revision))
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(data)
	if hex.EncodeToString(digest[:]) != revision {
		return nil, fmt.Errorf("%w: document digest mismatch", ErrPresentationCorrupt)
	}
	var doc presentation.Document
	if err := decodePresentation(data, &doc); err != nil {
		return nil, fmt.Errorf("%w: invalid document: %v", ErrPresentationCorrupt, err)
	}
	return &PresentationRevision{Revision: revision, Document: doc}, nil
}

func readPresentationFile(root *os.Root, path string) ([]byte, error) {
	file, err := root.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrPresentationNotFound
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxPresentationBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxPresentationBytes {
		return nil, fmt.Errorf("%w: file exceeds size bound", ErrPresentationCorrupt)
	}
	return data, nil
}

func presentationRevisionPath(variant, revision string) string {
	return filepath.Join("presentations", variant, "revisions", revision+".json")
}

func decodePresentation(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode presentation: %w", err)
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return errors.New("presentation must contain exactly one JSON value")
	}
	return nil
}

func checkPresentationGeneration(state *PresentationState, expected uint64) error {
	if state.Generation != expected {
		return fmt.Errorf("%w: expected %d, current %d", ErrPresentationConflict, expected, state.Generation)
	}
	if state.Generation == math.MaxUint64 {
		return fmt.Errorf("%w: generation exhausted", ErrPresentationConflict)
	}
	return nil
}

func (cs *ConfigStore) writePresentationState(ctx context.Context, root *os.Root, variant string, state *PresentationState) (*PresentationState, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	state.Generation++
	data, err := json.Marshal(state)
	if err != nil {
		return nil, err
	}
	if err := storage.WriteFileAtomicInRoot(root, filepath.Join("presentations", variant, "head.json"), data, 0600); err != nil {
		return nil, fmt.Errorf("commit presentation pointer: %w", err)
	}
	cs.presentationRoots.RecordWrite(ctx)
	return state, nil
}
