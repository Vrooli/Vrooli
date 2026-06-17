package dependencyhealth

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_health"
)

func (h *connectHandler) checkGoSurface(ctx context.Context, runner commandRunner, surface *healthv1.DependencyHealthSurface) []*healthv1.DependencyHealthFinding {
	root := surface.GetRootPath()
	modPath := filepath.Join(root, "go.mod")
	if !fileExists(modPath) {
		return []*healthv1.DependencyHealthFinding{readinessFinding("go."+surfaceID(surface)+".missing-go-mod", "WARNING", "Go surface has no go.mod", "A discovered Go surface does not have a go.mod at its root.", "Expose the correct parse-unit root through Code Facts or add a go.mod for this Go module.", surface, "dependency.go.mod_present", "missing go.mod", "go.mod at the Go surface root")}
	}
	var findings []*healthv1.DependencyHealthFinding
	for _, missing := range missingLocalReplaces(root, modPath) {
		findings = append(findings, readinessFinding("go."+surfaceID(surface)+".replace."+slug(missing), "ERROR", "Local replace target missing", "A go.mod replace directive points to a local path that does not exist.", "Update the replace directive so the local target exists, or remove the stale replacement.", surface, "dependency.go.local_replace_resolves", missing, "existing local replace target"))
	}
	out, err := runner.Run(ctx, root, "go", "mod", "tidy", "-diff")
	if err != nil {
		observed := strings.TrimSpace(out)
		if observed == "" {
			observed = err.Error()
		}
		findings = append(findings, readinessFinding("go."+surfaceID(surface)+".tidy-diff", "ERROR", "Go module metadata is not tidy", "go mod tidy -diff reported module metadata drift or failed.", "Run `cd "+filepath.ToSlash(root)+" && GOWORK=off go mod tidy`, then rerun dependency health.", surface, "dependency.go.tidy", observed, "go mod tidy -diff exits cleanly"))
	}
	return findings
}

func missingLocalReplaces(moduleRoot, modPath string) []string {
	data, err := os.ReadFile(modPath)
	if err != nil {
		return nil
	}
	var missing []string
	inBlock := false
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		if inBlock {
			if line == ")" {
				inBlock = false
				continue
			}
			if miss := missingReplaceTarget(moduleRoot, line); miss != "" {
				missing = append(missing, miss)
			}
			continue
		}
		if strings.HasPrefix(line, "replace (") {
			inBlock = true
			continue
		}
		if strings.HasPrefix(line, "replace ") {
			if miss := missingReplaceTarget(moduleRoot, strings.TrimPrefix(line, "replace ")); miss != "" {
				missing = append(missing, miss)
			}
		}
	}
	sort.Strings(missing)
	return missing
}

var localReplaceRE = regexp.MustCompile(`^([^\s]+)(?:\s+v\S+)?\s+=>\s+(\S+)`)

func missingReplaceTarget(moduleRoot, line string) string {
	match := localReplaceRE.FindStringSubmatch(line)
	if match == nil || !isLocalPath(match[2]) {
		return ""
	}
	target := match[2]
	if !filepath.IsAbs(target) {
		target = filepath.Join(moduleRoot, filepath.FromSlash(target))
	}
	if dirExists(target) {
		return ""
	}
	return filepath.ToSlash(match[2])
}

func isLocalPath(path string) bool {
	return strings.HasPrefix(path, ".") || filepath.IsAbs(path)
}
