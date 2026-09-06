package validationbroker

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"test-genie/internal/execution"
	"test-genie/internal/orchestrator/phases"

	"github.com/vrooli/freshness-go/treedigest"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation"
	"google.golang.org/protobuf/encoding/protojson"
)

type IdentityResolver interface {
	Resolve(context.Context, *validationv1.ValidationIntent) (*validationv1.SourceIdentity, error)
}

type ContentIdentityResolver struct {
	repoRoot    string
	builder     *treedigest.ManifestBuilder
	buildInputs func(context.Context, string) (*cliv1.ScenarioFreshnessInputs, error)
	planner     execution.ExecutionPlanner
}

// WithExecutionPlanner resolves effective phase descriptors and configuration
// through the same owner used to start suite children.
func (r *ContentIdentityResolver) WithExecutionPlanner(planner execution.ExecutionPlanner) *ContentIdentityResolver {
	r.planner = planner
	return r
}

// WithControlPlaneInputs uses lifecycle's authoritative input expansion and
// build keys. A missing or old provider is an admission error, never incomplete
// evidence silently labelled fresh. The subprocess only reads; it never starts
// a scenario or stamps a build manifest.
func (r *ContentIdentityResolver) WithControlPlaneInputs() *ContentIdentityResolver {
	r.buildInputs = func(ctx context.Context, scenario string) (*cliv1.ScenarioFreshnessInputs, error) {
		ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, "vrooli", "scenario", "freshness", scenario, "--inputs", "--json")
		cmd.Dir = r.repoRoot
		output, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("control-plane input resolution for %s: %w", scenario, err)
		}
		response := &cliv1.ScenarioFreshnessResponse{}
		if err := protojson.Unmarshal(output, response); err != nil {
			return nil, err
		}
		if !response.GetSuccess() || response.GetScenario() != scenario || response.GetInputs() == nil {
			return nil, fmt.Errorf("control plane returned no authoritative inputs for %s", scenario)
		}
		return response.GetInputs(), nil
	}
	return r
}

func NewContentIdentityResolver(repoRoot string, cacheBytes int) *ContentIdentityResolver {
	return &ContentIdentityResolver{repoRoot: filepath.Clean(repoRoot), builder: treedigest.NewManifestBuilder(cacheBytes)}
}

