package runs

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"connectrpc.com/connect"
	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
	"test-genie/internal/executionevidence"
	sharedartifacts "test-genie/internal/shared/artifacts"
	sharedruns "test-genie/internal/shared/runs"
)

const strictInputArtifactKind = "strict.input-manifest"

type strictInputKey struct{ directory, run, artifact string }

// StrictInputReadBinding is supplied only by installed owner construction.
// It crossbinds opaque strict digests to exact tagged terminal identities;
// no conversion, artifact-derived epoch, or public request flag is trusted.
type StrictInputReadBinding struct {
	MappingVersion           int
	Target                   string
	RunID                    string
	ArtifactID               string
	ByteSHA256               string
	SizeBytes                int64
	CatalogDigest            string
	Execution                executionevidence.StrictExecutionBinding
	ConfigurationFingerprint string
	DescriptorDigest         string
	PhaseSetDigest           string
}

func strictTaggedDigest(s, prefix string) bool {
	if !strings.HasPrefix(s, prefix) || len(s) != len(prefix)+64 || s != strings.ToLower(s) {
		return false
	}
	_, e := hex.DecodeString(s[len(prefix):])
	return e == nil
}
func validStrictIdentity(s string) bool {
	return s != "" && s == strings.TrimSpace(s) && !strings.ContainsAny(s, "\x00/\\")
}

// StrictInputTerminalDigests v1 binds exact tagged terminal identities without
// stripping or aliasing their algorithm domains. This hashes references only.
func StrictInputTerminalDigests(configuration, descriptor, phases string) ([3]string, error) {
	if !strictTaggedDigest(configuration, "execution-config:") || !strictTaggedDigest(descriptor, "ds:sha256:") || !strictTaggedDigest(phases, "phase-set:") {
		return [3]string{}, fmt.Errorf("invalid terminal identity domain")
	}
	var out [3]string
	for i, v := range []string{configuration, descriptor, phases} {
		field := []string{"configuration", "descriptor", "phases"}[i]
		sum := sha256.Sum256([]byte("test-genie.strict-input-binding/v1\x00" + field + "\x00" + v))
		out[i] = "sha256:" + hex.EncodeToString(sum[:])
	}
	return out, nil
}

// NewServiceWithStrictInputBindings freezes an explicit observation table.
// The default constructor has no table and refuses the strict kind only.
func NewServiceWithStrictInputBindings(base *Service, bindings []StrictInputReadBinding) (*Service, error) {
	if base == nil {
		return nil, fmt.Errorf("base service required")
	}
	copyService := *base
	s := &copyService
	s.strictInputBindings = make(map[strictInputKey]StrictInputReadBinding, len(bindings))
	for _, b := range bindings {
		if b.MappingVersion != 1 || !validStrictIdentity(b.Target) || !validStrictIdentity(b.RunID) || !validStrictIdentity(b.ArtifactID) || b.Execution.Target != b.Target || b.Execution.RunID != b.RunID || b.Execution.ExecutionEpoch == 0 || b.SizeBytes < 1 || b.SizeBytes > executionevidence.StrictInputArtifactMaxBytes {
			return nil, fmt.Errorf("invalid strict input owner binding")
		}
		for _, d := range []string{b.ByteSHA256, b.Execution.OwnerBindingDigest, b.Execution.ConfigurationDigest, b.Execution.DescriptorDigest, b.Execution.PhaseSetDigest} {
			if !strictTaggedDigest(d, "sha256:") {
				return nil, fmt.Errorf("invalid strict digest")
			}
		}
		if !strictTaggedDigest(b.CatalogDigest, "ac:sha256:") || !strictTaggedDigest(b.ConfigurationFingerprint, "execution-config:") || !strictTaggedDigest(b.DescriptorDigest, "ds:sha256:") || !strictTaggedDigest(b.PhaseSetDigest, "phase-set:") {
			return nil, fmt.Errorf("invalid terminal identity domain")
		}
		mapped, err := StrictInputTerminalDigests(b.ConfigurationFingerprint, b.DescriptorDigest, b.PhaseSetDigest)
		if err != nil {
			return nil, err
		}
		if b.Execution.ConfigurationDigest != mapped[0] || b.Execution.DescriptorDigest != mapped[1] || b.Execution.PhaseSetDigest != mapped[2] {
			return nil, fmt.Errorf("strict terminal reference digest mismatch")
		}
		_, dir, err := base.targetDirs(b.Target)
		if err != nil {
			return nil, err
		}
		dir, err = filepath.Abs(dir)
		if err != nil {
			return nil, err
		}
		key := strictInputKey{dir, b.RunID, b.ArtifactID}
		if _, exists := s.strictInputBindings[key]; exists {
			return nil, fmt.Errorf("duplicate strict owner binding")
		}
		s.strictInputBindings[key] = b
	}
	return s, nil
}

