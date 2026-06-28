package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// roleHeadBytes is how many leading bytes are read for the generated-marker
// scan. Generated headers always sit at the top of the file.
const roleHeadBytes = 1024

// roleCache classifies repo-relative file paths into FileRoles, caching results
// (and the leading-bytes read needed for the generated-marker scan) so each
// file is classified at most once per scan.
type roleCache struct {
	scenarioPath string
	cache        map[string]FileRole
}

// newRoleCache returns a roleCache rooted at scenarioPath.
func newRoleCache(scenarioPath string) *roleCache {
	return &roleCache{scenarioPath: scenarioPath, cache: make(map[string]FileRole)}
}

// role returns the FileRole for a repo-relative path, reading the file's leading
// bytes only when the path alone has not already determined the role.
func (rc *roleCache) role(relPath string) FileRole {
	if relPath == "" {
		return FileRoleProduction
	}
	if cached, ok := rc.cache[relPath]; ok {
		return cached
	}
	// Cheap path-only pass first; only read the file when it is still ambiguous
	// (path conventions did not already classify it as Generated).
	role := ClassifyFileRole(relPath, nil)
	if role == FileRoleProduction || role == FileRoleCompositionRoot || role == FileRoleDeclarativeWiring {
		if head := rc.readHead(relPath); len(head) > 0 && isGeneratedMarker(head) {
			role = FileRoleGenerated
		}
	}
	rc.cache[relPath] = role
	return role
}

// readHead reads up to roleHeadBytes from the file; returns nil on any error.
func (rc *roleCache) readHead(relPath string) []byte {
	file, err := os.Open(filepath.Join(rc.scenarioPath, relPath))
	if err != nil {
		return nil
	}
	defer file.Close()
	buf := make([]byte, roleHeadBytes)
	n, _ := file.Read(buf)
	return buf[:n]
}

// duplicationFindings converts detected duplicate blocks into tidiness findings,
// applying role-aware severity. Generated files are excluded entirely; structural
// duplication in a cappable role (test/test-support/declarative-wiring) is
// downgraded to a warning, while genuine logic keeps its base severity.
func duplicationFindings(scenarioName, scenarioPath string, roles *roleCache, dup *DuplicateResult) []TidinessFinding {
	if dup == nil || dup.Skipped {
		return nil
	}
	findings := make([]TidinessFinding, 0, len(dup.DuplicateBlocks))
	for i, block := range dup.DuplicateBlocks {
		primaryPath := ""
		line := 0
		if len(block.Files) > 0 {
			primaryPath = block.Files[0].Path
			line = block.Files[0].StartLine
		}
		primaryRole := roles.role(primaryPath)
		// Generated code is fully excluded.
		if primaryRole == FileRoleGenerated {
			continue
		}
		// Cap structural duplication at warning: a uniform descriptor list or
		// test-scaffolding block is enforced consistency, not logic debt.
		// Content-confirm (IsStructuralBlock) guards against masking — a genuine
		// logic block in a role-named file keeps its normal severity.
		var blockLines []string
		if roleAllowsStructuralCap(primaryRole) && primaryPath != "" {
			blockLines, _ = readBlockLines(filepath.Join(scenarioPath, primaryPath), line, block.Lines)
		}
		severity, structural := resolveDuplicationSeverity(severityForDuplication(float64(block.Lines), 10), primaryRole, blockLines)
		evidence := map[string]any{"lines": block.Lines, "locations": block.Files, "tool": dup.Tool, "file_role": primaryRole.String()}
		if structural {
			evidence["structural"] = true
		}
		findings = append(findings, newTidinessFinding(scenarioName, "duplicated-code", "duplication", severity, primaryPath, "", line,
			fmt.Sprintf("Duplicated block spans %d lines", block.Lines),
			fmt.Sprintf("Duplicated code block #%d spans %d lines across %d locations.", i+1, block.Lines, len(block.Files)),
			evidence,
			"Duplicated code multiplies future fixes and makes behavior drift likely.",
			"Extract the shared behavior or intentionally document why the copies must diverge.",
			"duplication"))
	}
	return findings
}

// resolveDuplicationSeverity returns the final severity for a duplicated block
// and whether it was confirmed structural. Structural duplication in a
// cappable role (test, test-support, declarative-wiring) is downgraded to a
// warning; everything else — including genuine logic in a role-named file —
// keeps its base severity. blockLines may be nil (treated as non-structural).
func resolveDuplicationSeverity(baseSeverity string, role FileRole, blockLines []string) (severity string, structural bool) {
	if roleAllowsStructuralCap(role) && IsStructuralBlock(blockLines) {
		return capSeverityAtMedium(baseSeverity), true
	}
	return baseSeverity, false
}

// roleAllowsStructuralCap reports whether duplication in a file of this role may
// be downgraded to a warning when its content is confirmed structural.
func roleAllowsStructuralCap(role FileRole) bool {
	switch role {
	case FileRoleTest, FileRoleTestSupport, FileRoleDeclarativeWiring:
		return true
	default:
		return false
	}
}

// FileRole is the structural role a file plays in a screaming-architecture
// codebase. Tidiness checks consult the role before deciding whether uniform
// duplication, high coupling, or file length is real maintainability debt or
// the very consistency the architecture enforces.
//
// Role names echo architecture-cartographer's zone vocabulary where they
// overlap (notably CompositionRoot). The alignment is intentional but informal:
// there is no runtime dependency on cartographer and no shared SSOT artifact yet
// (deferred — see plan §4/§12). If a shared cross-scenario role vocabulary is
// ever extracted, this enum is the producer side to reconcile.
type FileRole int

