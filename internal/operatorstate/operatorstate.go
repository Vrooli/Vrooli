// Package operatorstate owns the mutable operator-state document.
//
// The document is intentionally written through a merge-patch boundary. A
// caller only names the decisions it owns, while fields introduced by a newer
// schema or another writer remain opaque and survive the round trip.
package operatorstate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	"github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/buildinfo"
	capacityengine "github.com/vrooli/vrooli/internal/capacity"
	"github.com/vrooli/vrooli/internal/repocontractmeta"
)

const (
	SchemaPath                    = ".vrooli/schemas/operator-state.schema.json"
	StateFile                     = "operator-state.json"
	DefaultDraftRetention         = 30 * 24 * time.Hour
	hostWorkloadPostureVrooliOnly = "vrooli_only"
	capacityPostureBalanced       = "balanced"
	AccelPreferenceAuto           = "auto"
	AccelPreferencePreferGPU      = "prefer_gpu"
	AccelPreferenceForceCPU       = "force_cpu"
)

type ScenarioChoice struct {
	Enabled     *bool `json:"enabled,omitempty"`
	AutoRestart *bool `json:"auto_restart,omitempty"`
}

type EnabledChoice struct {
	Enabled  *bool           `json:"enabled,omitempty"`
	Capacity *CapacityChoice `json:"capacity,omitempty"`
}

type CapacityChoice struct {
	Rung             string         `json:"rung,omitempty"`
	Tunables         map[string]any `json:"tunables,omitempty"`
	GPUIndex         *int           `json:"gpu_index,omitempty"`
	Priority         string         `json:"priority,omitempty"`
	YieldWhenIdle    *bool          `json:"yield_when_idle,omitempty"`
	IdleGraceSeconds *int           `json:"idle_grace_seconds,omitempty"`
}

type CapacitySettings struct {
	TransientHeadroomReserveBytes *int64 `json:"transient_headroom_reserve_bytes,omitempty"`
}

// ValidateCapacityChoices keeps manifest-backed capacity validation beside
// the operator-state write authority. Scenario clients should not import the
// capacity engine merely to validate this document's fields.
func ValidateCapacityChoices(repoRoot string, doc Document) error {
	manifests, err := capacityengine.LoadDeclaredResources(repoRoot)
	if err != nil {
		return err
	}
	choices := make(map[string]capacityengine.FitResourceChoice, len(doc.Resources))
	for name, resource := range doc.Resources {
		if resource.Capacity == nil {
			continue
		}
		choices[name] = capacityengine.FitResourceChoice{
			Rung: resource.Capacity.Rung, Tunables: resource.Capacity.Tunables,
			Priority: resource.Capacity.Priority, GPUIndex: resource.Capacity.GPUIndex,
		}
	}
	return capacityengine.ValidateResourceCapacityChoices(manifests, choices)
}

type OptInChoice struct {
	OptedIn *bool          `json:"opted_in,omitempty"`
	Config  map[string]any `json:"config,omitempty"`
}

type CoreSet struct {
	Seed        []string `json:"seed"`
	TrustedBase []string `json:"trusted_base"`
}

type Completion struct {
	SelectionDigest string `json:"selection_digest"`
	AppliedAt       string `json:"applied_at"`
	// DegradedAcknowledgement records that the operator accepted a specific set
	// of degraded optional items. It carries the digest of the set that was
	// acknowledged rather than a bare flag, so an acknowledgement of one gap
	// cannot authorise completion over a different gap later, and a page reload
	// cannot silently discard it.
	DegradedAcknowledgement *DegradedAcknowledgement `json:"degraded_acknowledgement,omitempty"`
}

type DegradedAcknowledgement struct {
	ReadinessDigest string `json:"readiness_digest"`
	AcknowledgedAt  string `json:"acknowledged_at"`
}

type Session struct {
	Step    int             `json:"step"`
	StepID  string          `json:"step_id,omitempty"`
	Profile *ProfileSession `json:"profile,omitempty"`
}

// ProfileSession is the durable, non-secret guided/manual selection context.
// Answers are JSON values because profile questions have typed controls; raw
// credential input and executable data are rejected by validateProfileSession.
type ProfileSession struct {
	Target          string                     `json:"target"`
	Actor           string                     `json:"actor"`
	Mode            string                     `json:"mode"`
	ProfileID       string                     `json:"profile_id,omitempty"`
	ProfileVersion  string                     `json:"profile_version,omitempty"`
	CatalogRevision string                     `json:"catalog_revision,omitempty"`
	BaseRevision    string                     `json:"base_revision"`
	Answers         map[string]json.RawMessage `json:"answers,omitempty"`
	ManualDecisions map[string]bool            `json:"manual_decisions,omitempty"`
	TargetContext   map[string]string          `json:"target_context,omitempty"`
	UpdatedAt       string                     `json:"updated_at"`
}