// validateStrictInputArtifact validates this metadata read only. The separate
// REST route separately requests the captured validated bytes from this stage.
func (s *Service) validateStrictInputArtifact(ctx context.Context, target, dir, runID, artifactID string, ref sharedartifacts.ArtifactRef, path string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	resolved, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	b, ok := s.strictInputBindings[strictInputKey{resolved, runID, artifactID}]
	if !ok {
		return nil, fmt.Errorf("strict input owner binding unavailable")
	}
	catalog, err := sharedartifacts.ReadArtifactCatalog(dir, runID)
	if err != nil {
		return nil, err
	}
	if catalog.LegacyDiscovered || catalog.RunID != runID || catalog.Digest != b.CatalogDigest || ref.Provenance == sharedartifacts.ArtifactProvenanceLegacy || ref.Kind != strictInputArtifactKind || ref.ID != artifactID || ref.SizeBytes != b.SizeBytes {
		return nil, fmt.Errorf("strict artifact catalog binding mismatch")
	}
	matched := false
	for _, retained := range catalog.Artifacts {
		if retained.ID == artifactID {
			matched = reflect.DeepEqual(retained, ref)
			break
		}
	}
	if !matched {
		return nil, fmt.Errorf("strict artifact catalog reference changed")
	}
	terminal, err := s.GetRun(ctx, connect.NewRequest(&runspb.GetRunRequest{Target: b.Target, RunId: runID}))
	if err != nil {
		return nil, err
	}
	run := terminal.Msg.GetRun()
	if terminal.Msg.GetTerminalSnapshotSchemaVersion() != sharedruns.TerminalSnapshotSchemaVersion || len(terminal.Msg.GetDegradedReasons()) != 0 || run == nil || run.GetRunId() != runID || run.GetTarget() != b.Target || run.GetExecutionConfigurationFingerprint() != b.ConfigurationFingerprint || run.GetDescriptorSnapshotDigest() != b.DescriptorDigest || run.GetPhaseSetDigest() != b.PhaseSetDigest {
		return nil, fmt.Errorf("strict artifact terminal binding mismatch")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() != b.SizeBytes {
		return nil, fmt.Errorf("strict artifact size or regular-file mismatch")
	}
	raw, err := io.ReadAll(io.LimitReader(f, b.SizeBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) != b.SizeBytes {
		return nil, fmt.Errorf("strict artifact byte size mismatch")
	}
	_, err = executionevidence.DecodeStrictInputArtifact(ctx, bytes.NewReader(raw), b.SizeBytes, b.ByteSHA256, b.Execution)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

// ReadStrictInputArtifact returns the same captured bytes that passed validation.
// It is an internal reader, not a new RPC, producer or custody capability.
func (s *Service) ReadStrictInputArtifact(ctx context.Context, target, runID, artifactID string) (sharedartifacts.ArtifactRef, []byte, error) {
	runID = strings.TrimSpace(runID)
	artifactID = strings.TrimSpace(artifactID)
	_, dir, err := s.targetDirs(target)
	if err != nil {
		return sharedartifacts.ArtifactRef{}, nil, err
	}
	ref, path, err := sharedartifacts.ResolveCatalogArtifact(dir, runID, artifactID, artifactPhaseDeclarations(dir, runID))
	if err != nil {
		return sharedartifacts.ArtifactRef{}, nil, err
	}
	if ref.Kind != strictInputArtifactKind {
		return sharedartifacts.ArtifactRef{}, nil, fmt.Errorf("strict artifact kind required")
	}
	raw, err := s.validateStrictInputArtifact(ctx, target, dir, runID, artifactID, ref, path)
	return ref, raw, err
}

// RequiresStrictInputValidation is a read-only routing decision, not authority.
// An installed strict binding cannot be downgraded by changing offered kind or
// using another supported spelling of the same routed artifact directory.
func (s *Service) RequiresStrictInputValidation(target, runID, artifactID, kind string) (bool, error) {
	_, dir, err := s.targetDirs(target)
	if err != nil {
		return false, err
	}
	dir, err = filepath.Abs(dir)
	if err != nil {
		return false, err
	}
	_, bound := s.strictInputBindings[strictInputKey{dir, strings.TrimSpace(runID), strings.TrimSpace(artifactID)}]
	return kind == strictInputArtifactKind || bound, nil
}
func (s *Service) requiresStrictInputValidationInDirectory(dir, runID, artifactID, kind string) bool {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return true
	}
	_, bound := s.strictInputBindings[strictInputKey{dir, runID, artifactID}]
	return kind == strictInputArtifactKind || bound
}
