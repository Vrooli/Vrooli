package maintenance

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// Codex quota frames belong to imported transcript runs, which have no
// process identity. An imported run is physically present only while an
// observed executor holds its transcript open: the activity read asks the
// control plane about live imports' transcripts and counts a held one as a
// physical run, without counting a second executor. Finished imports, unclean
// paths and paths the request never named are never claimed.
func TestActivityInventoryObservesImportedRunsWhoseTranscriptIsHeldOpen(t *testing.T) {
	const (
		heldPath    = "/home/op/.codex/sessions/2026/11/20/rollout-held.jsonl"
		unheldPath  = "/home/op/.codex/sessions/2026/11/20/rollout-unheld.jsonl"
		donePath    = "/home/op/.codex/sessions/2026/11/20/rollout-done.jsonl"
		strayPath   = "/home/op/.codex/sessions/2026/11/20/rollout-stray.jsonl"
		heldImport  = "imported-held"
		unheldLabel = "imported-unheld"
	)
	started := time.Date(2026, 11, 20, 14, 25, 22, 0, time.UTC)
	attached := &domain.Run{ID: uuid.New(), Status: domain.RunStatusRunning, ExecutionMode: domain.ExecutionModeAttached, HarnessKind: "codex", HarnessSessionID: "codex-339c4402", RunnerPID: 451992, StartedAt: &started, Tag: "attached-codex"}
	_, db := openGate(t, filepath.Join(t.TempDir(), "activity-imported-transcripts.db"))
	if _, err := db.Exec(`CREATE TABLE runs(id TEXT,status TEXT,execution_mode TEXT,runner_pid INTEGER,runner_pgid INTEGER,finalization_status TEXT,transcript_path TEXT);
CREATE TABLE workflow_executions(id TEXT,status TEXT);
INSERT INTO runs VALUES(?,'running','attached',451992,0,'none',''),
 (?,'running','imported',0,0,'none',?),
 (?,'running','imported',0,0,'none',?),
 ('imported-done','complete','imported',0,0,'none',?),
 ('imported-unclean','running','imported',0,0,'none','/home/op/.codex/../.codex/rollout.jsonl');`,
		attached.ID.String(), heldImport, heldPath, unheldLabel, unheldPath, donePath); err != nil {
		t.Fatal(err)
	}
	var asked []string
	owner := &ControlPlaneObserver{
		lookupBatch: func(context.Context, []ExecutorRef) ([]*domain.Run, error) { return []*domain.Run{attached}, nil },
		read: func(_ context.Context, input []byte) ([]byte, error) {
			var request struct {
				Executors   []executorScopeRef `json:"executors"`
				Transcripts []string           `json:"transcripts"`
			}
			if err := json.Unmarshal(input, &request); err != nil {
				t.Fatal(err)
			}
			asked = request.Transcripts
			held, _ := json.Marshal([]string{heldPath, donePath, strayPath})
			return []byte(fmt.Sprintf(`{"schemaVersion":"executor-scope-v1","identityVersion":"attached-key-v1","complete":true,"executors":[{"runId":%q,"state":"present","pids":[451992],"reasons":[],"openTranscripts":%s}]}`, attached.ID, held)), nil
		},
	}
	reader := NewBatchedInventory(db, owner.Observe)
	reader.transcriptPhysical = owner.ObserveTranscripts
	state, err := reader.ObserveActivity(t.Context())
	if err != nil || !state.PhysicalComplete {
		t.Fatalf("activity inventory: state=%+v err=%v", state, err)
	}
	if !slices.Equal(asked, []string{heldPath, unheldPath}) {
		t.Fatalf("asked about transcripts %v; want only live imports with clean paths", asked)
	}
	if got := strings.Join(state.PhysicalRunIDs, ","); got != attached.ID.String()+","+heldImport || state.ObservedExecutors != 1 {
		t.Fatalf("physical runs=%s executors=%d; want the attached executor and its held import only", got, state.ObservedExecutors)
	}

	// The maintenance read never asks; its verdicts are unchanged.
	asked = nil
	if _, err := reader.Observe(t.Context()); err != nil || asked != nil {
		t.Fatalf("maintenance observe asked about transcripts %v (err=%v)", asked, err)
	}
}