var ErrProfileSessionConflict = errors.New("profile session conflict")

func validateProfileSession(value *ProfileSession) error {
	if value == nil {
		return nil
	}
	if strings.TrimSpace(value.Target) == "" || len(value.Target) > 256 {
		return errors.New("profile session target is required and must be at most 256 characters")
	}
	if strings.TrimSpace(value.Actor) == "" || len(value.Actor) > 512 {
		return errors.New("profile session actor is required and must be at most 512 characters")
	}
	if value.Mode != "guided" && value.Mode != "manual" {
		return errors.New("profile session mode must be guided or manual")
	}
	if len(value.ProfileID) > 128 || len(value.ProfileVersion) > 64 || len(value.CatalogRevision) > 256 || len(value.BaseRevision) > 256 {
		return errors.New("profile session revision or profile reference is too long")
	}
	if len(value.Answers) > 250 {
		return errors.New("profile session answers exceed the maximum of 250 entries")
	}
	for key, answer := range value.Answers {
		if err := validateProfileSessionKey(key); err != nil {
			return err
		}
		if len(answer) == 0 || len(answer) > 16*1024 || !json.Valid(answer) {
			return fmt.Errorf("profile session answer %q must be valid JSON of at most 16 KiB", key)
		}
		var decoded any
		if err := json.Unmarshal(answer, &decoded); err != nil {
			return fmt.Errorf("profile session answer %q: %w", key, err)
		}
		if !profileAnswerValueAllowed(decoded) {
			return fmt.Errorf("profile session answer %q must be a scalar or an array of scalars", key)
		}
	}
	if len(value.ManualDecisions) > 256 {
		return errors.New("profile session manual decisions exceed the maximum of 256 entries")
	}
	for key := range value.ManualDecisions {
		if err := validateProfileSessionKey(key); err != nil {
			return err
		}
	}
	if len(value.TargetContext) > 64 {
		return errors.New("profile session target context exceeds the maximum of 64 entries")
	}
	for key, item := range value.TargetContext {
		if err := validateProfileSessionKey(key); err != nil {
			return err
		}
		if len(item) > 512 {
			return fmt.Errorf("profile session target context %q is too long", key)
		}
	}
	if _, err := time.Parse(time.RFC3339Nano, value.UpdatedAt); err != nil {
		return fmt.Errorf("profile session updated_at must be RFC3339: %w", err)
	}
	return nil
}

func validateProfileSessionKey(key string) error {
	key = strings.TrimSpace(key)
	if key == "" || len(key) > 128 {
		return errors.New("profile session key must be between 1 and 128 characters")
	}
	for _, secretWord := range []string{"password", "passphrase", "secret", "token", "private_key", "credential"} {
		if strings.Contains(strings.ToLower(key), secretWord) {
			return fmt.Errorf("profile session key %q may not contain secret material", key)
		}
	}
	return nil
}

