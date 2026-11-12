package inference

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"ai-gateway/internal/catalog"
	internalinference "ai-gateway/internal/inference"
	"ai-gateway/internal/providers"
	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	apidb "github.com/vrooli/api-core/database"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	inferenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference/inference_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
	"google.golang.org/protobuf/encoding/protojson"
	_ "modernc.org/sqlite"
)

func packagedGateway(t *testing.T, bundle catalog.Bundle, resolve func(context.Context, string, string) (string, error)) inferenceconnect.InferenceServiceClient {
	server := packagedGatewayServer(t, bundle, resolve, nil)
	return inferenceconnect.NewInferenceServiceClient(server.Client(), server.URL)
}

func packagedGatewayServer(t *testing.T, bundle catalog.Bundle, resolve func(context.Context, string, string) (string, error), door *providers.HostedDoorClient, ledgers ...internalinference.AttemptLedger) *httptest.Server {
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
	repository, err := internalinference.NewResourceRepository(roles, adapters, ledgers...)
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
	var ledgers []internalinference.AttemptLedger
	if path := os.Getenv("VROOLI_JOURNEY_LEDGER"); path != "" {
		db, err := sql.Open("sqlite", path)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		if err := apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(internalinference.AttemptSchema)); err != nil {
			t.Fatal(err)
		}
		ledgers = append(ledgers, internalinference.NewSQLAttemptLedger(db))
	}
	server := packagedGatewayServer(t, bundle, resolve, door, ledgers...)
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
	// These synthetic inputs are explicitly public so remote route tests exercise
	// their intended provider contracts under the privacy gate.
	return connect.NewRequest(&inferencev1.RunRequest{Source: "Return the answer ok.", Instruction: "Return a JSON object with answer equal to ok.", SchemaJson: `{"type":"object","properties":{"answer":{"type":"string"}},"required":["answer"],"additionalProperties":false}`, Role: "extract.structured", MaxOutputTokens: 32, PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_PUBLIC})
}

type gatewayProcess struct {
	cmd    *exec.Cmd
	url    string
	reader *bufio.Reader
}

func startGatewayProcess(t *testing.T, doorURL, credentialsPath, ledgerPath string) *gatewayProcess {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestJourneyEmbeddedGatewayServer$")
	cmd.Env = append(os.Environ(),
		"VROOLI_JOURNEY_DOOR="+doorURL,
		"VROOLI_JOURNEY_CREDENTIALS="+credentialsPath,
		"VROOLI_JOURNEY_LEDGER="+ledgerPath,
	)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	process := &gatewayProcess{cmd: cmd, reader: bufio.NewReader(stdout)}
	t.Cleanup(func() {
		if cmd.ProcessState == nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	})
	for {
		line, err := process.reader.ReadString('\n')
		if err != nil {
			if exitErr := cmd.Wait(); exitErr != nil {
				t.Fatalf("Gateway helper exited before announcing its URL: %v", exitErr)
			}
			t.Fatal("Gateway helper exited before announcing its URL")
		}
		if strings.HasPrefix(line, "EMBEDDED_GATEWAY_URL=") {
			process.url = strings.TrimSpace(strings.TrimPrefix(line, "EMBEDDED_GATEWAY_URL="))
			return process
		}
	}
}

func (p *gatewayProcess) stop(t *testing.T) {
	t.Helper()
	response, err := http.Get(p.url + "/__journey/stop")
	if err != nil {
		t.Fatalf("stop Gateway process: %v", err)
	}
	_ = response.Body.Close()
	if err := p.cmd.Wait(); err != nil {
		t.Fatalf("Gateway process did not exit cleanly: %v", err)
	}
}

