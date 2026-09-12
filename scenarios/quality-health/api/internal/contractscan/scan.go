// Package contractscan owns repository-wide static policy checks that are
// quality concerns rather than repository-shape concerns.
package contractscan

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type Finding struct {
	Code        string
	Message     string
	Location    string
	Remediation string
}

var (
	personalPathPattern     = regexp.MustCompile(`(?:^|[^A-Za-z0-9._-])(?:/home|/Users)/[A-Za-z0-9._-]+(?:/|[[:space:]"']|$)`)
	operatorIdentityPattern = regexp.MustCompile(`(?i)\b(?:matthalloran8|matt(?:hew)?[[:space:]_-]*halloran)\b`)
	runtimeHomeJoinPattern  = regexp.MustCompile(`filepath\.Join\([^,)]*(?i:home)[^,)]*,\s*"\.vrooli"`)
	homeSubpathPattern      = regexp.MustCompile(`"[~/][^"]*\.vrooli/`)
)

var adoptionPatterns = []struct {
	name    string
	pattern *regexp.Regexp
}{
	{"ad_hoc_repo_root_detector", regexp.MustCompile(`(?m)^func (?:\([^)]*\) )?(?:findRepoRoot|getVrooliRoot|FindRepoRoot|DetectVrooliRoot)\(`)},
	{"legacy_vrooli_home_fallback", regexp.MustCompile(`\$HOME/Vrooli|filepath\.Join\([^,\n]+,\s*"Vrooli"(?:,\s*"scenarios")?`)},
	{"canonical_service_manifest_join", regexp.MustCompile(`filepath\.Join\([^,\n]+,\s*"scenarios",\s*[^,\n]+,\s*"\.vrooli",\s*"service\.json"\)`)},
	{"app_root_repo_env", regexp.MustCompile(`os\.Getenv\("APP_ROOT"\)`)},
	{"git_marker_repo_root", regexp.MustCompile(`(?s)func (?:\([^)]*\) )?(?:findRepoRoot|FindRepoRoot|resolveRepoRoot|ResolveRepoRoot|detectRepoRoot)\([^)]*\).*?filepath\.Join\([^,\n]+,\s*"\.git"\)`)},
	{"pnpm_workspace_repo_root", regexp.MustCompile(`(?s)func (?:\([^)]*\) )?(?:findRepoRoot|FindRepoRoot|resolveRepoRoot|ResolveRepoRoot|detectRepoRoot)\([^)]*\).*?filepath\.Join\([^,\n]+,\s*"pnpm-workspace\.yaml"\)`)},
}

var hostPatterns = []struct {
	name    string
	pattern *regexp.Regexp
}{
	{"proc_meminfo", regexp.MustCompile(`/proc/meminfo`)},
	{"proc_loadavg", regexp.MustCompile(`/proc/loadavg`)},
	{"nvidia_smi_inventory", regexp.MustCompile(`(?i)(LookPath\("nvidia-smi"\)|\bnvidia-smi\b[^\n]*(--query-gpu|--query-compute-apps)|--query-compute-apps)`)},
	{"system_profiler_gpu_inventory", regexp.MustCompile(`system_profiler[^\n]*SPDisplaysDataType|SPDisplaysDataType[^\n]*system_profiler`)},
	{"wmic_gpu_inventory", regexp.MustCompile(`(?i)\bwmic\b[^\n]*(VideoController|win32_VideoController)|VideoController[^\n]*\bwmic\b`)},
	{"docker_info_nvidia_probe", regexp.MustCompile(`(?i)docker\s+info[^\n]*(nvidia|runtime)|nvidia[^\n]*docker\s+info`)},
	{"grdctl_remote_desktop", regexp.MustCompile(`(?i)(?:Output|CombinedOutput|Run|LookPath|commandStatus|commandFn)\([^\n]*["']grdctl["']`)},
	{"loginctl_session", regexp.MustCompile(`(?i)(?:Output|CombinedOutput|Run|LookPath|commandFn|run)\([^\n]*["']loginctl["'][^\n]*["']show-session["']`)},
	{"gnome_remote_desktop_service", regexp.MustCompile(`(?i)(?:Output|CombinedOutput|Run|LookPath|commandStatus)\([^\n]*(?:is-enabled|is-active|list-unit-files)[^\n]*gnome-remote-desktop\.service`)},
	{"xrandr_display", regexp.MustCompile(`(?i)(?:Output|CombinedOutput|Run|LookPath)\([^\n]*["']xrandr["']`)},
	{"windows_termservice", regexp.MustCompile(`(?i)(?:Output|CombinedOutput|Run|LookPath|commandFn|run)\([^\n]*["']sc(?:\.exe)?["'][^\n]*["']query["'][^\n]*["']TermService["']`)},
	{"gdm_udev_marker", regexp.MustCompile(`(?:ReadFile|Open|Stat|Lstat)\([^\n]*/run/udev/gdm-machine-has-vendor-nvidia-driver`)},
	{"gdm_custom_conf", regexp.MustCompile(`(?:ReadFile|Open|Stat|Lstat)\([^\n]*/((etc|run)/gdm3?)/custom\.conf`)},
}

