package ai

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/ai"
	internalai "web-console/internal/ai"
)

type fakeAIService struct {
	generateErr error
	suggestErr  error
	updateErr   error
	snapshot    ConfigSnapshot
	updated     UpdateConfigRequest
}

func TestRESTGenerateCreditsRefusalUsesPaymentRequired(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/ai/generate", strings.NewReader(`{"prompt":"route this"}`))
	request.Host = "127.0.0.1:16382"
	request.Header.Set("Authorization", "Bearer signed-access")
	restGenerateHandler(&fakeAIService{generateErr: internalai.ErrCreditsRequired}).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusPaymentRequired {
		t.Fatalf("status = %d, want 402; body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"error_type":"credits_required"`) {
		t.Fatalf("body = %s, want typed credits error", recorder.Body.String())
	}
}

func (f *fakeAIService) Generate(context.Context, string, string) (string, string, error) {
	if f.generateErr != nil {
		return "", "", f.generateErr
	}
	return "printf ok", "ollama", nil
}

func (f *fakeAIService) Suggest(context.Context, string, string) ([]string, string, error) {
	if f.suggestErr != nil {
		return nil, "", f.suggestErr
	}
	return []string{"printf one", "printf two"}, "openrouter", nil
}
func (f *fakeAIService) GetConfig(context.Context) ConfigSnapshot { return f.snapshot }
func (f *fakeAIService) UpdateConfig(_ context.Context, req UpdateConfigRequest) (ConfigSnapshot, error) {
	f.updated = req
	if f.updateErr != nil {
		return ConfigSnapshot{}, f.updateErr
	}
	return f.snapshot, nil
}
func (f *fakeAIService) GetHealth(context.Context) []ProviderHealth { return f.snapshot.Health }

func TestConnectHandlerOperationsAndMappings(t *testing.T) {
	fake := &fakeAIService{snapshot: ConfigSnapshot{
		Providers: []ProviderConfig{{Name: "ollama", Enabled: true, Priority: 1, TimeoutSec: 30, MaxRetries: 2}},
		Health:    []ProviderHealth{{Name: "ollama", Available: true, ErrorRate: .1}},
	}}
	h := NewConnectHandler(Deps{Service: fake})
	ctx := context.Background()

	gen, err := h.Generate(ctx, connect.NewRequest(&aiv1.GenerateRequest{Prompt: "list files", Context: "cwd=/tmp"}))
	if err != nil || gen.Msg.Command != "printf ok" || gen.Msg.Provider != "ollama" {
		t.Fatalf("generate: %#v %v", gen, err)
	}
	sug, err := h.Suggest(ctx, connect.NewRequest(&aiv1.SuggestRequest{Prompt: "show status"}))
	if err != nil || len(sug.Msg.Commands) != 2 {
		t.Fatalf("suggest: %#v %v", sug, err)
	}
	cfg, err := h.GetConfig(ctx, connect.NewRequest(&aiv1.GetConfigRequest{}))
	if err != nil || len(cfg.Msg.Providers) != 1 || len(cfg.Msg.Health) != 1 {
		t.Fatalf("config: %#v %v", cfg, err)
	}
	upd, err := h.UpdateConfig(ctx, connect.NewRequest(&aiv1.UpdateConfigRequest{Name: "ollama", HasEnabled: true, Enabled: false, HasPriority: true, Priority: 3}))
	if err != nil || len(upd.Msg.Providers) != 1 || !fake.updated.HasEnabled || fake.updated.Priority != 3 {
		t.Fatalf("update: %#v %v", upd, err)
	}
	health, err := h.GetHealth(ctx, connect.NewRequest(&aiv1.GetHealthRequest{}))
	if err != nil || len(health.Msg.Health) != 1 {
		t.Fatalf("health: %#v %v", health, err)
	}
}

func TestConnectHandlerValidationAndServiceErrors(t *testing.T) {
	tests := []struct {
		name string
		call func(*connectHandler) error
		want connect.Code
	}{
		{"generate empty", func(h *connectHandler) error {
			_, e := h.Generate(context.Background(), connect.NewRequest(&aiv1.GenerateRequest{}))
			return e
		}, connect.CodeInvalidArgument},
		{"suggest empty", func(h *connectHandler) error {
			_, e := h.Suggest(context.Background(), connect.NewRequest(&aiv1.SuggestRequest{}))
			return e
		}, connect.CodeInvalidArgument},
		{"update empty", func(h *connectHandler) error {
			_, e := h.UpdateConfig(context.Background(), connect.NewRequest(&aiv1.UpdateConfigRequest{}))
			return e
		}, connect.CodeInvalidArgument},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call(NewConnectHandler(Deps{Service: &fakeAIService{}}))
			var ce *connect.Error
			if !errors.As(err, &ce) || ce.Code() != tt.want {
				t.Fatalf("got %v, want %v", err, tt.want)
			}
		})
	}
	for _, tc := range []struct {
		name string
		svc  *fakeAIService
		want connect.Code
	}{
		{"generate unavailable", &fakeAIService{generateErr: errors.New("down")}, connect.CodeUnavailable},
		{"suggest unavailable", &fakeAIService{suggestErr: errors.New("down")}, connect.CodeUnavailable},
		{"update unknown", &fakeAIService{updateErr: ErrUnknownProvider}, connect.CodeInvalidArgument},
		{"update invalid", &fakeAIService{updateErr: ErrInvalidBody}, connect.CodeInvalidArgument},
		{"update internal", &fakeAIService{updateErr: errors.New("db")}, connect.CodeInternal},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := NewConnectHandler(Deps{Service: tc.svc})
			var err error
			switch tc.name[:1] {
			case "g":
				_, err = h.Generate(context.Background(), connect.NewRequest(&aiv1.GenerateRequest{Prompt: "x"}))
			case "s":
				_, err = h.Suggest(context.Background(), connect.NewRequest(&aiv1.SuggestRequest{Prompt: "x"}))
			default:
				_, err = h.UpdateConfig(context.Background(), connect.NewRequest(&aiv1.UpdateConfigRequest{Name: "x"}))
			}
			var ce *connect.Error
			if !errors.As(err, &ce) || ce.Code() != tc.want {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}
}
