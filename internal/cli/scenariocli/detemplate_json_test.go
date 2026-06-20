package scenariocli

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPruneJSONLocaleKeys(t *testing.T) {
	in := `{
  "_comment": "keep me",
  "app": {
    "title": "App"
  },
  "layout": {
    "nav": {
      "dashboard": "Dashboard",
      "notes": "Notes",
      "settings": "Settings"
    }
  },
  "pages": {
    "dashboard": {
      "title": "Dash"
    },
    "notes": {
      "title": "Notes"
    }
  },
  "notes": {
    "title": "ノート",
    "empty": "الملاحظات"
  }
}
`
	out, removed, err := PruneJSON([]byte(in), []string{"notes", "pages.notes", "layout.nav.notes"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if removed != 3 {
		t.Fatalf("expected 3 deletions, got %d", removed)
	}
	s := string(out)
	if strings.Contains(s, "notes") {
		t.Fatalf("notes residue survived:\n%s", s)
	}
	// Order + siblings preserved.
	if !strings.Contains(s, `"_comment": "keep me"`) {
		t.Errorf("sentinel lost:\n%s", s)
	}
	if !(strings.Index(s, "dashboard") < strings.Index(s, "settings")) {
		t.Errorf("key order not preserved:\n%s", s)
	}
	// Non-ASCII preserved (no \u escapes) — the surviving siblings keep UTF-8.
	if strings.Contains(s, `\u`) {
		t.Errorf("unexpected unicode escaping:\n%s", s)
	}
	// Still valid JSON.
	var v any
	if err := json.Unmarshal(out, &v); err != nil {
		t.Fatalf("invalid JSON after prune: %v\n%s", err, s)
	}
}

func TestPruneJSONUnicodeRoundTrip(t *testing.T) {
	in := `{
  "keep": "ノート / الملاحظات / <b>",
  "drop": "x"
}
`
	out, removed, err := PruneJSON([]byte(in), []string{"drop"}, nil)
	if err != nil || removed == 0 {
		t.Fatalf("prune failed: removed=%d err=%v", removed, err)
	}
	if !strings.Contains(string(out), "ノート / الملاحظات / <b>") {
		t.Errorf("unicode/html not preserved literally:\n%s", out)
	}
}

func TestPruneJSONArrayMatch(t *testing.T) {
	in := `{
  "groups": [
    {
      "name": "health",
      "commands": []
    },
    {
      "name": "notes",
      "commands": []
    }
  ],
  "measures": {
    "domains": [
      {
        "domain": "notes",
        "stateful": true
      }
    ]
  }
}
`
	matches := []TemplateJSONArrayMatch{
		{Path: "groups", Where: map[string]string{"name": "notes"}},
		{Path: "measures.domains", Where: map[string]string{"domain": "notes"}},
	}
	out, removed, err := PruneJSON([]byte(in), nil, matches)
	if err != nil || removed == 0 {
		t.Fatalf("prune failed: removed=%d err=%v", removed, err)
	}
	s := string(out)
	if strings.Contains(s, "notes") {
		t.Fatalf("notes array residue survived:\n%s", s)
	}
	if !strings.Contains(s, `"name": "health"`) {
		t.Errorf("health group lost:\n%s", s)
	}
	if !strings.Contains(s, `"domains": []`) {
		t.Errorf("emptied measures.domains not rendered:\n%s", s)
	}
	var v any
	if err := json.Unmarshal(out, &v); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
}

func TestPruneJSONNoop(t *testing.T) {
	in := `{
  "a": 1
}
`
	out, removed, err := PruneJSON([]byte(in), []string{"nonexistent", "a.b.c"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if removed != 0 {
		t.Fatalf("expected no change, got:\n%s", out)
	}
	if string(out) != in {
		t.Errorf("noop should return input unchanged")
	}
}
