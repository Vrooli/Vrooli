package validationbroker

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
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

const buildInputResolveWorkers = 4

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
		// Authoritative input expansion can walk a large scenario's declared
		// build closure. Keep the resolver bounded, but allow that existing
		// control-plane operation enough time to complete under a busy shared
		// workspace; timing out here incorrectly turns valid evidence into an
		// unresolved-input receipt.
		ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
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
		var targetNames []string
		for _, target := range intent.GetTargets() {
			if target.GetKind() != commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SCENARIO || seen[target.GetId()] {
				continue
			}
			name := target.GetId()
			if name == "" || strings.HasPrefix(name, "-") || strings.ContainsAny(name, `/\\`) {
				return nil, fmt.Errorf("invalid scenario input target %q", name)
			}
			seen[name] = true
			targetNames = append(targetNames, name)
		}
		// Freshness input expansion is an owner read, but it can take several
		// seconds per scenario because it crosses the lifecycle/build graph.
		// Resolve a bounded number concurrently so a certification intent over
		// the complete affected collection does not serialize into the client
		// deadline. Results remain indexed by targetNames, preserving the
		// deterministic manifest/toolchain order below.
		inputsByTarget, err := r.resolveBuildInputs(ctx, targetNames)
		if err != nil {
			return nil, err
		}
		for index, name := range targetNames {
			inputs := inputsByTarget[index]
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

func (r *ContentIdentityResolver) resolveBuildInputs(ctx context.Context, targetNames []string) ([]*cliv1.ScenarioFreshnessInputs, error) {
	if len(targetNames) == 0 {
		return nil, nil
	}
	workerCount := buildInputResolveWorkers
	if len(targetNames) < workerCount {
		workerCount = len(targetNames)
	}
	inputCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	results := make([]*cliv1.ScenarioFreshnessInputs, len(targetNames))
	jobs := make(chan int)
	errs := make(chan error, 1)
	var workers sync.WaitGroup
	for worker := 0; worker < workerCount; worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for {
				select {
				case <-inputCtx.Done():
					return
				case index, ok := <-jobs:
					if !ok {
						return
					}
					inputs, err := r.buildInputs(inputCtx, targetNames[index])
					if err != nil {
						select {
						case errs <- fmt.Errorf("control-plane input resolution for %s: %w", targetNames[index], err):
						default:
						}
						cancel()
						return
					}
					results[index] = inputs
				}
			}
		}()
	}
sendJobs:
	for index := range targetNames {
		select {
		case jobs <- index:
		case <-inputCtx.Done():
			break sendJobs
		}
	}
	close(jobs)
	workers.Wait()
	select {
	case err := <-errs:
		return nil, err
	default:
	}
	if err := inputCtx.Err(); err != nil && ctx.Err() != nil {
		return nil, err
	}
	return results, nil
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