func Scan(root string) ([]Finding, error) {
	var out []Finding
	var outMu sync.Mutex
	walkFiles(root, []string{"cmd", "internal", "packages", "resources", "scenarios", "templates", "docs", ".vrooli"}, func(rel string, data []byte) {
		var fileFindings []Finding
		personal := under(rel, []string{"cmd", "internal", "packages", "resources", "scenarios", "templates", "docs", ".vrooli"}) && scannable(rel) && !isBinary(data) && !skipPersonal(rel) && !isContractScanAuthority(rel)
		runtimeHome := under(rel, []string{"cmd", "internal", "packages"}) && filepath.Ext(rel) == ".go" && !strings.HasSuffix(rel, "_test.go") && !strings.HasPrefix(rel, "packages/repo-contract-go/") && !strings.HasPrefix(rel, "internal/repocontractmeta/") && !isContractScanAuthority(rel)
		host := under(rel, []string{"cmd", "internal", "packages", "resources", "scenarios"}) && isHostScannable(rel) && !strings.HasSuffix(rel, "_test.go") && !strings.Contains(rel, "/testdata/") && !strings.Contains(rel, "/tests/") && !strings.HasPrefix(rel, "internal/hostinventory/") && !strings.Contains(rel, "/gen/") && !strings.Contains(rel, "/generated/") && !isContractScanAuthority(rel)
		if personal || runtimeHome || host {
			for line, text := range strings.Split(string(data), "\n") {
				if personal && (personalPathPattern.MatchString(text) || operatorIdentityPattern.MatchString(text)) {
					fileFindings = append(fileFindings, Finding{"QUALITY_PERSONAL_ABSOLUTE_PATHS", "personal absolute path or operator identity found", relLine(rel, line+1), "Use repository-relative/configured paths and remove operator-specific identities."})
				}
				if runtimeHome {
					if !strings.Contains(text, "repo-contract:project-config") && !strings.HasPrefix(strings.TrimSpace(text), "//") && (runtimeHomeJoinPattern.MatchString(text) || homeSubpathPattern.MatchString(text)) {
						fileFindings = append(fileFindings, Finding{"QUALITY_NO_RUNTIME_HOME_LITERALS", "runtime-home literal drift", relLine(rel, line+1), "Use config.VrooliPath or repository runtime-home helpers; annotate genuine project-config joins with // repo-contract:project-config."})
					}
				}
				if host && !strings.HasPrefix(strings.TrimSpace(text), "//") && !strings.Contains(text, "hostinventory:remote-snapshot-parser") {
					if !hostCandidateLine(text) {
						continue
					}
					for _, rule := range hostPatterns {
						if rule.pattern.MatchString(text) {
							fileFindings = append(fileFindings, Finding{"QUALITY_HOST_INVENTORY_AUTHORITY", "host-inventory authority violation: " + rule.name, relLine(rel, line+1), "Use internal/hostinventory, or annotate a reviewed remote snapshot parser with hostinventory:remote-snapshot-parser."})
						}
					}
				}
			}
		}
		if under(rel, []string{"cmd", "internal", "packages", "scenarios"}) && filepath.Ext(rel) == ".go" && !strings.HasSuffix(rel, "_test.go") && !strings.HasPrefix(rel, "packages/repo-contract-go/") && !strings.HasPrefix(rel, "internal/repocontract/") && !isContractScanAuthority(rel) {
			text := string(data)
			for _, rule := range adoptionPatterns {
				if rule.pattern.MatchString(text) && !(rule.name == "ad_hoc_repo_root_detector" && strings.Contains(text, "repocontract.")) {
					fileFindings = append(fileFindings, Finding{"QUALITY_ADOPTION_RULES_ALIGNMENT", "repository contract adoption violation: " + rule.name, rel, "Use shared repository-contract helpers instead of local path or root detection."})
				}
			}
		}
		if len(fileFindings) > 0 {
			outMu.Lock()
			out = append(out, fileFindings...)
			outMu.Unlock()
		}
	})
	sort.Slice(out, func(i, j int) bool {
		if out[i].Code != out[j].Code {
			return out[i].Code < out[j].Code
		}
		return out[i].Location < out[j].Location
	})
	return out, nil
}

