package runannotations

import (
	"path/filepath"
	"testing"
)

func TestGoldRootResolvesAgainstTheDurableStateRoot(t *testing.T) {
	state := t.TempDir()
	elsewhere := filepath.Join(t.TempDir(), "private-gold")
	cases := []struct {
		name, runtime, stateRoot, want string
	}{
		{"default is gold under the state root", `{"mode":"passive"}`, state, filepath.Join(state, "gold")},
		{"relative resolves under the state root", `{"mode":"passive","gold_root":"reference/gold"}`, state, filepath.Join(state, "reference", "gold")},
		{"absolute is used as written", `{"mode":"passive","gold_root":"` + elsewhere + `"}`, state, elsewhere},
		{"absolute needs no state root", `{"mode":"passive","gold_root":"` + elsewhere + `"}`, "", elsewhere},
		{"relative without a state root stays unresolved", `{"mode":"passive"}`, "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			service, _ := testServiceRuntime(t, tc.runtime, `[`+labelJSON("a")+`]`, nil, nil, Options{StateRoot: tc.stateRoot})
			if got := service.GoldRoot(); got != tc.want {
				t.Fatalf("GoldRoot() = %q, want %q", got, tc.want)
			}
			if got := service.PackSummary().GoldRoot; got != tc.want {
				t.Fatalf("PackSummary().GoldRoot = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestGoldRootRefusesPaddedPath(t *testing.T) {
	_, err := LoadCompiled(writePack(t, `{"pack":{"name":"test","version":"1","runtime":{"mode":"passive","gold_root":" gold"},"labels":[`+labelJSON("a")+`]}}`))
	if err == nil {
		t.Fatal("a gold_root with leading space was accepted")
	}
}
