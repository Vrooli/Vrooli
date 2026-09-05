package hygiene

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const pathAuthorityProviderID = "runtime-path-authority"

var handBuiltRuntimeHomeJoin = regexp.MustCompile("(?i)(filepath\\.Join\\([^\\n)]*(home|userHome)[^\\n)]*\\\"\\.vrooli\\\"|filepath\\.Join\\([^\\n)]*\\\"\\.vrooli\\\"[^\\n)]*(home|userHome)|\\b(home|userHome)\\s*\\+\\s*[\\\"`]\\.?/?\\.vrooli)")

type pathAuthorityProvider struct{ root string }

func (p pathAuthorityProvider) ID() string            { return pathAuthorityProviderID }
func (p pathAuthorityProvider) Budget() time.Duration { return 5 * time.Second }
func (p pathAuthorityProvider) Run(_ context.Context, _ Request, report *Report) error {
	var violations []string
	for _, scenario := range []string{"vrooli-autoheal", "agent-manager", "prompt-manager"} {
		base := filepath.Join(p.root, "scenarios", scenario)
		err := filepath.WalkDir(base, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			for lineNumber, line := range strings.Split(string(data), "\n") {
				if handBuiltRuntimeHomeJoin.MatchString(line) {
					location, _ := filepath.Rel(p.root, path)
					violations = append(violations, fmt.Sprintf("%s:%d", filepath.ToSlash(location), lineNumber+1))
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
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
		Message:    "runtime paths must be resolved through api-core/storage; home/.vrooli joins are not portable",
		Why:        "A custom VROOLI_STORAGE_ROOT must move runtime state without source changes.",
		Fixability: FixabilityManual,
	})
	return nil
}