func (r *ContentIdentityResolver) Resolve(ctx context.Context, intent *validationv1.ValidationIntent) (*validationv1.SourceIdentity, error) {
	if r == nil || strings.TrimSpace(r.repoRoot) == "" {
		return nil, fmt.Errorf("repository root is unavailable")
	}
	request := treedigest.ManifestRequest{}
	for _, input := range intent.GetContentInputs() {
		root, err := r.resolveRoot(input)
		if err != nil {
			return nil, err
		}
		if input.GetDependency() {
			request.Dependencies = append(request.Dependencies, root)
		} else if request.Primary.Name != "" {
			return nil, fmt.Errorf("multiple primary content roots")
		} else {
			request.Primary = root
		}
	}
	if request.Primary.Name == "" {
		return nil, fmt.Errorf("primary content root is required")
	}
	expected := intent.GetExpectedIdentity()
	request.Configuration = cloneStringMap(expected.GetConfiguration())
	if r.planner != nil {
		if request.Configuration == nil {
			request.Configuration = map[string]string{}
		}
		for key := range request.Configuration {
			if strings.HasPrefix(key, "suite/") {
				delete(request.Configuration, key)
			}
		}
		for _, target := range intent.GetTargets() {
			if target.GetKind() != commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SCENARIO {
				continue
			}
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			suite, preview, err := resolvedSuiteRequest(ctx, r.planner, target.GetId(), intent)
			if err != nil {
				return nil, fmt.Errorf("resolve suite configuration: %w", err)
			}
			if preview == nil || preview.ConfigurationFingerprint == "" {
				return nil, fmt.Errorf("suite configuration identity is unavailable")
			}
			request.Configuration["suite/"+target.GetId()] = preview.ConfigurationFingerprint
			request.Configuration["suite/"+target.GetId()+"/phases"] = phases.PhaseSetDigest(suite.ResolvedPhases)
		}
	}
	request.Toolchain = cloneStringMap(expected.GetToolchain())
	if r.buildInputs != nil {
		if request.Toolchain == nil {
			request.Toolchain = map[string]string{}
		}
		for key := range request.Toolchain {
			if strings.HasPrefix(key, "build/") {
				delete(request.Toolchain, key)
			}
		}
		seen := map[string]bool{}
		for _, target := range intent.GetTargets() {
			if target.GetKind() != commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SCENARIO || seen[target.GetId()] {
				continue
			}
			name := target.GetId()
			if name == "" || strings.HasPrefix(name, "-") || strings.ContainsAny(name, `/\\`) {
				return nil, fmt.Errorf("invalid scenario input target %q", name)
			}
			seen[name] = true
			inputs, err := r.buildInputs(ctx, name)
			if err != nil {
				return nil, err
			}
			root := treedigest.RootSpec{Name: "build-inputs/" + name, Path: r.repoRoot}
			for _, input := range inputs.GetPaths() {
				path := filepath.Clean(filepath.FromSlash(input))
				if filepath.IsAbs(path) || path == ".." || strings.HasPrefix(path, ".."+string(filepath.Separator)) {
					return nil, fmt.Errorf("build input escapes repository: %q", input)
				}
				root.Files = append(root.Files, filepath.ToSlash(path))
			}
			// Components without file inputs contribute only their build keys.
			if len(root.Files) > 0 {
				request.Dependencies = append(request.Dependencies, root)
			}
			for key, value := range inputs.GetBuildKeys() {
				request.Toolchain["build/"+name+"/"+key] = value
			}
		}
	}
	request.Attribution = treedigest.ManifestAttribution{Commit: expected.GetCommit(), Branch: expected.GetBranch(), Dirty: expected.GetDirty()}
	manifest, err := r.builder.BuildContext(ctx, request)
	if err != nil {
		return nil, err
	}
	identity := &validationv1.SourceIdentity{SchemaVersion: uint32(manifest.SchemaVersion), Identity: manifest.Identity, Configuration: manifest.Configuration, Toolchain: manifest.Toolchain, Commit: manifest.Attribution.Commit, Branch: manifest.Attribution.Branch, Dirty: manifest.Attribution.Dirty}
	for _, root := range manifest.Roots {
		frozen := &validationv1.ContentRootIdentity{Name: root.Name, Identity: root.Identity}
		for _, file := range root.Files {
			frozen.Files = append(frozen.Files, &validationv1.ContentFileIdentity{Path: file.Path, Digest: file.Digest, Size: file.Size})
		}
		identity.Roots = append(identity.Roots, frozen)
	}
	return identity, nil
}

func (r *ContentIdentityResolver) resolveRoot(input *validationv1.ContentInputRoot) (treedigest.RootSpec, error) {
	rel := filepath.Clean(filepath.FromSlash(strings.TrimSpace(input.GetRoot())))
	if filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return treedigest.RootSpec{}, fmt.Errorf("content root %q must be repository-relative", input.GetRoot())
	}
	abs := filepath.Join(r.repoRoot, rel)
	within, err := filepath.Rel(r.repoRoot, abs)
	if err != nil || within == ".." || strings.HasPrefix(within, ".."+string(filepath.Separator)) {
		return treedigest.RootSpec{}, fmt.Errorf("content root %q escapes the repository", input.GetRoot())
	}
	root := treedigest.RootSpec{Name: strings.TrimSpace(input.GetName()), Path: abs}
	for _, selection := range input.GetSelections() {
		root.Selections = append(root.Selections, treedigest.InputSelection{Glob: selection.GetGlob(), Required: selection.GetRequired()})
	}
	return root, nil
}

func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	output := make(map[string]string, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}
