package fixtures

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"agent-manager/internal/adapters/runner/codecs"
	"agent-manager/internal/domain"
	"agent-manager/internal/structuredresult"
	"agent-manager/internal/testutil"

	"github.com/google/uuid"
)

// PathologyReplay is a finalized offline run and its decoded transcript
// evidence. The fake-agent process proves the corpus can drive a runner-shaped
// process without consuming a live-agent slot.
type PathologyReplay struct {
	Run    *domain.Run
	Events []*domain.RunEvent
}

// ReplayPathology executes one named corpus through fake-agent and decodes its
// stdout with the Codex codec. Corpus fixtures intentionally use the portable
// Codex stream shape so one loader covers the investigation pathologies.
func ReplayPathology(t testing.TB, name string) PathologyReplay {
	t.Helper()
	corpus := pathologyCorpusPath(t, name)
	command := exec.Command(testutil.BuildFakeAgent(t))
	command.Env = append(os.Environ(), "FAKE_AGENT_CORPUS="+corpus)
	output, err := command.CombinedOutput()
	if err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			t.Fatalf("replay %s: %v", name, err)
		}
	}

	runID := uuid.New()
	codec := codecs.NewCodexForTest()
	state := codec.NewState()
	var events []*domain.RunEvent
	for lineNumber, line := range strings.Split(string(output), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		decoded, decodeErr := codec.DecodeStreamLine(state, runID, line)
		if decodeErr != nil {
			t.Fatalf("decode %s line %d: %v", name, lineNumber+1, decodeErr)
		}
		events = append(events, decoded...)
	}
	result := domain.ResolveRunResult(events, true, 0, "completed")
	if spec := pathologyResultSpec(name); spec != nil {
		result.Structured = structuredresult.Resolver{}.Resolve(context.Background(), spec, result)
	}
	return PathologyReplay{Run: &domain.Run{ID: runID, Status: domain.RunStatusComplete, Result: result}, Events: events}
}

func pathologyResultSpec(name string) *domain.ResultSpec {
	schema := json.RawMessage(`{"type":"object","properties":{"answer":{"type":"string"}},"required":["answer"]}`)
	switch name {
	case "invalid-structured-result":
		return &domain.ResultSpec{Kind: domain.ResultSpecKindJSONSchema, Schema: schema, ExtractionMode: domain.StructuredExtractionDeterministic}
	case "abstained-structured-result":
		return &domain.ResultSpec{Kind: domain.ResultSpecKindJSONSchema, Schema: schema, ExtractionMode: domain.StructuredExtractionConstrained}
	default:
		return nil
	}
}

func pathologyCorpusPath(t testing.TB, name string) string {
	t.Helper()
	if strings.TrimSpace(name) == "" || strings.Contains(name, string(filepath.Separator)) {
		t.Fatalf("invalid pathology name %q", name)
	}
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate pathology corpus")
	}
	path := filepath.Join(filepath.Dir(file), "..", "..", "adapters", "runner", "codecs", "testdata", "corpus", "pathology", name+".jsonl")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("pathology corpus %s: %v", name, err)
	}
	return path
}

// String keeps test failure output compact without exposing transcript bodies.
func (r PathologyReplay) String() string {
	if r.Run == nil || r.Run.Result == nil {
		return "pathology replay without result"
	}
	return fmt.Sprintf("selection=%s structured=%v events=%d", r.Run.Result.Selection.Status, r.Run.Result.Structured, len(r.Events))
}
