package hygiene

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
)

const pathAuthorityProviderID = "runtime-path-authority"

// handBuiltRuntimeHomeJoin matches a home directory joined inline with the
// ".vrooli" runtime-home name.
var handBuiltRuntimeHomeJoin = regexp.MustCompile("(?i)(filepath\\.Join\\([^\\n)]*(home|userHome)[^\\n)]*\\\"\\.vrooli\\\"|filepath\\.Join\\([^\\n)]*\\\"\\.vrooli\\\"[^\\n)]*(home|userHome)|\\b(home|userHome)\\s*\\+\\s*[\\\"`]\\.?/?\\.vrooli)")

// runtimeHomeRelLiteral matches a named constant or variable holding a
// runtime-home-relative path (".vrooli/..."). Joined to a home directory it is
// the same bypass as an inline join, one declaration removed.
var runtimeHomeRelLiteral = regexp.MustCompile("\\b([A-Za-z_][A-Za-z0-9_]*)\\s*=\\s*[\"`]\\.vrooli/")

var homeJoin = regexp.MustCompile("(?i)filepath\\.Join\\([^\\n]*\\b(home|userHome)\\b")

// pathAuthorityProvider scans every source root the repo contract declares.
// debt is the per-file ratchet of known sites (nil means pathAuthorityDebt).
type pathAuthorityProvider struct {
	root string
	debt map[string]int
}

// runtimeHomeBypassLines returns the 1-based lines of source that build a
// runtime-home path from a home directory by hand: an inline home/.vrooli
// join, or a home join with a declared ".vrooli/..." relative path.
func runtimeHomeBypassLines(source string) []int {
	// Both forms name ".vrooli" somewhere in the file; most source never does.
	if !strings.Contains(source, ".vrooli") {
		return nil
	}
	lines := strings.Split(source, "\n")
	var relNames []string
	for _, line := range lines {
		if !strings.Contains(line, ".vrooli/") {
			continue
		}
		if match := runtimeHomeRelLiteral.FindStringSubmatch(line); match != nil {
			relNames = append(relNames, regexp.QuoteMeta(match[1]))
		}
	}
	var relUse *regexp.Regexp
	if len(relNames) > 0 {
		relUse = regexp.MustCompile("\\b(" + strings.Join(relNames, "|") + ")\\b")
	}
	var hits []int
	for index, line := range lines {
		inline := strings.Contains(line, ".vrooli") && handBuiltRuntimeHomeJoin.MatchString(line)
		if inline || (relUse != nil && relUse.MatchString(line) && homeJoin.MatchString(line)) {
			hits = append(hits, index+1)
		}
	}
	return hits
}

func (p pathAuthorityProvider) ID() string            { return pathAuthorityProviderID }
func (p pathAuthorityProvider) Budget() time.Duration { return 5 * time.Second }
func (p pathAuthorityProvider) Run(_ context.Context, _ Request, report *Report) error {
	root, err := filepath.Abs(p.root)
	if err != nil {
		return err
	}
	contract, err := repocontract.LoadDefault(root)
	if err != nil {
		return fmt.Errorf("load repo contract: %w", err)
	}
	hits := map[string][]int{}
	err = contract.WalkSource(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return skipUnreadable(walkErr, entry)
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			if isUnreadable(err) {
				return nil
			}
			return err
		}
		if lines := runtimeHomeBypassLines(string(data)); len(lines) > 0 {
			location, _ := filepath.Rel(root, path)
			hits[filepath.ToSlash(location)] = lines
		}
		return nil
	})
	if err != nil {
		return err
	}
	debt := p.debt
	if debt == nil {
		debt = pathAuthorityDebt
	}
	var violations, ledgered, stale []string
	for _, file := range sortedKeys(hits) {
		target := &violations
		if len(hits[file]) <= debt[file] {
			target = &ledgered
		}
		for _, line := range hits[file] {
			*target = append(*target, fmt.Sprintf("%s:%d", file, line))
		}
	}
	for _, file := range sortedKeys(debt) {
		if found := len(hits[file]); found < debt[file] {
			stale = append(stale, fmt.Sprintf("%s (ledger %d, found %d)", file, debt[file], found))
		}
	}
	if len(ledgered) > 0 {
		report.addFinding(Finding{
			Severity:   SeverityWarning,
			Code:       "runtime_path_authority_debt",
			Locations:  ledgered,
			Message:    "known hand-built runtime-home paths remain; each owner should resolve them through api-core/storage or the repo contract's runtime_home entries",
			Why:        "The ledger in path_authority_debt.go may only shrink.",
			Fixability: FixabilityManual,
		})
	}
	if len(stale) > 0 {
		report.addFinding(Finding{
			Severity:   SeverityWarning,
			Code:       "runtime_path_authority_debt_stale",
			Locations:  stale,
			Message:    "the path authority ledger allows more sites than these files hold; lower or delete their entries in path_authority_debt.go",
			Why:        "A ledger that outlives its debt would let new bypasses back in silently.",
			Fixability: FixabilityManual,
		})
	}
	if len(violations) == 0 {
		report.addCheck(pathAuthorityProviderID, true, SeverityInfo, "runtime paths use the storage/contract authority")
		return nil
	}
	report.addCheck(pathAuthorityProviderID, false, SeverityError, "runtime path joins bypass the storage authority")
	report.addFinding(Finding{
		Severity:   SeverityError,
		Code:       "runtime_path_authority_bypass",
		Locations:  violations,
		Message:    "runtime paths must be resolved through api-core/storage (scenario storage classes) or the repo contract's runtime_home entries; hand-built home/.vrooli paths are not portable",
		Why:        "A custom VROOLI_STORAGE_ROOT must move scenario runtime state without source changes, and runtime-home entries must follow the contract that declares them.",
		Fixability: FixabilityManual,
	})
	return nil
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
