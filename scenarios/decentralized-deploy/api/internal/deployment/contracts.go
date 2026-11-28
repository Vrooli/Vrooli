package deployment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

type Catalog struct {
	SchemaVersion int          `json:"schema_version"`
	SourcePolicy  string       `json:"source_policy"`
	Records       []PathRecord `json:"records"`
}

type PathRecord struct {
	Provider     string   `json:"provider"`
	Path         string   `json:"path"`
	Category     string   `json:"category"`
	Accepts      []string `json:"accepts"`
	KYC          string   `json:"kyc"`
	Minimum      string   `json:"minimum"`
	Result       string   `json:"result"`
	DeadEnd      string   `json:"dead_end"`
	Evidence     []string `json:"evidence"`
	UploadURL    string   `json:"upload_url,omitempty"`
	GatewayURL   string   `json:"gateway_url,omitempty"`
	MaxItemBytes int      `json:"max_item_bytes,omitempty"`
}

func LoadCatalog(r io.Reader) (Catalog, error) {
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	var catalog Catalog
	if err := decoder.Decode(&catalog); err != nil {
		return Catalog{}, fmt.Errorf("decode provider/path catalogue: %w", err)
	}
	if catalog.SchemaVersion != 1 || !strings.Contains(catalog.SourcePolicy, "recorded-fixtures-only") {
		return Catalog{}, errors.New("provider/path catalogue must use schema version 1 and recorded-fixture policy")
	}
	seen := map[string]bool{}
	for _, record := range catalog.Records {
		key := record.Provider + "/" + record.Path
		if record.Provider == "" || record.Path == "" || record.Category == "" || record.Result == "" || record.KYC == "" || record.Minimum == "" || record.DeadEnd == "" {
			return Catalog{}, fmt.Errorf("catalogue row %q is missing required evidence state", key)
		}
		if seen[key] {
			return Catalog{}, fmt.Errorf("duplicate catalogue row %q", key)
		}
		seen[key] = true
	}
	if len(catalog.Records) == 0 {
		return Catalog{}, errors.New("provider/path catalogue has no records")
	}
	return catalog, nil
}

func (c Catalog) Path(provider, path string) (PathRecord, error) {
	for _, record := range c.Records {
		if record.Provider == provider && record.Path == path {
			return record, nil
		}
	}
	return PathRecord{}, fmt.Errorf("provider path %s/%s is not recorded in the catalogue", provider, path)
}

type Effect string

const (
	EffectDevnet   Effect = "devnet"
	EffectSandbox  Effect = "sandbox"
	EffectFunded   Effect = "funded"
	EffectPublic   Effect = "public"
	EffectIdentity Effect = "identity-bound"
)

type GateMode string

const (
	GateStub GateMode = "stub"
	GateLive GateMode = "live"
)

// Gate is the shared effect boundary. Local devnet effects are allowed; all
// other effects require an explicit approval callback. Live mode is not an
// approval by itself.
type Gate struct {
	Mode             GateMode
	AllowFreeSandbox bool
	Authorize        func(context.Context, Effect, string) error
}

func (g Gate) Check(ctx context.Context, effect Effect, adapter string) error {
	if effect == EffectDevnet {
		return nil
	}
	if effect == EffectSandbox && g.AllowFreeSandbox {
		return nil
	}
	if g.Mode != GateLive || g.Authorize == nil {
		return fmt.Errorf("effect %q through %s refused: stub gate is active", effect, adapter)
	}
	if err := g.Authorize(ctx, effect, adapter); err != nil {
		return fmt.Errorf("effect %q through %s refused by authorization: %w", effect, adapter, err)
	}
	return nil
}

type Fault string

const (
	FaultNone    Fault = ""
	FaultRefuse  Fault = "refuse"
	FaultTimeout Fault = "timeout"
	FaultVanish  Fault = "vanish"
)

type Faults struct {
	mu      sync.RWMutex
	byName  map[string]Fault
	Timeout time.Duration
}

func NewFaults() *Faults { return &Faults{byName: map[string]Fault{}, Timeout: 30 * time.Second} }

