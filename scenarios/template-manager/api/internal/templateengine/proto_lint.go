package templateengine

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/internal/scenarioexec"
	templatecontracts "github.com/vrooli/vrooli/scenarios/template-manager/api/internal/templatecontracts"
)

// validateRelocationProtoSources runs `buf lint` against template-side
// proto source folders so schema-level mistakes (missing package
// directive, syntax errors, naming convention violations) surface in
// template validation rather than after a real scenario generation.
//
// The "is this proto?" decision is heuristic: any relocation source that
// contains a .proto file is treated as one. Future non-proto relocations
// (e.g., scripts) won't be confused for protos because they won't have
// .proto files inside.
//
// Implementation note: `buf lint --path` only accepts paths inside the
// buf module (packages/proto/schemas/). The template's protos live
// outside that module pre-substitution, so we copy them into a temp
// subdirectory under schemas/ with template-validation seed values
// applied, lint there, and clean up. The temp directory name is
// prefixed with `.tmp-validate-` so it can never collide with a real
// scenario schema directory.
//
// Skipped entirely when deps.RunSubprocess is nil (mirrors the pattern
// used by validateGeneratedScenario for `go mod tidy`).
func validateRelocationProtoSources[C any](deps HandlerDeps[C], ctx C, info templatecontracts.TemplateInfo) []templatecontracts.TemplateValidationIssue {
	if deps.RunSubprocess == nil {
		return nil
	}
	if len(info.Manifest.Relocations) == 0 {
		return nil
	}
	repoRoot := deps.Root(ctx)
	protoPackageDir := filepath.Join(repoRoot, "packages", "proto")
	schemasDir := filepath.Join(protoPackageDir, "schemas")
	if _, err := os.Stat(schemasDir); err != nil {
		// No proto module in this repo (e.g., test fixtures with a
		// minimal repo-contract). The template's claim is that protos
		// belong here, so absence isn't a per-template issue — the
		// generator would fail at make-generate time, which is a
		// separate failure mode.
		return nil
	}
	var issues []templatecontracts.TemplateValidationIssue
	values := templateValidationSeedValues(info)
	if id, ok := values["SCENARIO_ID"]; ok && strings.TrimSpace(id) != "" {
		values["SCENARIO_ID_SNAKE"] = strings.ReplaceAll(id, "-", "_")
	}
	for _, reloc := range info.Manifest.Relocations {
		from := strings.TrimSpace(reloc.From)
		if from == "" {
			continue
		}
		srcDir := filepath.Join(info.Path, filepath.FromSlash(from))
		if !directoryContainsProto(srcDir) {
			continue
		}
		tmpDir, err := os.MkdirTemp(schemasDir, ".tmp-validate-"+info.Name+"-")
		if err != nil {
			issues = append(issues, templatecontracts.TemplateValidationIssue{
				Template: info.Name,
				Path:     filepath.ToSlash(from),
				Message:  fmt.Sprintf("create lint temp dir: %v", err),
			})
			continue
		}
		// Best-effort cleanup; lint failures are surfaced through
		// `issues` regardless of whether the cleanup succeeds.
		shouldClean := true
		defer func(path string, doClean *bool) {
			if *doClean {
				_ = os.RemoveAll(path)
			}
		}(tmpDir, &shouldClean)
		if err := copyRelocationTree(srcDir, tmpDir, values); err != nil {
			issues = append(issues, templatecontracts.TemplateValidationIssue{
				Template: info.Name,
				Path:     filepath.ToSlash(from),
				Message:  fmt.Sprintf("substitute proto sources for lint: %v", err),
			})
			continue
		}
		// `buf lint --path` is now scoped to the temp dir which lives
		// inside the buf module, so the lint succeeds.
		//
		// `buf lint` writes lint diagnostics to stdout (one per line) and
		// exits non-zero. We capture both streams and prefer stdout for the
		// surfaced message because that's where the actionable detail lives.
		// The temp-dir path prefix in each diagnostic line is also stripped
		// so the surfaced message matches what an author would see if they
		// ran `buf lint` directly against the template's proto/.
		var stdout, stderr bytes.Buffer
		relTmp, err := filepath.Rel(protoPackageDir, tmpDir)
		if err != nil {
			relTmp = tmpDir
		}
		err = deps.RunSubprocess(ctx, scenarioexec.SubprocessSpec{
			Name:   "bash",
			Args:   []string{"-lc", fmt.Sprintf("buf lint --path %s", shellQuote(relTmp))},
			Dir:    protoPackageDir,
			Env:    deps.CommandEnv(ctx),
			Stdout: &stdout,
			Stderr: &stderr,
		})
		if err != nil {
			msg := strings.TrimSpace(stdout.String())
			if msg == "" {
				msg = strings.TrimSpace(stderr.String())
			}
			if msg == "" {
				msg = err.Error()
			}
			// Strip the temp-dir prefix so diagnostics read as if buf lint
			// had been run against the template's source proto/ directly.
			fromPrefix := strings.TrimRight(filepath.ToSlash(from), "/") + "/"
			msg = strings.ReplaceAll(msg, filepath.ToSlash(relTmp)+"/", fromPrefix)
			issues = append(issues, templatecontracts.TemplateValidationIssue{
				Template: info.Name,
				Path:     filepath.ToSlash(from),
				Message:  fmt.Sprintf("buf lint failed: %s", msg),
			})
		}
	}
	return issues
}

// directoryContainsProto reports whether the directory tree rooted at
// path contains any .proto files. Walks until the first match.
func directoryContainsProto(path string) bool {
	stat, err := os.Stat(path)
	if err != nil || !stat.IsDir() {
		return false
	}
	found := false
	_ = filepath.WalkDir(path, func(p string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(p), ".proto") {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

// shellQuote returns a single-quoted shell argument that survives buf's
// `bash -lc` invocation. Used for absolute paths that may contain
// shell-special characters; deliberately conservative.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
