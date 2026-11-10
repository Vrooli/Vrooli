package scenarios_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/connectx"
	connectxtest "github.com/vrooli/api-core/connectxtest"

	scenariosH "flow-verifier/handlers/scenarios"
	"flow-verifier/internal/artifacts"
	"flow-verifier/internal/flows"
	"flow-verifier/internal/lifecycleinference"
	"flow-verifier/internal/scenarios"

	scenariosv1 "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/scenarios"
	scenariosconnect "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/scenarios/scenarios_v1connect"
)

func newClient(t *testing.T, scenariosSvc scenariosH.Service, artifactsSvc scenariosH.ArtifactsService) scenariosconnect.ScenariosServiceClient {
	t.Helper()
	logger, _ := connectxtest.NewLogger(t)
	handler := scenariosH.NewStreamHandler(scenariosH.StreamDeps{
		Scenarios: scenariosSvc,
		Artifacts: artifactsSvc,
		Logger:    logger,
	})
	path, h := scenariosconnect.NewScenariosServiceHandler(handler)
	server := connectxtest.StartTestServer(t, connectx.ServiceMount{Path: path, Handler: h})
	return scenariosconnect.NewScenariosServiceClient(server.Client(), server.URL)
}

func TestListScenariosHappyPath(t *testing.T) {
	scenariosSvc := &fakeScenarios{
		root: "/tmp/v",
		list: []scenarios.Summary{
			{ID: "alpha", DisplayName: "Alpha", Path: "/tmp/v/scenarios/alpha", FlowCount: 2},
		},
	}
	client := newClient(t, scenariosSvc, nil)

	resp, err := client.ListScenarios(context.Background(), connect.NewRequest(&scenariosv1.ListScenariosRequest{}))
	require.NoError(t, err)
	require.Equal(t, "/tmp/v", resp.Msg.VrooliRoot)
	require.Len(t, resp.Msg.Scenarios, 1)
	require.Equal(t, "alpha", resp.Msg.Scenarios[0].Id)
	require.Equal(t, int32(2), resp.Msg.Scenarios[0].FlowCount)
}

