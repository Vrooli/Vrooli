package deps

import (
	"strings"
	"testing"
)

func TestClassify(t *testing.T) {
	cases := []struct {
		name     string
		declared string
		target   string
		wantKind IssueKind
	}{
		{"wildcard", "*", "18.0.0", ""},
		{"empty range", "", "18.0.0", ""},
		{"caret exact match", "^18.0.0", "18.0.0", ""},
		{"caret newer minor ok", "^18.0.0", "18.2.0", ""},
		{"caret newer patch ok", "^18.0.0", "18.0.5", ""},
		{"caret below minimum", "^18.2.0", "18.1.0", IssueRangeDoesNotMatch},
		{"caret major mismatch", "^18.0.0", "19.0.0", IssueIncompatibleMajor},
		{"caret major mismatch down", "^18.0.0", "17.0.0", IssueIncompatibleMajor},
		{"tilde match", "~1.2.3", "1.2.5", ""},
		{"tilde minor mismatch", "~1.2.3", "1.3.0", IssueRangeDoesNotMatch},
		{"tilde major mismatch", "~1.2.3", "2.2.3", IssueIncompatibleMajor},
		{"gte ok", ">=1.0.0", "2.5.0", ""},
		{"gte below", ">=2.0.0", "1.9.9", IssueIncompatibleMajor},
		{"exact match", "1.2.3", "1.2.3", ""},
		{"exact mismatch minor", "1.2.3", "1.2.4", IssueRangeDoesNotMatch},
		{"exact mismatch major", "1.2.3", "2.0.0", IssueIncompatibleMajor},
		{"target with caret prefix", "^18.0.0", "^18.2.0", ""},
		{"target with tilde prefix", "^18.0.0", "~18.2.0", ""},
		{"unparseable target", "^18.0.0", "garbage", IssueUnparseableTarget},
		{"unparseable range", "huh", "1.0.0", IssueUnparseableRange},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, detail := classify(tc.declared, tc.target)
			if got != tc.wantKind {
				t.Fatalf("classify(%q, %q) = %q; want %q (detail: %s)", tc.declared, tc.target, got, tc.wantKind, detail)
			}
		})
	}
}

func TestResolveRangeToLatest(t *testing.T) {
	candidates := []string{"17.0.2", "18.2.0", "18.3.1", "19.1.0"}
	cases := []struct {
		name      string
		declared  string
		want      string
		wantFound bool
	}{
		{"wildcard chooses latest", "*", "19.1.0", true},
		{"empty chooses latest", "", "19.1.0", true},
		{"caret stays in major", "^18.0.0", "18.3.1", true},
		{"tilde stays in minor", "~18.2.0", "18.2.0", true},
		{"gte chooses newest", ">=17.0.0", "19.1.0", true},
		{"compound upper bound", ">=18.0.0 <19.0.0", "18.3.1", true},
		{"exact match", "17.0.2", "17.0.2", true},
		{"no satisfying candidate", "^16.0.0", "", false},
		{"unparseable", "workspace:*", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := ResolveRangeToLatest(tc.declared, candidates)
			if ok != tc.wantFound || got != tc.want {
				t.Fatalf("ResolveRangeToLatest(%q) = %q, %v; want %q, %v", tc.declared, got, ok, tc.want, tc.wantFound)
			}
		})
	}
}

func TestParseHeaderField(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		out, err := ParseHeaderField("")
		if err != nil || out != nil {
			t.Fatalf("empty: got %v, %v", out, err)
		}
	})
	t.Run("object form", func(t *testing.T) {
		out, err := ParseHeaderField(`{"react": "^18.0.0", "lodash": "*"}`)
		if err != nil {
			t.Fatal(err)
		}
		if len(out) != 2 {
			t.Fatalf("want 2, got %d", len(out))
		}
		got := declarationsByName(out)
		if got["react"].VersionRange != "^18.0.0" || got["react"].Kind != DepKindRuntime {
			t.Fatalf("bad react declaration: %+v", got["react"])
		}
		if got["lodash"].VersionRange != "*" || got["lodash"].Kind != DepKindRuntime {
			t.Fatalf("bad lodash declaration: %+v", got["lodash"])
		}
	})
	t.Run("array form", func(t *testing.T) {
		out, err := ParseHeaderField(`[{"name":"react","range":"^18","kind":"peer"}]`)
		if err != nil {
			t.Fatal(err)
		}
		if len(out) != 1 || out[0].DepName != "react" || out[0].VersionRange != "^18" || out[0].Kind != DepKindPeer {
			t.Fatalf("bad parse: %+v", out)
		}
	})
	t.Run("array form defaults missing kind to runtime", func(t *testing.T) {
		out, err := ParseHeaderField(`[{"name":"react","range":"^18"}]`)
		if err != nil {
			t.Fatal(err)
		}
		if len(out) != 1 || out[0].DepName != "react" || out[0].VersionRange != "^18" || out[0].Kind != DepKindRuntime {
			t.Fatalf("bad parse: %+v", out)
		}
	})
	t.Run("detailed object form", func(t *testing.T) {
		out, err := ParseHeaderField(`{"lucide-react":{"range":"^0.424.0","kind":"dev"}}`)
		if err != nil {
			t.Fatal(err)
		}
		if len(out) != 1 || out[0].DepName != "lucide-react" || out[0].VersionRange != "^0.424.0" || out[0].Kind != DepKindDev {
			t.Fatalf("bad parse: %+v", out)
		}
	})
	t.Run("explicit invalid kind rejected", func(t *testing.T) {
		_, err := ParseHeaderField(`[{"name":"react","range":"^18","kind":"optional"}]`)
		if err == nil {
			t.Fatal("want error")
		}
		if !strings.Contains(err.Error(), "invalid @deps kind") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	t.Run("malformed object", func(t *testing.T) {
		_, err := ParseHeaderField(`{bad`)
		if err == nil {
			t.Fatal("want error")
		}
	})
	t.Run("malformed array", func(t *testing.T) {
		_, err := ParseHeaderField(`[bad`)
		if err == nil {
			t.Fatal("want error")
		}
	})
	t.Run("bare string rejected", func(t *testing.T) {
		_, err := ParseHeaderField(`react`)
		if err == nil {
			t.Fatal("want error")
		}
	})
}

func declarationsByName(in []DeclarationFields) map[string]DeclarationFields {
	out := make(map[string]DeclarationFields, len(in))
	for _, d := range in {
		out[d.DepName] = d
	}
	return out
}