func (f *Faults) Set(adapter string, fault Fault) error {
	if strings.TrimSpace(adapter) == "" {
		return errors.New("adapter is required")
	}
	if fault != FaultNone && fault != FaultRefuse && fault != FaultTimeout && fault != FaultVanish {
		return fmt.Errorf("unknown fault %q", fault)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if fault == FaultNone {
		delete(f.byName, adapter)
	} else {
		f.byName[adapter] = fault
	}
	return nil
}

func (f *Faults) Before(ctx context.Context, adapter string) error {
	f.mu.RLock()
	fault := f.byName[adapter]
	f.mu.RUnlock()
	switch fault {
	case FaultNone:
		return nil
	case FaultRefuse:
		return fmt.Errorf("provider %s refused the operation", adapter)
	case FaultVanish:
		return fmt.Errorf("provider %s became unavailable", adapter)
	case FaultTimeout:
		timeout := f.Timeout
		if timeout <= 0 {
			timeout = 30 * time.Second
		}
		timer := time.NewTimer(timeout)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return fmt.Errorf("provider %s timed out: %w", adapter, ctx.Err())
		case <-timer.C:
			return fmt.Errorf("provider %s timed out", adapter)
		}
	default:
		return fmt.Errorf("provider %s has unknown fault state %q", adapter, fault)
	}
}

type Artifact struct {
	Path            string
	Name            string
	DeploymentID    string
	StorageProvider string
	EntryPath       string
}

type StoredContent struct {
	Provider  string `json:"provider"`
	CID       string `json:"cid"`
	URI       string `json:"uri"`
	EntryPath string `json:"entry_path,omitempty"`
	RepoID    string `json:"repo_id,omitempty"`
}

type StorageAdapter interface {
	Name() string
	Deploy(context.Context, Artifact) (StoredContent, error)
	Renew(context.Context, StoredContent) error
	Teardown(context.Context, StoredContent) error
	Resolve(context.Context, StoredContent) ([]byte, error)
}

type storageArtifactProvider interface {
	ProviderForArtifact(Artifact) (string, error)
}

type storageContentProvider interface {
	ProviderForContent(StoredContent) (string, error)
}

func storageProviderForArtifact(storage StorageAdapter, artifact Artifact) (string, error) {
	if selected, ok := storage.(storageArtifactProvider); ok {
		return selected.ProviderForArtifact(artifact)
	}
	return storage.Name(), nil
}

func storageProviderForContent(storage StorageAdapter, content StoredContent) (string, error) {
	if selected, ok := storage.(storageContentProvider); ok {
		return selected.ProviderForContent(content)
	}
	return storage.Name(), nil
}

// StorageRouter dispatches lifecycle operations through the selected provider
// while preserving the existing DeploymentService and CLI contract.
type StorageRouter struct {
	Default  string
	Adapters map[string]StorageAdapter
}

func (r StorageRouter) Name() string { return "storage-router" }

func (r StorageRouter) ProviderForArtifact(artifact Artifact) (string, error) {
	adapter, err := r.adapter(artifact.StorageProvider)
	if err != nil {
		return "", err
	}
	return adapter.Name(), nil
}

func (r StorageRouter) ProviderForContent(content StoredContent) (string, error) {
	adapter, err := r.adapter(content.Provider)
	if err != nil {
		return "", err
	}
	return adapter.Name(), nil
}

func (r StorageRouter) adapter(provider string) (StorageAdapter, error) {
	if provider == "" {
		provider = r.Default
	}
	adapter := r.Adapters[provider]
	if adapter == nil {
		return nil, fmt.Errorf("storage provider %q is not configured", provider)
	}
	return adapter, nil
}

func (r StorageRouter) Deploy(ctx context.Context, artifact Artifact) (StoredContent, error) {
	adapter, err := r.adapter(artifact.StorageProvider)
	if err != nil {
		return StoredContent{}, err
	}
	return adapter.Deploy(ctx, artifact)
}

func (r StorageRouter) Renew(ctx context.Context, content StoredContent) error {
	adapter, err := r.adapter(content.Provider)
	if err != nil {
		return err
	}
	return adapter.Renew(ctx, content)
}

func (r StorageRouter) Teardown(ctx context.Context, content StoredContent) error {
	adapter, err := r.adapter(content.Provider)
	if err != nil {
		return err
	}
	return adapter.Teardown(ctx, content)
}

func (r StorageRouter) Resolve(ctx context.Context, content StoredContent) ([]byte, error) {
	adapter, err := r.adapter(content.Provider)
	if err != nil {
		return nil, err
	}
	return adapter.Resolve(ctx, content)
}

type NamedContent struct {
	Provider string `json:"provider"`
	Name     string `json:"name"`
	Target   string `json:"target"`
}

type NamingAdapter interface {
	Name() string
	Deploy(context.Context, string, string) (NamedContent, error)
	Renew(context.Context, NamedContent) error
	Teardown(context.Context, NamedContent) error
	Resolve(context.Context, NamedContent) (string, error)
}

type Record struct {
	ID               string         `json:"id"`
	ComputeReference string         `json:"compute_reference"`
	Storage          StoredContent  `json:"storage"`
	Names            []NamedContent `json:"names"`
	State            string         `json:"state"`
}

