package proposals

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"git-control-tower/internal/policygate"

	"github.com/vrooli/cli-core/cliutil"
	_ "modernc.org/sqlite"
)

// --- shared fixtures ---

const testRun = "70dd81be-0c15-4ef3-9b05-e18aa6ca7ccb"

var (
	agentActor = Actor{Subject: "agent-run", Kind: "vrooli-agent", Verified: true, RunID: testRun}
	humanActor = Actor{Subject: "operator", Kind: "human", Verified: true}
)

func testService(t *testing.T) *Service {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+strings.ReplaceAll(t.Name(), "/", "_")+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	clock := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	return NewService(store).WithClock(func() time.Time {
		clock = clock.Add(time.Second)
		return clock
	})
}

func repoFor(git Git) Repo { return Repo{ID: "1", Git: git} }

func epochWork() Work {
	return Work{EffortRef: "effort:bas", EffortRevision: "9", Epoch: "E27", RunIDs: []string{testRun}}
}

func createProposal(t *testing.T, svc *Service, repo Repo, paths ...string) View {
	t.Helper()
	result, err := svc.Create(context.Background(), repo, CreateRequest{Work: epochWork(), Subject: "bas: unify timeline (E27)", Body: "Outcome text.", Paths: paths, Actor: agentActor})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	return result.View
}

// humanApplyContext carries a verified human and a consumed apply intent,
// as the transport handler does after Consume.
func humanApplyContext(operation string) context.Context {
	ctx := policygate.WithPrincipal(context.Background(), policygate.Principal{Kind: cliutil.CallerKindHuman, Subject: "operator", Verified: true})
	consumedAt := time.Now().UTC()
	return policygate.WithIntent(ctx, policygate.HumanIntent{ID: "intent-1", PrincipalID: "operator", Operation: operation, ExpiresAt: consumedAt.Add(time.Minute), ConsumedAt: &consumedAt, Consumed: true})
}

func flagCodes(file File) []string {
	var codes []string
	for _, flag := range file.Flags {
		codes = append(codes, flag.Code)
	}
	return codes
}

func fileByPath(t *testing.T, files []File, path string) File {
	t.Helper()
	for _, file := range files {
		if file.Path == path {
			return file
		}
	}
	t.Fatalf("file %s not in %#v", path, files)
	return File{}
}

func agentContextWithIntent() context.Context {
	ctx := policygate.WithPrincipal(context.Background(), policygate.Principal{Kind: cliutil.CallerKindVrooliAgent, Subject: "agent", Verified: true})
	consumedAt := time.Now().UTC()
	return policygate.WithIntent(ctx, policygate.HumanIntent{ID: "intent-2", PrincipalID: "agent", Operation: OperationApply, ExpiresAt: consumedAt.Add(time.Minute), ConsumedAt: &consumedAt, Consumed: true})
}
