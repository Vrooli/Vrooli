package journeytest

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	inferenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference/inference_v1connect"
	"landing-page-business-suite-api/internal/aidoor"
)

func startEmbeddedGateway(t *testing.T, door, key string) inferenceconnect.InferenceServiceClient {
	t.Helper()
	path := filepath.Join(t.TempDir(), "authority.enc")
	authority, err := credentialauthority.OpenEncryptedFile(path, "fixture encrypted authority passphrase")
	if err != nil {
		t.Fatal(err)
	}
	identity, err := credentialauthority.ParseIdentity("vrooli/ai-device")
	if err != nil {
		t.Fatal(err)
	}
	if err = authority.Put(identity, "key", key); err != nil {
		t.Fatal(err)
	}
	dir, err := filepath.Abs("../../../ai-gateway/api")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "test", "./handlers/inference", "-run", "^TestJourneyEmbeddedGatewayServer$", "-count=1", "-v", "-timeout", "10m")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off", "VROOLI_JOURNEY_DOOR="+door, "VROOLI_JOURNEY_CREDENTIALS="+path)
	input, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	output, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		input.Close()
		if err := cmd.Wait(); err != nil {
			t.Errorf("embedded gateway: %v: %s", err, stderr.String())
		}
	})
	scanner := bufio.NewScanner(output)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "EMBEDDED_GATEWAY_URL=") {
			go io.Copy(io.Discard, output)
			base := strings.TrimPrefix(line, "EMBEDDED_GATEWAY_URL=")
			t.Cleanup(func() {
				response, err := http.Post(base+"/__journey/stop", "text/plain", nil)
				if err == nil {
					response.Body.Close()
				}
			})
			return inferenceconnect.NewInferenceServiceClient(http.DefaultClient, base)
		}
	}
	t.Fatalf("embedded gateway failed to start: %s", stderr.String())
	return nil
}

func TestJ12_FallsBackToHostedDoorWithDeviceKey(t *testing.T) {
	var calls int
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"{\\\"answer\\\":\\\"ok\\\"}\"}}]}\n\ndata: {\"choices\":[],\"usage\":{\"prompt_tokens\":4,\"completion_tokens\":3,\"cost\":0.000002}}\n\ndata: [DONE]\n\n")
	}))
	defer upstream.Close()
	t.Setenv("VROOLI_JOURNEY_ROUTER_URL", upstream.URL)
	t.Setenv("VROOLI_JOURNEY_REAL_DECISIONS", "1")
	s := newSpine(t)
	client := startEmbeddedGateway(t, s.server.URL, s.key.Token)
	request := connect.NewRequest(&inferencev1.RunRequest{Source: "Return ok", Instruction: "Return an object with answer ok", SchemaJson: `{"type":"object","properties":{"answer":{"type":"string"}},"required":["answer"],"additionalProperties":false}`, Role: "extract.structured", MaxOutputTokens: 32})
	response, err := client.Run(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.GetError() != nil {
		t.Fatalf("fallback refused: %v", response.Msg.GetError())
	}
	var result map[string]string
	if err = json.Unmarshal([]byte(response.Msg.GetValueJson()), &result); err != nil || result["answer"] != "ok" {
		t.Fatalf("result %s: %v", response.Msg.GetValueJson(), err)
	}
	assertCharged(t, s, 200)
	var charges int
	if err = s.db.QueryRow(`SELECT COUNT(*) FROM ai_key_charges WHERE key_id=$1 AND actual_credits=200`, s.key.ID).Scan(&charges); err != nil || charges != 1 {
		t.Fatalf("issued device key charge rows=%d: %v", charges, err)
	}
	if err := (aidoor.KeyStore{DB: s.db}).Revoke(context.Background(), s.key, s.key.ID); err != nil {
		t.Fatal(err)
	}
	response, err = client.Run(t.Context(), request)
	if err == nil && response.Msg.GetError() == nil {
		t.Fatal("revoked device key answered")
	}
	if calls != 1 {
		t.Fatalf("upstream calls=%d after revoked-key refusal", calls)
	}
	assertCharged(t, s, 200)
}
