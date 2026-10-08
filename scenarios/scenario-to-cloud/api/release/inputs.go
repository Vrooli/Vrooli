package release

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"scenario-to-cloud/domain"
)

// sourceFacts observes the VCS state of the included roots. It is read-only
// (`git rev-parse`, `git status --porcelain`) and every failure becomes a
// recorded limitation instead of a guess.
func sourceFacts(ctx context.Context, repoRoot string, roots []string) (commit string, dirty bool, dirtyPaths int, limitations []string) {
	git, err := exec.LookPath("git")
	if err != nil {
		return "", false, 0, []string{"source commit not recorded: git is not available on the builder"}
	}
	head := exec.CommandContext(ctx, git, "rev-parse", "HEAD")
	head.Dir = repoRoot
	out, err := head.Output()
	if err != nil {
		return "", false, 0, []string{"source commit not recorded: the repo root is not a git work tree"}
	}
	commit = strings.TrimSpace(string(out))
	args := append([]string{"status", "--porcelain", "--untracked-files=all", "--"}, roots...)
	status := exec.CommandContext(ctx, git, args...)
	status.Dir = repoRoot
	out, err = status.Output()
	if err != nil {
		return commit, false, 0, []string{"working-tree dirtiness not observed: git status failed; content hashes in the content manifest are the only source identity"}
	}
	for _, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
		if strings.TrimSpace(line) != "" {
			dirtyPaths++
		}
	}
	return commit, dirtyPaths > 0, dirtyPaths, nil
}

// resourceArtifacts projects the closure's per-platform artifacts and the
// license references the resource declarations carry. A nil closure yields an
// empty list plus a limitation: the release still binds the closure digest.
func resourceArtifacts(repoRoot string, closure *domain.Closure) ([]domain.ReleaseResourceArtifact, []string) {
	if closure == nil {
		return []domain.ReleaseResourceArtifact{}, []string{"resource artifact references not recorded: closure document not supplied to the build (closure_digest is still bound)"}
	}
	out := make([]domain.ReleaseResourceArtifact, 0)
	for _, component := range closure.Components {
		if component.Artifact == nil {
			continue
		}
		out = append(out, domain.ReleaseResourceArtifact{
			Component:   component.ID,
			Platform:    component.Artifact.Platform,
			Name:        component.Artifact.Name,
			Digest:      component.Artifact.Digest,
			Mode:        component.Artifact.Mode,
			Eligibility: string(component.Artifact.Eligibility),
			LicenseRefs: resourceLicenseRefs(repoRoot, component.ID),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Component != out[j].Component {
			return out[i].Component < out[j].Component
		}
		return out[i].Platform < out[j].Platform
	})
	return out, nil
}

// resourceLicenseRefs collects every "license" string a resource declaration
// carries (top level, deployment block, artifact entries). References only.
func resourceLicenseRefs(repoRoot, resourceID string) []string {
	refs := []string{}
	raw, err := os.ReadFile(filepath.Join(repoRoot, "resources", resourceID, "resource.json"))
	if err != nil {
		return refs
	}
	var doc any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return refs
	}
	seen := map[string]struct{}{}
	var walk func(node any, depth int)
	walk = func(node any, depth int) {
		if depth > 6 {
			return
		}
		switch typed := node.(type) {
		case map[string]any:
			for key, value := range typed {
				if key == "license" {
					if text, ok := value.(string); ok && strings.TrimSpace(text) != "" {
						seen[strings.TrimSpace(text)] = struct{}{}
					}
					continue
				}
				walk(value, depth+1)
			}
		case []any:
			for _, item := range typed {
				walk(item, depth+1)
			}
		}
	}
	walk(doc, 0)
	for ref := range seen {
		refs = append(refs, ref)
	}
	sort.Strings(refs)
	return refs
}

// credentialRefs lists descriptor references from the manifest's secret
// plan. Generators, prompts and any value-bearing field are dropped here and
// the canary test proves nothing leaks.
func credentialRefs(m domain.CloudManifest) []domain.ReleaseCredentialRef {
	refs := []domain.ReleaseCredentialRef{}
	if m.Secrets == nil {
		return refs
	}
	for _, plan := range m.Secrets.BundleSecrets {
		ref := domain.ReleaseCredentialRef{ID: plan.ID, Class: plan.Class, Required: plan.Required, TargetType: plan.Target.Type, TargetName: plan.Target.Name}
		if plan.Descriptor != nil {
			descriptor := *plan.Descriptor
			ref.Descriptor = &descriptor
		}
		refs = append(refs, ref)
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i].ID < refs[j].ID })
	return refs
}
