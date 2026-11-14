package coreactivation

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/artifactlock"
	"github.com/vrooli/vrooli/internal/config"
)

const (
	imageLimit      = 512 << 20
	definitionLimit = 64 << 10
)

// Layout is frozen from the canonical runtime-home and native unit owners at
// admission. Recovery uses these paths without consulting a source checkout.
// Its digest is part of Record; changed path metadata is never repair input.
type Layout struct {
	Home       string `json:"home"`
	Executable string `json:"executable"`
	UnitPath   string `json:"unit_path"`
	UnitName   string `json:"unit_name"`
	LogPath    string `json:"log_path"`
}

func (l Layout) validate(home string) error {
	unit, _ := platform.CoreUnitByID(platform.CoreUnitRuntimeSupervisor)
	if l.Home != home || l.UnitName != unit.NativeName("linux") || l.UnitPath != platform.RuntimeSupervisorUnitPath("linux", home) || filepath.Base(l.Executable) != "vrooli" {
		return fmt.Errorf("core activation frozen layout does not identify the native root owner")
	}
	for _, path := range []string{l.Executable, l.UnitPath, l.LogPath} {
		if !filepath.IsAbs(path) || filepath.Clean(path) != path {
			return ErrConflict
		}
		rel, err := filepath.Rel(home, path)
		if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return fmt.Errorf("core activation frozen path is outside its user home")
		}
	}
	return nil
}

// InputSources are already resolved, qualified local image paths. Freeze does
// not execute these files or claim build-role/state compatibility qualification.
type InputSources struct {
	Candidate, Prior, Worker string
	CandidateUnit, PriorUnit []byte
}

type InputDigests struct {
	Layout, Candidate, Prior, Worker, CandidateUnit, PriorUnit string
}

type Preparer struct {
	mu     sync.Mutex
	store  *Store
	active bool
}

// Prepare serializes private input creation through admission and dispatch.
// The callback must retain the operation's identity on an ambiguous result.
// It must not call Recover while holding the root binary install lock.
func (s *Store) Prepare(ctx context.Context, fn func(*Preparer) error) error {
	if fn == nil {
		return ErrConflict
	}
	release, err := artifactlock.AcquireWithPathContext(ctx, "core-activation-preparation:"+s.home, func(string) (string, error) { return filepath.Join(s.Directory(), "preparation.lock"), nil })
	if err != nil {
		return err
	}
	defer release()
	current, exists, err := s.Current()
	if err != nil {
		return err
	}
	if exists && !current.Terminal() {
		return fmt.Errorf("%w: another root activation remains unresolved", ErrConflict)
	}
	p := &Preparer{store: s, active: true}
	defer func() { p.mu.Lock(); p.active = false; p.mu.Unlock() }()
	return fn(p)
}

func (s *Store) inputDirectory(id string) (string, error) {
	if !idPattern.MatchString(id) {
		return "", ErrConflict
	}
	return filepath.Join(s.Directory(), "inputs", id), nil
}

