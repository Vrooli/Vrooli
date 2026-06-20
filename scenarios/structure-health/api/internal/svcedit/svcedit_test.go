package svcedit

import (
	"strings"
	"testing"
)

const sample = `{
  "$schema": "../schema.json",
  "version": "1.0.0",
  "service": {
    "name": "old-name",
    "displayName": "Old Name",
    "customField": "preserved"
  },
  "lifecycle": {
    "setup": {
      "steps": [
        {
          "name": "build",
          "run": "go build ./..."
        }
      ]
    }
  },
  "unknownTopLevel": {
    "deeply": {
      "nested": [
        1,
        2,
        3
      ]
    }
  }
}
`

func TestRoundTripIsByteIdentical(t *testing.T) {
	doc, err := Parse([]byte(sample))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	out, err := doc.Bytes()
	if err != nil {
		t.Fatalf("bytes: %v", err)
	}
	if string(out) != sample {
		t.Fatalf("round trip not byte-identical.\n--- want ---\n%s\n--- got ---\n%s", sample, out)
	}
}

func TestEditPreservesKeyOrderAndUnknownFields(t *testing.T) {
	doc, err := Parse([]byte(sample))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	svc := EnsureMap(doc.Root(), "service")
	svc.Set("name", "new-name")

	out, err := doc.Bytes()
	if err != nil {
		t.Fatalf("bytes: %v", err)
	}
	got := string(out)

	if !strings.Contains(got, `"name": "new-name"`) {
		t.Fatalf("name not updated: %s", got)
	}
	// Unknown / custom fields survive.
	for _, want := range []string{`"customField": "preserved"`, `"unknownTopLevel"`, `"deeply"`} {
		if !strings.Contains(got, want) {
			t.Fatalf("dropped field %q:\n%s", want, got)
		}
	}
	// Top-level key order is preserved.
	order := []string{`"$schema"`, `"version"`, `"service"`, `"lifecycle"`, `"unknownTopLevel"`}
	last := -1
	for _, k := range order {
		idx := strings.Index(got, k)
		if idx < 0 {
			t.Fatalf("missing key %q", k)
		}
		if idx < last {
			t.Fatalf("key order changed at %q:\n%s", k, got)
		}
		last = idx
	}
}

// TestHTMLCharactersStayEscaped proves shell metacharacters (&, <, >) survive a
// round trip in the generated \uXXXX escaped form (matching how service.json is
// generated), so an edit never silently rewrites a lifecycle command.
func TestHTMLCharactersStayEscaped(t *testing.T) {
	in := "{\n  \"run\": \"a \\u0026\\u0026 b\"\n}\n"
	doc, err := Parse([]byte(in))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	out, err := doc.Bytes()
	if err != nil {
		t.Fatalf("bytes: %v", err)
	}
	if string(out) != in {
		t.Fatalf("escaped chars not preserved.\nwant %q\ngot  %q", in, out)
	}
}

func TestAppendToSliceCreatesArray(t *testing.T) {
	doc, err := Parse([]byte(`{"a": 1}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	AppendToSlice(doc.Root(), "items", NewObject("k", "v"))
	out, err := doc.Bytes()
	if err != nil {
		t.Fatalf("bytes: %v", err)
	}
	if !strings.Contains(string(out), `"items"`) || !strings.Contains(string(out), `"k": "v"`) {
		t.Fatalf("array not appended: %s", out)
	}
}
