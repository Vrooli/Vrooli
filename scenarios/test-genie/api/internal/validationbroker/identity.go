package validationbroker

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"

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

// WorkspaceSandboxSealOwner resolves only owner-authorized, content-addressed
// seal references. Paths returned by ResolveReplaySeal remain server-internal.
type WorkspaceSandboxSealOwner interface {
	ResolveReplaySeal(context.Context, *validationv1.WorkspaceSandboxSealReference) (string, error)
	VerifyReplaySeal(context.Context, *validationv1.WorkspaceSandboxSealReference) error
}

type FrozenEvaluatorFile struct {
	RelativePath string
	Content      []byte
	SHA256       string
}

type FrozenEvaluatorBundle struct {
	EvaluatorBundleID string
	CaseID            string
	EntryPoint        string
	EvaluatorDigest   string
	ContentDigest     string
	Files             []FrozenEvaluatorFile
}

type frozenEvaluatorDigestTuple struct {
	RelativePath string `json:"relativePath"`
	SHA256       string `json:"sha256"`
	Size         int64  `json:"size"`
}

type FrozenEvaluatorBundleOwner interface {
	ResolveFrozenEvaluatorBundle(context.Context, *validationv1.FrozenEvaluatorBundleReference) (FrozenEvaluatorBundle, error)
}

// matchExpectedIdentity checks retained preconditions before admission replaces
// them with the full resolved identity. A named root asserts its digest and/or
// complete file set; it is not a subset of files. Unasserted roots may be added by
// the execution/build owners and are retained in the admitted identity.
func matchExpectedIdentity(expected, resolved *validationv1.SourceIdentity) error {
	if resolved == nil || resolved.GetIdentity() == "" {
		return fmt.Errorf("resolved content identity is unavailable")
	}
	if expected == nil {
		return nil
	}
	if expected.GetIdentity() != "" && expected.GetIdentity() != resolved.GetIdentity() {
		return fmt.Errorf("expected content identity %s but resolved %s", expected.GetIdentity(), resolved.GetIdentity())
	}
	if expected.GetSchemaVersion() != 0 && expected.GetSchemaVersion() != resolved.GetSchemaVersion() {
		return fmt.Errorf("expected content identity schema differs from resolved schema")
	}
	roots := map[string]*validationv1.ContentRootIdentity{}
	for _, root := range resolved.GetRoots() {
		if root.GetName() == "" || roots[root.GetName()] != nil {
			return fmt.Errorf("resolved content roots are ambiguous")
		}
		roots[root.GetName()] = root
	}
	for _, want := range expected.GetRoots() {
		got := roots[want.GetName()]
		if got == nil {
			return fmt.Errorf("expected content root %q is missing or repeated", want.GetName())
		}
		delete(roots, want.GetName())
		if want.GetIdentity() != "" && want.GetIdentity() != got.GetIdentity() {
			return fmt.Errorf("expected content root %q identity differs", want.GetName())
		}
		if want.GetIdentity() != "" && len(want.GetFiles()) == 0 {
			continue // A digest-only assertion does not also assert an empty tree.
		}
		if len(want.GetFiles()) != len(got.GetFiles()) {
			return fmt.Errorf("expected content root %q file set differs", want.GetName())
		}
		files := map[string]*validationv1.ContentFileIdentity{}
		for _, file := range got.GetFiles() {
			files[file.GetPath()] = file
		}
		for _, file := range want.GetFiles() {
			actual := files[file.GetPath()]
			if actual == nil || file.GetDigest() != actual.GetDigest() || file.GetSize() != actual.GetSize() {
				return fmt.Errorf("expected content root %q file %q differs or is repeated", want.GetName(), file.GetPath())
			}
			delete(files, file.GetPath())
		}
	}
	return nil
}

type ContentIdentityResolver struct {
	repoRoot    string
	builder     *treedigest.ManifestBuilder
	buildInputs func(context.Context, string) (*cliv1.ScenarioFreshnessInputs, error)
	planner     execution.ExecutionPlanner
	seals       WorkspaceSandboxSealOwner
	evaluators  FrozenEvaluatorBundleOwner
}

