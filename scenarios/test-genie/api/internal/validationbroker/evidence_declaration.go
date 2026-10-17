package validationbroker

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation"

	"test-genie/internal/orchestrator/providerdescriptor"
)

type DescriptorEvidenceResolver struct{ RepoRoot string }

var scenarioNamePattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func (r DescriptorEvidenceResolver) ResolveEvidenceProducer(ctx context.Context, provider, name, candidate string) (*validationv1.PinnedEvidenceProducer, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	provider = strings.TrimSpace(provider)
	name = strings.TrimSpace(name)
	candidate = strings.TrimSpace(candidate)
	if !scenarioNamePattern.MatchString(provider) || !scenarioNamePattern.MatchString(candidate) || name == "" {
		return nil, fmt.Errorf("provider, producer, and candidate scenario must be valid names")
	}
	path := filepath.Join(r.RepoRoot, "scenarios", provider, ".vrooli", "test-genie.json")
	descriptors := providerdescriptor.Load(providerdescriptor.LoadOptions{RepoRoot: r.RepoRoot, Paths: []string{path}})
	if err := descriptors.Err(); err != nil {
		return nil, fmt.Errorf("load provider declaration: %w", err)
	}
	if len(descriptors.Descriptors) != 1 {
		return nil, fmt.Errorf("provider descriptor %s is unavailable", provider)
	}
	descriptor := descriptors.Descriptors[0]
	root := filepath.Join(r.RepoRoot, "scenarios", provider)
	resolved, err := descriptor.ResolveEvidenceProducer(root, name, "pending-run-id")
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(raw)
	return &validationv1.PinnedEvidenceProducer{
		Provider: provider, Producer: name, Argv: append([]string(nil), descriptor.EvidenceProducers[name].Argv...),
		WorkingDirectory: resolved.WorkingDirectory, OutputRoot: resolved.OutputRoot,
		TimeoutMilliseconds: uint64(resolved.TimeoutValue.Milliseconds()), MaximumOutputBytes: uint64(resolved.MaximumOutputBytes),
		MutatesLifecycle: resolved.MutatesLifecycle, DescriptorDigest: fmt.Sprintf("sha256:%x", digest),
	}, nil
}

func producerSourceIntent(pin *validationv1.PinnedEvidenceProducer) *validationv1.ValidationIntent {
	return &validationv1.ValidationIntent{
		Purpose:          validationv1.ValidationPurpose_VALIDATION_PURPOSE_EVIDENCE_PRODUCTION,
		Targets:          []*commonv1.ValidationTarget{{Kind: commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SCENARIO, Id: pin.GetProvider()}},
		ExpectedIdentity: &validationv1.SourceIdentity{},
		ContentInputs:    []*validationv1.ContentInputRoot{{Name: "provider-source", Root: "scenarios/" + pin.GetProvider(), Selections: []*validationv1.InputSelection{{Glob: "**", Required: true}}}},
	}
}