func TestListLifecyclesReturnsRankedEvidenceAndHuntReviewStatus(t *testing.T) {
	scenariosSvc := &fakeScenarios{
		root: "/tmp/v",
		inventory: lifecycleinference.Inventory{
			Root: "/tmp/v", ScenarioCount: 1, CandidateCount: 1,
			Scenarios: []lifecycleinference.ScenarioSummary{{ID: "alpha", SourceFiles: 3, CandidateCount: 1, ExplicitFlows: 1}},
			Candidates: []lifecycleinference.Candidate{{
				ID: "alpha/api/job#Job.State", ScenarioID: "alpha", Package: "api/job", TypeName: "Job", FieldName: "State",
				Declaration: lifecycleinference.Evidence{Path: "scenarios/alpha/api/job.go", Line: 10, Text: "State JobState"},
				States:      []lifecycleinference.StateEvidence{{Name: "JobQueued", Value: "queued"}},
				HuntRecords: []lifecycleinference.HuntReference{{EntryID: "hunt-1", Round: "Round 4", Status: "review"}},
				ExplicitFlows: []lifecycleinference.FlowReference{{
					FlowID: "alpha.job", Path: "scenarios/alpha/api/job/flow/flow.json", StatusType: "JobState",
					MatchedStates: []string{"queued"}, CandidateStateCount: 1, ContractStateCount: 1, ExactLifecycleMatch: true,
				}},
				TransitionOwners:  []lifecycleinference.Evidence{{Path: "scenarios/alpha/api/job.go", Line: 20, Text: "func (job *Job) TransitionStatus(next JobState) error"}},
				ExplicitlyModeled: true,
				Persistence: lifecycleinference.PersistenceClassification{
					Class: lifecycleinference.ClassPersistedLifecycle, Reason: "candidate data reaches Save", Scope: "scenario production Go source", FilesScanned: 12, ScanComplete: true,
					Evidence: []lifecycleinference.Evidence{{Path: "scenarios/alpha/api/job.go", Line: 30, Text: "store.Save(job)"}},
				},
				BurdenScore: 21, ScoreBreakdown: []string{"typed constants × 3"}, Confidence: "candidate-needs-review",
			}},
		},
	}
	candidate := scenariosSvc.inventory.Candidates[0]
	candidate.PreselectionStatus = "owner_adjudication_candidate"
	scenariosSvc.inventory.Candidates[0] = candidate
	scenariosSvc.inventory.HuntCoverageComplete = true
	scenariosSvc.inventory.HuntCoverageScope = "ecosystem-hunt"
	scenariosSvc.inventory.PreselectionCandidateCount = 1
	scenariosSvc.inventory.PreselectionCandidates = []lifecycleinference.Candidate{candidate}
	client := newClient(t, scenariosSvc, nil)

	resp, err := client.ListLifecycles(context.Background(), connect.NewRequest(&scenariosv1.ListLifecyclesRequest{}))
	require.NoError(t, err)
	require.Equal(t, int32(1), resp.Msg.ScenarioCount)
	require.Equal(t, int32(1), resp.Msg.CandidateCount)
	require.Len(t, resp.Msg.Candidates, 1)
	require.Equal(t, "Job.State", resp.Msg.Candidates[0].TypeName+"."+resp.Msg.Candidates[0].FieldName)
	require.Equal(t, "queued", resp.Msg.Candidates[0].States[0].Value)
	require.Equal(t, "review", resp.Msg.Candidates[0].HuntRecords[0].Status)
	require.True(t, resp.Msg.Candidates[0].ExplicitlyModeled)
	require.Len(t, resp.Msg.Candidates[0].ExplicitFlows, 1)
	require.True(t, resp.Msg.Candidates[0].ExplicitFlows[0].ExactLifecycleMatch)
	require.Len(t, resp.Msg.Candidates[0].TransitionOwners, 1)
	require.Contains(t, resp.Msg.Candidates[0].TransitionOwners[0].Text, "TransitionStatus")
	require.Equal(t, string(lifecycleinference.ClassPersistedLifecycle), resp.Msg.Candidates[0].PersistenceClass)
	require.Contains(t, resp.Msg.Candidates[0].PersistenceReason, "Save")
	require.Equal(t, "store.Save(job)", resp.Msg.Candidates[0].PersistenceEvidence[0].Text)
	require.Equal(t, "scenario production Go source", resp.Msg.Candidates[0].PersistenceScope)
	require.Equal(t, int32(12), resp.Msg.Candidates[0].PersistenceFilesScanned)
	require.True(t, resp.Msg.Candidates[0].PersistenceScanComplete)
	require.True(t, resp.Msg.HuntCoverageComplete)
	require.Equal(t, "ecosystem-hunt", resp.Msg.HuntCoverageScope)
	require.Equal(t, int32(1), resp.Msg.PreselectionCandidateCount)
	require.Equal(t, "owner_adjudication_candidate", resp.Msg.Candidates[0].PreselectionStatus)
	require.Len(t, resp.Msg.PreselectionCandidates, 1)
}