func walkFiles(root string, topLevels []string, visit func(string, []byte)) {
	var wg sync.WaitGroup
	for _, top := range topLevels {
		wg.Add(1)
		go func(top string) {
			defer wg.Done()
			base := filepath.Join(root, filepath.FromSlash(top))
			_ = filepath.WalkDir(base, func(path string, entry os.DirEntry, err error) error {
				if err != nil {
					if os.IsNotExist(err) {
						return filepath.SkipDir
					}
					return err
				}
				rel, relErr := filepath.Rel(root, path)
				if relErr != nil {
					return relErr
				}
				rel = filepath.ToSlash(rel)
				if entry.IsDir() {
					if skipDir(rel) {
						return filepath.SkipDir
					}
					return nil
				}
				if entry.Type()&os.ModeSymlink != 0 || !scannable(rel) {
					return nil
				}
				data, readErr := os.ReadFile(path)
				if readErr == nil {
					visit(rel, data)
				}
				return nil
			})
		}(top)
	}
	wg.Wait()
}

func under(rel string, roots []string) bool {
	for _, root := range roots {
		if rel == root || strings.HasPrefix(rel, root+"/") {
			return true
		}
	}
	return false
}

func skipDir(rel string) bool {
	base := filepath.Base(rel)
	switch base {
	case ".git", "node_modules", ".venv", "vendor", "dist", "build", "coverage", ".cache", ".gocache", ".nyc_output", ".claude", "tmp", "temp", "logs", "data", "investigations", "report", ".swarm", "review", "evidence", "captures", "handoff", "gen", "generated":
		return true
	}
	return false
}

func skipPersonal(rel string) bool {
	if strings.HasSuffix(rel, ".log") || strings.HasSuffix(rel, "/acceptance-validation.json") || filepath.Base(rel) == "test_output.txt" || strings.EqualFold(filepath.Ext(rel), ".md") {
		return true
	}
	return skipDir(filepath.Dir(rel))
}

func scannable(rel string) bool {
	switch strings.ToLower(filepath.Ext(rel)) {
	case ".go", ".sh", ".bash", ".js", ".cjs", ".mjs", ".ts", ".tsx", ".json", ".jsonl", ".yaml", ".yml", ".toml":
		return true
	default:
		return false
	}
}

func isHostScannable(rel string) bool {
	switch strings.ToLower(filepath.Ext(rel)) {
	case ".go", ".sh", ".bash":
		return true
	}
	return false
}

func hostCandidateLine(line string) bool {
	lower := strings.ToLower(line)
	for _, marker := range []string{
		"/proc/", "nvidia-smi", "system_profiler", "wmic", "docker info", "grdctl", "loginctl", "gnome-remote-desktop", "xrandr", "termservice", "gdm-machine-has-vendor-nvidia-driver", "/custom.conf",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func isContractScanAuthority(rel string) bool {
	return strings.HasPrefix(rel, "scenarios/quality-health/api/internal/contractscan/")
}

func isBinary(data []byte) bool {
	for _, b := range data {
		if b == 0 {
			return true
		}
	}
	return false
}
func relLine(rel string, line int) string { return rel + ":" + strconv.Itoa(line) }
