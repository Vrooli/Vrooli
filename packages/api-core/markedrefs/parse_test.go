package markedrefs

import (
	"reflect"
	"testing"
)

func TestParseToken(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want Reference
		ok   bool
	}{
		{
			name: "simple topic",
			in:   "topic:friction-inbox/*",
			want: Reference{Marker: "topic", Value: "friction-inbox/*", Raw: "topic:friction-inbox/*"},
			ok:   true,
		},
		{
			name: "qualified topic",
			in:   "topic[example]:foo/bar/*",
			want: Reference{Marker: "topic", Qualifiers: []string{"example"}, Value: "foo/bar/*", Raw: "topic[example]:foo/bar/*"},
			ok:   true,
		},
		{
			name: "multiple qualifiers",
			in:   "path[future,optional]:docs/future.md",
			want: Reference{Marker: "path", Qualifiers: []string{"future", "optional"}, Value: "docs/future.md", Raw: "path[future,optional]:docs/future.md"},
			ok:   true,
		},
		{
			name: "trims value",
			in:   "path: docs/README.md ",
			want: Reference{Marker: "path", Value: "docs/README.md", Raw: "path: docs/README.md "},
			ok:   true,
		},
		{
			name: "colon inside value",
			in:   "url:https://example.com/a:b",
			want: Reference{Marker: "url", Value: "https://example.com/a:b", Raw: "url:https://example.com/a:b"},
			ok:   true,
		},
		{
			name: "unknown marker still parses",
			in:   "custom:value",
			want: Reference{Marker: "custom", Value: "value", Raw: "custom:value"},
			ok:   true,
		},
		{name: "unmarked slash string", in: "docs/README.md", ok: false},
		{name: "missing value", in: "path:", ok: false},
		{name: "uppercase marker rejected", in: "PATH:docs/README.md", ok: false},
		{name: "space in qualifier rejected", in: "path[future optional]:docs/file.md", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseToken(tt.in)
			if ok != tt.ok {
				t.Fatalf("ParseToken() ok = %v, want %v", ok, tt.ok)
			}
			if !ok {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ParseToken() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestParseInlineCode(t *testing.T) {
	line := "Use `topic:friction-inbox/*`, ignore `docs/README.md`, then `path[old]:docs/old.md`."
	got := ParseInlineCode(line, 12)
	want := []Reference{
		{
			Marker: "topic",
			Value:  "friction-inbox/*",
			Raw:    "`topic:friction-inbox/*`",
			Line:   12,
			Column: 5,
		},
		{
			Marker:     "path",
			Qualifiers: []string{"old"},
			Value:      "docs/old.md",
			Raw:        "`path[old]:docs/old.md`",
			Line:       12,
			Column:     61,
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseInlineCode() = %#v, want %#v", got, want)
	}
}

func TestParseInlineCodeUnclosedSpan(t *testing.T) {
	got := ParseInlineCode("Use `topic:foo", 1)
	if len(got) != 0 {
		t.Fatalf("expected no refs from unclosed inline-code span, got %#v", got)
	}
}
