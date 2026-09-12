package invokers

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
)

// [REQ:CLI-ARGV-003] Every registered invoker's argv resolves through the
// real root parser with no unknown command and no retired global.
func TestEveryInvokerResolvesThroughTheRootParser(t *testing.T) {
	items, err := All()
	if err != nil {
		t.Fatalf("All() error = %v", err)
	}
	if len(items) < 20 {
		t.Fatalf("registry has %d invokers, expected the catalog, fixture and CI entries", len(items))
	}
	for _, item := range items {
		argv, err := item.Argv()
		if err != nil {
			t.Errorf("%s (%s): argv error %v", item.Name, item.Owner, err)
			continue
		}
		res, err := rootcli.ResolveArgv(argv)
		if err != nil {
			t.Errorf("%s (%s): %v — argv %q does not resolve", item.Name, item.Owner, err, strings.Join(argv, " "))
			continue
		}
		if len(res.Warnings) != 0 {
			t.Errorf("%s (%s): argv %q relies on the retired-globals tolerance table: %v", item.Name, item.Owner, strings.Join(argv, " "), res.Warnings)
			continue
		}
		t.Logf("%-45s -> %s %s", item.Name, res.Command, res.Subcommand)
	}
}

// invocationSite matches a production call that builds an Invocation.
var invocationSite = regexp.MustCompile(`cliinvoke\.Invocation\{`)

// [REQ:CLI-ARGV-003] Every file that spawns the CLI through the invoker is a
// registered owner, so a new spawn site cannot appear without a registry entry.
func TestEveryDirectSpawnIsRegistered(t *testing.T) {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	items, err := All()
	if err != nil {
		t.Fatal(err)
	}
	owners := map[string]bool{}
	for _, item := range items {
		owners[item.Owner] = true
	}
	for _, runner := range Runners {
		owners[runner] = true
	}
	var missing []string
	for _, base := range []string{"internal", "cmd", "packages", filepath.Join("scenarios", "vrooli-autoheal")} {
		_ = filepath.WalkDir(filepath.Join(root, base), func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				switch d.Name() {
				case "node_modules", "vendor", "testdata", ".git":
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			rel, _ := filepath.Rel(root, path)
			rel = filepath.ToSlash(rel)
			if strings.HasPrefix(rel, "packages/repo-contract-go/cliinvoke/") {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil || !invocationSite.Match(data) {
				return nil
			}
			if !owners[rel] {
				missing = append(missing, rel)
			}
			return nil
		})
	}
	sort.Strings(missing)
	if len(missing) != 0 {
		t.Fatalf("files build a cliinvoke.Invocation but are not registered owners:\n  %s", strings.Join(missing, "\n  "))
	}
}

func TestWorkflowScannerExtractsArgv(t *testing.T) {
	cases := map[string][]string{
		`        run: go run ./cmd/vrooli scenario setup test-genie`:                     {"scenario", "setup", "test-genie"},
		`            go run ./cmd/vrooli scenario setup test-genie 2>&1 `:                {"scenario", "setup", "test-genie"},
		`          go run ./cmd/vrooli hygiene --contract-only --fail-on error --json`:   {"hygiene", "--contract-only", "--fail-on", "error", "--json"},
		`          vrooli scenario test scenario-to-plugin --preset comprehensive`:       {"scenario", "test", "scenario-to-plugin", "--preset", "comprehensive"},
		`        run: ssh -i "$KEY" host 'cd ${VROOLI_ROOT:-~/Vrooli} && vrooli deploy'`: {"deploy"},
		`          go run ./cmd/vrooli credentials doctor --format json > doctor.json`:   {"credentials", "doctor", "--format", "json"},
		`        run: go run ./cmd/vrooli-manifest-check`:                                nil,
		`          # go run ./cmd/vrooli scenario setup commented`:                       nil,
	}
	for line, want := range cases {
		got, ok := argvFromSegment(line)
		if want == nil {
			if ok {
				t.Errorf("%q: expected no argv, got %q", line, got)
			}
			continue
		}
		if !ok || strings.Join(got, " ") != strings.Join(want, " ") {
			t.Errorf("%q: got %q (ok=%v), want %q", line, got, ok, want)
		}
	}
}
