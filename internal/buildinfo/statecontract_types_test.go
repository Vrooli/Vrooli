package buildinfo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const codecFixtureBase = `package fixture

import "encoding/json"

// Phase is a stored enum.
type Phase string

const (
	PhaseReady   Phase = "ready"
	PhaseRunning Phase = "running"
)

type Level int

const (
	LevelLow Level = iota
	LevelHigh
)

type Mode string

const ModeFast = Mode("fast")

var AllowedBackends = []string{"cuda", "rocm"}

const retryLimit = 3

type Spec struct {
	Backend string
}

func (s Spec) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{"backend": s.Backend})
}

func (s *Spec) UnmarshalJSON(data []byte) error {
	var raw map[string]string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	for _, allowed := range AllowedBackends {
		if raw["backend"] == allowed {
			s.Backend = allowed
		}
	}
	return nil
}

func Process(p Phase) bool {
	return p == PhaseReady
}
`

func scanFixture(t *testing.T, files map[string]string) map[string]string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	digests, violations, err := scanCodecPackage(dir, ".")
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) > 0 {
		t.Fatalf("fixture produced codec violations: %v", violations)
	}
	return digests
}

func TestCodecStampCoversEnumsEncodersAndCodecConsultedValues(t *testing.T) {
	base := scanFixture(t, map[string]string{"fixture.go": codecFixtureBase})
	for _, typeName := range []string{"Phase", "Level", "Mode", "Spec"} {
		if len(base[typeName]) != 32 {
			t.Fatalf("type %s has no codec stamp entry: %v", typeName, base)
		}
	}
	if _, ok := base["retryLimit"]; ok || len(base) != 4 {
		t.Fatalf("unexpected codec stamp entries: %v", base)
	}
}

func TestBehaviourOnlyEditsKeepTheCodecStamp(t *testing.T) {
	base := scanFixture(t, map[string]string{"fixture.go": codecFixtureBase})
	edits := map[string]string{
		"function body":   strings.Replace(codecFixtureBase, "return p == PhaseReady", "if p == \"\" {\n\t\treturn false\n\t}\n\treturn p == PhaseReady || p == PhaseRunning", 1),
		"decoder body":    strings.Replace(codecFixtureBase, "s.Backend = allowed", "s.Backend = allowed\n\t\t\tbreak", 1),
		"untyped const":   strings.Replace(codecFixtureBase, "const retryLimit = 3", "const retryLimit = 5\n\nconst pendingSuffix = \".pending\"", 1),
		"helper type":     codecFixtureBase + "\ntype stagingPlan struct{ Path string }\n\nfunc stage(p stagingPlan) string { return p.Path + pendingSuffix }\n\nconst pendingSuffix = \".pending\"\n",
		"comments/layout": strings.Replace(strings.Replace(codecFixtureBase, "PhaseReady   Phase = \"ready\"", "// PhaseReady is first.\n\tPhaseReady Phase = /* initial */ \"ready\"", 1), "AllowedBackends = []string{\"cuda\", \"rocm\"}", "AllowedBackends = []string{\n\t\"cuda\",\n\t\"rocm\",\n}", 1),
		"constant rename": strings.Replace(strings.Replace(codecFixtureBase, "PhaseReady ", "PhaseIdle ", 1), "p == PhaseReady", "p == PhaseIdle", 1),
	}
	for name, source := range edits {
		t.Run(name, func(t *testing.T) {
			got := scanFixture(t, map[string]string{"fixture.go": source, "extra_test.go": "package fixture\n\nconst PhaseTest Phase = \"test\"\n"})
			for typeName, digest := range base {
				if got[typeName] != digest {
					t.Fatalf("behaviour-only edit changed the %s codec stamp", typeName)
				}
			}
		})
	}
}

func TestStoredValueEditsChangeOnlyTheirTypesStamp(t *testing.T) {
	base := scanFixture(t, map[string]string{"fixture.go": codecFixtureBase})
	edits := map[string]struct {
		source  string
		changed string
	}{
		"new enum value":        {strings.Replace(codecFixtureBase, "PhaseRunning Phase = \"running\"", "PhaseRunning Phase = \"running\"\n\tPhaseDone Phase = \"done\"", 1), "Phase"},
		"changed enum value":    {strings.Replace(codecFixtureBase, "\"running\"", "\"active\"", 1), "Phase"},
		"implicit iota":         {strings.Replace(codecFixtureBase, "\tLevelHigh\n", "\tLevelMid\n\tLevelHigh\n", 1), "Level"},
		"conversion constant":   {strings.Replace(codecFixtureBase, "Mode(\"fast\")", "Mode(\"quick\")", 1), "Mode"},
		"encoder body":          {strings.Replace(codecFixtureBase, "map[string]string{\"backend\": s.Backend}", "map[string]string{\"engine\": s.Backend}", 1), "Spec"},
		"decoder-consulted var": {strings.Replace(codecFixtureBase, "\"cuda\", \"rocm\"", "\"cuda\", \"rocm\", \"vulkan\"", 1), "Spec"},
	}
	for name, tc := range edits {
		t.Run(name, func(t *testing.T) {
			got := scanFixture(t, map[string]string{"fixture.go": tc.source})
			for typeName, digest := range base {
				if (got[typeName] != digest) != (typeName == tc.changed) {
					t.Fatalf("edit changed %s stamp=%t; only %s should change", typeName, got[typeName] != digest, tc.changed)
				}
			}
		})
	}
}

