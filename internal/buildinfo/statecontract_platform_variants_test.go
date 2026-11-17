package buildinfo

import (
	"strings"
	"testing"
)

func TestCodecDigestIncludesEveryPlatformDependency(t *testing.T) {
	files := map[string]string{
		"codec.go": `package fixture
import "encoding/json"
type Wire struct{ V string }
func (w Wire) MarshalJSON() ([]byte,error) { return json.Marshal(makeValue(w.V)) }
`,
		"variant_linux.go": `//go:build linux

package fixture
const prefix = "linux"
type wrapper struct { Value string }
func makeValue(v string) wrapper { return wrapper{prefix+v} }
`,
		"variant_windows.go": `//go:build windows

package fixture
const prefix = "windows"
type wrapper struct { Value string }
func makeValue(v string) wrapper { return wrapper{prefix+v} }
`,
	}
	baseline := scanFixture(t, files)["Wire"]
	for _, tc := range []struct{ old, new string }{
		{`"linux"`, `"linux-v2"`},
		{`Value string`, `Value string ` + "`json:\"value\"`"},
		{`prefix+v`, `v+prefix`},
	} {
		changed := make(map[string]string, len(files))
		for k, v := range files {
			changed[k] = v
		}
		changed["variant_linux.go"] = strings.Replace(files["variant_linux.go"], tc.old, tc.new, 1)
		if changed["variant_linux.go"] == files["variant_linux.go"] {
			t.Fatalf("fixture mutation %q did not apply", tc.old)
		}
		if got := scanFixture(t, changed)["Wire"]; got == baseline {
			t.Errorf("linux mutation %q escaped codec digest", tc.old)
		}
	}
}

func TestOwnerVocabularyIncludesEveryPlatformVariant(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.test/owner\n\ngo 1.25\n")
	writeTestFile(t, root, "pkg/status_linux.go", "//go:build linux\n\npackage pkg\nconst Status = \"linux\"\n")
	writeTestFile(t, root, "pkg/status_windows.go", "//go:build windows\n\npackage pkg\nconst Status = \"windows\"\n")
	digest := func() string {
		t.Helper()
		a, err := analyzeOwnerPackages(root, []string{"pkg"}, nil, true)
		if err != nil {
			t.Fatal(err)
		}
		return a.values["Status"]
	}
	before := digest()
	writeTestFile(t, root, "pkg/status_linux.go", "//go:build linux\n\npackage pkg\nconst Status = \"linux-v2\"\n")
	if got := digest(); got == before {
		t.Fatal("linux vocabulary escaped owner digest")
	}
}