// Freeze writes one immutable private input set. An existing identity is never
// overwritten, including after a lost acknowledgement. The caller qualifies
// these frozen bytes and admits their digests before any publication effect.
func (p *Preparer) Freeze(ctx context.Context, id string, l Layout, in InputSources) (InputDigests, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.active || p.store == nil {
		return InputDigests{}, ErrConflict
	}
	if ctx == nil {
		return InputDigests{}, ErrConflict
	}
	if err := ctx.Err(); err != nil {
		return InputDigests{}, err
	}
	if err := l.validate(p.store.home); err != nil {
		return InputDigests{}, err
	}
	dir, err := p.store.inputDirectory(id)
	if err != nil {
		return InputDigests{}, err
	}
	if _, err := config.EnsureOwnedDir(filepath.Dir(dir)); err != nil {
		return InputDigests{}, err
	}
	current, exists, err := p.store.Current()
	if err != nil {
		return InputDigests{}, err
	}
	if exists && (!current.Terminal() || current.ID == id) {
		return InputDigests{}, ErrConflict
	}
	if err := p.store.pruneUnadmittedInputs(id, current.ID); err != nil {
		return InputDigests{}, err
	}
	if err := os.Mkdir(dir, 0o700); err != nil {
		return InputDigests{}, fmt.Errorf("core activation input identity already exists or cannot be created")
	}
	// Do not remove this directory on an ambiguous durable write. The next
	// owner must inspect the same operation, never infer absence from return.
	created, err := os.Open(dir)
	if err != nil {
		return InputDigests{}, err
	}
	if err := config.ChownFileToInvokingUser(created); err != nil {
		_ = created.Close()
		return InputDigests{}, err
	}
	if err := created.Close(); err != nil {
		return InputDigests{}, err
	}
	layout, err := json.Marshal(l)
	if err != nil {
		return InputDigests{}, err
	}
	d := InputDigests{Layout: bytesDigest(layout), CandidateUnit: bytesDigest(in.CandidateUnit), PriorUnit: bytesDigest(in.PriorUnit)}
	for _, item := range []struct {
		name string
		data []byte
	}{{"layout.json", layout}, {"candidate.unit", in.CandidateUnit}, {"prior.unit", in.PriorUnit}} {
		if len(item.data) == 0 || len(item.data) > definitionLimit {
			return InputDigests{}, fmt.Errorf("core activation frozen definition exceeds its bounds")
		}
		if err := config.WriteOwnedFileDurable(filepath.Join(dir, item.name), item.data, 0o600); err != nil {
			return InputDigests{}, err
		}
	}
	for _, item := range []struct {
		name, source string
		digest       *string
	}{{"candidate", in.Candidate, &d.Candidate}, {"prior", in.Prior, &d.Prior}, {"worker", in.Worker, &d.Worker}} {
		sha, err := copyFrozenImage(ctx, item.source, filepath.Join(dir, item.name), "", 0o700)
		if err != nil {
			return InputDigests{}, err
		}
		*item.digest = sha
	}
	return d, nil
}

func bytesDigest(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }

// Assets is a source-independent reader/publisher for one admitted input set.
// The native adapter remains responsible for build-role/compatibility proof,
// install locking, scheduler actions and exact native tree exclusion.
type Assets struct {
	Store          *Store
	confirmRemoval func(string) error
}

func (a Assets) path(r Record, name string) (string, error) {
	if a.Store == nil || r.Home != a.Store.home {
		return "", ErrConflict
	}
	dir, err := a.Store.inputDirectory(r.ID)
	if err != nil {
		return "", err
	}
	switch name {
	case "layout.json", "candidate", "prior", "worker", "candidate.unit", "prior.unit":
		return filepath.Join(dir, name), nil
	}
	return "", ErrConflict
}

func readFrozen(path string, limit int64) ([]byte, error) {
	f, err := openFrozen(path, limit)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil || len(data) == 0 || int64(len(data)) > limit {
		return nil, fmt.Errorf("core activation frozen input exceeded its bounds")
	}
	return data, nil
}