type Service struct {
	Gate    Gate
	Faults  *Faults
	Storage StorageAdapter
	Names   []NamingAdapter
	Records RecordRepository
	mu      sync.Mutex
}

func (s *Service) namingAdapter(provider string) (NamingAdapter, error) {
	for _, adapter := range s.Names {
		if adapter.Name() == provider {
			return adapter, nil
		}
	}
	return nil, fmt.Errorf("naming provider %q from deployment record is not configured", provider)
}

func NewService(g Gate, f *Faults, storage StorageAdapter, names ...NamingAdapter) (*Service, error) {
	return NewServiceWithRecords(g, f, storage, NewMemoryRecordRepository(), names...)
}

func NewServiceWithRecords(g Gate, f *Faults, storage StorageAdapter, records RecordRepository, names ...NamingAdapter) (*Service, error) {
	if f == nil || storage == nil {
		return nil, errors.New("fault hooks and a storage adapter are required")
	}
	if records == nil {
		return nil, errors.New("deployment record repository is required")
	}
	for i, n := range names {
		if n == nil {
			return nil, fmt.Errorf("naming adapter %d is nil", i)
		}
	}
	return &Service{Gate: g, Faults: f, Storage: storage, Names: names, Records: records}, nil
}

func (s *Service) Deploy(ctx context.Context, id string, artifact Artifact, name string) (Record, error) {
	return s.DeployWithComputeReference(ctx, id, artifact, name, "")
}

// DeployWithComputeReference records the Compute Manager-owned compute handle
// without selecting or provisioning a compute provider in this scenario.
func (s *Service) DeployWithComputeReference(ctx context.Context, id string, artifact Artifact, name, computeReference string) (Record, error) {
	if id == "" || name == "" || artifact.Path == "" {
		return Record{}, errors.New("deployment id, name, and artifact path are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.Records.Get(ctx, id); err == nil {
		return Record{}, fmt.Errorf("deployment %q already exists", id)
	} else if !errors.Is(err, ErrRecordNotFound) {
		return Record{}, fmt.Errorf("check deployment record: %w", err)
	}
	storageProvider, err := storageProviderForArtifact(s.Storage, artifact)
	if err != nil {
		return Record{}, err
	}
	if err := s.Gate.Check(ctx, EffectDevnet, storageProvider); err != nil {
		return Record{}, err
	}
	if err := s.Faults.Before(ctx, storageProvider); err != nil {
		return Record{}, err
	}
	artifact.DeploymentID = id
	stored, err := s.Storage.Deploy(ctx, artifact)
	if err != nil {
		return Record{}, fmt.Errorf("storage deploy: %w", err)
	}
	record := Record{ID: id, ComputeReference: strings.TrimSpace(computeReference), Storage: stored, State: "deployed"}
	for _, adapter := range s.Names {
		if err := s.Gate.Check(ctx, EffectDevnet, adapter.Name()); err != nil {
			_ = s.Storage.Teardown(ctx, stored)
			return Record{}, err
		}
		if err := s.Faults.Before(ctx, adapter.Name()); err != nil {
			_ = s.Storage.Teardown(ctx, stored)
			return Record{}, fmt.Errorf("naming deploy through %s: %w", adapter.Name(), err)
		}
		named, err := adapter.Deploy(ctx, name, stored.URI)
		if err != nil {
			for i := len(record.Names) - 1; i >= 0; i-- {
				_ = s.Names[i].Teardown(ctx, record.Names[i])
			}
			_ = s.Storage.Teardown(ctx, stored)
			return Record{}, fmt.Errorf("naming deploy through %s: %w", adapter.Name(), err)
		}
		record.Names = append(record.Names, named)
	}
	if err := s.Records.Put(ctx, record); err != nil {
		for i := len(record.Names) - 1; i >= 0; i-- {
			_ = s.Names[i].Teardown(ctx, record.Names[i])
		}
		_ = s.Storage.Teardown(ctx, stored)
		return Record{}, fmt.Errorf("persist deployment record: %w", err)
	}
	return record, nil
}

func (s *Service) Renew(ctx context.Context, id string) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, err := s.Records.Get(ctx, id)
	if err != nil {
		return Record{}, fmt.Errorf("read deployment record: %w", err)
	}
	if record.State != "deployed" && record.State != "renewed" {
		return Record{}, fmt.Errorf("deployment %q is not active", id)
	}
	storageProvider, err := storageProviderForContent(s.Storage, record.Storage)
	if err != nil {
		return Record{}, err
	}
	if err := s.Gate.Check(ctx, EffectDevnet, storageProvider); err != nil {
		return Record{}, err
	}
	if err := s.Faults.Before(ctx, storageProvider); err != nil {
		return Record{}, err
	}
	nameAdapters := make([]NamingAdapter, len(record.Names))
	for i, named := range record.Names {
		adapter, err := s.namingAdapter(named.Provider)
		if err != nil {
			return Record{}, err
		}
		nameAdapters[i] = adapter
	}
	if err := s.Storage.Renew(ctx, record.Storage); err != nil {
		return Record{}, fmt.Errorf("storage renew: %w", err)
	}
	for i, named := range record.Names {
		adapter := nameAdapters[i]
		if err := s.Gate.Check(ctx, EffectDevnet, adapter.Name()); err != nil {
			return Record{}, err
		}
		if err := s.Faults.Before(ctx, adapter.Name()); err != nil {
			return Record{}, err
		}
		if err := adapter.Renew(ctx, named); err != nil {
			return Record{}, fmt.Errorf("naming renew through %s: %w", adapter.Name(), err)
		}
	}
	record.State = "renewed"
	if err := s.Records.Put(ctx, record); err != nil {
		return Record{}, fmt.Errorf("persist renewed deployment record: %w", err)
	}
	return record, nil
}