func TestGetScenarioNotFound(t *testing.T) {
	client := newClient(t, &fakeScenarios{detailErr: scenarios.ErrScenarioNotFound}, nil)
	_, err := client.GetScenario(context.Background(), connect.NewRequest(&scenariosv1.GetScenarioRequest{Id: "ghost"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestListLifecyclesRouteBoundsDeadlineAndPropagatesCancellation(t *testing.T) {
	t.Run("large full response survives ordinary write timeout", func(t *testing.T) {
		const largeReasonSize = 3_200_000
		svc := &slowLifecycleScenarios{delay: 150 * time.Millisecond, inventory: lifecycleinference.Inventory{
			Root: "/tmp/v", ScenarioCount: 128, CandidateCount: 1722,
			Candidates: []lifecycleinference.Candidate{{
				ID: "alpha/api/jobs#Attempt.State", TypeName: "Attempt", FieldName: "State",
				Persistence: lifecycleinference.PersistenceClassification{Class: lifecycleinference.ClassPersistedLifecycle, Reason: strings.Repeat("x", largeReasonSize)},
			}},
		}}
		server := startLifecycleRouteServer(t, svc)
		client := scenariosconnect.NewScenariosServiceClient(server.Client(), server.URL)
		resp, err := client.ListLifecycles(context.Background(), connect.NewRequest(&scenariosv1.ListLifecyclesRequest{}))
		require.NoError(t, err)
		require.Equal(t, int32(1722), resp.Msg.CandidateCount)
		require.Len(t, resp.Msg.Candidates, 1)
		require.Len(t, resp.Msg.Candidates[0].PersistenceReason, largeReasonSize)
	})

	t.Run("client cancellation stops inventory work", func(t *testing.T) {
		stopped := make(chan struct{})
		svc := &slowLifecycleScenarios{stopped: stopped, waitForCancel: true}
		server := startLifecycleRouteServer(t, svc)
		client := scenariosconnect.NewScenariosServiceClient(server.Client(), server.URL)
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Millisecond)
		defer cancel()
		_, err := client.ListLifecycles(ctx, connect.NewRequest(&scenariosv1.ListLifecyclesRequest{}))
		require.Error(t, err)
		select {
		case <-stopped:
		case <-time.After(time.Second):
			t.Fatal("service work continued after the HTTP caller canceled")
		}
	})
}

func startLifecycleRouteServer(t *testing.T, svc *slowLifecycleScenarios) *httptest.Server {
	t.Helper()
	router := mux.NewRouter()
	scenariosH.ModuleWithLogger(svc, nil, nil).Mount(router)
	server := httptest.NewUnstartedServer(router)
	server.Config.WriteTimeout = 50 * time.Millisecond
	server.Start()
	t.Cleanup(server.Close)
	server.Client().Timeout = 5 * time.Second
	return server
}

func TestGenerateScenarioArtifactsStreams(t *testing.T) {
	scenariosSvc := &fakeScenarios{
		detail: scenarios.Detail{
			Summary: scenarios.Summary{ID: "s1", Path: "/tmp/v/s1"},
			Flows:   []flows.Summary{{FlowID: "fa"}, {FlowID: "fb"}},
		},
	}
	artifactsSvc := &fakeArtifacts{
		progress: []progressItem{
			{flowID: "fa", report: artifacts.Report{FlowID: "fa", Status: artifacts.StatusFresh}},
			{flowID: "fb", err: errors.New("boom")},
		},
	}
	client := newClient(t, scenariosSvc, artifactsSvc)

	stream, err := client.GenerateScenarioArtifacts(context.Background(), connect.NewRequest(&scenariosv1.GenerateScenarioArtifactsRequest{ScenarioId: "s1"}))
	require.NoError(t, err)
	var received []*scenariosv1.GenerateScenarioArtifactsResponse
	for stream.Receive() {
		received = append(received, stream.Msg())
	}
	require.NoError(t, stream.Err())
	require.Len(t, received, 2)
	require.Equal(t, "fa", received[0].FlowId)
	require.NotNil(t, received[0].Report)
	require.Equal(t, "fb", received[1].FlowId)
	require.Equal(t, "boom", received[1].ErrorMessage)
}

// --- fakes ---

type fakeScenarios struct {
	root      string
	list      []scenarios.Summary
	detail    scenarios.Detail
	detailErr error
	inventory lifecycleinference.Inventory
}

type slowLifecycleScenarios struct {
	delay         time.Duration
	waitForCancel bool
	stopped       chan struct{}
	inventory     lifecycleinference.Inventory
}

func (s *slowLifecycleScenarios) List() ([]scenarios.Summary, error) { return nil, nil }
func (s *slowLifecycleScenarios) Detail(string) (scenarios.Detail, error) {
	return scenarios.Detail{}, scenarios.ErrScenarioNotFound
}
func (s *slowLifecycleScenarios) Root() string { return s.inventory.Root }
func (s *slowLifecycleScenarios) ListLifecycles(ctx context.Context, _ string, _ []byte) (lifecycleinference.Inventory, error) {
	if s.waitForCancel {
		<-ctx.Done()
		close(s.stopped)
		return lifecycleinference.Inventory{}, ctx.Err()
	}
	timer := time.NewTimer(s.delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return s.inventory, nil
	case <-ctx.Done():
		if s.stopped != nil {
			close(s.stopped)
		}
		return lifecycleinference.Inventory{}, ctx.Err()
	}
}

func (f *fakeScenarios) List() ([]scenarios.Summary, error) { return f.list, nil }
func (f *fakeScenarios) Detail(_ string) (scenarios.Detail, error) {
	if f.detailErr != nil {
		return scenarios.Detail{}, f.detailErr
	}
	return f.detail, nil
}
func (f *fakeScenarios) Root() string { return f.root }
func (f *fakeScenarios) ListLifecycles(context.Context, string, []byte) (lifecycleinference.Inventory, error) {
	return f.inventory, nil
}

type progressItem struct {
	flowID string
	report artifacts.Report
	err    error
}

type fakeArtifacts struct {
	progress []progressItem
}

func (f *fakeArtifacts) GenerateForScenarioStream(_ context.Context, _ string, onProgress func(string, artifacts.Report, error) error) error {
	for _, p := range f.progress {
		if err := onProgress(p.flowID, p.report, p.err); err != nil {
			return err
		}
	}
	return nil
}

func (f *fakeArtifacts) ClearForScenario(_ string) ([]artifacts.ClearResult, error) {
	return nil, nil
}
