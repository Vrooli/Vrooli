package executionevidence

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

const SchemaVersion = 1

var (
	ErrUnsupportedSchema = errors.New("unsupported execution evidence schema")
	ErrCorruptEvidence   = errors.New("corrupt execution evidence")
	ErrArtifactTooLarge  = errors.New("evidence artifact exceeds configured budget")
)

// ArtifactRef is an immutable, path-safe reference to one evidence payload.
// RelativePath is private durable storage metadata; transports project an
// opaque artifact ID rather than exposing the filesystem layout.
type ArtifactRef struct {
	ID           string `json:"id"`
	Kind         string `json:"kind"`
	RelativePath string `json:"relativePath"`
	SizeBytes    int64  `json:"sizeBytes"`
	SHA256       string `json:"sha256"`
	ContentType  string `json:"contentType,omitempty"`
	Phase        string `json:"phase,omitempty"`
}

// PhaseSummary is the only phase projection allowed outside durable evidence.
// It intentionally contains counts and references, never observations,
// findings, logs, screenshots, or provider response bodies.
type PhaseSummary struct {
	Name              string                       `json:"name"`
	Status            string                       `json:"status"`
	DurationSeconds   int                          `json:"durationSeconds"`
	FindingCount      int                          `json:"findingCount"`
	ObservationCount  int                          `json:"observationCount"`
	Findings          *ArtifactRef                 `json:"findings,omitempty"`
	FindingSource     string                       `json:"findingSource,omitempty"`
	PhasePresentation *commonv1.PhasePresentation  `json:"phasePresentation,omitempty"`
	FindingsSummary   *runspb.PhaseFindingsSummary `json:"findingsSummary,omitempty"`
}

// Manifest is the canonical durable run-evidence index. Detailed structured
// findings have one owner (Findings); every other durable payload is referenced
// through Artifacts and cannot be embedded in a summary or snapshot.
type Manifest struct {
	SchemaVersion int            `json:"schemaVersion"`
	RunID         string         `json:"runId"`
	Scenario      string         `json:"scenario"`
	CreatedAt     time.Time      `json:"createdAt"`
	Verdict       string         `json:"verdict"`
	Findings      ArtifactRef    `json:"findings"`
	Phases        []PhaseSummary `json:"phases"`
	Artifacts     []ArtifactRef  `json:"artifacts,omitempty"`
}

func (m Manifest) Validate() error {
	if m.SchemaVersion != SchemaVersion {
		return fmt.Errorf("%w: got %d, want %d", ErrUnsupportedSchema, m.SchemaVersion, SchemaVersion)
	}
	if strings.TrimSpace(m.RunID) == "" || strings.TrimSpace(m.Scenario) == "" || strings.TrimSpace(m.Verdict) == "" || m.CreatedAt.IsZero() {
		return fmt.Errorf("%w: run identity, verdict, and timestamp are required", ErrCorruptEvidence)
	}
	if err := m.Findings.validate(); err != nil {
		return fmt.Errorf("%w: findings: %v", ErrCorruptEvidence, err)
	}
	seen := map[string]struct{}{m.Findings.ID: {}}
	for _, phase := range m.Phases {
		if strings.TrimSpace(phase.Name) == "" || strings.TrimSpace(phase.Status) == "" || phase.FindingCount < 0 || phase.ObservationCount < 0 {
			return fmt.Errorf("%w: invalid phase summary", ErrCorruptEvidence)
		}
		if phase.Findings != nil {
			if err := phase.Findings.validate(); err != nil {
				return fmt.Errorf("%w: phase %s findings: %v", ErrCorruptEvidence, phase.Name, err)
			}
			if phase.Findings.ID != m.Findings.ID {
				return fmt.Errorf("%w: phase %s embeds a noncanonical findings artifact", ErrCorruptEvidence, phase.Name)
			}
		}
	}
	for _, artifact := range m.Artifacts {
		if err := artifact.validate(); err != nil {
			return fmt.Errorf("%w: artifact: %v", ErrCorruptEvidence, err)
		}
		if _, duplicate := seen[artifact.ID]; duplicate {
			return fmt.Errorf("%w: duplicate artifact id %q", ErrCorruptEvidence, artifact.ID)
		}
		seen[artifact.ID] = struct{}{}
	}
	return nil
}

func (r ArtifactRef) validate() error {
	if strings.TrimSpace(r.ID) == "" || strings.TrimSpace(r.Kind) == "" || strings.TrimSpace(r.RelativePath) == "" || r.SizeBytes < 0 || strings.TrimSpace(r.SHA256) == "" {
		return errors.New("id, kind, relative path, nonnegative size, and digest are required")
	}
	if filepath.IsAbs(r.RelativePath) || r.RelativePath == "." || strings.HasPrefix(filepath.Clean(r.RelativePath), ".."+string(filepath.Separator)) || filepath.Clean(r.RelativePath) == ".." {
		return errors.New("artifact path escapes run root")
	}
	return nil
}
