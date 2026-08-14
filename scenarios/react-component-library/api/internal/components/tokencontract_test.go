package components

import (
	"reflect"
	"testing"
)

func TestExtractRequiredTokens(t *testing.T) {
	tests := []struct {
		name  string
		files []ComponentVersionFile
		want  []string
	}{
		{name: "self-defined property is excluded", files: []ComponentVersionFile{{Content: `:root { --local: red; color: var(--local); }`}}, want: []string{}},
		{name: "external property is included", files: []ComponentVersionFile{{Content: `color: var(--foreground);`}}, want: []string{"--foreground"}},
		{name: "template literal is scanned", files: []ComponentVersionFile{{Path: "styles.ts", Content: "const css = `background: var(--surface);`;"}}, want: []string{"--surface"}},
		{name: "color mix argument is scanned", files: []ComponentVersionFile{{Content: `background: color-mix(in srgb, var(--primary) 12%, white);`}}, want: []string{"--primary"}},
		{name: "nested fallback references are both scanned", files: []ComponentVersionFile{{Content: `color: var(--a, var(--b));`}}, want: []string{"--a", "--b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExtractRequiredTokens(tt.files); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ExtractRequiredTokens() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestExtractRequiredTokenPatterns(t *testing.T) {
	got := ExtractRequiredTokenPatterns([]ComponentVersionFile{{Content: "var(--space-${gap}); var(--dur-${duration}); var(--space-${other})"}})
	want := []string{"--dur-*", "--space-*"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ExtractRequiredTokenPatterns() = %#v, want %#v", got, want)
	}
}
