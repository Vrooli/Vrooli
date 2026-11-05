package glossary

import (
	"context"
	"strings"
	"testing"
)

func TestSearchConfigurationReturnsStableSafeMetadata(t *testing.T) {
	response, err := SearchConfiguration(context.Background(), "credentials", "local")
	if err != nil {
		t.Fatal(err)
	}
	if response.GetFallback() != true || response.GetTarget() != "local" {
		t.Fatalf("response context = fallback %t target %q", response.GetFallback(), response.GetTarget())
	}
	if len(response.GetResults()) == 0 {
		t.Fatal("expected a credentials configuration result")
	}
	for _, result := range response.GetResults() {
		if result.GetId() == "" || result.GetRoute() == "" || result.GetStepId() == "" {
			t.Fatalf("unstable configuration descriptor: %+v", result)
		}
		if strings.Contains(strings.ToLower(result.String()), "password") || strings.Contains(strings.ToLower(result.String()), "secret") || strings.Contains(strings.ToLower(result.String()), "token") {
			t.Fatalf("restricted configuration detail leaked into descriptor: %s", result.String())
		}
	}
}

func TestSearchConfigurationUnknownQueryDoesNotInventResult(t *testing.T) {
	response, err := SearchConfiguration(context.Background(), "florbnax-zxcv", "local")
	if err != nil {
		t.Fatal(err)
	}
	if len(response.GetResults()) != 0 {
		t.Fatalf("unknown query returned %d results", len(response.GetResults()))
	}
}

func TestSearchConfigurationMatchesIntentTokens(t *testing.T) {
	response, err := SearchConfiguration(context.Background(), "apply consent", "local")
	if err != nil {
		t.Fatal(err)
	}
	if len(response.GetResults()) == 0 || response.GetResults()[0].GetId() != "setup.apply" {
		t.Fatalf("intent query results = %+v, want setup.apply first", response.GetResults())
	}
}

func TestSearchConfigurationRanksNaturalLanguageSetting(t *testing.T) {
	response, err := SearchConfiguration(context.Background(), "where do I configure credentials", "local")
	if err != nil {
		t.Fatal(err)
	}
	if len(response.GetResults()) == 0 || response.GetResults()[0].GetId() != "setup.credentials" {
		t.Fatalf("natural-language results = %+v, want setup.credentials first", response.GetResults())
	}
}