func TestGenericCustomEncodersAreStampedUnderTheirDeclaredType(t *testing.T) {
	source := "package fixture\n\nimport \"encoding/json\"\n\ntype Box[T any] struct{ V T }\n\nfunc (b Box[T]) MarshalJSON() ([]byte, error) { return json.Marshal(b.V) }\n\ntype Pair[K comparable, V any] struct{}\n\nfunc (p *Pair[K, V]) MarshalText() ([]byte, error) { return nil, nil }\n"
	digests := scanFixture(t, map[string]string{"fixture.go": source})
	changed := scanFixture(t, map[string]string{"fixture.go": strings.Replace(source, "json.Marshal(b.V)", "json.Marshal([]any{b.V})", 1)})
	if len(digests["Box"]) != 32 || len(digests["Pair"]) != 32 || changed["Box"] == digests["Box"] || changed["Pair"] != digests["Pair"] {
		t.Fatalf("generic encoders were not stamped per declared type: %v %v", digests, changed)
	}
}

func TestCodecScanSkipsIgnoredFilesAndRefusesUnboundedSources(t *testing.T) {
	base := scanFixture(t, map[string]string{"fixture.go": codecFixtureBase})
	withIgnored := scanFixture(t, map[string]string{"fixture.go": codecFixtureBase, "gen.go": "//go:build ignore\n\npackage main\n\nconst X Phase = \"x\"\n"})
	if withIgnored["Phase"] != base["Phase"] {
		t.Fatal("a //go:build ignore file participated")
	}
	for _, mode := range []string{"symlink", "oversize", "empty"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "codec.go")
			switch mode {
			case "symlink":
				other := filepath.Join(t.TempDir(), "codec.go")
				if err := os.WriteFile(other, []byte(codecFixtureBase), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(other, path); err != nil {
					t.Fatal(err)
				}
			case "oversize":
				if err := os.WriteFile(path, []byte("package fixture\n"+strings.Repeat("//x\n", maxCodecSourceBytes/4)), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if _, _, err := scanCodecPackage(dir, "."); err == nil {
				t.Fatal("unbounded or missing codec source was silently accepted")
			}
		})
	}
}

// The hostile review's synthetic package: each edit changes stored bytes
// through a path a one-level codec stamp could not see.
const codecClosureFixture = `package fixture

import (
	"encoding/json"
	"strconv"
)

type Wire struct{ A string }
type wire struct {
	A string ` + "`json:\"a\"`" + `
}

func (w Wire) MarshalJSON() ([]byte, error) { return json.Marshal(wire{A: w.A}) }

type Window struct{ dur int64 }

func (w Window) MarshalJSON() ([]byte, error) { return json.Marshal(map[string]any{"dur": w.dur}) }

const modePrefix = "v1-"

type Mode string

const ModeA Mode = modePrefix + "a"

type Phase int

const (
	PhaseIdle Phase = iota
	PhaseRun
)

var phaseNames = []string{"idle", "run"}

func (p Phase) String() string               { return phaseNames[p] }
func (p Phase) MarshalText() ([]byte, error) { return []byte(p.String()), nil }

type Set[T comparable] struct{ m map[T]bool }

func (s Set[T]) MarshalJSON() ([]byte, error) { return []byte(strconv.Itoa(len(s.m))), nil }

type Stamp struct{ N int }

func (s Stamp) MarshalJSON() ([]byte, error) { return encodeStamp(s.N) }
func encodeStamp(n int) ([]byte, error)      { return json.Marshal(strconv.Itoa(n)) }
`

func TestCodecStampFollowsTheSamePackageClosure(t *testing.T) {
	base := scanFixture(t, map[string]string{"fixture.go": codecClosureFixture})
	for _, typeName := range []string{"Wire", "Window", "Mode", "Phase", "Set", "Stamp"} {
		if len(base[typeName]) != 32 {
			t.Fatalf("%s has no stamp entry: %v", typeName, base)
		}
	}
	for name, tc := range map[string]struct{ old, new, changed string }{
		"wire struct tag":        {"`json:\"a\"`", "`json:\"alpha\"`", "Wire"},
		"unexported field type":  {"dur int64", "dur string", "Window"},
		"enum value via const":   {`modePrefix = "v1-"`, `modePrefix = "v2-"`, "Mode"},
		"names table via String": {`[]string{"idle", "run"}`, `[]string{"IDLE", "RUNNING"}`, "Phase"},
		"generic encoder body":   {"strconv.Itoa(len(s.m))), nil", "\"\\\"\" + strconv.Itoa(len(s.m)) + \"\\\"\"), nil", "Set"},
		"encoder helper func":    {"json.Marshal(strconv.Itoa(n))", "json.Marshal(n)", "Stamp"},
	} {
		t.Run(name, func(t *testing.T) {
			if strings.Count(codecClosureFixture, tc.old) != 1 {
				t.Fatalf("fixture anchor %q is not unique", tc.old)
			}
			got := scanFixture(t, map[string]string{"fixture.go": strings.Replace(codecClosureFixture, tc.old, tc.new, 1)})
			if got[tc.changed] == base[tc.changed] {
				t.Fatalf("%s did not change the %s stamp", name, tc.changed)
			}
		})
	}
}