func openFrozen(path string, limit int64) (*os.File, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > limit {
		return nil, fmt.Errorf("core activation input must be a bounded nonempty regular file")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	opened, err := f.Stat()
	if err != nil || !os.SameFile(info, opened) {
		_ = f.Close()
		return nil, fmt.Errorf("core activation input changed while opening")
	}
	return f, nil
}

func (a Assets) Layout(r Record) (Layout, error) {
	path, err := a.path(r, "layout.json")
	if err != nil {
		return Layout{}, err
	}
	data, err := readFrozen(path, definitionLimit)
	if err != nil {
		return Layout{}, err
	}
	if bytesDigest(data) != r.LayoutSHA {
		return Layout{}, ErrConflict
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := validateJSONShape(decoder, reflect.TypeFor[Layout](), 0); err != nil {
		return Layout{}, err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return Layout{}, ErrConflict
	}
	var l Layout
	if err := json.Unmarshal(data, &l); err != nil {
		return Layout{}, ErrConflict
	}
	return l, l.validate(r.Home)
}

func (a Assets) Definition(r Record, candidate bool) ([]byte, error) {
	name, expected := "prior.unit", r.PriorUnitSHA
	if candidate {
		name, expected = "candidate.unit", r.CandidateUnitSHA
	}
	path, err := a.path(r, name)
	if err != nil {
		return nil, err
	}
	data, err := readFrozen(path, definitionLimit)
	if err != nil {
		return nil, err
	}
	if bytesDigest(data) != expected {
		return nil, ErrConflict
	}
	return data, nil
}

func (a Assets) Image(ctx context.Context, r Record, name string) (string, error) {
	path, err := a.path(r, name)
	if err != nil {
		return "", err
	}
	var expected string
	switch name {
	case "candidate":
		expected = r.CandidateSHA
	case "prior":
		expected = r.PriorSHA
	case "worker":
		expected = r.WorkerSHA
	default:
		return "", ErrConflict
	}
	sha, err := imageDigest(ctx, path)
	if err != nil {
		return "", err
	}
	if sha != expected {
		return "", ErrConflict
	}
	return path, nil
}

func imageDigest(ctx context.Context, path string) (string, error) {
	if ctx == nil {
		return "", ErrConflict
	}
	f, err := openFrozen(path, imageLimit)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(&contextReader{ctx: ctx, reader: f}, imageLimit+1))
	if err != nil {
		return "", err
	}
	if n <= 0 || n > imageLimit {
		return "", ErrConflict
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// ImageDigest applies the same regular-file and size bounds used by frozen
// input admission to an owner's installed executable observation.
func ImageDigest(ctx context.Context, path string) (string, error) { return imageDigest(ctx, path) }

// VerifyInstalled checks both installed targets against one frozen side. It
// never accepts an unexpected definition merely because its process is healthy.
func (a Assets) VerifyInstalled(ctx context.Context, r Record, candidate bool) error {
	l, err := a.Layout(r)
	if err != nil {
		return err
	}
	expectedImage, expectedUnit := r.PriorSHA, r.PriorUnitSHA
	if candidate {
		expectedImage, expectedUnit = r.CandidateSHA, r.CandidateUnitSHA
	}
	image, err := imageDigest(ctx, l.Executable)
	if err != nil {
		return err
	}
	unit, err := readFrozen(l.UnitPath, definitionLimit)
	if err != nil {
		return err
	}
	if image != expectedImage || bytesDigest(unit) != expectedUnit {
		return ErrConflict
	}
	return nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}

func copyFrozenImage(ctx context.Context, source, destination, expected string, mode os.FileMode) (string, error) {
	if ctx == nil {
		return "", ErrConflict
	}
	f, err := openFrozen(source, imageLimit)
	if err != nil {
		return "", err
	}
	defer f.Close()
	var sha string
	err = config.WriteOwnedFileDurableStream(destination, mode, func(w io.Writer) error {
		h := sha256.New()
		n, err := io.Copy(io.MultiWriter(w, h), io.LimitReader(&contextReader{ctx: ctx, reader: f}, imageLimit+1))
		if err != nil {
			return err
		}
		if n <= 0 || n > imageLimit {
			return ErrConflict
		}
		sha = hex.EncodeToString(h.Sum(nil))
		if expected != "" && sha != expected {
			return ErrConflict
		}
		return nil
	})
	return sha, err
}

func (a Assets) publicationGuard(r Record, restoring bool, phase Phase) error {
	if a.Store == nil {
		return ErrConflict
	}
	current, exists, err := a.Store.Current()
	if err != nil {
		return err
	}
	if !exists || current.ID != r.ID || current.LayoutSHA != r.LayoutSHA || current.CandidateSHA != r.CandidateSHA || current.PriorSHA != r.PriorSHA || current.CandidateUnitSHA != r.CandidateUnitSHA || current.PriorUnitSHA != r.PriorUnitSHA || current.WorkerSHA != r.WorkerSHA {
		return ErrConflict
	}
	if restoring {
		if current.Phase != Restoring || current.Fence != Revoked || !current.RestorationRequired {
			return ErrClaimed
		}
	} else if current.Phase != phase || current.Fence != Claimed {
		return ErrClaimed
	}
	return nil
}

// PublishBinary requires the adapter's shared root install lock. Unexpected
// current bytes are a conflict; only the frozen current/prior images may be
// replaced. Missing bytes are restorable only after a durable restore intent.
func (a Assets) PublishBinary(ctx context.Context, r Record, restoring bool) error {
	if err := a.publicationGuard(r, restoring, Publishing); err != nil {
		return err
	}
	l, err := a.Layout(r)
	if err != nil {
		return err
	}
	if err := checkInstalledInputs(ctx, r, l, restoring); err != nil {
		return err
	}
	name, sha := "candidate", r.CandidateSHA
	if restoring {
		name, sha = "prior", r.PriorSHA
	}
	source, err := a.path(r, name)
	if err != nil {
		return err
	}
	_, err = copyFrozenImage(ctx, source, l.Executable, sha, 0o755)
	return err
}

// PublishDefinition changes only the frozen native definition. The adapter
// validates native syntax and serializes daemon reload/start/stop separately.
// It must first exclude the candidate's pending/native tree when restoring.
func (a Assets) PublishDefinition(ctx context.Context, r Record, restoring bool) error {
	if ctx == nil {
		return ErrConflict
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := a.publicationGuard(r, restoring, Activating); err != nil {
		return err
	}
	l, err := a.Layout(r)
	if err != nil {
		return err
	}
	if err := checkInstalledInputs(ctx, r, l, restoring); err != nil {
		return err
	}
	data, err := a.Definition(r, !restoring)
	if err != nil {
		return err
	}
	return config.WriteOwnedFileDurable(l.UnitPath, data, 0o644)
}

// CheckPublicationInputs refuses foreign installed bytes before a native stop
// or reload, as well as before the later atomic publication itself.
func (a Assets) CheckPublicationInputs(ctx context.Context, r Record, restoring bool) error {
	l, err := a.Layout(r)
	if err != nil {
		return err
	}
	return checkInstalledInputs(ctx, r, l, restoring)
}

func checkInstalledInputs(ctx context.Context, r Record, l Layout, restoring bool) error {
	image, err := imageDigest(ctx, l.Executable)
	if err != nil && !(restoring && errors.Is(err, os.ErrNotExist)) {
		return err
	}
	if err == nil && image != r.CandidateSHA && image != r.PriorSHA {
		return ErrConflict
	}
	data, err := readFrozen(l.UnitPath, definitionLimit)
	if err != nil && !(restoring && errors.Is(err, os.ErrNotExist)) {
		return err
	}
	if err == nil && bytesDigest(data) != r.CandidateUnitSHA && bytesDigest(data) != r.PriorUnitSHA {
		return ErrConflict
	}
	return ctx.Err()
}

func boundedEntries(dir string) ([]os.DirEntry, error) {
	info, err := os.Lstat(dir)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, ErrConflict
	}
	f, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return nil, ErrConflict
	}
	entries, err := f.ReadDir(65)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	if len(entries) > 64 {
		return nil, fmt.Errorf("core activation private input inventory exceeds its bound")
	}
	return entries, nil
}

// removeInputDirectory touches only this domain's reserved regular files and
// bounded atomic-write temporaries. It refuses unexpected and nonregular files
// before deleting anything; unknown content requires owner inspection.
func removeInputDirectory(dir string, confirm func(string) error) error {
	entries, err := boundedEntries(dir)
	if errors.Is(err, os.ErrNotExist) {
		return confirm(filepath.Dir(dir))
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		switch entry.Name() {
		case "layout.json", "candidate", "prior", "worker", "candidate.unit", "prior.unit":
		default:
			if !strings.HasPrefix(entry.Name(), ".vrooli-owned-") {
				return ErrConflict
			}
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			return ErrConflict
		}
	}
	for _, entry := range entries {
		if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	if err := os.Remove(dir); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return confirm(filepath.Dir(dir))
}

func confirmRemovedInputs(parent string) error {
	for {
		f, err := os.Open(parent)
		if errors.Is(err, os.ErrNotExist) {
			next := filepath.Dir(parent)
			if next == parent {
				return err
			}
			parent = next
			continue
		}
		if err != nil {
			return err
		}
		syncErr := f.Sync()
		closeErr := f.Close()
		if syncErr != nil {
			return syncErr
		}
		return closeErr
	}
}

func (s *Store) pruneUnadmittedInputs(requestID, currentID string) error {
	root := filepath.Join(s.Directory(), "inputs")
	entries, err := boundedEntries(root)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !idPattern.MatchString(entry.Name()) || !entry.IsDir() {
			return ErrConflict
		}
	}
	for _, entry := range entries {
		if entry.Name() == requestID || entry.Name() == currentID {
			continue
		}
		if err := removeInputDirectory(filepath.Join(root, entry.Name()), confirmRemovedInputs); err != nil {
			return err
		}
	}
	return nil
}

// Retire requires the native adapter to have excluded/released the exact
// worker first. Terminal state alone does not prove a native tree is empty.
func (a Assets) Retire(r Record) error {
	if a.Store == nil || !r.Terminal() {
		return ErrConflict
	}
	current, exists, err := a.Store.Current()
	if err != nil {
		return err
	}
	if !exists || current.ID != r.ID || !current.Terminal() || current.Fence != Revoked {
		return ErrConflict
	}
	dir, err := a.Store.inputDirectory(r.ID)
	if err != nil {
		return err
	}
	confirm := a.confirmRemoval
	if confirm == nil {
		confirm = confirmRemovedInputs
	}
	return removeInputDirectory(dir, confirm)
}
