package validationbroker

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/vrooli/freshness-go/treedigest"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation"
)

type IdentityResolver interface {
	Resolve(context.Context, *validationv1.ValidationIntent) (*validationv1.SourceIdentity, error)
}

type ContentIdentityResolver struct {
	repoRoot string
	builder  *treedigest.ManifestBuilder
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
	request.Toolchain = cloneStringMap(expected.GetToolchain())
	request.Attribution = treedigest.ManifestAttribution{Commit: expected.GetCommit(), Branch: expected.GetBranch(), Dirty: expected.GetDirty()}
	manifest, err := r.builder.BuildContext(ctx, request)
	if err != nil {
		return nil, err
	}
	identity := &validationv1.SourceIdentity{SchemaVersion: uint32(manifest.SchemaVersion), Identity: manifest.Identity, Configuration: manifest.Configuration, Toolchain: manifest.Toolchain, Commit: manifest.Attribution.Commit, Branch: manifest.Attribution.Branch, Dirty: manifest.Attribution.Dirty}
	for _, root := range manifest.Roots {
		identity.Roots = append(identity.Roots, &validationv1.ContentRootIdentity{Name: root.Name, Identity: root.Identity})
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
