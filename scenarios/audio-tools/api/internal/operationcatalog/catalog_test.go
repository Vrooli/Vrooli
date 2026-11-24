package operationcatalog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	generatedUIPath        = "../../../ui/src/features/diagnostics/audioOps.generated.ts"
	generatedReferencePath = "../../../docs/reference/audio-operations.md"
)

func TestEmbeddedCatalogValidates(t *testing.T) {
	c, err := Default()
	if err != nil {
		t.Fatalf("Default: %v", err)
	}
	if c.Version != 1 {
		t.Fatalf("version = %d, want 1", c.Version)
	}
	if len(c.Operations) < 13 {
		t.Fatalf("operations = %d, want at least 13", len(c.Operations))
	}
}

func TestTransformFamilyIsComplete(t *testing.T) {
	c := MustDefault()
	transforms := c.Transform()
	want := []string{"transcode", "trim", "merge", "split", "fade", "volume", "normalize", "extract_metadata"}
	if len(transforms) != len(want) {
		t.Fatalf("transform operations = %d, want %d", len(transforms), len(want))
	}
	for i, id := range want {
		if transforms[i].ID != id {
			t.Fatalf("transform[%d] = %q, want %q (presentation order)", i, transforms[i].ID, id)
		}
	}
	for _, op := range transforms {
		if len(op.Params) == 0 {
			t.Errorf("%s: no params", op.ID)
		}
		if op.CLI.Group != "audio" {
			t.Errorf("%s: cli.group = %q, want audio", op.ID, op.CLI.Group)
		}
		if op.CLI.Service != "AudioProcessingService" {
			t.Errorf("%s: cli.service = %q", op.ID, op.CLI.Service)
		}
		if op.Presentation.Panel != "diagnostics" {
			t.Errorf("%s: presentation.panel = %q", op.ID, op.Presentation.Panel)
		}
	}
}

func TestNonTransformOperationsKeepCLIOwnership(t *testing.T) {
	c := MustDefault()
	for _, id := range []string{"transcribe", "transcribe_stream", "synthesize", "synthesize_stream", "summarize"} {
		op, ok := c.Lookup(id)
		if !ok {
			t.Fatalf("%s missing", id)
		}
		if op.Family == "transform" {
			t.Errorf("%s must not be a transform", id)
		}
		if op.CLI.Group == "" || op.CLI.Verb == "" || op.CLI.Method == "" {
			t.Errorf("%s: incomplete CLI ownership", id)
		}
	}
}

func TestValidationRejectsBrokenCatalogs(t *testing.T) {
	base := func() Catalog {
		return MustDefault()
	}
	tests := []struct {
		name   string
		mutate func(*Catalog)
		want   string
	}{
		{
			name: "duplicate id",
			mutate: func(c *Catalog) {
				c.Operations[1].ID = c.Operations[0].ID
			},
			want: "duplicate id",
		},
		{
			name: "unknown family",
			mutate: func(c *Catalog) {
				c.Operations[0].Family = "video"
			},
			want: "family",
		},
		{
			name: "engine family mismatch",
			mutate: func(c *Catalog) {
				c.Operations[0].Engine = "tts"
			},
			want: "does not match family",
		},
		{
			name: "unknown mechanism",
			mutate: func(c *Catalog) {
				c.Operations[0].Mechanism = "stream"
			},
			want: "mechanism",
		},
		{
			name: "unknown transport",
			mutate: func(c *Catalog) {
				c.Operations[0].Transport = "duplex"
			},
			want: "transport",
		},
		{
			name: "transform missing cli",
			mutate: func(c *Catalog) {
				c.Operations[0].CLI = CLISpec{}
			},
			want: "CLI spec",
		},
		{
			name: "transform missing presentation",
			mutate: func(c *Catalog) {
				c.Operations[0].Presentation.LabelKey = ""
			},
			want: "presentation spec",
		},
		{
			name: "duplicate param",
			mutate: func(c *Catalog) {
				c.Operations[0].Params[1].Name = c.Operations[0].Params[0].Name
			},
			want: "declared more than once",
		},
		{
			name: "select without options",
			mutate: func(c *Catalog) {
				c.Operations[0].Params[1].Options = nil
			},
			want: "requires options",
		},
		{
			name: "number without bounds",
			mutate: func(c *Catalog) {
				c.Operations[0].Params[4].Min = nil
			},
			want: "requires min and max",
		},
		{
			name: "inverted bounds",
			mutate: func(c *Catalog) {
				lo, hi := 10.0, 1.0
				c.Operations[0].Params[4].Min = &lo
				c.Operations[0].Params[4].Max = &hi
			},
			want: "must be less than max",
		},
		{
			name: "required file missing",
			mutate: func(c *Catalog) {
				c.Operations[0].Params[0].Required = false
			},
			want: "required file input",
		},
		{
			name: "cli flag maps to unknown param",
			mutate: func(c *Catalog) {
				c.Operations[0].CLI.Flags[0].Field = "missing"
			},
			want: "unknown param",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := base()
			tc.mutate(&c)
			err := c.Validate()
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %q, want substring %q", err, tc.want)
			}
		})
	}
}

func TestGeneratedUISpecsAreCurrent(t *testing.T) {
	c := MustDefault()
	rendered, err := RenderTypeScript(c)
	if err != nil {
		t.Fatalf("RenderTypeScript: %v", err)
	}
	path := filepath.Clean(generatedUIPath)
	current, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v (run: cd api && go run ./cmd/gen-operations)", path, err)
	}
	if string(current) != string(rendered) {
		t.Fatalf("%s is stale; run: cd api && go run ./cmd/gen-operations", path)
	}
}

func TestGeneratedOperationReferenceIsCurrent(t *testing.T) {
	want := RenderReferenceMarkdown(MustDefault())
	path := filepath.Clean(generatedReferencePath)
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v (run: cd api && go run ./cmd/gen-operations)", path, err)
	}
	if string(got) != string(want) {
		t.Fatalf("%s is stale; run: cd api && go run ./cmd/gen-operations", path)
	}
}
