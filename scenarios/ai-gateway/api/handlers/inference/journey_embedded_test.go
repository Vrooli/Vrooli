package inference

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"ai-gateway/internal/catalog"
	internalinference "ai-gateway/internal/inference"
	"ai-gateway/internal/providers"
	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	inferenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference/inference_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

func packagedGateway(t *testing.T, bundle catalog.Bundle, resolve func(context.Context, string, string) (string, error)) inferenceconnect.InferenceServiceClient {
	server := packagedGatewayServer(t, bundle, resolve, nil)
	return inferenceconnect.NewInferenceServiceClient(server.Client(), server.URL)
}

func packagedGatewayServer(t *testing.T, bundle catalog.Bundle, resolve func(context.Context, string, string) (string, error), door *providers.HostedDoorClient) *httptest.Server {
	t.Helper()
	var roles internalinference.RoleCatalog
	if err := json.Unmarshal(bundle.Roles, &roles); err != nil {
		t.Fatal(err)
	}
	adapters := providers.NewPackagedAdapters(bundle)
	for i := range adapters {
		adapters[i].HTTP.ResolveKey = resolve
	}
	if door != nil {
		adapters = append(adapters, providers.Adapter{Provider: providers.ProviderMetered, Locality: "remote", Door: door})
	}
	repository, err := internalinference.NewResourceRepository(roles, adapters)
	if err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	Module(Deps{Service: internalinference.NewService(repository)}).Mount(router)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	return server
}

// The LPBS journey runner starts this real typed Gateway in an isolated test
// process; only the router on the other side of LPBS is a fixture.
func TestJourneyEmbeddedGatewayServer(t *testing.T) {
	base := os.Getenv("VROOLI_JOURNEY_DOOR")
	if base == "" && os.Getenv("VROOLI_JOURNEY_TYPED_ONLY") != "1" {
		t.Skip("helper invoked by embedded journey runner")
	}
	authority, err := credentialauthority.OpenEncryptedFile(os.Getenv("VROOLI_JOURNEY_CREDENTIALS"), "fixture encrypted authority passphrase")
	if err != nil {
		t.Fatal(err)
	}
	resolve := func(_ context.Context, id, field string) (string, error) {
		identity, err := credentialauthority.ParseIdentity(id)
		if err != nil {
			return "", err
		}
		return authority.Require(identity, field)
	}
	bundle, err := catalog.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	var door *providers.HostedDoorClient
	if base != "" {
		local := bundle.Providers["ollama"]
		local.Endpoint = "http://127.0.0.1:1"
		bundle.Providers["ollama"] = local
		door = &providers.HostedDoorClient{BaseURL: base, ResolveDeviceKey: func(ctx context.Context) (string, error) {
			return resolve(ctx, providers.DeviceKeyIdentity, providers.DeviceKeyField)
		}}
	}
	server := packagedGatewayServer(t, bundle, resolve, door)
	done := make(chan struct{})
	var calls atomic.Int32
	handler := server.Config.Handler
	server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/__journey/stop" {
			close(done)
			return
		}
		if r.URL.Path == "/__journey/evidence" {
			_ = json.NewEncoder(w).Encode(map[string]int32{"calls": calls.Load()})
			return
		}
		calls.Add(1)
		handler.ServeHTTP(w, r)
	})
	fmt.Println("EMBEDDED_GATEWAY_URL=" + server.URL)
	fmt.Printf("EMBEDDED_GATEWAY_PID=%d\n", os.Getpid())
	<-done
}

func journeyRequest() *connect.Request[inferencev1.RunRequest] {
	return connect.NewRequest(&inferencev1.RunRequest{Source: "Return the answer ok.", Instruction: "Return a JSON object with answer equal to ok.", SchemaJson: `{"type":"object","properties":{"answer":{"type":"string"}},"required":["answer"],"additionalProperties":false}`, Role: "extract.structured", MaxOutputTokens: 32})
}

func TestJ12_LocalModelAnswersLocally(t *testing.T) {
	bundle, err := catalog.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	// The provider is the real local Ollama service. There is no command runner,
	// upstream fixture, repository lookup or provider credential on this path.
	client := packagedGateway(t, bundle, func(context.Context, string, string) (string, error) {
		t.Fatal("local request attempted a credential lookup")
		return "", nil
	})
	request := journeyRequest()
	request.Msg.Profile = sharedv1.Profile_PROFILE_LOCAL_ONLY
	ctx, cancel := context.WithTimeout(t.Context(), 4*time.Minute)
	defer cancel()
	response, err := client.Run(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.GetError() != nil {
		t.Fatalf("local gateway refused: %v", response.Msg.GetError())
	}
	if response.Msg.GetValueJson() == "" {
		t.Fatal("local gateway returned no JSON")
	}
	t.Logf("real local gateway result: %s", response.Msg.GetValueJson())
}

func TestJ12_OwnKeyWorks(t *testing.T) {
	bundle, err := catalog.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	// Only the upstream router is fake; the key uses a real encrypted authority
	// and the request crosses the real Gateway transport, service and adapter.
	const key = "synthetic-own-router-key"
	authority, err := credentialauthority.OpenEncryptedFile(t.TempDir()+"/credentials.enc", "fixture encrypted authority passphrase")
	if err != nil {
		t.Fatal(err)
	}
	identity, err := credentialauthority.ParseIdentity(bundle.Providers["openrouter"].CredentialIdentity)
	if err != nil {
		t.Fatal(err)
	}
	field := bundle.Providers["openrouter"].CredentialField
	if err := authority.Put(identity, field, key); err != nil {
		t.Fatal(err)
	}
	calls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "Bearer "+key {
			t.Error("own key did not reach router")
		}
		if r.URL.Path != "/chat/completions" {
			t.Errorf("path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"answer\":\"ok\"}"}}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`))
	}))
	defer upstream.Close()
	p := bundle.Providers["openrouter"]
	p.Endpoint = upstream.URL
	bundle.Providers["openrouter"] = p
	client := packagedGateway(t, bundle, func(_ context.Context, id, field string) (string, error) {
		identity, err := credentialauthority.ParseIdentity(id)
		if err != nil {
			return "", err
		}
		return authority.Require(identity, field)
	})
	request := journeyRequest()
	request.Msg.Profile = sharedv1.Profile_PROFILE_REMOTE_ONLY
	response, err := client.Run(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.GetError() != nil {
		t.Fatalf("own key route refused: %v", response.Msg.GetError())
	}
	if calls != 1 {
		t.Fatalf("router calls %d", calls)
	}
	if response.Msg.GetValueJson() != `{"answer":"ok"}` {
		t.Fatalf("result %s", response.Msg.GetValueJson())
	}
}
