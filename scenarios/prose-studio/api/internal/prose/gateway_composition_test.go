package prose

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	inferenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference/inference_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

type gatewayWireEvidence struct {
	Calls         int32 `json:"calls"`
	EmbedRequests []struct {
		Role              string `json:"role"`
		Profile           string `json:"profile"`
		PrivacyClass      string `json:"privacy_class"`
		PrivacyProvenance string `json:"privacy_provenance"`
	} `json:"embed_requests"`
}

func TestProseServiceComposesWithGatewayAndRefusesBeforeProviderDispatch(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()

	var localTagCalls, localDispatchCalls, remoteCalls atomic.Int32
	localProvider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/tags":
			localTagCalls.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"models":[{"name":"nomic-embed-text:latest"}]}`))
		case r.Method == http.MethodPost:
			localDispatchCalls.Add(1)
			http.Error(w, "task-owned fake provider must not execute", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer localProvider.Close()
	remoteProvider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		remoteCalls.Add(1)
		http.Error(w, "task-owned fake remote provider must not execute", http.StatusForbidden)
	}))
	defer remoteProvider.Close()

	gatewayURL, stopGateway := startCompositionGateway(ctx, t, localProvider.URL, remoteProvider.URL)
	defer stopGateway()
	service := &Service{gateway: HTTPGateway{BaseURL: gatewayURL}}

	// This production Prose service path builds its confidential, local-only
	// embedding request and sends it over the generated Connect client to the
	// actual AI Gateway Connect handler running on loopback. The packaged model
	// has no trusted rights receipt in this isolated fixture, so admission must
	// fail before the fake provider's embedding endpoint is dispatched.
	_, _, err := service.semanticSectionRepetition(ctx, []string{"private section one", "private section two"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "embedding_unavailable")
	require.GreaterOrEqual(t, localTagCalls.Load(), int32(1), "Gateway should resolve the local role after receiving the sourced task contract")
	require.Zero(t, localDispatchCalls.Load(), "rights refusal must happen before local provider execution")
	require.Zero(t, remoteCalls.Load(), "local-only Prose requests must never contact a remote provider")

	evidence := readGatewayWireEvidence(t, gatewayURL)
	require.Len(t, evidence.EmbedRequests, 1)
	require.Equal(t, "embedding.default", evidence.EmbedRequests[0].Role)
	require.Equal(t, sharedv1.Profile_PROFILE_LOCAL_ONLY.String(), evidence.EmbedRequests[0].Profile)
	require.Equal(t, sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL.String(), evidence.EmbedRequests[0].PrivacyClass)
	require.Equal(t, prosePrivacySource, evidence.EmbedRequests[0].PrivacyProvenance)

	// An invalid provenance contract submitted at the public Connect route is
	// rejected by the real Gateway service before repository resolution. The
	// provider's inventory and execution counts therefore stay unchanged.
	client := inferenceconnect.NewInferenceServiceClient(http.DefaultClient, gatewayURL)
	bad, err := client.Embed(ctx, connect.NewRequest(&inferencev1.EmbedRequest{
		Role: "embedding.default", Texts: []string{"private fixture"},
		Profile: sharedv1.Profile_PROFILE_LOCAL_ONLY, PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL,
		PrivacyProvenance: "malformed private source !",
	}))
	require.NoError(t, err)
	require.Equal(t, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, bad.Msg.GetError().GetCode())
	require.Equal(t, localTagCalls.Load(), int32(1), "invalid provenance must be refused before provider role resolution")
	require.Zero(t, localDispatchCalls.Load())
	require.Zero(t, remoteCalls.Load())

	remote, err := client.Embed(ctx, connect.NewRequest(&inferencev1.EmbedRequest{
		Role: "embedding.default", Texts: []string{"private fixture"},
		Profile: sharedv1.Profile_PROFILE_REMOTE_ONLY, PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL,
		PrivacyProvenance: prosePrivacySource,
	}))
	require.NoError(t, err)
	require.Equal(t, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, remote.Msg.GetError().GetCode())
	require.Equal(t, int32(1), localTagCalls.Load(), "remote-only embeddings must be refused before local role resolution")
	require.Zero(t, localDispatchCalls.Load())
	require.Zero(t, remoteCalls.Load(), "confidential content must not cross to the remote provider")

	// Prose also refuses a mismatched source at its own client boundary, before
	// making another Connect request.
	_, err = (HTTPGateway{BaseURL: gatewayURL}).Embed(ctx, EmbeddingRequest{
		Role: "embedding.default", Texts: []string{"private fixture"},
		Profile: sharedv1.Profile_PROFILE_LOCAL_ONLY, PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL,
		PrivacyProvenance: "another_scenario.public_content",
	})
	require.ErrorContains(t, err, "sourced confidential local-only contract")
	evidence = readGatewayWireEvidence(t, gatewayURL)
	require.Len(t, evidence.EmbedRequests, 3, "the Prose caller must not transmit mismatched provenance")
	require.Zero(t, localDispatchCalls.Load())
	require.Zero(t, remoteCalls.Load())
}

func startCompositionGateway(ctx context.Context, t *testing.T, localURL, remoteURL string) (string, func()) {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	repositoryRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "..", ".."))
	gatewayDir := filepath.Join(repositoryRoot, "scenarios", "ai-gateway", "api")
	serverDir := t.TempDir()
	binary := filepath.Join(serverDir, "ai-gateway-inference.test")
	build := exec.CommandContext(ctx, "go", "test", "-c", "-o", binary, "./handlers/inference")
	build.Dir = gatewayDir
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build AI Gateway handler test server: %v\n%s", err, output)
	}

	cmd := exec.CommandContext(ctx, binary, "-test.v", "-test.run=^TestJourneyEmbeddedGatewayServer$")
	cmd.Env = append(os.Environ(),
		"VROOLI_JOURNEY_TYPED_ONLY=1",
		"VROOLI_JOURNEY_DOOR="+remoteURL,
		"VROOLI_JOURNEY_OLLAMA_ENDPOINT="+localURL,
		"VROOLI_JOURNEY_DEVICE_KEY=lm4-loopback-test-device-key",
		"VROOLI_JOURNEY_CREDENTIALS="+filepath.Join(serverDir, "credentials.enc"),
		"VROOLI_JOURNEY_LEDGER="+filepath.Join(serverDir, "gateway.sqlite"),
	)
	cmd.Dir = gatewayDir
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Start())
	gatewayURL := ""
	stopped := false
	stop := func() {
		if stopped {
			return
		}
		stopped = true
		if cmd.ProcessState != nil {
			return
		}
		if gatewayURL != "" {
			stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
			request, requestErr := http.NewRequestWithContext(stopCtx, http.MethodGet, gatewayURL+"/__journey/stop", nil)
			if requestErr == nil {
				response, getErr := http.DefaultClient.Do(request)
				if getErr == nil {
					_ = response.Body.Close()
				} else {
					t.Errorf("stop loopback AI Gateway test process: %v", getErr)
				}
			} else {
				t.Errorf("create loopback AI Gateway stop request: %v", requestErr)
			}
			stopCancel()
		}
		if err := cmd.Wait(); err != nil {
			t.Errorf("loopback AI Gateway test process did not exit cleanly: %v", err)
		}
	}
	t.Cleanup(func() {
		if !stopped && cmd.ProcessState == nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	})

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "EMBEDDED_GATEWAY_URL=") {
			gatewayURL = strings.TrimPrefix(line, "EMBEDDED_GATEWAY_URL=")
			return gatewayURL, stop
		}
	}
	t.Fatalf("AI Gateway test server exited before announcing its URL: %v", scanner.Err())
	return "", stop
}

func readGatewayWireEvidence(t *testing.T, gatewayURL string) gatewayWireEvidence {
	t.Helper()
	response, err := http.Get(gatewayURL + "/__journey/evidence")
	require.NoError(t, err)
	defer response.Body.Close()
	var evidence gatewayWireEvidence
	require.NoError(t, json.NewDecoder(response.Body).Decode(&evidence))
	return evidence
}