const (
	// FileRoleProduction is ordinary hand-written application logic (default).
	FileRoleProduction FileRole = iota
	// FileRoleTest is a test file (e.g. *_test.go, *.spec.ts).
	FileRoleTest
	// FileRoleTestSupport is hand-written test scaffolding (mocks, fixtures,
	// testutil, testdata). Relaxed thresholds, never excluded — its findings
	// stay visible as warnings.
	FileRoleTestSupport
	// FileRoleGenerated is machine-generated code (marker or path convention).
	// Fully excluded from the scan.
	FileRoleGenerated
	// FileRoleCompositionRoot is a high-fan-in aggregator whose job is to wire
	// many collaborators together (handler.go, app/modules.go, registry.go).
	// High import counts are by design, not coupling debt.
	FileRoleCompositionRoot
	// FileRoleDeclarativeWiring is a uniform descriptor/registration file or a
	// declarative const registry (endpoints.go, module.go, register.go,
	// selectors.ts). Cross-file uniformity here is enforced consistency, not
	// duplication debt.
	FileRoleDeclarativeWiring
)

// String returns a stable, lower-kebab label for the role (used in evidence).
func (r FileRole) String() string {
	switch r {
	case FileRoleProduction:
		return "production"
	case FileRoleTest:
		return "test"
	case FileRoleTestSupport:
		return "test-support"
	case FileRoleGenerated:
		return "generated"
	case FileRoleCompositionRoot:
		return "composition-root"
	case FileRoleDeclarativeWiring:
		return "declarative-wiring"
	default:
		return "production"
	}
}

// generatedMarkerRe matches the canonical Go "Code generated … DO NOT EDIT"
// header. Markers must appear in the file's leading bytes.
var generatedMarkerRe = regexp.MustCompile(`(?m)^//\s*Code generated .* DO NOT EDIT\.?`)

// ClassifyFileRole returns the structural role of a file. It is a pure function
// of the repo-relative path plus (for the Generated marker scan only) the file's
// leading bytes. headBytes may be nil — path conventions still classify, but
// marker-only generated files will not be detected without their leading bytes.
//
// Precedence (most specific first): Generated → Test → TestSupport →
// DeclarativeWiring → CompositionRoot → Production.
func ClassifyFileRole(repoRelPath string, headBytes []byte) FileRole {
	p := filepath.ToSlash(repoRelPath)
	base := path.Base(p)

	switch {
	case isGeneratedMarker(headBytes) || isGeneratedPath(p, base):
		return FileRoleGenerated
	case IsTestFilePath(repoRelPath):
		return FileRoleTest
	case isTestSupportPath(p):
		return FileRoleTestSupport
	case isDeclarativeWiringPath(p, base):
		return FileRoleDeclarativeWiring
	case isCompositionRootPath(base):
		return FileRoleCompositionRoot
	default:
		return FileRoleProduction
	}
}

// isGeneratedMarker reports whether the leading bytes carry a generated-code
// marker. Two markers are recognized: the Go "Code generated … DO NOT EDIT"
// header and the language-agnostic "AUTO-GENERATED" tag.
func isGeneratedMarker(headBytes []byte) bool {
	if len(headBytes) == 0 {
		return false
	}
	if generatedMarkerRe.Match(headBytes) {
		return true
	}
	return strings.Contains(string(headBytes), "AUTO-GENERATED")
}

// isGeneratedPath reports whether the path follows a generated-code naming
// convention. Note: /mocks/ is deliberately NOT here — hand-written mocks are
// TestSupport (relaxed, kept visible); a mock is only Generated when it also
// carries a marker.
func isGeneratedPath(p, base string) bool {
	switch {
	case strings.HasSuffix(base, ".pb.go"),
		strings.HasSuffix(base, ".connect.go"),
		strings.HasSuffix(base, ".pb.gw.go"),
		strings.Contains(base, ".generated."):
		return true
	case strings.Contains(p, "/gen/"), strings.HasPrefix(p, "gen/"):
		return true
	default:
		return false
	}
}

// isTestSupportPath reports whether the path is hand-written test scaffolding.
func isTestSupportPath(p string) bool {
	return strings.Contains(p, "/mocks/") ||
		strings.Contains(p, "/fixtures/") ||
		strings.Contains(p, "/testutil/") ||
		strings.Contains(p, "/testdata/") ||
		strings.HasPrefix(p, "testdata/")
}

// declarativeWiringBases are filenames whose entire job is uniform descriptor
// lists or declarative registries — the consistency screaming architecture
// enforces, reported by naive duplication detection as debt.
var declarativeWiringBases = map[string]bool{
	"endpoints.go": true,
	"module.go":    true,
	"proto.go":     true,
	"register.go":  true,
	// Declarative const registries (UI). Long flat constant tables, not logic.
	"selectors.ts": true,
	"constants.ts": true,
}

// isDeclarativeWiringPath reports whether the file is a declarative wiring or
// const-registry file.
func isDeclarativeWiringPath(p, base string) bool {
	if declarativeWiringBases[base] {
		return true
	}
	// CLI command wiring: cli/.../handlers.go is a declarative dispatch table,
	// unlike an api-side handlers.go which may carry real logic.
	if base == "handlers.go" && (strings.Contains(p, "/cli/") || strings.HasPrefix(p, "cli/")) {
		return true
	}
	return false
}

// compositionRootBases are filenames that aggregate many collaborators. Their
// high fan-in (import count) is structural, not coupling debt.
var compositionRootBases = map[string]bool{
	"handler.go":  true,
	"modules.go":  true,
	"registry.go": true,
	"routes.go":   true,
	"wire.go":     true,
	"server.go":   true,
	"app.go":      true,
	"main.go":     true,
}

// isCompositionRootPath reports whether the file is a high-fan-in composition
// root.
func isCompositionRootPath(base string) bool {
	return compositionRootBases[base]
}