const buildInputResolveWorkers = 4

// WithExecutionPlanner resolves effective phase descriptors and configuration
// through the same owner used to start suite children.
func (r *ContentIdentityResolver) WithExecutionPlanner(planner execution.ExecutionPlanner) *ContentIdentityResolver {
	r.planner = planner
	return r
}

func (r *ContentIdentityResolver) WithWorkspaceSandboxSeals(owner WorkspaceSandboxSealOwner) *ContentIdentityResolver {
	r.seals = owner
	return r
}

func (r *ContentIdentityResolver) WithFrozenEvaluatorBundles(owner FrozenEvaluatorBundleOwner) *ContentIdentityResolver {
	r.evaluators = owner
	return r
}

func (r *ContentIdentityResolver) ResolveFrozenEvaluatorBundle(ctx context.Context, execution *validationv1.FrozenReplayExecution) (FrozenEvaluatorBundle, error) {
	if r == nil || r.evaluators == nil || execution == nil {
		return FrozenEvaluatorBundle{}, fmt.Errorf("frozen evaluator bundle owner is unavailable")
	}
	ref := &validationv1.FrozenEvaluatorBundleReference{EvaluatorBundleId: execution.GetEvaluatorBundleId(), EvaluatorDigest: execution.GetEvaluatorDigest(), EntryPoint: execution.GetEvaluatorEntryPoint()}
	bundle, err := r.evaluators.ResolveFrozenEvaluatorBundle(ctx, ref)
	if err != nil {
		return FrozenEvaluatorBundle{}, err
	}
	if _, err := frozenEvaluatorContentRoot(bundle, ref); err != nil {
		return FrozenEvaluatorBundle{}, err
	}
	return bundle, nil
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
	var sealedRoots []*validationv1.WorkspaceSandboxSealReference
	var sealDigest string
	var evaluatorRoots []*validationv1.ContentRootIdentity
	for _, input := range intent.GetContentInputs() {
		if bundleRef := input.GetFrozenEvaluatorBundle(); bundleRef != nil {
			if input.GetWorkspaceSandboxSeal() != nil || strings.TrimSpace(input.GetRoot()) != "" || !input.GetDependency() || input.GetName() != "frozen-evaluator" || len(input.GetSelections()) != 0 {
				return nil, fmt.Errorf("frozen evaluator input must be a dependency named frozen-evaluator with no caller path or selections")
			}
			if r.evaluators == nil {
				return nil, fmt.Errorf("frozen evaluator bundle owner is unavailable")
			}
			bundle, err := r.evaluators.ResolveFrozenEvaluatorBundle(ctx, bundleRef)
			if err != nil {
				return nil, fmt.Errorf("resolve frozen evaluator bundle: %w", err)
			}
			root, err := frozenEvaluatorContentRoot(bundle, bundleRef)
			if err != nil {
				return nil, err
			}
			evaluatorRoots = append(evaluatorRoots, root)
			continue
		}
		root, err := r.resolveRoot(ctx, input)
		if err != nil {
			return nil, err
		}
		if input.GetWorkspaceSandboxSeal() != nil {
			if input.GetDependency() {
				return nil, fmt.Errorf("workspace sandbox seal must be the primary content root")
			}
			sealedRoots = append(sealedRoots, proto.Clone(input.GetWorkspaceSandboxSeal()).(*validationv1.WorkspaceSandboxSealReference))
			if sealDigest != "" {
				return nil, fmt.Errorf("exactly one workspace sandbox seal is allowed per frozen replay intent")
			}
			sealDigest = input.GetWorkspaceSandboxSeal().GetSealDigest()
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
	if r.planner != nil && intent.GetPurpose() != validationv1.ValidationPurpose_VALIDATION_PURPOSE_EVIDENCE_PRODUCTION {
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
	for _, seal := range sealedRoots {
		if err := r.seals.VerifyReplaySeal(ctx, seal); err != nil {
			return nil, fmt.Errorf("workspace sandbox seal changed during identity resolution: %w", err)
		}
	}
	identity := &validationv1.SourceIdentity{SchemaVersion: uint32(manifest.SchemaVersion), Identity: manifest.Identity, Configuration: manifest.Configuration, Toolchain: manifest.Toolchain, Commit: manifest.Attribution.Commit, Branch: manifest.Attribution.Branch, Dirty: manifest.Attribution.Dirty}
	for _, root := range manifest.Roots {
		frozen := &validationv1.ContentRootIdentity{Name: root.Name, Identity: root.Identity}
		for _, file := range root.Files {
			frozen.Files = append(frozen.Files, &validationv1.ContentFileIdentity{Path: file.Path, Digest: file.Digest, Size: file.Size})
		}
		identity.Roots = append(identity.Roots, frozen)
	}
	identity.Roots = append(identity.Roots, evaluatorRoots...)
	if sealDigest != "" {
		identity.Identity = "sha256:" + sealDigest
	}
	return identity, nil
}

func frozenEvaluatorContentRoot(bundle FrozenEvaluatorBundle, ref *validationv1.FrozenEvaluatorBundleReference) (*validationv1.ContentRootIdentity, error) {
	if bundle.EvaluatorBundleID == "" || bundle.EvaluatorBundleID != ref.GetEvaluatorBundleId() || bundle.EvaluatorDigest != ref.GetEvaluatorDigest() || bundle.EntryPoint != ref.GetEntryPoint() || bundle.ContentDigest == "" {
		return nil, fmt.Errorf("frozen evaluator owner returned a mismatched bundle identity")
	}
	if bundle.CaseID == "" || !safeRelativeContentPath(bundle.EntryPoint) {
		return nil, fmt.Errorf("frozen evaluator bundle case or entry point is invalid")
	}
	if len(bundle.Files) == 0 || len(bundle.Files) > 128 {
		return nil, fmt.Errorf("frozen evaluator bundle file count is outside its bound")
	}
	contentBytes := 0
	seen := make(map[string]bool, len(bundle.Files))
	tuples := make([]frozenEvaluatorDigestTuple, 0, len(bundle.Files))
	root := &validationv1.ContentRootIdentity{Name: "frozen-evaluator", Identity: "sha256:" + strings.TrimPrefix(strings.ToLower(bundle.ContentDigest), "sha256:")}
	entrypointFound := false
	for _, file := range bundle.Files {
		if !safeRelativeContentPath(file.RelativePath) || seen[file.RelativePath] {
			return nil, fmt.Errorf("frozen evaluator manifest contains an unsafe or duplicate path")
		}
		seen[file.RelativePath] = true
		contentBytes += len(file.Content)
		if contentBytes > 16<<20 {
			return nil, fmt.Errorf("frozen evaluator bundle exceeds the 16 MiB bound")
		}
		digest := sha256.Sum256(file.Content)
		if len(file.SHA256) != 64 || hex.EncodeToString(digest[:]) != strings.ToLower(file.SHA256) {
			return nil, fmt.Errorf("frozen evaluator file %q failed owner digest verification", file.RelativePath)
		}
		root.Files = append(root.Files, &validationv1.ContentFileIdentity{Path: file.RelativePath, Digest: strings.ToLower(file.SHA256), Size: int64(len(file.Content))})
		tuples = append(tuples, frozenEvaluatorDigestTuple{RelativePath: file.RelativePath, SHA256: strings.ToLower(file.SHA256), Size: int64(len(file.Content))})
		if file.RelativePath == bundle.EntryPoint {
			entrypointFound = strings.ToLower(file.SHA256) == strings.TrimPrefix(strings.ToLower(bundle.EvaluatorDigest), "sha256:")
		}
	}
	if !entrypointFound {
		return nil, fmt.Errorf("frozen evaluator entry point digest differs from the catalog digest")
	}
	sort.Slice(root.Files, func(i, j int) bool { return root.Files[i].GetPath() < root.Files[j].GetPath() })
	sort.Slice(tuples, func(i, j int) bool { return tuples[i].RelativePath < tuples[j].RelativePath })
	canonical, err := json.Marshal(tuples)
	if err != nil {
		return nil, fmt.Errorf("encode frozen evaluator manifest: %w", err)
	}
	manifestDigest := sha256.Sum256(canonical)
	if hex.EncodeToString(manifestDigest[:]) != strings.TrimPrefix(strings.ToLower(bundle.ContentDigest), "sha256:") {
		return nil, fmt.Errorf("frozen evaluator content manifest digest differs from owner bundle")
	}
	return root, nil
}

func safeRelativeContentPath(value string) bool {
	if value == "" || filepath.IsAbs(value) || strings.ContainsAny(value, "\\\x00\r\n") {
		return false
	}
	clean := filepath.Clean(filepath.FromSlash(value))
	return clean != "." && clean != ".." && !strings.HasPrefix(clean, ".."+string(filepath.Separator))
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

func (r *ContentIdentityResolver) resolveRoot(ctx context.Context, input *validationv1.ContentInputRoot) (treedigest.RootSpec, error) {
	if input == nil || strings.TrimSpace(input.GetName()) == "" {
		return treedigest.RootSpec{}, fmt.Errorf("content root requires a name")
	}
	if seal := input.GetWorkspaceSandboxSeal(); seal != nil {
		if strings.TrimSpace(input.GetRoot()) != "" || input.GetFrozenEvaluatorBundle() != nil {
			return treedigest.RootSpec{}, fmt.Errorf("workspace sandbox content root must not include a caller filesystem path")
		}
		if r.seals == nil {
			return treedigest.RootSpec{}, fmt.Errorf("workspace sandbox seal owner is unavailable")
		}
		if err := validateWorkspaceSandboxSealReference(seal); err != nil {
			return treedigest.RootSpec{}, err
		}
		if err := r.seals.VerifyReplaySeal(ctx, seal); err != nil {
			return treedigest.RootSpec{}, fmt.Errorf("verify workspace sandbox seal: %w", err)
		}
		rootPath, err := r.seals.ResolveReplaySeal(ctx, seal)
		if err != nil {
			return treedigest.RootSpec{}, fmt.Errorf("resolve workspace sandbox seal: %w", err)
		}
		if strings.TrimSpace(rootPath) == "" || !filepath.IsAbs(rootPath) {
			return treedigest.RootSpec{}, fmt.Errorf("workspace sandbox returned no owner-resolved absolute root")
		}
		if err := r.seals.VerifyReplaySeal(ctx, seal); err != nil {
			return treedigest.RootSpec{}, fmt.Errorf("workspace sandbox seal changed while resolving: %w", err)
		}
		root := treedigest.RootSpec{Name: strings.TrimSpace(input.GetName()), Path: rootPath}
		for _, selection := range input.GetSelections() {
			root.Selections = append(root.Selections, treedigest.InputSelection{Glob: selection.GetGlob(), Required: selection.GetRequired()})
		}
		return root, nil
	}
	if len(input.GetSelections()) == 0 {
		return treedigest.RootSpec{}, fmt.Errorf("content root requires at least one selection")
	}
	if input.GetRoot() == "" {
		return treedigest.RootSpec{}, fmt.Errorf("repository content root path is required")
	}
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

func validateWorkspaceSandboxSealReference(ref *validationv1.WorkspaceSandboxSealReference) error {
	if ref == nil || strings.TrimSpace(ref.GetCaseId()) == "" || len(strings.TrimSpace(ref.GetSealDigest())) != 64 {
		return fmt.Errorf("workspace sandbox seal requires case_id and a SHA-256 seal_digest")
	}
	if _, err := hex.DecodeString(ref.GetSealDigest()); err != nil {
		return fmt.Errorf("workspace sandbox seal digest is not SHA-256 hex")
	}
	switch ref.GetRole() {
	case validationv1.WorkspaceSandboxSealRole_WORKSPACE_SANDBOX_SEAL_ROLE_SOURCE,
		validationv1.WorkspaceSandboxSealRole_WORKSPACE_SANDBOX_SEAL_ROLE_CANDIDATE,
		validationv1.WorkspaceSandboxSealRole_WORKSPACE_SANDBOX_SEAL_ROLE_REFERENCE:
		return nil
	default:
		return fmt.Errorf("workspace sandbox seal role is required")
	}
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
