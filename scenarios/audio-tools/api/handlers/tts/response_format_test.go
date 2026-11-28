package tts

import (
	"context"
	"testing"
)

type responseFormatConfigStore struct{ value string }

func (s *responseFormatConfigStore) Get(context.Context) (string, string, bool, error) {
	return s.value, "", true, nil
}

func (*responseFormatConfigStore) Set(context.Context, string, string) error { return nil }

func TestEffectiveResponseFormatUsesSavedDefaultAndHonorsRequestOverride(t *testing.T) {
	h := &connectHandler{deps: Deps{ConfigStore: &responseFormatConfigStore{value: `{"DefaultResponseFormat":"opus"}`}}}
	if got := h.effectiveResponseFormat(context.Background(), ""); got != "opus" {
		t.Fatalf("effective default = %q, want opus", got)
	}
	if got := h.effectiveResponseFormat(context.Background(), "wav"); got != "wav" {
		t.Fatalf("explicit request format = %q, want wav", got)
	}
}

func TestEffectiveResponseFormatMigratesSavedLegacyValue(t *testing.T) {
	h := &connectHandler{deps: Deps{ConfigStore: &responseFormatConfigStore{value: `{"Backend":"flac"}`}}}
	if got := h.effectiveResponseFormat(context.Background(), ""); got != "flac" {
		t.Fatalf("migrated legacy format = %q, want flac", got)
	}
}
