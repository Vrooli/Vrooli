package tiered_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"audio-tools/internal/ai/chains/tiered"
)

type credentials struct{ byok, vrooli bool }

type sampleProvider struct{ available bool }

type captureLogger struct{ lines []string }

func (l *captureLogger) Printf(format string, args ...any) { l.lines = append(l.lines, format) }

func TestCredentialPolicyPreservesTierEligibilityAndTerminalErrors(t *testing.T) {
	route := tiered.CredentialRoute(func(req credentials) bool { return req.byok }, func(req credentials) bool { return req.vrooli })
	if route(tiered.SlotBYOK, credentials{}) || route(tiered.SlotVrooli, credentials{}) || !route(tiered.SlotLocal, credentials{}) {
		t.Fatal("credential route must require tokens only for remote tiers")
	}
	if !route(tiered.SlotBYOK, credentials{byok: true}) || !route(tiered.SlotVrooli, credentials{vrooli: true}) {
		t.Fatal("credential route rejected supplied credentials")
	}
	unknown, missing, credits := errors.New("unknown"), errors.New("missing"), errors.New("credits")
	terminal := tiered.CredentialTerminal(unknown, missing, credits)
	if !terminal(tiered.SlotBYOK, unknown) || !terminal(tiered.SlotBYOK, missing) || !terminal(tiered.SlotVrooli, credits) || terminal(tiered.SlotLocal, credits) {
		t.Fatal("terminal policy did not preserve tier-specific behavior")
	}
}

func TestFallbackLoggerIsOptionalAndStructured(t *testing.T) {
	if tiered.FallbackLogger("stt", nil) != nil {
		t.Fatal("nil logger must not allocate a fallback hook")
	}
	logger := &captureLogger{}
	tiered.FallbackLogger("tts", logger)(context.Background(), tiered.FallbackEvent{From: tiered.SlotBYOK, To: tiered.SlotLocal, Reason: "unavailable"})
	if len(logger.lines) != 1 || !strings.Contains(logger.lines[0], "capability=%s") {
		t.Fatalf("fallback log format = %v", logger.lines)
	}
}

func TestTierForPreservesTypedNilAndDelegates(t *testing.T) {
	var absent *sampleProvider
	if got := tiered.TierFor(absent, sampleExecute, sampleAvailable); got != nil {
		t.Fatal("typed-nil provider must not produce a tier")
	}
	provider := &sampleProvider{available: true}
	tier := tiered.TierFor(provider, sampleExecute, sampleAvailable)
	got, err := tier.Execute(context.Background(), "request")
	if err != nil || got != "request" || !tier.IsAvailable(context.Background()) {
		t.Fatalf("tier delegation got result=%q err=%v available=%v", got, err, tier.IsAvailable(context.Background()))
	}
}

func TestExecuteBYOKPreservesCredentialGuardsAndDecoration(t *testing.T) {
	missing, unknown := errors.New("missing provider"), errors.New("unknown provider")
	registry := map[string]string{"known": "adapter"}

	_, err := tiered.ExecuteBYOK(registry, "", "known", "test", missing, unknown,
		func(string) (*string, error) { t.Fatal("call must not run without a key"); return nil, nil },
		func(*string, string) { t.Fatal("decoration must not run without a key") })
	if err == nil || !strings.Contains(err.Error(), "audio-tools/test: BYOK key required") {
		t.Fatalf("missing key error = %v", err)
	}

	_, err = tiered.ExecuteBYOK(registry, "key", "", "test", missing, unknown,
		func(string) (*string, error) { t.Fatal("call must not run without a provider"); return nil, nil },
		func(*string, string) { t.Fatal("decoration must not run without a provider") })
	if !errors.Is(err, missing) {
		t.Fatalf("missing provider error = %v", err)
	}

	value, err := tiered.ExecuteBYOK(registry, "key", "known", "test", missing, unknown,
		func(adapter string) (*string, error) { return ptr(adapter + " result"), nil },
		func(result *string, adapter string) { *result += " via " + adapter })
	if err != nil || value == nil || *value != "adapter result via adapter" {
		t.Fatalf("execution result = %v, %v", value, err)
	}
}

func TestNewCredentialChainAppliesPrecedenceAndTerminalErrors(t *testing.T) {
	unknown, missing, credits, allFailed := errors.New("unknown"), errors.New("missing"), errors.New("credits"), errors.New("all failed")
	byok := &tiered.Tier[credentials, string]{
		Execute:     func(context.Context, credentials) (string, error) { return "", unknown },
		IsAvailable: func(context.Context) bool { return true },
	}
	vrooli := &tiered.Tier[credentials, string]{
		Execute:     func(context.Context, credentials) (string, error) { return "vrooli", nil },
		IsAvailable: func(context.Context) bool { return true },
	}
	chain := tiered.NewCredentialChain(tiered.CredentialSet[credentials, string]{
		BYOK: byok, Vrooli: vrooli,
		HasBYOK: func(req credentials) bool { return req.byok }, HasVrooli: func(req credentials) bool { return req.vrooli },
		UnknownBYOK: unknown, MissingBYOK: missing, InsufficientCredits: credits, AllFailed: allFailed,
	}, tiered.ChainOptions{EnableBYOK: true, EnableVrooli: true})

	_, err := chain.Execute(context.Background(), credentials{byok: true, vrooli: true})
	if !errors.Is(err, unknown) {
		t.Fatalf("terminal BYOK error = %v", err)
	}
	result, err := chain.Execute(context.Background(), credentials{vrooli: true})
	if err != nil || result != "vrooli" {
		t.Fatalf("vrooli fallback result = %q, %v", result, err)
	}
}

func ptr(value string) *string { return &value }

func sampleExecute(_ *sampleProvider, _ context.Context, request string) (string, error) {
	return request, nil
}

func sampleAvailable(provider *sampleProvider, _ context.Context) bool { return provider.available }
