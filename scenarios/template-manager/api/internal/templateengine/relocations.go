package templateengine

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/internal/scenarioexec"
	"github.com/vrooli/vrooli/internal/templatevalidation"
	templatecontracts "github.com/vrooli/vrooli/scenarios/template-manager/api/internal/templatecontracts"
)

// resolveRelocations renders each relocation's To path against the
// substitution values and returns absolute repo-rooted destinations.
// The From paths are left template-relative — the caller resolves them
// against info.Path when copying. Errors signal misconfigured manifests
// (empty From/To, From referencing outside the template tree, etc.).
func resolveRelocations(root string, info templatecontracts.TemplateInfo, values map[string]string) ([]templatecontracts.ResolvedRelocation, error) {
	if len(info.Manifest.Relocations) == 0 {
		return nil, nil
	}
	repoRoot := filepath.Clean(root)
	resolved := make([]templatecontracts.ResolvedRelocation, 0, len(info.Manifest.Relocations))
	for index, reloc := range info.Manifest.Relocations {
		from := strings.TrimSpace(reloc.From)
		if from == "" {
			return nil, fmt.Errorf("relocation %d: from is required", index)
		}
		// Reject path traversal so a manifest can't escape the template tree.
		cleanFrom := filepath.Clean(filepath.FromSlash(from))
		if cleanFrom == ".." || strings.HasPrefix(cleanFrom, ".."+string(filepath.Separator)) || filepath.IsAbs(cleanFrom) {
			return nil, fmt.Errorf("relocation %d: from %q must be a template-relative path", index, reloc.From)
		}
		toRendered := strings.TrimSpace(renderTemplateString(reloc.To, values))
		if toRendered == "" {
			return nil, fmt.Errorf("relocation %d: to is required (rendered from %q)", index, reloc.To)
		}
		toAbs := toRendered
		if !filepath.IsAbs(toAbs) {
			toAbs = filepath.Join(repoRoot, filepath.FromSlash(toAbs))
		}
		toAbs = filepath.Clean(toAbs)
		// The resolved To must stay within the repo root — relocations
		// declare in-repo placement, not arbitrary writes.
		rel, err := filepath.Rel(repoRoot, toAbs)
		if err != nil || strings.HasPrefix(rel, "..") {
			return nil, fmt.Errorf("relocation %d: to %q resolves outside repo root", index, reloc.To)
		}
		resolved = append(resolved, templatecontracts.ResolvedRelocation{
			Description: reloc.Description,
			From:        cleanFrom,
			To:          toAbs,
			Post:        reloc.Post,
		})
	}
	return resolved, nil
}

// runRelocations writes each resolved relocation to disk, substituting
// {{...}} placeholders in both file content and path components, and
// then invokes Post hooks at the repo root. It must run AFTER
// copyTemplate so the in-tree skip-list has already filtered the
// relocated source dirs out of the scenario destination.
func runRelocations[C any](deps HandlerDeps[C], ctx C, templateDir string, relocations []templatecontracts.ResolvedRelocation, values map[string]string, output io.Writer) error {
	if len(relocations) == 0 {
		return nil
	}
	if output == nil {
		output = io.Discard
	}
	for _, reloc := range relocations {
		srcDir := filepath.Join(templateDir, reloc.From)
		stat, err := os.Stat(srcDir)
		if err != nil {
			return fmt.Errorf("relocation source %q: %w", reloc.From, err)
		}
		if !stat.IsDir() {
			return fmt.Errorf("relocation source %q is not a directory", reloc.From)
		}
		if err := copyRelocationTree(srcDir, reloc.To, values); err != nil {
			return fmt.Errorf("relocate %s -> %s: %w", reloc.From, reloc.To, err)
		}
		if err := verifyTemplate(reloc.To); err != nil {
			return fmt.Errorf("relocate %s -> %s: %w", reloc.From, reloc.To, err)
		}
	}
	// Post hooks run from the repo root, NOT the scenario destination —
	// this is the structural difference from runTemplateHooks. They're
	// declared per-relocation but executed in declaration order after every
	// relocation has been written, so a single `make generate` covers all
	// of them when multiple relocations are siblings.
	repoRoot := deps.Root(ctx)
	for _, reloc := range relocations {
		for index, hook := range reloc.Post {
			name, args, description, err := resolveTemplateHook(hook)
			if err != nil {
				return fmt.Errorf("relocation post hook %d: %w", index+1, err)
			}
			cwd := repoRoot
			if hookCwd := strings.TrimSpace(hook.Cwd); hookCwd != "" && hookCwd != "." {
				cwd = filepath.Join(repoRoot, filepath.FromSlash(hookCwd))
			}
			env, err := templateHookEnv(deps.CommandEnv(ctx), hook.Env)
			if err != nil {
				return fmt.Errorf("relocation post hook %d: %w", index+1, err)
			}
			_, _ = fmt.Fprintf(output, "[Relocation post] %s\n", description)
			if err := deps.RunSubprocess(ctx, scenarioexec.SubprocessSpec{
				Name:   name,
				Args:   args,
				Dir:    cwd,
				Env:    env,
				Stdout: output,
				Stderr: deps.Stderr(ctx),
			}); err != nil {
				return fmt.Errorf("relocation post hook %q: %w", description, err)
			}
		}
	}
	return nil
}

// cleanupRelocationTargets removes the resolved To paths and any artifacts
// each post hook would have produced under packages/proto/gen/ for the
// validation scenario. Best-effort: errors are swallowed because the
// validation flow has already completed by the time cleanup runs.
//
// The proto/gen/ artifact paths mirror the repository's generated output
// layout. Go and TypeScript use the scenario id directly, and Python rewrites
// hyphens to underscores for package names.
func cleanupRelocationTargets(relocations []templatecontracts.ResolvedRelocation) {
	for _, path := range relocationArtifactPaths(relocations) {
		_ = os.RemoveAll(path)
	}
}

func relocationArtifactPaths(relocations []templatecontracts.ResolvedRelocation) []string {
	targets := make([]string, 0, len(relocations))
	for _, reloc := range relocations {
		targets = append(targets, reloc.To)
	}
	return templatevalidation.RelocationArtifactPaths(targets)
}

// copyRelocationTree mirrors copyTemplate's substitution logic but writes
// into an arbitrary repo-relative target instead of the scenario destination.
// File mode is preserved from the source; path components and text content
// are both rendered through renderTemplateString.
func copyRelocationTree(srcDir, destDir string, values map[string]string) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	return filepath.WalkDir(srcDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == srcDir {
			return nil
		}
		if entry.IsDir() && shouldSkipTemplateCopyDir(entry.Name()) {
			return filepath.SkipDir
		}
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		if filepath.Base(path) == ".DS_Store" {
			return nil
		}
		targetPath := filepath.Join(destDir, renderTemplateString(relPath, values))
		if entry.IsDir() {
			return os.MkdirAll(targetPath, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if looksLikeTextFile(data) {
			data = []byte(renderTemplateString(string(data), values))
		}
		return os.WriteFile(targetPath, data, info.Mode())
	})
}
