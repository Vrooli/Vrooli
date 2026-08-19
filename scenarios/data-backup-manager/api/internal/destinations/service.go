package destinations

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"data-backup-manager/internal/engine"
)

// defaultListLimit caps List when the caller passes 0.
const defaultListLimit = 100

// nearCapThreshold is the fraction of cap_bytes at which usage transitions from
// WITHIN to NEAR (i.e. >= 90 % of cap).
const nearCapThreshold = 0.90

// Service is the application surface the destinations handlers depend on. It
// owns validation, the separate-root rule, encryption-on proof, and the
// cap-decision method the runs domain calls.
type Service interface {
	// CreateDestination validates and creates a destination, provisioning the
	// underlying kopia repository. Returns ErrInvalidDestination on validation
	// failure (including separate-root violation).
	CreateDestination(ctx context.Context, in CreateInput) (Destination, error)

	// GetDestination returns a destination by id. ErrDestinationNotFound
	// propagates verbatim.
	GetDestination(ctx context.Context, id string) (Destination, error)

	// ListDestinations returns up to pageSize destinations ordered by name.
	// pageSize <= 0 uses the default.
	ListDestinations(ctx context.Context, pageSize int) ([]Destination, error)

	// UpdateDestination updates the mutable fields (cap_bytes, cap_policy) of an
	// existing destination.
	UpdateDestination(ctx context.Context, in UpdateInput) (Destination, error)

	// DeleteDestination removes a destination's catalog row. When
	// deleteRepository is true, it first removes local resource-kopia metadata
	// and credential-authority refs for the underlying repository.
	DeleteDestination(ctx context.Context, id string, deleteRepository bool) (bool, error)

	// GetDestinationUsage returns current usage bytes, cap, state, and policy
	// for the given destination.
	GetDestinationUsage(ctx context.Context, id string) (UsageReport, error)

	// WouldBlock reports whether a write of pendingBytes to the given destination
	// would be blocked by the cap policy. With CAP_POLICY_ALERT_BLOCK and
	// cap_bytes > 0, returns blocked=true when current usage + pendingBytes
	// exceeds cap_bytes. CAP_POLICY_ALERT_ONLY never blocks; cap_bytes == 0
	// (no cap) never blocks. WouldBlock never deletes anything.
	WouldBlock(ctx context.Context, destinationID string, pendingBytes int64) (blocked bool, reason string, err error)

	// ReconcileCredentialReferences upgrades legacy destination references and
	// generated filesystem recovery metadata to the canonical authority
	// identity. It is safe to run on every boot; a read-only destination remains
	// pending for a later retry.
	ReconcileCredentialReferences(ctx context.Context) error
}

// UsageReport is the result type returned by GetDestinationUsage.
type UsageReport struct {
	UsageBytes int64
	CapBytes   int64
	UsageState UsageState
	CapPolicy  CapPolicy
}

type service struct {
	repo          Repository
	eng           engine.KopiaEngine
	bundle        BundleWriter
	protectedRoot string
}

// NewService constructs the production Service. bundle materializes the
// self-describing filesystem destination bundle (README/RECOVERY/manifest +
// repositories/<slug>.kopia); pass nil for an S3-only deployment or a fake in
// tests.
func NewService(repo Repository, eng engine.KopiaEngine, bundle BundleWriter, protectedRoot string) Service {
	return &service{
		repo:          repo,
		eng:           eng,
		bundle:        bundle,
		protectedRoot: filepath.Clean(protectedRoot),
	}
}

// Compile-time guarantee.
var _ Service = (*service)(nil)

