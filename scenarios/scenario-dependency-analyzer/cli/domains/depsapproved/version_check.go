package depsapproved

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/mod/modfile"

	"github.com/vrooli/cli-core/cliutil"
	"scenario-dependency-analyzer/cli/internal/support"
)

const oneVersionCheck = "one-version-per-shared-package"

type versionCheckReport struct {
	Check      string              `json:"check"`
	Packages   []string            `json:"packages"`
	Versions   map[string][]string `json:"versions"`
	Violations []versionViolation  `json:"violations,omitempty"`
}

type versionViolation struct {
	Package  string   `json:"package"`
	Versions []string `json:"versions"`
}

func runCheck(args []string) error {
	fs := support.NewFlagSet("deps check")
	var jsonOutput, includeIndirect bool
	var packages string
	fs.BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	fs.BoolVar(&includeIndirect, "include-indirect", true, "Include indirect requirements")
	fs.StringVar(&packages, "packages", "modernc.org/sqlite,modernc.org/libc", "Comma-separated modules to check")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) != 1 || fs.Args()[0] != oneVersionCheck {
		return fmt.Errorf("usage: %s deps check %s [--packages <module,...>] [--json]", support.AppName, oneVersionCheck)
	}
	selected := make([]string, 0)
	for _, item := range strings.Split(packages, ",") {
		if item = strings.TrimSpace(item); item != "" {
			selected = append(selected, item)
		}
	}
	report, err := checkModuleVersions(cliutil.ResolveRepoRoot(), selected, includeIndirect)
	if err != nil {
		return err
	}
	if jsonOutput {
		if err := support.PrintReportJSON(report); err != nil {
			return err
		}
	} else {
		for _, pkg := range report.Packages {
			fmt.Printf("%s: %s\n", pkg, strings.Join(report.Versions[pkg], ", "))
		}
		if len(report.Violations) == 0 {
			fmt.Println("one-version-per-shared-package: pass")
		}
	}
	if len(report.Violations) > 0 {
		return fmt.Errorf("%s found %d package(s) with version drift", oneVersionCheck, len(report.Violations))
	}
	return nil
}

func checkModuleVersions(root string, packages []string, includeIndirect bool) (versionCheckReport, error) {
	wanted := make(map[string]struct{}, len(packages))
	for _, pkg := range packages {
		if pkg = strings.TrimSpace(pkg); pkg != "" {
			wanted[pkg] = struct{}{}
		}
	}
	versions := make(map[string]map[string]struct{}, len(wanted))
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "node_modules", "vendor", "templates":
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Name() != "go.mod" {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		file, parseErr := modfile.Parse(path, data, nil)
		if parseErr != nil {
			return fmt.Errorf("parse %s: %w", path, parseErr)
		}
		for _, requirement := range file.Require {
			if !includeIndirect && requirement.Indirect {
				continue
			}
			if _, ok := wanted[requirement.Mod.Path]; !ok {
				continue
			}
			if versions[requirement.Mod.Path] == nil {
				versions[requirement.Mod.Path] = make(map[string]struct{})
			}
			versions[requirement.Mod.Path][requirement.Mod.Version] = struct{}{}
		}
		return nil
	})
	if err != nil {
		return versionCheckReport{}, err
	}
	report := versionCheckReport{Check: oneVersionCheck, Packages: append([]string(nil), packages...), Versions: make(map[string][]string, len(wanted))}
	sort.Strings(report.Packages)
	for pkg := range wanted {
		for version := range versions[pkg] {
			report.Versions[pkg] = append(report.Versions[pkg], version)
		}
		sort.Strings(report.Versions[pkg])
		if len(report.Versions[pkg]) > 1 {
			report.Violations = append(report.Violations, versionViolation{Package: pkg, Versions: report.Versions[pkg]})
		}
	}
	sort.Slice(report.Violations, func(i, j int) bool { return report.Violations[i].Package < report.Violations[j].Package })
	return report, nil
}
