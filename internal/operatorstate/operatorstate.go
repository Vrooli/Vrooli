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
	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/repocontractmeta"
)

const (
	SchemaPath                    = ".vrooli/schemas/operator-state.schema.json"
	StateFile                     = "operator-state.json"
	hostWorkloadPostureVrooliOnly = "vrooli_only"
)

type ScenarioChoice struct {
	Enabled     *bool `json:"enabled,omitempty"`
	AutoRestart *bool `json:"auto_restart,omitempty"`
}

type EnabledChoice struct {
	Enabled *bool `json:"enabled,omitempty"`
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
	Step int `json:"step"`
}

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
	UpdateControl       string                     `json:"update_control,omitempty"`
	Core                *CoreSet                   `json:"core,omitempty"`
	ActiveProfile       *string                    `json:"active_profile,omitempty"`
	Scenarios           map[string]ScenarioChoice  `json:"scenarios,omitempty"`
	Resources           map[string]EnabledChoice   `json:"resources,omitempty"`
	HostTools           map[string]OptInChoice     `json:"host_tools,omitempty"`
	HostSafeguards      map[string]OptInChoice     `json:"host_safeguards,omitempty"`
	Notifications       *NotificationsChoice       `json:"notifications,omitempty"`
	Completion          *Completion                `json:"completion,omitempty"`
	Session             *Session                   `json:"session,omitempty"`
	RawFields           map[string]json.RawMessage `json:"-"`
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
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.loadLocked(path)
}

func (s *Service) Apply(ctx context.Context, patch []byte) (Document, error) {
	return s.apply(ctx, patch, nil)
}

// ApplyValidated performs the read/merge/validate/write sequence while the
// path lock is held. Callers can add domain validation without creating a
// second, racy read-modify-write sequence.
func (s *Service) ApplyValidated(ctx context.Context, patch []byte, validate DocumentValidator) (Document, error) {
	return s.apply(ctx, patch, validate)
}

func (s *Service) apply(ctx context.Context, patch []byte, validate DocumentValidator) (Document, error) {
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
	s.lock.Lock()
	defer s.lock.Unlock()
	current, err := s.loadLocked(path)
	if err != nil {
		return Document{}, err
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
	next.UpdatedAt = s.cfg.Now().UTC().Format(time.RFC3339)
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
	return doc, nil
}

func Default() Document {
	return Document{
		Schema: SchemaPath, Version: "1.0.0",
		HostWorkloadPosture: hostWorkloadPostureVrooliOnly,
		Scenarios:           map[string]ScenarioChoice{}, Resources: map[string]EnabledChoice{},
		HostTools: map[string]OptInChoice{}, HostSafeguards: map[string]OptInChoice{},
		RawFields: map[string]json.RawMessage{},
	}
}

var knownFields = map[string]bool{
	"$schema": true, "version": true, "updated_at": true, "trust_posture": true, "host_workload_posture": true,
	"core": true, "active_profile": true, repocontractmeta.ScenarioDir: true, "resources": true,
	"host_tools": true, "host_safeguards": true, "completion": true, "session": true,
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
	if doc.TrustPosture != "" && doc.TrustPosture != "personal" && doc.TrustPosture != "shared" && doc.TrustPosture != "hosted" {
		return errors.New("operator state validation failed at /trust_posture: must be personal, shared, or hosted")
	}
	if doc.HostWorkloadPosture != "" && doc.HostWorkloadPosture != "whole_host" && doc.HostWorkloadPosture != hostWorkloadPostureVrooliOnly {
		return errors.New("operator state validation failed at /host_workload_posture: must be whole_host or vrooli_only")
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
	owned, err := marshalDocument(Document{Schema: doc.Schema, Version: doc.Version, UpdatedAt: doc.UpdatedAt, TrustPosture: doc.TrustPosture, HostWorkloadPosture: doc.HostWorkloadPosture, Core: doc.Core, ActiveProfile: doc.ActiveProfile, Scenarios: doc.Scenarios, Resources: doc.Resources, HostTools: doc.HostTools, HostSafeguards: doc.HostSafeguards, Completion: doc.Completion})
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