func (s *service) CreateDestination(ctx context.Context, in CreateInput) (Destination, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return Destination{}, ErrInvalidDestination{Field: "name", Reason: "required"}
	}
	// A destination name doubles as its kopia repository name, so it must be a
	// stable slug — never an arbitrary label with spaces or slashes.
	if !ValidSlug(name) {
		return Destination{}, ErrInvalidDestination{
			Field:  "name",
			Reason: "must be slug-safe: lowercase letters, digits and hyphens only (e.g. elements-local)",
		}
	}
	if !in.Backend.Valid() {
		return Destination{}, ErrInvalidDestination{Field: "backend_kind", Reason: "must be filesystem or s3"}
	}
	location := strings.TrimSpace(in.Location)
	if location == "" {
		return Destination{}, ErrInvalidDestination{Field: "location", Reason: "required"}
	}

	capPolicy := in.CapPolicy
	if capPolicy == "" {
		capPolicy = CapPolicyAlertBlock
	}

	// repositoryLocation is the concrete backend path resource-kopia targets.
	// For filesystem it nests under the operator-facing bundle root; for S3 it
	// is the bucket/prefix itself.
	var repositoryLocation string

	// Build the kopia RepoSpec and create the repository.
	spec := engine.RepoSpec{
		Name:    name,
		Backend: string(in.Backend),
	}
	switch in.Backend {
	case BackendFilesystem:
		// Separate-root rule: a filesystem destination must not point under the
		// protectedRoot (the storage root the manager protects).
		clean := filepath.Clean(location)
		if clean == s.protectedRoot || strings.HasPrefix(clean, s.protectedRoot+string(filepath.Separator)) {
			return Destination{}, ErrInvalidDestination{
				Field:  "location",
				Reason: "filesystem destination must not point inside the protected root",
			}
		}
		repositoryLocation = RepositoryPathFor(location, name)
		// Materialize the bundle root + repository directory before creating the
		// repository so kopia writes into repositories/<slug>.kopia, not the bare
		// operator-facing root.
		if s.bundle != nil {
			if err := s.bundle.PrepareRepository(ctx, location, repositoryLocation); err != nil {
				return Destination{}, err
			}
		}
		spec.Path = repositoryLocation
	case BackendS3:
		repositoryLocation = location
		spec.Bucket = location
	}

	if err := s.eng.RepoCreate(ctx, spec); err != nil {
		return Destination{}, err
	}

	// RepoStatus proves encryption is on (EncryptionAlgorithm must be non-empty).
	status, err := s.eng.RepoStatus(ctx, name)
	if err != nil {
		return Destination{}, err
	}
	if status.EncryptionAlgorithm == "" {
		return Destination{}, ErrInvalidDestination{
			Field:  "encryption_algorithm",
			Reason: "kopia repository did not report an encryption algorithm; encryption must be on",
		}
	}

	// The encryption passphrase is generated and owned by resource-kopia; DBM
	// records only the deterministic credential-authority *reference* (never the value) so
	// a detached bundle can point an operator at the key for standalone recovery.
	d := Destination{
		Name:                name,
		BackendKind:         in.Backend,
		Location:            location,
		RepositoryLocation:  repositoryLocation,
		CapBytes:            in.CapBytes,
		CapPolicy:           capPolicy,
		EncryptionAlgorithm: status.EncryptionAlgorithm,
		SecretRef:           s.eng.PassphraseRef(name),
	}

	saved, err := s.repo.Create(ctx, d)
	if err != nil {
		return Destination{}, err
	}

	// Write the human-facing bundle files now that the destination id and
	// encryption algorithm are known. Filesystem backends only; never carries a
	// secret value.
	if in.Backend == BackendFilesystem && s.bundle != nil {
		if err := s.bundle.WriteMetadata(ctx, BundleMetadata{
			DestinationID:       saved.ID,
			Name:                saved.Name,
			Backend:             string(saved.BackendKind),
			BundleRoot:          saved.Location,
			RepositoryPath:      saved.RepositoryLocation,
			EncryptionAlgorithm: saved.EncryptionAlgorithm,
			SecretRef:           saved.SecretRef,
			CreatedAt:           saved.CreatedAt,
			Host:                hostname(),
			User:                username(),
		}); err != nil {
			return Destination{}, err
		}
	}

	return saved, nil
}

// hostname / username best-effort host identity for the manifest. Empty on
// error — they are optional, non-secret context only.
func hostname() string {
	h, err := os.Hostname()
	if err != nil {
		return ""
	}
	return h
}

func username() string {
	if u := os.Getenv("USER"); u != "" {
		return u
	}
	return os.Getenv("USERNAME")
}

func (s *service) GetDestination(ctx context.Context, id string) (Destination, error) {
	if strings.TrimSpace(id) == "" {
		return Destination{}, ErrInvalidDestination{Field: "id", Reason: "required"}
	}
	return s.repo.GetByID(ctx, id)
}

func (s *service) ListDestinations(ctx context.Context, pageSize int) ([]Destination, error) {
	if pageSize <= 0 {
		pageSize = defaultListLimit
	}
	return s.repo.List(ctx, pageSize)
}

