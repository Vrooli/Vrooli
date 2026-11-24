//go:build journey

package journeytest

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/consumeridentity"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	inferenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference/inference_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"landing-page-business-suite-api/internal/aidoor"
	"landing-page-business-suite-api/internal/commerce"
)

const (
	ac3ManagedGatewayScenario = "ai-gateway-ac3-managed"
	ac3ManagedDeviceKey       = "synthetic-ac3-managed-gateway-device-key"
	ac3ManagedPrivateReceipt  = "private-ac3-managed-gateway-receipt-must-not-escape"
	ac3ManagedRunCallerHeader = "X-Vrooli-LPBS-Run-Assertion"
	ac3ManagedIssuer          = "https://lpbs.ac3.fixture"
)

// TestAC3_ManagedGatewayScenarioRestartRecoversPendingReceipt proves receipt
// recovery after the scenario lifecycle has stopped and started a Gateway API
// process. The temporary scenario source disables every provider adapter but
// LPBS and supplies the same local fixture seams used by the helper-process
// journey. Vrooli still builds, supervises, health-checks and restarts the
// actual Gateway executable; no test helper acts as the Gateway process.
func TestAC3_ManagedGatewayScenarioRestartRecoversPendingReceiptThroughFundedDoor(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
	defer cancel()

	var mu sync.Mutex
	modelReads, providerPosts := 0, 0
	receiptReads := map[string]int{}
	providerPostsByKey := map[string]int{}
	fakeLPBS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+ac3ManagedDeviceKey {
			t.Errorf("fixture LPBS authorization = %q", r.Header.Get("Authorization"))
		}
		mu.Lock()
		defer mu.Unlock()
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/ai/receipt":
			key := r.Header.Get("Idempotency-Key")
			receiptReads[key]++
			w.Header().Set("Content-Type", "application/json")
			switch receiptReads[key] {
			case 1:
				w.WriteHeader(http.StatusNotFound)
			case 2:
				_, _ = fmt.Fprintf(w, `{"status":"pending","charged_credits":4,"private_receipt":%q}`, ac3ManagedPrivateReceipt)
			default:
				_, _ = fmt.Fprintf(w, `{"status":"settled","charged_credits":7,"private_receipt":%q}`, ac3ManagedPrivateReceipt)
			}
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/ai/models":
			modelReads++
			_, _ = w.Write([]byte(`{"data":[{"id":"synthetic-ac3-model"}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/ai/chat/completions":
			providerPosts++
			providerPostsByKey[r.Header.Get("Idempotency-Key")]++
			w.Header().Set("Content-Type", "text/event-stream")
			// The provider accepted one request, but its terminal frame was lost.
			_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"partial\"}}]}\n\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer fakeLPBS.Close()

	repoRoot := ac3RepositoryRoot(t)
	scenarioRoot, err := os.MkdirTemp(filepath.Join(repoRoot, "scenarios"), ".ai-gateway-ac3-managed-")
	if err != nil {
		t.Fatal(err)
	}
	storageRoot := filepath.Join(scenarioRoot, "runtime-storage")
	if err := copyAC3GatewaySource(filepath.Join(repoRoot, "scenarios", "ai-gateway"), scenarioRoot); err != nil {
		_ = os.RemoveAll(scenarioRoot)
		t.Fatal(err)
	}
	signer, jwksPath := ac3ManagedSigner(t, scenarioRoot)
	if err := prepareAC3GatewayVariant(scenarioRoot, storageRoot, fakeLPBS.URL, jwksPath); err != nil {
		_ = os.RemoveAll(scenarioRoot)
		t.Fatal(err)
	}
	started := false
	t.Cleanup(func() {
		if started {
			stopCtx, stopCancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer stopCancel()
			if output, stopErr := ac3Vrooli(stopCtx, "scenario", "stop", ac3ManagedGatewayScenario, "--json"); stopErr != nil {
				t.Errorf("stop managed Gateway instance during cleanup: %v: %s", stopErr, output)
			}
		}
		_ = os.RemoveAll(scenarioRoot)
	})

	startManaged := func() string {
		t.Helper()
		startOutput, startErr := ac3Vrooli(ctx, "scenario", "start", ac3ManagedGatewayScenario, "--path", scenarioRoot, "--timeout", "600", "--json")
		if startErr != nil {
			t.Fatalf("start managed Gateway scenario: %v: %s", startErr, startOutput)
		}
		t.Logf("managed Gateway start: %s", startOutput)
		started = true
		var portReport struct {
			Ports []struct {
				Key  string `json:"key"`
				Port int    `json:"port"`
			} `json:"ports"`
		}
		portOutput, portErr := ac3Vrooli(ctx, "scenario", "port", ac3ManagedGatewayScenario, "--path", scenarioRoot, "--json")
		if portErr != nil {
			t.Fatalf("read managed Gateway port: %v: %s", portErr, portOutput)
		}
		if err := json.Unmarshal([]byte(portOutput), &portReport); err != nil {
			t.Fatalf("decode managed Gateway port response %q: %v", portOutput, err)
		}
		apiPort := 0
		for _, port := range portReport.Ports {
			if port.Key == "API_PORT" {
				apiPort = port.Port
				break
			}
		}
		if apiPort <= 0 {
			t.Fatalf("managed Gateway has no API_PORT: %s", portOutput)
		}
		return fmt.Sprintf("http://127.0.0.1:%d", apiPort)
	}

	dbFixture := newSpineDB(t)
	keys := aidoor.KeyStore{DB: dbFixture.db}
	for _, query := range []string{
		`INSERT INTO business_accounts(id,display_name,billing_email) VALUES('account-b','Journey B','actor-b@example.invalid')`,
		`INSERT INTO business_account_members(business_account_id,user_id,role) VALUES('account-b','actor-b','owner')`,
	} {
		if _, err := dbFixture.db.Exec(query); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := dbFixture.db.Exec(`INSERT INTO credit_funding_lots(id,business_account_id,source_kind,provider,provider_ref,livemode,settlement_state,granted,remaining,status,created_at) VALUES(gen_random_uuid(),'account-b','paid','stripe','fixture-payment-b',FALSE,'settled',10000,10000,'open',$1)`, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	keyB, err := keys.IssueDevice(ctx, "account-b", "actor-b", "web-console", 10000)
	if err != nil {
		t.Fatal(err)
	}
	live := false
	wallet := commerce.NewBusinessAccountUsageService(commerce.BusinessAccountUsageOptions{DB: dbFixture.db, Limits: commerce.NewLimitsService(dbFixture.db, "postgres", nil), Dialect: "postgres", LiveMode: &live})
	door := &aidoor.Door{Keys: keys, Wallet: wallet, SignCreditAssertion: func(key aidoor.Key, operation, featureID string) (string, error) {
		return func() (string, error) {
			token, _, signErr := signer.SignCreditAssertion(consumeridentity.CreditAssertionExpectation{
				BusinessAccountID: key.Account, ActorID: key.Actor, AppKey: key.App, FeatureID: featureID,
				MaxCredits: key.MaxCredits, Audience: consumeridentity.CreditGatewayAudience,
				OperationID: operation, Source: consumeridentity.CreditAssertionSource,
			}, 5*time.Minute)
			return token, signErr
		}()
	}}

	baseURL := startManaged()
	door.GatewayURL = baseURL
	doorServer := httptest.NewServer(door)
	defer doorServer.Close()

	request := &inferencev1.RunRequest{
		Source:      "Read this synthetic input and return its result.",
		Instruction: "Return a JSON object with answer equal to ok.",
		SchemaJson:  `{"type":"object","properties":{"answer":{"type":"string"}},"required":["answer"],"additionalProperties":false}`,
		Role:        "author.generator", MaxOutputTokens: 32,
		PrivacyClass:      sharedv1.PrivacyClass_PRIVACY_CLASS_PUBLIC,
		PrivacyProvenance: "journey_test.explicit_public_sample",
		Profile:           sharedv1.Profile_PROFILE_REMOTE_ONLY,
	}
	requestBody, err := protojson.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	type doorResult struct {
		key       aidoor.Key
		status    int
		operation string
		body      []byte
	}
	results := make(chan doorResult, 2)
	var group sync.WaitGroup
	for _, key := range []aidoor.Key{dbFixture.key, keyB} {
		group.Add(1)
		go func(key aidoor.Key) {
			defer group.Done()
			req, requestErr := http.NewRequestWithContext(ctx, http.MethodPost, doorServer.URL+"/api/v1/ai/run", bytes.NewReader(requestBody))
			if requestErr != nil {
				t.Errorf("build account %s run request: %v", key.Account, requestErr)
				return
			}
			req.Header.Set("Authorization", "Bearer "+key.Token)
			req.Header.Set("Idempotency-Key", "shared-client-operation")
			req.Header.Set("Content-Type", "application/json")
			response, requestErr := http.DefaultClient.Do(req)
			if requestErr != nil {
				t.Errorf("account %s run request: %v", key.Account, requestErr)
				return
			}
			body, readErr := io.ReadAll(response.Body)
			response.Body.Close()
			if readErr != nil {
				t.Errorf("read account %s response: %v", key.Account, readErr)
				return
			}
			results <- doorResult{key: key, status: response.StatusCode, operation: response.Header.Get("X-Operation-ID"), body: body}
		}(key)
	}
	group.Wait()
	close(results)
	byAccount := map[string]doorResult{}
	for result := range results {
		if result.status != http.StatusAccepted || result.operation == "" {
			var gatewayDetail string
			token, _, signErr := signer.SignCreditAssertion(consumeridentity.CreditAssertionExpectation{
				BusinessAccountID: result.key.Account, ActorID: result.key.Actor, AppKey: result.key.App,
				FeatureID: "ai-gateway.run/author.generator", MaxCredits: result.key.MaxCredits,
				Audience: consumeridentity.CreditGatewayAudience, OperationID: result.operation,
				Source: consumeridentity.CreditAssertionSource,
			}, 5*time.Minute)
			if signErr == nil {
				directRequest := proto.Clone(request).(*inferencev1.RunRequest)
				directRequest.RootOperationId = result.operation
				connectRequest := connect.NewRequest(directRequest)
				connectRequest.Header().Set("Authorization", "Bearer "+token)
				connectRequest.Header().Set(ac3ManagedRunCallerHeader, token)
				connectRequest.Header().Set("X-Vrooli-AI-Source", "vrooli-credits")
				if direct, directErr := inferenceconnect.NewInferenceServiceClient(http.DefaultClient, baseURL).Run(ctx, connectRequest); directErr != nil {
					gatewayDetail = directErr.Error()
				} else if direct.Msg.GetError() != nil {
					gatewayDetail = direct.Msg.GetError().GetMessage()
				} else {
					gatewayDetail = fmt.Sprintf("calls=%d recovery=%v", direct.Msg.GetProviderCallsMade(), direct.Msg.GetRecovery())
				}
			}
			mu.Lock()
			postCount, reads := providerPosts, fmt.Sprintf("%v", receiptReads)
			mu.Unlock()
			t.Fatalf("account %s did not retain its pending typed operation: status=%d operation=%q body=%s Gateway detail=%q sign_err=%v provider_posts=%d receipt_reads=%s", result.key.Account, result.status, result.operation, result.body, gatewayDetail, signErr, postCount, reads)
		}
		var public struct {
			Status string `json:"status"`
		}
		if json.Unmarshal(result.body, &public) != nil || public.Status != "pending" || bytes.Contains(result.body, []byte(ac3ManagedPrivateReceipt)) {
			t.Fatalf("account %s received a private or non-pending response: %s", result.key.Account, result.body)
		}
		byAccount[result.key.Account] = result
	}
	accountA, accountB := byAccount["account"], byAccount["account-b"]
	if accountA.operation == "" || accountB.operation == "" || accountA.operation == accountB.operation {
		t.Fatalf("LPBS did not namespace identical client idempotency keys by account key: A=%q B=%q", accountA.operation, accountB.operation)
	}
	mu.Lock()
	if providerPosts != 2 || len(providerPostsByKey) != 2 {
		mu.Unlock()
		t.Fatalf("concurrent LPBS accounts did not get isolated provider attempts: provider POSTs=%d keys=%v", providerPosts, providerPostsByKey)
	}
	mu.Unlock()

	// Replay account A's exact root and body with a valid, signed account B
	// principal. Gateway must bind the recovery digest to the verified caller
	// and refuse the cross-account ledger read before another provider call.
	accountBReplayToken, _, err := signer.SignCreditAssertion(consumeridentity.CreditAssertionExpectation{
		BusinessAccountID: keyB.Account, ActorID: keyB.Actor, AppKey: keyB.App,
		FeatureID: "ai-gateway.run/author.generator", MaxCredits: keyB.MaxCredits,
		Audience: consumeridentity.CreditGatewayAudience, OperationID: accountA.operation,
		Source: consumeridentity.CreditAssertionSource,
	}, 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	crossLedgerRequest := proto.Clone(request).(*inferencev1.RunRequest)
	crossLedgerRequest.RootOperationId = accountA.operation
	crossLedgerConnect := connect.NewRequest(crossLedgerRequest)
	crossLedgerConnect.Header().Set("Authorization", "Bearer "+accountBReplayToken)
	crossLedgerConnect.Header().Set(ac3ManagedRunCallerHeader, accountBReplayToken)
	crossLedgerConnect.Header().Set("X-Vrooli-AI-Source", "vrooli-credits")
	crossLedgerResponse, err := inferenceconnect.NewInferenceServiceClient(http.DefaultClient, baseURL).Run(ctx, crossLedgerConnect)
	if err != nil {
		t.Fatalf("account B cross-ledger replay transport: %v", err)
	}
	if crossLedgerResponse.Msg.GetError() == nil || crossLedgerResponse.Msg.GetError().GetCode() != inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST || crossLedgerResponse.Msg.GetProviderCallsMade() != 0 {
		t.Fatalf("Gateway ledger accepted cross-account recovery replay: error=%v provider_calls=%d", crossLedgerResponse.Msg.GetError(), crossLedgerResponse.Msg.GetProviderCallsMade())
	}
	mu.Lock()
	if providerPosts != 2 {
		mu.Unlock()
		t.Fatalf("cross-account ledger replay dispatched an extra provider request: %d", providerPosts)
	}
	mu.Unlock()

	stopOutput, err := ac3Vrooli(ctx, "scenario", "stop", ac3ManagedGatewayScenario, "--json")
	if err != nil {
		t.Fatalf("stop managed Gateway scenario: %v: %s", err, stopOutput)
	}
	started = false
	baseURL = startManaged()
	door.GatewayURL = baseURL
	retry, err := http.NewRequestWithContext(ctx, http.MethodPost, doorServer.URL+"/api/v1/ai/run", bytes.NewReader(requestBody))
	if err != nil {
		t.Fatal(err)
	}
	retry.Header.Set("Authorization", "Bearer "+dbFixture.key.Token)
	retry.Header.Set("Idempotency-Key", "shared-client-operation")
	retry.Header.Set("Content-Type", "application/json")
	retryResponse, err := http.DefaultClient.Do(retry)
	if err != nil {
		t.Fatal(err)
	}
	retryBody, err := io.ReadAll(retryResponse.Body)
	retryResponse.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if retryResponse.StatusCode != http.StatusOK || !bytes.Contains(retryBody, []byte(`"status":"settled"`)) || !bytes.Contains(retryBody, []byte(`"charged_credits":7`)) || bytes.Contains(retryBody, []byte(ac3ManagedPrivateReceipt)) {
		t.Fatalf("managed Gateway did not recover through the funded LPBS Door: status=%d body=%s", retryResponse.StatusCode, retryBody)
	}

	var accountAState, accountBState, accountAOwner, accountAActor, accountBOwner, accountBActor string
	var accountAReserved, accountBReserved int64
	if err := dbFixture.db.QueryRow(`SELECT status,business_account_id,actor_identity,reserved_amount FROM business_account_credit_reservations WHERE id=$1`, accountA.operation).Scan(&accountAState, &accountAOwner, &accountAActor, &accountAReserved); err != nil {
		t.Fatalf("LPBS reservation for recovered account is not durable: %v", err)
	}
	if err := dbFixture.db.QueryRow(`SELECT status,business_account_id,actor_identity,reserved_amount FROM business_account_credit_reservations WHERE id=$1`, accountB.operation).Scan(&accountBState, &accountBOwner, &accountBActor, &accountBReserved); err != nil {
		t.Fatal(err)
	}
	if accountAState != "finalized" || accountBState != "pending" {
		t.Fatalf("LPBS reservation state crossed accounts: A=%s B=%s", accountAState, accountBState)
	}
	if accountAOwner != "account" || accountAActor != "actor" || accountBOwner != "account-b" || accountBActor != "actor-b" || accountAReserved != 10000 || accountBReserved != 10000 {
		t.Fatalf("LPBS reservations lost tenant ownership or bounded amounts: A=(%s,%s,%d) B=(%s,%s,%d)", accountAOwner, accountAActor, accountAReserved, accountBOwner, accountBActor, accountBReserved)
	}
	var charged int64
	if err := dbFixture.db.QueryRow(`SELECT credits FROM business_account_usage_events WHERE operation_id=$1 AND business_account_id='account'`, accountA.operation).Scan(&charged); err != nil || charged != 7 {
		t.Fatalf("recovered account wallet charge=%d err=%v, want 7", charged, err)
	}
	var accountBUsageEvents, accountBWalletRemaining int64
	if err := dbFixture.db.QueryRow(`SELECT COUNT(*) FROM business_account_usage_events WHERE business_account_id='account-b'`).Scan(&accountBUsageEvents); err != nil {
		t.Fatal(err)
	}
	if err := dbFixture.db.QueryRow(`SELECT COALESCE(SUM(remaining),0) FROM credit_funding_lots WHERE business_account_id='account-b'`).Scan(&accountBWalletRemaining); err != nil {
		t.Fatal(err)
	}
	if accountBUsageEvents != 0 || accountBWalletRemaining != 10000 {
		t.Fatalf("account A settlement changed account B wallet: usage events=%d remaining=%d", accountBUsageEvents, accountBWalletRemaining)
	}
	// An account can only recover the operation derived from its own key. Reusing
	// the other account's opaque operation as an idempotency key derives a fresh
	// root and cannot read that tenant's receipt.
	crossReceipt, err := http.NewRequestWithContext(ctx, http.MethodGet, doorServer.URL+"/api/v1/ai/receipt", nil)
	if err != nil {
		t.Fatal(err)
	}
	crossReceipt.Header.Set("Authorization", "Bearer "+keyB.Token)
	crossReceipt.Header.Set("Idempotency-Key", accountA.operation)
	crossResponse, err := http.DefaultClient.Do(crossReceipt)
	if err != nil {
		t.Fatal(err)
	}
	crossBody, err := io.ReadAll(crossResponse.Body)
	crossResponse.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if crossResponse.StatusCode != http.StatusNotFound {
		t.Fatalf("account B read account A receipt: HTTP %d body=%s", crossResponse.StatusCode, crossBody)
	}
	accountAReadsB, err := http.NewRequestWithContext(ctx, http.MethodGet, doorServer.URL+"/api/v1/ai/receipt", nil)
	if err != nil {
		t.Fatal(err)
	}
	accountAReadsB.Header.Set("Authorization", "Bearer "+dbFixture.key.Token)
	accountAReadsB.Header.Set("Idempotency-Key", accountB.operation)
	accountAReadsBResponse, err := http.DefaultClient.Do(accountAReadsB)
	if err != nil {
		t.Fatal(err)
	}
	accountAReadsBBody, err := io.ReadAll(accountAReadsBResponse.Body)
	accountAReadsBResponse.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if accountAReadsBResponse.StatusCode != http.StatusNotFound {
		t.Fatalf("account A read account B receipt: HTTP %d body=%s", accountAReadsBResponse.StatusCode, accountAReadsBBody)
	}

	mu.Lock()
	defer mu.Unlock()
	if providerPosts != 2 {
		t.Fatalf("managed Gateway redispatched after restart: provider POST count=%d models=%d receipts=%v", providerPosts, modelReads, receiptReads)
	}
	// Provider-door calls are keyed by Gateway attempt IDs (SHA-256 strings),
	// while the funded LPBS responses expose per-account root operation IDs
	// (UUID-shaped). Keep those namespaces separate: the exact B-as-A root replay
	// above proves ledger isolation, and these counts prove two receipts were
	// polled and each of two attempts dispatched exactly once.
	if len(receiptReads) != 2 || len(providerPostsByKey) != 2 {
		t.Fatalf("two account operations did not retain independent Gateway attempts: roots=(%q,%q) reads=%v providerPosts=%v", accountA.operation, accountB.operation, receiptReads, providerPostsByKey)
	}
	var receiptReadTotal int
	for _, count := range receiptReads {
		receiptReadTotal += count
	}
	if receiptReadTotal != 5 {
		t.Fatalf("unexpected Gateway receipt recovery read count %d, want 5 across two pending operations and the post-restart retry: %v", receiptReadTotal, receiptReads)
	}
	for attemptKey, count := range providerPostsByKey {
		if attemptKey == "" || count != 1 {
			t.Fatalf("Gateway attempt %q dispatched %d times across concurrent runs and restart; want one: %v", attemptKey, count, providerPostsByKey)
		}
	}
}

func ac3ManagedSigner(t *testing.T, scenarioRoot string) (*consumeridentity.Signer, string) {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := consumeridentity.NewSigner("ac3-managed-fixture", privateKey, ac3ManagedIssuer, 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	jwks, err := consumeridentity.NewKeySet(consumeridentity.PublicKey{ID: signer.KeyID, Key: &privateKey.PublicKey}).JWKS()
	if err != nil {
		t.Fatal(err)
	}
	jwksPath := filepath.Join(scenarioRoot, "ac3-fixture-jwks.json")
	if err := os.WriteFile(jwksPath, jwks, 0o600); err != nil {
		t.Fatal(err)
	}
	return signer, jwksPath
}

func ac3RepositoryRoot(t *testing.T) string {
	t.Helper()
	current, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for dir := current; ; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "VISION.md")); err == nil {
			return dir
		}
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not locate repository root from %s", current)
		}
	}
}

func copyAC3GatewaySource(source, destination string) error {
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return nil
		}
		if entry.IsDir() && (entry.Name() == ".git" || entry.Name() == "node_modules" || entry.Name() == "dist") {
			return filepath.SkipDir
		}
		if entry.IsDir() {
			return os.MkdirAll(filepath.Join(destination, relative), 0o755)
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		input, err := os.Open(path)
		if err != nil {
			return err
		}
		defer input.Close()
		outputPath := filepath.Join(destination, relative)
		if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
			return err
		}
		output, err := os.OpenFile(outputPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(output, input)
		closeErr := output.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
}

func prepareAC3GatewayVariant(scenarioRoot, storageRoot, fakeLPBSURL, jwksPath string) error {
	manifestPath := filepath.Join(scenarioRoot, ".vrooli", "service.json")
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	var manifest map[string]any
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return err
	}
	service, _ := manifest["service"].(map[string]any)
	service["name"] = ac3ManagedGatewayScenario
	service["displayName"] = "AI Gateway AC3 Managed Restart Fixture"
	manifest["dependencies"] = map[string]any{"scenarios": map[string]any{}, "resources": map[string]any{}}
	components, _ := manifest["components"].(map[string]any)
	for name := range components {
		if name != "api" {
			delete(components, name)
		}
	}
	ports, _ := manifest["ports"].(map[string]any)
	delete(ports, "ui")
	if lifecycle, ok := manifest["lifecycle"].(map[string]any); ok {
		if health, ok := lifecycle["health"].(map[string]any); ok {
			if endpoints, ok := health["endpoints"].(map[string]any); ok {
				delete(endpoints, "ui")
			}
			if checks, ok := health["checks"].([]any); ok {
				filtered := checks[:0]
				for _, check := range checks {
					if row, ok := check.(map[string]any); !ok || row["name"] != "ui_endpoint" {
						filtered = append(filtered, check)
					}
				}
				health["checks"] = filtered
			}
		}
	}
	environment, _ := manifest["environment"].(map[string]any)
	if environment == nil {
		environment = map[string]any{}
		manifest["environment"] = environment
	}
	environment["LPBS_AI_DOOR_URL"] = fakeLPBSURL
	environment["LPBS_CONSUMER_TOKEN_ISSUER"] = ac3ManagedIssuer
	environment["AC3_MANAGED_JWKS_PATH"] = jwksPath
	environment["VROOLI_STORAGE_ROOT"] = storageRoot
	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(manifestPath, append(encoded, '\n'), 0o600); err != nil {
		return err
	}
	mainPath := filepath.Join(scenarioRoot, "api", "main.go")
	mainSource, err := os.ReadFile(mainPath)
	if err != nil {
		return err
	}
	mainSource = bytes.Replace(mainSource, []byte(`"context"`), []byte("\"context\"\n\n\t\"github.com/vrooli/api-core/consumeridentity\""), 1)
	defaultAdapters := []byte("adapters := providers.NewDefaultAdapters(resourceRunner)")
	isolatedAdapters := []byte("adapters := providers.NewDefaultAdapters(resourceRunner)")
	if bytes.Count(mainSource, defaultAdapters) != 1 {
		return fmt.Errorf("expected one Gateway default-adapter composition site, found %d", bytes.Count(mainSource, defaultAdapters))
	}
	mainSource = bytes.Replace(mainSource, defaultAdapters, isolatedAdapters, 1)
	defaultWarmup := []byte("go providers.WarmFacts(context.Background(), providers.NewDefaultAdapters(resourceRunner))")
	isolatedWarmup := []byte("if os.Getenv(\"AC3_MANAGED_JOURNEY\") == \"1\" {\n\t\tgo providers.WarmFacts(context.Background(), nil)\n\t} else {\n\t\tgo providers.WarmFacts(context.Background(), providers.NewDefaultAdapters(resourceRunner))\n\t}")
	if bytes.Count(mainSource, defaultWarmup) != 1 {
		return fmt.Errorf("expected one Gateway startup-warmup site, found %d", bytes.Count(mainSource, defaultWarmup))
	}
	mainSource = bytes.Replace(mainSource, defaultWarmup, isolatedWarmup, 1)
	defaultDoor := []byte("Door: &providers.HostedDoorClient{BaseURL: endpoint}")
	fixtureDoor := []byte("Door: &providers.HostedDoorClient{BaseURL: endpoint, ResolveDeviceKey: func(context.Context) (string, error) { return \"" + ac3ManagedDeviceKey + "\", nil }}")
	if bytes.Count(mainSource, defaultDoor) != 1 {
		return fmt.Errorf("expected one Gateway hosted-door composition site, found %d", bytes.Count(mainSource, defaultDoor))
	}
	mainSource = bytes.Replace(mainSource, defaultDoor, fixtureDoor, 1)
	defaultIdentity := []byte(`lpbsConsumerProvider := internalidentity.NewLPBSConsumerProvider(os.Getenv("LPBS_CONSUMER_TOKEN_ISSUER"))`)
	fixtureIdentity := []byte(`issuer := os.Getenv("LPBS_CONSUMER_TOKEN_ISSUER")
	lpbsJWKSCache := consumeridentity.NewCache(issuer, 0, time.Hour, func(*http.Request) ([]byte, error) {
		return os.ReadFile(os.Getenv("AC3_MANAGED_JWKS_PATH"))
	})
	lpbsConsumerProvider := internalidentity.NewLPBSConsumerProviderWithCache(issuer, lpbsJWKSCache)`)
	if bytes.Count(mainSource, defaultIdentity) != 1 {
		return fmt.Errorf("expected one Gateway consumer identity composition site, found %d", bytes.Count(mainSource, defaultIdentity))
	}
	mainSource = bytes.Replace(mainSource, defaultIdentity, fixtureIdentity, 1)
	databaseScenario := []byte(`Scenario:     "ai-gateway",`)
	isolatedDatabaseScenario := []byte(`Scenario:     "` + ac3ManagedGatewayScenario + `",`)
	if bytes.Count(mainSource, databaseScenario) != 1 {
		return fmt.Errorf("expected one Gateway database scenario binding, found %d", bytes.Count(mainSource, databaseScenario))
	}
	mainSource = bytes.Replace(mainSource, databaseScenario, isolatedDatabaseScenario, 1)
	if err := os.WriteFile(mainPath, mainSource, 0o644); err != nil {
		return err
	}

	manifestBytes, err = os.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return err
	}
	environment, _ = manifest["environment"].(map[string]any)
	environment["AC3_MANAGED_JOURNEY"] = "1"
	encoded, err = json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(manifestPath, append(encoded, '\n'), 0o600)
}

func ac3Vrooli(ctx context.Context, args ...string) (string, error) {
	path, err := exec.LookPath("vrooli")
	if err != nil {
		return "", err
	}
	command := exec.CommandContext(ctx, path, args...)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return strings.TrimSpace(stdout.String() + "\n" + stderr.String()), err
	}
	return strings.TrimSpace(stdout.String()), nil
}
