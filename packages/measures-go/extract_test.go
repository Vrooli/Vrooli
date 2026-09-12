package measures

import (
	"context"
	"strings"
	"testing"
)

func TestLLMExtractor_Found(t *testing.T) {
	c := CompleterFunc(func(_ context.Context, _ string) (string, error) {
		return `{"found":true,"value":"desktop-release","confidence":0.9}`, nil
	})
	x := NewLLMExtractor(c)
	r, err := x.Extract(context.Background(), "work in the desktop-release initiative", Param{Name: "initiative"}, []string{"desktop-release", "mobile"})
	if err != nil {
		t.Fatal(err)
	}
	if !r.Found || r.Value != "desktop-release" || r.Confidence != 0.9 {
		t.Fatalf("got %+v", r)
	}
}

func TestLLMExtractor_Abstains(t *testing.T) {
	c := CompleterFunc(func(_ context.Context, _ string) (string, error) {
		return `{"found":false,"value":"","confidence":0}`, nil
	})
	r, err := NewLLMExtractor(c).Extract(context.Background(), "how many items", Param{Name: "initiative"}, []string{"a"})
	if err != nil {
		t.Fatal(err)
	}
	if r.Found {
		t.Fatalf("expected abstention, got %+v", r)
	}
}

func TestLLMExtractor_DefaultsConfidenceWhenMissing(t *testing.T) {
	c := CompleterFunc(func(_ context.Context, _ string) (string, error) {
		return `{"found":true,"value":"mobile"}`, nil
	})
	r, _ := NewLLMExtractor(c).Extract(context.Background(), "q", Param{Name: "i"}, []string{"mobile"})
	if !r.Found || r.Confidence != extractedConfidence {
		t.Fatalf("expected default confidence %v, got %+v", extractedConfidence, r)
	}
}

func TestLLMExtractor_UnwrapsGatewayEnvelopeAndStripsThink(t *testing.T) {
	// The gateway wraps the completion in {"response":"…"} and qwen3 may emit an
	// empty <think> block before the JSON.
	c := CompleterFunc(func(_ context.Context, _ string) (string, error) {
		return `{"response":"<think></think>\n{\"found\":true,\"value\":\"42\",\"confidence\":0.8}"}`, nil
	})
	r, err := NewLLMExtractor(c).Extract(context.Background(), "q", Param{Name: "limit"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !r.Found || r.Value != "42" {
		t.Fatalf("got %+v", r)
	}
}

func TestLLMExtractor_ConfidenceAsString(t *testing.T) {
	c := CompleterFunc(func(_ context.Context, _ string) (string, error) {
		return `{"found":true,"value":"x","confidence":"0.72"}`, nil
	})
	r, _ := NewLLMExtractor(c).Extract(context.Background(), "q", Param{Name: "i"}, []string{"x"})
	if !r.Found || r.Confidence != 0.72 {
		t.Fatalf("string confidence not coerced: %+v", r)
	}
}

func TestLLMExtractor_GarbageOutputAbstains(t *testing.T) {
	c := CompleterFunc(func(_ context.Context, _ string) (string, error) {
		return "I cannot help with that.", nil
	})
	r, err := NewLLMExtractor(c).Extract(context.Background(), "q", Param{Name: "i"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if r.Found {
		t.Fatalf("garbage must abstain, got %+v", r)
	}
}

func TestLLMExtractor_BuildPrompt_ConstrainsToAllowed(t *testing.T) {
	p := buildExtractionPrompt("q", Param{Name: "initiative", Description: "the initiative filter"}, []string{"a", "b"})
	for _, want := range []string{"initiative", "the initiative filter", "exactly one of: a, b", "found=false"} {
		if !strings.Contains(p, want) {
			t.Fatalf("prompt missing %q:\n%s", want, p)
		}
	}
}

func TestLLMExtractor_BuildPrompt_NumericBounds(t *testing.T) {
	lo, hi := int64(1), int64(100)
	p := buildExtractionPrompt("q", Param{Name: "limit", Min: &lo, Max: &hi}, nil)
	if !strings.Contains(p, "between 1 and 100") {
		t.Fatalf("numeric-bound prompt wrong:\n%s", p)
	}
}