func (s *service) UpdateDestination(ctx context.Context, in UpdateInput) (Destination, error) {
	if strings.TrimSpace(in.ID) == "" {
		return Destination{}, ErrInvalidDestination{Field: "id", Reason: "required"}
	}
	existing, err := s.repo.GetByID(ctx, in.ID)
	if err != nil {
		return Destination{}, err
	}
	existing.CapBytes = in.CapBytes
	if in.CapPolicy != "" {
		existing.CapPolicy = in.CapPolicy
	}
	return s.repo.Update(ctx, existing)
}

func (s *service) ReconcileCredentialReferences(ctx context.Context) error {
	dests, err := s.repo.List(ctx, defaultListLimit)
	if err != nil {
		return fmt.Errorf("list destinations for credential-reference reconciliation: %w", err)
	}
	var failures []string
	for _, d := range dests {
		canonical := s.eng.PassphraseRef(d.Name)
		if canonical == "" || d.SecretRef == canonical {
			continue
		}
		// The catalog is the authoritative runtime record consumed by DBM's
		// orchestration. Update it even when a read-only filesystem prevents the
		// generated bundle metadata from being refreshed; leaving the catalog on
		// a retired provider-shaped reference makes a healthy repository appear
		// unavailable and keeps Vault paths alive in shipped/read APIs.
		d.SecretRef = canonical
		if _, err := s.repo.Update(ctx, d); err != nil {
			failures = append(failures, fmt.Sprintf("%s catalog: %v", d.Name, err))
			continue
		}
		if d.BackendKind == BackendFilesystem && s.bundle != nil {
			if err := s.bundle.RefreshMetadata(ctx, BundleMetadata{
				DestinationID:       d.ID,
				Name:                d.Name,
				Backend:             string(d.BackendKind),
				BundleRoot:          d.Location,
				RepositoryPath:      d.RepositoryLocation,
				EncryptionAlgorithm: d.EncryptionAlgorithm,
				SecretRef:           canonical,
				CreatedAt:           d.CreatedAt,
			}); err != nil {
				failures = append(failures, fmt.Sprintf("%s metadata: %v", d.Name, err))
			}
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("destination credential-reference reconciliation incomplete: %s", strings.Join(failures, "; "))
	}
	return nil
}

func (s *service) DeleteDestination(ctx context.Context, id string, deleteRepository bool) (bool, error) {
	if strings.TrimSpace(id) == "" {
		return false, ErrInvalidDestination{Field: "id", Reason: "required"}
	}
	if deleteRepository {
		d, err := s.repo.GetByID(ctx, id)
		if err != nil {
			return false, err
		}
		if err := s.eng.RepoDelete(ctx, d.Name); err != nil {
			return false, err
		}
	}
	return s.repo.Delete(ctx, id)
}

func (s *service) GetDestinationUsage(ctx context.Context, id string) (UsageReport, error) {
	d, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return UsageReport{}, err
	}
	stats, err := s.eng.RepoStats(ctx, d.Name)
	if err != nil {
		return UsageReport{}, err
	}
	state := computeUsageState(stats.SizeBytes, d.CapBytes)
	return UsageReport{
		UsageBytes: stats.SizeBytes,
		CapBytes:   d.CapBytes,
		UsageState: state,
		CapPolicy:  d.CapPolicy,
	}, nil
}

func (s *service) WouldBlock(ctx context.Context, destinationID string, pendingBytes int64) (bool, string, error) {
	d, err := s.repo.GetByID(ctx, destinationID)
	if err != nil {
		return false, "", err
	}
	// No cap or alert-only policy: never block.
	if d.CapBytes == 0 || d.CapPolicy == CapPolicyAlertOnly {
		return false, "", nil
	}
	// ALERT_BLOCK: block when current usage + pending would exceed cap.
	stats, err := s.eng.RepoStats(ctx, d.Name)
	if err != nil {
		return false, "", err
	}
	if stats.SizeBytes+pendingBytes > d.CapBytes {
		return true, "storage cap exceeded; write blocked by alert+block policy", nil
	}
	return false, "", nil
}

// computeUsageState classifies usage against cap. cap == 0 means no cap → always WITHIN.
func computeUsageState(usageBytes, capBytes int64) UsageState {
	if capBytes == 0 {
		return UsageStateWithin
	}
	if usageBytes >= capBytes {
		return UsageStateOver
	}
	if float64(usageBytes) >= float64(capBytes)*nearCapThreshold {
		return UsageStateNear
	}
	return UsageStateWithin
}