func TestJ12_LocalModelWithUnknownRightsIsRefusedBeforeInference(t *testing.T) {
	bundle, err := catalog.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	// The local transport is a fake inventory endpoint. The packaged provider
	// cannot supply artifact rights/runtime identity, so no generation request
	// may follow role resolution.
	tagCalls, generationCalls := 0, 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			tagCalls++
			_, _ = w.Write([]byte(`{"models":[{"name":"qwen3.5:9b"}]}`))
		case "/api/generate":
			generationCalls++
			_, _ = w.Write([]byte(`{"response":"{\"answer\":\"ok\"}"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()
	local := bundle.Providers["ollama"]
	local.Endpoint = upstream.URL
	bundle.Providers["ollama"] = local
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
	if response.Msg.GetError().GetCode() != inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_UNAVAILABLE {
		t.Fatalf("unknown local rights must fail closed: %v", response.Msg.GetError())
	}
	if !strings.Contains(response.Msg.GetError().GetMessage(), "rights, model identity, or trusted organization context is missing") {
		t.Fatalf("unexpected refusal reason: %v", response.Msg.GetError())
	}
	if tagCalls != 1 || generationCalls != 0 {
		t.Fatalf("local calls tags=%d generation=%d; unknown rights must stop before inference", tagCalls, generationCalls)
	}
}

func TestJ12_OwnKeyUnavailableWithoutAnAdapter(t *testing.T) {
	bundle, err := catalog.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	// With all remote adapters absent, the own-key source is unavailable. The
	// journey must not manufacture a successful answer or contact a provider.
	local := bundle.Providers["ollama"]
	bundle.Providers = map[string]catalog.Provider{"ollama": local}
	client := packagedGateway(t, bundle, func(context.Context, string, string) (string, error) {
		t.Fatal("an unavailable adapter must not look up a credential")
		return "", nil
	})
	request := journeyRequest()
	request.Msg.Profile = sharedv1.Profile_PROFILE_REMOTE_ONLY
	request.Header().Set("X-Vrooli-AI-Source", "own-key")
	response, err := client.Run(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.GetError().GetCode() != inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_UNAVAILABLE {
		t.Fatalf("missing own-key adapter must be unavailable: %v", response.Msg.GetError())
	}
	if response.Msg.GetValueJson() != "" || response.Msg.GetProvider() != "" {
		t.Fatalf("unavailable route returned provider output: %v", response.Msg)
	}
}

func TestJ18_HostedUncertaintyRecoversThroughGeneratedConnectClient(t *testing.T) {
	bundle, err := catalog.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	ctx := t.Context()
	databasePath := filepath.Join(t.TempDir(), "gateway.sqlite")
	openLedger := func() (*sql.DB, *internalinference.SQLAttemptLedger) {
		t.Helper()
		db, err := sql.Open("sqlite", databasePath)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = db.Close() })
		if err := db.PingContext(ctx); err != nil {
			t.Fatal(err)
		}
		if err := apidb.EnsureSchemas(ctx, db, apidb.SchemaProviderFunc(internalinference.AttemptSchema)); err != nil {
			t.Fatal(err)
		}
		return db, internalinference.NewSQLAttemptLedger(db)
	}
	sqlDB, ledger := openLedger()

	const deviceKey = "synthetic-connect-journey-device-key"
	const privateReceipt = "private-lpbs-provider-receipt-must-not-escape"
	receiptReads, modelReads, providerPosts := 0, 0, 0
	var postedKey string
	var receiptKeys []string
	lpbs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+deviceKey {
			t.Errorf("LPBS authorization = %q", r.Header.Get("Authorization"))
		}
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/ai/receipt":
			receiptReads++
			receiptKeys = append(receiptKeys, r.Header.Get("Idempotency-Key"))
			w.Header().Set("Content-Type", "application/json")
			if receiptReads == 1 {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			if receiptReads == 2 {
				_, _ = w.Write([]byte(`{"status":"pending","charged_credits":4,"private_receipt":"` + privateReceipt + `"}`))
				return
			}
			_, _ = w.Write([]byte(`{"status":"settled","charged_credits":7,"private_receipt":"` + privateReceipt + `"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/ai/models":
			modelReads++
			_, _ = w.Write([]byte(`{"data":[{"id":"synthetic-lpbs-model"}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/ai/chat/completions":
			providerPosts++
			postedKey = r.Header.Get("Idempotency-Key")
			w.Header().Set("Content-Type", "text/event-stream")
			// Simulate a response lost after provider dispatch: a partial stream
			// ends without its terminal frame, so no output is recoverable.
			_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"partial\"}}]}\n\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer lpbs.Close()

	door := &providers.HostedDoorClient{
		BaseURL:          lpbs.URL,
		ResolveDeviceKey: func(context.Context) (string, error) { return deviceKey, nil },
	}
	server := packagedGatewayServer(t, bundle, nil, door, ledger)
	client := inferenceconnect.NewInferenceServiceClient(server.Client(), server.URL)
	request := journeyRequest()
	request.Msg.Role = "extract.structured"
	request.Msg.Profile = sharedv1.Profile_PROFILE_REMOTE_ONLY
	request.Msg.Source = "Read this synthetic input and return its result."
	request.Msg.SchemaJson = `{"type":"object","properties":{"answer":{"type":"string"}},"required":["answer"],"additionalProperties":false}`
	request.Msg.RootOperationId = "e3a19a8d3c2f4b75a04f1926738b5d61"
	request.Header().Set("X-Vrooli-AI-Source", "vrooli-credits")

	first, err := client.Run(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if first.Msg.GetError().GetCode() != inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_OUTCOME_UNCERTAIN {
		t.Fatalf("first call error = %v", first.Msg.GetError())
	}
	if first.Msg.GetRecovery().GetState() != inferencev1.RecoveryReceiptState_RECOVERY_RECEIPT_STATE_PENDING {
		t.Fatalf("first recovery state = %v", first.Msg.GetRecovery().GetState())
	}

	// Reconstruct the Gateway service and reopen its file-backed ledger before
	// retrying. This exercises the persisted root and child key across restart.
	server.Close()
	if err := sqlDB.Close(); err != nil {
		t.Fatal(err)
	}
	sqlDB, ledger = openLedger()
	server = packagedGatewayServer(t, bundle, nil, door, ledger)
	client = inferenceconnect.NewInferenceServiceClient(server.Client(), server.URL)
	second, err := client.Run(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if second.Msg.GetError().GetCode() != inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_OUTCOME_UNCERTAIN {
		t.Fatalf("recovery error = %v", second.Msg.GetError())
	}
	if second.Msg.GetRecovery().GetState() != inferencev1.RecoveryReceiptState_RECOVERY_RECEIPT_STATE_SETTLED {
		t.Fatalf("recovered state = %v", second.Msg.GetRecovery().GetState())
	}
	if second.Msg.GetRecovery().GetAccountingStatus() != "settled" || second.Msg.GetUsage().GetCreditsCharged() != 7 {
		t.Fatalf("settled accounting was not returned: recovery=%v usage=%v", second.Msg.GetRecovery(), second.Msg.GetUsage())
	}
	if second.Msg.GetValueJson() != "" || second.Msg.GetProvider() != "" || second.Msg.GetModel() != "" {
		t.Fatalf("recovery leaked or replayed provider output: %v", second.Msg)
	}
	if first.Msg.GetRecovery().GetAttemptReference() != second.Msg.GetRecovery().GetAttemptReference() {
		t.Fatalf("attempt reference changed: first=%v second=%v", first.Msg.GetRecovery(), second.Msg.GetRecovery())
	}
	encoded, err := protojson.Marshal(second.Msg)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), privateReceipt) {
		t.Fatalf("private receipt escaped the public response: %s", encoded)
	}
	if providerPosts != 1 || modelReads != 1 || receiptReads != 3 {
		t.Fatalf("unexpected upstream calls: POST=%d models=%d receipts=%d", providerPosts, modelReads, receiptReads)
	}
	if postedKey == "" || len(receiptKeys) != 3 {
		t.Fatalf("attempt key missing: posted=%q receipts=%v", postedKey, receiptKeys)
	}
	for _, key := range receiptKeys {
		if key != postedKey {
			t.Fatalf("receipt key %q does not match posted attempt %q", key, postedKey)
		}
	}
}

func TestJ18_HostedUncertaintyRecoversAcrossGatewayProcessRestart(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	const deviceKey = "synthetic-process-restart-device-key"
	const privateReceipt = "private-process-restart-receipt-must-not-escape"
	receiptReads, modelReads, providerPosts := 0, 0, 0
	var postedKey string
	var receiptKeys []string
	lpbs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+deviceKey {
			t.Errorf("LPBS authorization = %q", r.Header.Get("Authorization"))
		}
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/ai/receipt":
			receiptReads++
			receiptKeys = append(receiptKeys, r.Header.Get("Idempotency-Key"))
			w.Header().Set("Content-Type", "application/json")
			if receiptReads == 1 {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			if receiptReads == 2 {
				_, _ = w.Write([]byte(`{"status":"pending","charged_credits":4,"private_receipt":"` + privateReceipt + `"}`))
				return
			}
			_, _ = w.Write([]byte(`{"status":"settled","charged_credits":7,"private_receipt":"` + privateReceipt + `"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/ai/models":
			modelReads++
			_, _ = w.Write([]byte(`{"data":[{"id":"synthetic-lpbs-model"}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/ai/chat/completions":
			providerPosts++
			postedKey = r.Header.Get("Idempotency-Key")
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"partial\"}}]}\n\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer lpbs.Close()

	credentialsPath := filepath.Join(t.TempDir(), "credentials.enc")
	authority, err := credentialauthority.OpenEncryptedFile(credentialsPath, "fixture encrypted authority passphrase")
	if err != nil {
		t.Fatal(err)
	}
	if err := authority.Put(providers.DeviceKeyIdentity, providers.DeviceKeyField, deviceKey); err != nil {
		t.Fatal(err)
	}
	databasePath := filepath.Join(t.TempDir(), "gateway.sqlite")
	firstProcess := startGatewayProcess(t, lpbs.URL, credentialsPath, databasePath)
	client := inferenceconnect.NewInferenceServiceClient(http.DefaultClient, firstProcess.url)
	request := journeyRequest()
	request.Msg.Profile = sharedv1.Profile_PROFILE_REMOTE_ONLY
	request.Msg.RootOperationId = "f4c2d2aa77e64162b6b91d46c9a7b103"
	request.Header().Set("X-Vrooli-AI-Source", "vrooli-credits")

	first, err := client.Run(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if first.Msg.GetError().GetCode() != inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_OUTCOME_UNCERTAIN {
		t.Fatalf("first call error = %v", first.Msg.GetError())
	}
	if first.Msg.GetRecovery().GetState() != inferencev1.RecoveryReceiptState_RECOVERY_RECEIPT_STATE_PENDING {
		t.Fatalf("first recovery state = %v", first.Msg.GetRecovery().GetState())
	}
	firstProcess.stop(t)

	secondProcess := startGatewayProcess(t, lpbs.URL, credentialsPath, databasePath)
	client = inferenceconnect.NewInferenceServiceClient(http.DefaultClient, secondProcess.url)
	second, err := client.Run(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if second.Msg.GetError().GetCode() != inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_OUTCOME_UNCERTAIN {
		t.Fatalf("recovery error = %v", second.Msg.GetError())
	}
	if second.Msg.GetRecovery().GetState() != inferencev1.RecoveryReceiptState_RECOVERY_RECEIPT_STATE_SETTLED || second.Msg.GetUsage().GetCreditsCharged() != 7 {
		t.Fatalf("recovery did not return settled accounting: %v", second.Msg)
	}
	if second.Msg.GetValueJson() != "" || second.Msg.GetProvider() != "" || second.Msg.GetModel() != "" {
		t.Fatalf("recovery leaked or replayed provider output: %v", second.Msg)
	}
	if first.Msg.GetRecovery().GetAttemptReference() != second.Msg.GetRecovery().GetAttemptReference() {
		t.Fatalf("attempt reference changed across process restart: first=%v second=%v", first.Msg.GetRecovery(), second.Msg.GetRecovery())
	}
	encoded, err := protojson.Marshal(second.Msg)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), privateReceipt) {
		t.Fatalf("private receipt escaped the public response: %s", encoded)
	}
	if providerPosts != 1 || modelReads != 1 || receiptReads != 3 {
		t.Fatalf("unexpected upstream calls after Gateway restart: POST=%d models=%d receipts=%d", providerPosts, modelReads, receiptReads)
	}
	if postedKey == "" || len(receiptKeys) != 3 {
		t.Fatalf("attempt key missing: posted=%q receipts=%v", postedKey, receiptKeys)
	}
	for _, key := range receiptKeys {
		if key != postedKey {
			t.Fatalf("receipt key %q does not match posted attempt %q", key, postedKey)
		}
	}
}