func (s *Service) Teardown(ctx context.Context, id string) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, err := s.Records.Get(ctx, id)
	if err != nil {
		if !errors.Is(err, ErrRecordNotFound) {
			return Record{}, fmt.Errorf("read deployment record: %w", err)
		}
		return Record{}, fmt.Errorf("deployment %q was not found", id)
	}
	var errs []error
	for i := len(record.Names) - 1; i >= 0; i-- {
		adapter, err := s.namingAdapter(record.Names[i].Provider)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if err := s.Gate.Check(ctx, EffectDevnet, adapter.Name()); err != nil {
			errs = append(errs, err)
			continue
		}
		if err := s.Faults.Before(ctx, adapter.Name()); err != nil {
			errs = append(errs, err)
			continue
		}
		if err := adapter.Teardown(ctx, record.Names[i]); err != nil {
			errs = append(errs, fmt.Errorf("name teardown %s: %w", adapter.Name(), err))
		}
	}
	storageProvider, providerErr := storageProviderForContent(s.Storage, record.Storage)
	if providerErr != nil {
		errs = append(errs, providerErr)
	} else if err := s.Gate.Check(ctx, EffectDevnet, storageProvider); err != nil {
		errs = append(errs, err)
	} else if err := s.Faults.Before(ctx, storageProvider); err != nil {
		errs = append(errs, err)
	} else if err := s.Storage.Teardown(ctx, record.Storage); err != nil {
		errs = append(errs, fmt.Errorf("storage teardown: %w", err))
	}
	if len(errs) > 0 {
		return record, errors.Join(errs...)
	}
	record.State = "torn-down"
	if err := s.Records.Put(ctx, record); err != nil {
		return Record{}, fmt.Errorf("persist torn-down deployment record: %w", err)
	}
	return record, nil
}

func (s *Service) Resolve(ctx context.Context, id string) (Record, []byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, err := s.Records.Get(ctx, id)
	if err != nil {
		return Record{}, nil, fmt.Errorf("read deployment record: %w", err)
	}
	if record.State == "torn-down" {
		return Record{}, nil, fmt.Errorf("deployment %q is not resolvable", id)
	}
	storageProvider, err := storageProviderForContent(s.Storage, record.Storage)
	if err != nil {
		return Record{}, nil, err
	}
	if err := s.Gate.Check(ctx, EffectDevnet, storageProvider); err != nil {
		return Record{}, nil, err
	}
	if err := s.Faults.Before(ctx, storageProvider); err != nil {
		return Record{}, nil, err
	}
	for _, named := range record.Names {
		adapter, err := s.namingAdapter(named.Provider)
		if err != nil {
			return Record{}, nil, err
		}
		if err := s.Gate.Check(ctx, EffectDevnet, adapter.Name()); err != nil {
			return Record{}, nil, err
		}
		if err := s.Faults.Before(ctx, adapter.Name()); err != nil {
			return Record{}, nil, err
		}
		target, err := adapter.Resolve(ctx, named)
		if err != nil {
			return Record{}, nil, err
		}
		if target != record.Storage.URI {
			return Record{}, nil, fmt.Errorf("name %s resolved to %q, want %q", named.Name, target, record.Storage.URI)
		}
	}
	body, err := s.Storage.Resolve(ctx, record.Storage)
	if err != nil {
		return Record{}, nil, err
	}
	return record, body, nil
}