func profileAnswerValueAllowed(value any) bool {
	switch item := value.(type) {
	case nil, bool, string, float64:
		return true
	case []any:
		for _, element := range item {
			if !profileAnswerValueAllowed(element) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// Draft is a target- and actor-scoped, non-secret working copy. Drafts are
// deliberately separate from effective operator choices: saving one must not
// start a service, grant a permission, or change host state.
type Draft struct {
	Target       string            `json:"target"`
	Actor        string            `json:"actor"`
	BaseRevision string            `json:"base_revision"`
	Revision     string            `json:"revision"`
	StepID       string            `json:"step_id,omitempty"`
	Choices      map[string]string `json:"choices,omitempty"`
	UpdatedAt    string            `json:"updated_at"`
}

var ErrDraftConflict = errors.New("operator draft conflict")

// NotificationsChoice names the person the host's notifications go to.
// notification-hub reads it at event intake; a host with no recipient records
// every incident notification as unroutable with this setting named.
type NotificationsChoice struct {
	Recipient string `json:"recipient,omitempty"`
}

// Document is the typed projection of operator-state.json. RawFields holds
// fields not yet understood by this binary. Their JSON bytes are retained so
// a patch from this version cannot erase a field owned by a newer version.
type Document struct {
	Schema              string                     `json:"$schema,omitempty"`
	Version             string                     `json:"version"`
	UpdatedAt           string                     `json:"updated_at"`
	TrustPosture        string                     `json:"trust_posture,omitempty"`
	HostWorkloadPosture string                     `json:"host_workload_posture,omitempty"`
	CapacityPosture     string                     `json:"capacity_posture,omitempty"`
	AccelPreference     string                     `json:"accel_preference,omitempty"`
	UpdateControl       string                     `json:"update_control,omitempty"`
	Core                *CoreSet                   `json:"core,omitempty"`
	ActiveProfile       *string                    `json:"active_profile,omitempty"`
	Scenarios           map[string]ScenarioChoice  `json:"scenarios,omitempty"`
	Resources           map[string]EnabledChoice   `json:"resources,omitempty"`
	Capacity            *CapacitySettings          `json:"capacity,omitempty"`
	HostTools           map[string]OptInChoice     `json:"host_tools,omitempty"`
	HostSafeguards      map[string]OptInChoice     `json:"host_safeguards,omitempty"`
	Notifications       *NotificationsChoice       `json:"notifications,omitempty"`
	Completion          *Completion                `json:"completion,omitempty"`
	Session             *Session                   `json:"session,omitempty"`
	Drafts              map[string]Draft           `json:"drafts,omitempty"`
	RawFields           map[string]json.RawMessage `json:"-"`
}

func (d Document) EffectiveAccelPreference() string {
	switch d.AccelPreference {
	case AccelPreferenceAuto, AccelPreferencePreferGPU, AccelPreferenceForceCPU:
		return d.AccelPreference
	}
	if d.CapacityPosture == "minimal" {
		return AccelPreferenceForceCPU
	}
	return AccelPreferenceAuto
}

func (d Document) EffectiveUpdateControl() string {
	switch d.UpdateControl {
	case "observe", "guard", "own":
		return d.UpdateControl
	default:
		return "observe"
	}
}

type Config struct {
	RepoRoot    string
	StorageRoot string
	SchemaPath  string
	Roots       *filerouting.RoutedRoots
	// StatePath is reserved for isolated tests and embedded callers that own
	// their storage seam. Production callers should provide RepoRoot or
	// StorageRoot and let this package resolve the path.
	StatePath func(context.Context) (string, error)
	Now       func() time.Time
}

type DocumentValidator func(Document) error

type Service struct {
	cfg  Config
	lock *sync.Mutex
}

var locks sync.Map

var ErrRevisionConflict = errors.New("operator state revision conflict")

func New(cfg Config) *Service {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	key := filepath.Clean(strings.TrimSpace(cfg.RepoRoot))
	if key == "." || key == "" {
		key = filepath.Clean(strings.TrimSpace(cfg.StorageRoot))
	}
	if key == "." || key == "" {
		key = "operator-state-default"
	}
	actual, _ := locks.LoadOrStore(key, &sync.Mutex{})
	return &Service{cfg: cfg, lock: actual.(*sync.Mutex)}
}

func (s *Service) Path(ctx context.Context) (string, error) {
	if s == nil {
		return "", errors.New("operator state service is nil")
	}
	if s.cfg.StatePath != nil && s.cfg.Roots == nil {
		return s.cfg.StatePath(ctx)
	}
	if s.cfg.Roots != nil {
		root, err := s.cfg.Roots.Pick(ctx, storage.ClassConfig)
		if err != nil {
			return "", err
		}
		return filepath.Join(root, StateFile), nil
	}
	if root := strings.TrimSpace(s.cfg.RepoRoot); root != "" {
		return filepath.Join(root, filepath.Dir(filepath.Join(repocontractmeta.ProjectConfigDir, StateFile)), StateFile), nil
	}
	if root := strings.TrimSpace(s.cfg.StorageRoot); root != "" {
		return filepath.Join(root, StateFile), nil
	}
	return "", fmt.Errorf("%s or VROOLI_STORAGE_ROOT is required to locate operator state", buildinfo.SourceRootFallbackEnvVar)
}

func SchemaDir() string { return filepath.Dir(SchemaPath) }

func (s *Service) Load(ctx context.Context) (Document, error) {
	path, err := s.Path(ctx)
	if err != nil {
		return Document{}, err
	}
	if err := s.lockContext(ctx); err != nil {
		return Document{}, err
	}
	defer s.lock.Unlock()
	release, err := acquireStateFileLock(ctx, path)
	if err != nil {
		return Document{}, err
	}
	defer release()
	return s.loadLocked(path)
}

func (s *Service) Apply(ctx context.Context, patch []byte) (Document, error) {
	return s.applyAtRevision(ctx, "", patch, nil)
}

// ApplyAtRevision applies a field-scoped patch only when the durable document
// still has expectedRevision. The check and write share the same process and
// native file lock, so a stale caller cannot erase a newer choice.
func (s *Service) ApplyAtRevision(ctx context.Context, expectedRevision string, patch []byte) (Document, error) {
	return s.applyAtRevision(ctx, expectedRevision, patch, nil)
}

// SaveProfileSession persists the non-secret guided/manual context without
// changing effective operator choices. The session is revision-checked so a
// resumed client cannot overwrite another actor's newer selection context.
func (s *Service) SaveProfileSession(ctx context.Context, value ProfileSession, expectedRevision string) (Document, error) {
	if strings.TrimSpace(value.BaseRevision) == "" {
		return Document{}, errors.New("profile session base revision is required")
	}
	value.Target = strings.TrimSpace(value.Target)
	value.Actor = strings.TrimSpace(value.Actor)
	value.Mode = strings.TrimSpace(value.Mode)
	value.UpdatedAt = s.cfg.Now().UTC().Format(time.RFC3339Nano)
	if err := validateProfileSession(&value); err != nil {
		return Document{}, err
	}
	current, err := s.Load(ctx)
	if err != nil {
		return Document{}, err
	}
	if expectedRevision != "" && Revision(current) != expectedRevision {
		return Document{}, fmt.Errorf("%w: expected %q, found %q", ErrRevisionConflict, expectedRevision, Revision(current))
	}
	if current.Session == nil {
		current.Session = &Session{}
	}
	patch, err := json.Marshal(map[string]any{"session": map[string]any{"profile": value}})
	if err != nil {
		return Document{}, err
	}
	return s.ApplyAtRevision(ctx, Revision(current), patch)
}

// ProfileSession returns a copy of the durable profile context. Callers may
// safely mutate the returned maps while preparing a new revision.
func (s *Service) ProfileSession(ctx context.Context) (*ProfileSession, error) {
	document, err := s.Load(ctx)
	if err != nil {
		return nil, err
	}
	if document.Session == nil || document.Session.Profile == nil {
		return nil, nil
	}
	value := *document.Session.Profile
	value.Answers = cloneRawMessageMap(value.Answers)
	value.ManualDecisions = cloneBoolMap(value.ManualDecisions)
	value.TargetContext = cloneStringMap(value.TargetContext)
	return &value, nil
}

// DraftKey is stable across UI, CLI, and API clients. The actor is explicit so
// a draft from one authenticated operator cannot appear when another operator
// or target is selected.
func DraftKey(target, actor string) string {
	return strings.TrimSpace(target) + "::" + strings.TrimSpace(actor)
}

func validateDraftInput(target, actor, baseRevision, stepID string, choices map[string]string) error {
	if strings.TrimSpace(target) == "" {
		return errors.New("draft target is required")
	}
	if strings.TrimSpace(actor) == "" {
		return errors.New("draft actor is required")
	}
	if strings.TrimSpace(baseRevision) == "" {
		return errors.New("draft base revision is required")
	}
	if len(choices) > 256 {
		return errors.New("draft choices exceed the maximum of 256 entries")
	}
	for key, value := range choices {
		key = strings.ToLower(strings.TrimSpace(key))
		if key == "" || len(key) > 128 || len(value) > 4096 {
			return errors.New("draft choice has an invalid size")
		}
		for _, secretWord := range []string{"password", "passphrase", "secret", "token", "private_key", "credential"} {
			if strings.Contains(key, secretWord) {
				return fmt.Errorf("draft choice %q may not contain secret material", key)
			}
		}
	}
	if len(stepID) > 128 {
		return errors.New("draft step id is too long")
	}
	return nil
}

// SaveDraft persists only safe, non-secret choices under the state authority.
// expectedRevision is the document revision observed by the client. A stale
// client is rejected atomically and its existing draft remains inspectable.
func (s *Service) SaveDraft(ctx context.Context, target, actor, expectedRevision, baseRevision, stepID string, choices map[string]string) (Document, error) {
	if err := validateDraftInput(target, actor, baseRevision, stepID, choices); err != nil {
		return Document{}, err
	}
	current, err := s.Load(ctx)
	if err != nil {
		return Document{}, err
	}
	if expectedRevision != "" && Revision(current) != expectedRevision {
		return Document{}, fmt.Errorf("%w: expected %q, found %q", ErrDraftConflict, expectedRevision, Revision(current))
	}
	drafts := make(map[string]Draft, len(current.Drafts)+1)
	for key, draft := range current.Drafts {
		drafts[key] = draft
	}
	key := DraftKey(target, actor)
	prior := drafts[key]
	if prior.Revision != "" && expectedRevision == "" {
		return Document{}, fmt.Errorf("%w: an existing draft must be saved with its observed revision", ErrDraftConflict)
	}
	now := s.cfg.Now().UTC().Format(time.RFC3339Nano)
	drafts[key] = Draft{Target: strings.TrimSpace(target), Actor: strings.TrimSpace(actor), BaseRevision: strings.TrimSpace(baseRevision), Revision: now, StepID: strings.TrimSpace(stepID), Choices: cloneStringMap(choices), UpdatedAt: now}
	patch, err := json.Marshal(map[string]any{"drafts": drafts})
	if err != nil {
		return Document{}, err
	}
	return s.ApplyAtRevision(ctx, Revision(current), patch)
}

func (s *Service) DiscardDraft(ctx context.Context, target, actor string) (Document, error) {
	if strings.TrimSpace(target) == "" || strings.TrimSpace(actor) == "" {
		return Document{}, errors.New("draft target and actor are required")
	}
	current, err := s.Load(ctx)
	if err != nil {
		return Document{}, err
	}
	drafts := make(map[string]any, len(current.Drafts)+1)
	for key, draft := range current.Drafts {
		drafts[key] = draft
	}
	drafts[DraftKey(target, actor)] = nil
	patch, err := json.Marshal(map[string]any{"drafts": drafts})
	if err != nil {
		return Document{}, err
	}
	return s.ApplyAtRevision(ctx, Revision(current), patch)
}

// PruneDrafts removes only drafts whose explicit UpdatedAt timestamp is older
// than maxAge. Malformed timestamps are retained for manual recovery, and this
// method never changes effective operator choices.
func (s *Service) PruneDrafts(ctx context.Context, maxAge time.Duration) (Document, error) {
	if maxAge <= 0 {
		return Document{}, errors.New("draft retention must be positive")
	}
	current, err := s.Load(ctx)
	if err != nil {
		return Document{}, err
	}
	if len(current.Drafts) == 0 {
		return current, nil
	}
	cutoff := s.cfg.Now().UTC().Add(-maxAge)
	drafts := make(map[string]any, len(current.Drafts))
	changed := false
	for key, draft := range current.Drafts {
		updated, parseErr := time.Parse(time.RFC3339Nano, draft.UpdatedAt)
		if parseErr == nil && updated.Before(cutoff) {
			changed = true
			drafts[key] = nil
			continue
		}
		drafts[key] = draft
	}
	if !changed {
		return current, nil
	}
	patch, err := json.Marshal(map[string]any{"drafts": drafts})
	if err != nil {
		return Document{}, err
	}
	return s.ApplyAtRevision(ctx, Revision(current), patch)
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	result := make(map[string]string, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

func cloneBoolMap(values map[string]bool) map[string]bool {
	if len(values) == 0 {
		return nil
	}
	result := make(map[string]bool, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

func cloneRawMessageMap(values map[string]json.RawMessage) map[string]json.RawMessage {
	if len(values) == 0 {
		return nil
	}
	result := make(map[string]json.RawMessage, len(values))
	for key, value := range values {
		result[key] = append(json.RawMessage(nil), value...)
	}
	return result
}

// ApplyValidated performs the read/merge/validate/write sequence while the
// path lock is held. Callers can add domain validation without creating a
// second, racy read-modify-write sequence.
func (s *Service) ApplyValidated(ctx context.Context, patch []byte, validate DocumentValidator) (Document, error) {
	return s.applyAtRevision(ctx, "", patch, validate)
}

// ApplyAtRevisionValidated combines optimistic concurrency with the domain
// validator while keeping the read/merge/validate/write sequence atomic.
func (s *Service) ApplyAtRevisionValidated(ctx context.Context, expectedRevision string, patch []byte, validate DocumentValidator) (Document, error) {
	return s.applyAtRevision(ctx, expectedRevision, patch, validate)
}

func (s *Service) applyAtRevision(ctx context.Context, expectedRevision string, patch []byte, validate DocumentValidator) (Document, error) {
	if !json.Valid(patch) {
		return Document{}, errors.New("operator-state merge patch is invalid JSON")
	}
	var patchValue map[string]json.RawMessage
	if err := json.Unmarshal(patch, &patchValue); err != nil || patchValue == nil {
		return Document{}, errors.New("operator-state merge patch must be a JSON object")
	}
	path, err := s.Path(ctx)
	if err != nil {
		return Document{}, err
	}
	if err := s.lockContext(ctx); err != nil {
		return Document{}, err
	}
	defer s.lock.Unlock()
	release, err := acquireStateFileLock(ctx, path)
	if err != nil {
		return Document{}, err
	}
	defer release()
	current, err := s.loadLocked(path)
	if err != nil {
		return Document{}, err
	}
	if expectedRevision != "" && Revision(current) != expectedRevision {
		return Document{}, fmt.Errorf("%w: expected %q, found %q", ErrRevisionConflict, expectedRevision, Revision(current))
	}
	currentBytes, err := marshalDocument(current)
	if err != nil {
		return Document{}, err
	}
	merged, err := mergePatch(currentBytes, patch)
	if err != nil {
		return Document{}, err
	}
	var next Document
	if err := unmarshalDocument(merged, &next); err != nil {
		return Document{}, fmt.Errorf("decode merged operator state: %w", err)
	}
	next.Schema = SchemaPath
	if next.Version == "" {
		next.Version = "1.0.0"
	}
	next.UpdatedAt = nextRevisionTimestamp(current.UpdatedAt, s.cfg.Now()).Format(time.RFC3339Nano)
	if err := s.validate(merged, next); err != nil {
		return Document{}, err
	}
	if validate != nil {
		if err := validate(next); err != nil {
			return Document{}, err
		}
	}
	data, err := marshalDocument(next)
	if err != nil {
		return Document{}, err
	}
	if err := storage.WriteFileAtomic(path, append(data, '\n'), storage.SecretFilePerm); err != nil {
		return Document{}, fmt.Errorf("write operator state: %w", err)
	}
	if s.cfg.Roots != nil {
		s.cfg.Roots.RecordWrite(ctx)
	}
	return next, nil
}

func nextRevisionTimestamp(previous string, now time.Time) time.Time {
	next := now.UTC()
	if prior, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(previous)); err == nil && !next.After(prior) {
		return prior.Add(time.Nanosecond)
	}
	return next
}

func (s *Service) lockContext(ctx context.Context) error {
	for !s.lock.TryLock() {
		timer := time.NewTimer(10 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return nil
}

func acquireStateFileLock(ctx context.Context, path string) (func(), error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("prepare operator state lock directory: %w", err)
	}
	lockPath := filepath.Join(filepath.Dir(path), "."+filepath.Base(path)+".lock")
	release, err := platform.AcquireFileLockContext(ctx, lockPath)
	if err != nil {
		return nil, fmt.Errorf("lock operator state: %w", err)
	}
	return release, nil
}

// Revision returns the durable optimistic-concurrency token for a document.
// UpdatedAt is the normal token; the version fallback keeps an untouched
// historical document addressable before its first write.
func Revision(doc Document) string {
	if value := strings.TrimSpace(doc.UpdatedAt); value != "" {
		return value
	}
	return strings.TrimSpace(doc.Version)
}

func (s *Service) MarkApplied(ctx context.Context, selectionDigest string, at time.Time) (Document, error) {
	if at.IsZero() {
		at = s.cfg.Now()
	}
	patch, err := json.Marshal(map[string]any{
		"completion": map[string]any{
			"selection_digest": selectionDigest,
			"applied_at":       at.UTC().Format(time.RFC3339),
		},
	})
	if err != nil {
		return Document{}, err
	}
	return s.Apply(ctx, patch)
}

// RecordDegradedAcknowledgement stores the operator's acceptance of one named
// set of degraded optional items. The digest is computed by the caller from the
// sorted names of that set.
func (s *Service) RecordDegradedAcknowledgement(ctx context.Context, readinessDigest string, at time.Time) (Document, error) {
	if strings.TrimSpace(readinessDigest) == "" {
		return Document{}, fmt.Errorf("degraded acknowledgement requires a readiness digest")
	}
	if at.IsZero() {
		at = s.cfg.Now()
	}
	patch, err := json.Marshal(map[string]any{
		"completion": map[string]any{
			"degraded_acknowledgement": map[string]any{
				"readiness_digest": readinessDigest,
				"acknowledged_at":  at.UTC().Format(time.RFC3339),
			},
		},
	})
	if err != nil {
		return Document{}, err
	}
	return s.Apply(ctx, patch)
}

// Effective returns the current operator choices. Manifest defaults are
// resolved by the catalog consumers; this method is the single read boundary
// for the durable override values.
func (s *Service) Effective(ctx context.Context) (Document, error) { return s.Load(ctx) }

func (s *Service) loadLocked(path string) (Document, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Default(), nil
	}
	if err != nil {
		return Document{}, fmt.Errorf("read operator state: %w", err)
	}
	var doc Document
	if err := unmarshalDocument(data, &doc); err != nil {
		return Document{}, fmt.Errorf("decode operator state: %w", err)
	}
	if strings.TrimSpace(doc.Version) == "" {
		return Document{}, errors.New("operator state version is required")
	}
	if strings.TrimSpace(doc.UpdatedAt) != "" {
		if _, err := time.Parse(time.RFC3339, doc.UpdatedAt); err != nil {
			return Document{}, fmt.Errorf("operator state updated_at must be RFC3339: %w", err)
		}
	}
	if doc.CapacityPosture == "" {
		doc.CapacityPosture = capacityPostureBalanced
	}
	return doc, nil
}

func Default() Document {
	return Document{
		Schema: SchemaPath, Version: "1.0.0",
		HostWorkloadPosture: hostWorkloadPostureVrooliOnly,
		CapacityPosture:     capacityPostureBalanced,
		Scenarios:           map[string]ScenarioChoice{}, Resources: map[string]EnabledChoice{},
		HostTools: map[string]OptInChoice{}, HostSafeguards: map[string]OptInChoice{},
		Drafts:    map[string]Draft{},
		RawFields: map[string]json.RawMessage{},
	}
}

var knownFields = map[string]bool{
	"$schema": true, "version": true, "updated_at": true, "trust_posture": true, "host_workload_posture": true, "capacity_posture": true, "accel_preference": true,
	"core": true, "active_profile": true, repocontractmeta.ScenarioDir: true, "resources": true, "capacity": true,
	"host_tools": true, "host_safeguards": true, "completion": true, "session": true, "drafts": true,
}

func unmarshalDocument(data []byte, doc *Document) error {
	type plain Document
	var decoded plain
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	raw := make(map[string]json.RawMessage)
	for key, value := range fields {
		if !knownFields[key] {
			raw[key] = append(json.RawMessage(nil), value...)
		}
	}
	*doc = Document(decoded)
	if doc.HostWorkloadPosture == "" {
		doc.HostWorkloadPosture = hostWorkloadPostureVrooliOnly
	}
	doc.RawFields = raw
	return nil
}

func marshalDocument(doc Document) ([]byte, error) {
	type plain Document
	copyDoc := plain(doc)
	copyDoc.RawFields = nil
	data, err := json.Marshal(copyDoc)
	if err != nil {
		return nil, err
	}
	if len(doc.RawFields) == 0 {
		return data, nil
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, err
	}
	for key, value := range doc.RawFields {
		if _, owned := knownFields[key]; !owned {
			fields[key] = value
		}
	}
	return json.Marshal(fields)
}

func mergePatch(document, patch []byte) ([]byte, error) {
	var target any
	var patchValue any
	if err := json.Unmarshal(document, &target); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(patch, &patchValue); err != nil {
		return nil, err
	}
	merged := applyMerge(target, patchValue)
	return json.Marshal(merged)
}

func applyMerge(target, patch any) any {
	patchObject, ok := patch.(map[string]any)
	if !ok {
		return patch
	}
	targetObject, _ := target.(map[string]any)
	if targetObject == nil {
		targetObject = map[string]any{}
	}
	for key, value := range patchObject {
		if value == nil {
			delete(targetObject, key)
			continue
		}
		if nested, ok := value.(map[string]any); ok {
			targetObject[key] = applyMerge(targetObject[key], nested)
			continue
		}
		targetObject[key] = value
	}
	return targetObject
}

func (s *Service) validate(merged []byte, doc Document) error {
	if doc.Session != nil {
		if err := validateProfileSession(doc.Session.Profile); err != nil {
			return fmt.Errorf("operator state validation failed at /session/profile: %w", err)
		}
	}
	if doc.TrustPosture != "" && doc.TrustPosture != "personal" && doc.TrustPosture != "shared" && doc.TrustPosture != "hosted" {
		return errors.New("operator state validation failed at /trust_posture: must be personal, shared, or hosted")
	}
	if doc.HostWorkloadPosture != "" && doc.HostWorkloadPosture != "whole_host" && doc.HostWorkloadPosture != hostWorkloadPostureVrooliOnly {
		return errors.New("operator state validation failed at /host_workload_posture: must be whole_host or vrooli_only")
	}
	if doc.CapacityPosture != "" && doc.CapacityPosture != "responsive" && doc.CapacityPosture != capacityPostureBalanced && doc.CapacityPosture != "throughput" && doc.CapacityPosture != "minimal" {
		return errors.New("operator state validation failed at /capacity_posture: must be responsive, balanced, throughput, or minimal")
	}
	if doc.AccelPreference != "" && doc.AccelPreference != AccelPreferenceAuto && doc.AccelPreference != AccelPreferencePreferGPU && doc.AccelPreference != AccelPreferenceForceCPU {
		return errors.New("operator state validation failed at /accel_preference: must be auto, prefer_gpu, or force_cpu")
	}
	schemaPath := strings.TrimSpace(s.cfg.SchemaPath)
	if schemaPath == "" && strings.TrimSpace(s.cfg.RepoRoot) != "" {
		schemaPath = filepath.Join(s.cfg.RepoRoot, SchemaPath)
	}
	if schemaPath == "" {
		return validateRequired(doc)
	}
	schemaBytes, err := os.ReadFile(schemaPath)
	if errors.Is(err, os.ErrNotExist) {
		return validateRequired(doc)
	}
	if err != nil {
		return fmt.Errorf("read operator state schema: %w", err)
	}
	compiler := jsonschema.NewCompiler()
	if common, readErr := os.ReadFile(filepath.Join(filepath.Dir(schemaPath), "common.schema.json")); readErr == nil {
		if err := compiler.AddResource("common.schema.json", bytes.NewReader(common)); err != nil {
			return fmt.Errorf("compile operator state schema dependency: %w", err)
		}
		if err := compiler.AddResource("https://vrooli.com/schemas/common.schema.json", bytes.NewReader(common)); err != nil {
			return fmt.Errorf("compile operator state schema dependency: %w", err)
		}
	}
	if err := compiler.AddResource(filepath.Base(schemaPath), bytes.NewReader(schemaBytes)); err != nil {
		return fmt.Errorf("compile operator state schema: %w", err)
	}
	schema, err := compiler.Compile(filepath.Base(schemaPath))
	if err != nil {
		return fmt.Errorf("compile operator state schema: %w", err)
	}
	// Validate the fields owned by this version. RawFields are deliberately
	// excluded: they are future schema fields and must survive this writer.
	owned, err := marshalDocument(Document{Schema: doc.Schema, Version: doc.Version, UpdatedAt: doc.UpdatedAt, TrustPosture: doc.TrustPosture, HostWorkloadPosture: doc.HostWorkloadPosture, Core: doc.Core, ActiveProfile: doc.ActiveProfile, Scenarios: doc.Scenarios, Resources: doc.Resources, HostTools: doc.HostTools, HostSafeguards: doc.HostSafeguards, Completion: doc.Completion, Session: doc.Session, Drafts: doc.Drafts})
	if err != nil {
		return err
	}
	if err := schema.Validate(mustJSON(owned)); err != nil {
		return fmt.Errorf("operator state validation failed: %w", err)
	}
	_ = merged
	return nil
}

func validateRequired(doc Document) error {
	if strings.TrimSpace(doc.Version) == "" {
		return errors.New("operator state validation failed at /version: is required")
	}
	if _, err := time.Parse(time.RFC3339, doc.UpdatedAt); err != nil {
		return fmt.Errorf("operator state validation failed at /updated_at: %v", err)
	}
	return nil
}

func mustJSON(data []byte) any {
	var value any
	_ = json.Unmarshal(data, &value)
	return value
}

// WithTestMode is a small convenience for callers that use api-core's routed
// file roots. It keeps the service's write accounting explicit at the seam.
func WithTestMode(ctx context.Context) context.Context { return database.WithTestMode(ctx) }
