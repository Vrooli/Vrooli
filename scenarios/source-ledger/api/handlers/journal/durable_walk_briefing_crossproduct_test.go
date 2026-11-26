//go:build crossproduct

package journal

import (
	"context"
	"errors"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"connectrpc.com/connect"
	journalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal"
	"source-ledger/internal/testutil/mocks"
)

// This composes Command Center's no-base-URL fallback with the real Source
// Ledger Journal Connect handler and SQLite repository. A test-injected
// api-core resolver fails offline so SOURCE_LEDGER_API_PORT must select owner.
func TestCrossProductWalkRetainsBriefingAcrossProviderLossAndJournalRestart(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "source-ledger.sqlite")
	inference := &mocks.FakeInference{
		ClassifyOut: "episode",
		EmbedErr:    errors.New("embedding provider unavailable during briefing save"),
	}
	route := openDurableJournalRoute(t, dbPath, inference)
	ctx := context.Background()
	request := walkBriefingAppendRequest(t)
	seed, err := route.client.AppendEntry(ctx, connect.NewRequest(request))
	if err != nil {
		t.Fatal(err)
	}
	entryID := seed.Msg.GetEntry().GetId()
	if entryID == "" || seed.Msg.GetEntry().GetBody() != request.GetBody() {
		t.Fatalf("seed durable briefing = %+v", seed.Msg.GetEntry())
	}
	for _, mode := range []string{"explicit-url", "port-fallback"} {
		t.Run(mode, func(t *testing.T) {
			if err := runCommandCenterConfigurationJourney(t, mode, route.server.URL, entryID); err != nil {
				t.Fatal(err)
			}
		})
	}

	duringLoss, err := route.client.GetEntry(ctx, connect.NewRequest(&journalv1.GetEntryRequest{Scope: request.GetScope(), RequestKey: request.GetRequestKey()}))
	if err != nil {
		t.Fatal(err)
	}
	if duringLoss.Msg.GetEntry().GetId() != entryID || duringLoss.Msg.GetEntry().GetBody() != request.GetBody() || !strings.Contains(duringLoss.Msg.GetEntry().GetBody(), "Saved walk: investigate the throughput change and resume at phase 5.3.") {
		t.Fatalf("runtime Walk did not persist the expected briefing: %+v", duringLoss.Msg.GetEntry())
	}
	deferred, err := route.client.ProcessEmbeddingRetries(ctx, connect.NewRequest(&journalv1.ProcessEmbeddingRetriesRequest{Scope: request.GetScope(), Limit: 10}))
	if err != nil || deferred.Msg.GetProcessed() != 0 || deferred.Msg.GetDeferred() != 1 {
		t.Fatalf("retry while embedding provider is down = %+v, %v", deferred, err)
	}

	// Simulate owner process reconstruction using the same file-backed database.
	route.server.Close()
	if err := route.db.Close(); err != nil {
		t.Fatal(err)
	}
	inference.EmbedErr = nil
	inference.EmbedOut = []float64{0.125, 0.5}
	route = openDurableJournalRoute(t, dbPath, inference)
	reopened, err := route.client.GetEntry(ctx, connect.NewRequest(&journalv1.GetEntryRequest{Scope: request.GetScope(), RequestKey: request.GetRequestKey()}))
	if err != nil || reopened.Msg.GetEntry().GetId() != entryID || reopened.Msg.GetEntry().GetBody() != duringLoss.Msg.GetEntry().GetBody() {
		t.Fatalf("briefing after owner reconstruction = %+v, %v", reopened, err)
	}

	request.Body = reopened.Msg.GetEntry().GetBody()
	replayed, err := route.client.AppendEntry(ctx, connect.NewRequest(request))
	if err != nil || !replayed.Msg.GetExisting() || replayed.Msg.GetEntry().GetId() != entryID {
		t.Fatalf("same-key replay after provider recovery = %+v, %v", replayed, err)
	}
	processed, err := route.client.ProcessEmbeddingRetries(ctx, connect.NewRequest(&journalv1.ProcessEmbeddingRetriesRequest{Scope: request.GetScope(), Limit: 10}))
	if err != nil || processed.Msg.GetProcessed() != 1 {
		t.Fatalf("retry after provider recovery = %+v, %v", processed, err)
	}
	listed, err := route.client.ListEntries(ctx, connect.NewRequest(&journalv1.ListEntriesRequest{Scope: request.GetScope(), Kind: request.GetKind(), NewestFirst: true, Limit: 10}))
	if err != nil || len(listed.Msg.GetEntries()) != 1 || listed.Msg.GetEntries()[0].GetId() != entryID {
		t.Fatalf("durable briefing inventory = %+v, %v", listed, err)
	}
}

func runCommandCenterConfigurationJourney(t *testing.T, mode, sourceLedgerURL, briefingID string) error {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve cross-product test source path")
	}
	apiDir := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
	repoRoot := filepath.Clean(filepath.Join(apiDir, "..", "..", ".."))
	commandCenterDir := filepath.Join(repoRoot, "scenarios", "command-center", "api")
	parsed, err := url.Parse(sourceLedgerURL)
	if err != nil {
		return err
	}
	host, port, found := strings.Cut(parsed.Host, ":")
	if !found || host == "" || port == "" {
		return errors.New("isolated Source Ledger URL has no host port")
	}
	if mode != "explicit-url" && mode != "port-fallback" {
		return errors.New("unsupported Source Ledger owner selection mode: " + mode)
	}
	baseURL, fallbackPort := sourceLedgerURL, ""
	if mode == "port-fallback" {
		baseURL = ""
		fallbackPort = port
	}
	cmd := exec.Command("go", "test", "-count=1", "-run", "^TestWalkRuntimeSourceLedgerConfigurationRecovery$", ".")
	cmd.Dir = commandCenterDir
	cmd.Env = withTestEnv(os.Environ(), map[string]string{
		"SOURCE_LEDGER_BASE_URL":   baseURL,
		"SOURCE_LEDGER_API_PORT":   fallbackPort,
		"SE4_EXPECTED_BRIEFING_ID": briefingID,
		"SE4_WALK_RUNTIME_MODE":    mode,
	})
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.New("Command Center runtime-path test failed: " + string(output))
	}
	return nil
}

func withTestEnv(env []string, updates map[string]string) []string {
	out := make([]string, 0, len(env)+len(updates))
	for _, item := range env {
		key, _, _ := strings.Cut(item, "=")
		if _, replaced := updates[key]; !replaced {
			out = append(out, item)
		}
	}
	for key, value := range updates {
		out = append(out, key+"="+value)
	}
	return out
}
