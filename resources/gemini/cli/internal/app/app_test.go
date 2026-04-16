package app

import (
	"strings"
	"testing"
)

func TestResolvePromptPrefersFlag(t *testing.T) {
	t.Parallel()

	got, err := resolvePrompt("hello", []string{"ignored"}, strings.NewReader("stdin"))
	if err != nil {
		t.Fatalf("resolvePrompt() error = %v", err)
	}
	if got != "hello" {
		t.Fatalf("resolvePrompt() = %q", got)
	}
}

func TestResolvePromptUsesTrailingArgs(t *testing.T) {
	t.Parallel()

	got, err := resolvePrompt("", []string{"hello", "world"}, strings.NewReader(""))
	if err != nil {
		t.Fatalf("resolvePrompt() error = %v", err)
	}
	if got != "hello world" {
		t.Fatalf("resolvePrompt() = %q", got)
	}
}

func TestResolvePromptUsesStdin(t *testing.T) {
	t.Parallel()

	got, err := resolvePrompt("", nil, strings.NewReader("from stdin\n"))
	if err != nil {
		t.Fatalf("resolvePrompt() error = %v", err)
	}
	if got != "from stdin" {
		t.Fatalf("resolvePrompt() = %q", got)
	}
}

func TestGenerateResponsePrimaryText(t *testing.T) {
	t.Parallel()

	response := generateResponse{
		Candidates: []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		}{{
			Content: struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			}{
				Parts: []struct {
					Text string `json:"text"`
				}{{Text: "hello world"}},
			},
		}},
	}

	if got := response.PrimaryText(); got != "hello world" {
		t.Fatalf("PrimaryText() = %q", got)
	}
}
