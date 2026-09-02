package appctx

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	actionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/actions"
	actionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/actions/actions_v1connect"
	agentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/agents"
	agentsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/agents/agents_v1connect"
	aisearchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/aisearch"
	aisearchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/aisearch/aisearch_v1connect"
	discoveryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/discovery"
	discoveryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/discovery/discovery_v1connect"
	experimentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/experiments"
	experimentsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/experiments/experiments_v1connect"
	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/graph"
	graphconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/graph/graph_v1connect"
	heartbeatv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/heartbeat"
	heartbeatconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/heartbeat/heartbeat_v1connect"
	memberflowv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/memberflow"
	memberflowconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/memberflow/memberflow_v1connect"
	metadatav1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/metadata"
	metadataconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/metadata/metadata_v1connect"
	searchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/search"
	searchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/search/search_v1connect"
	skillsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/skills"
	skillsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/skills/skills_v1connect"
	tagsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/tags"
	tagsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/tags/tags_v1connect"
	teamsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/teams"
	teamsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/teams/teams_v1connect"
	testingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/testing"
	testingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/testing/testing_v1connect"
	topicsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/topics"
	topicsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/topics/topics_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/structpb"
)

// connectRequest routes migrated domains through generated clients while the
// command packages retain their established parsing and human-output logic.
// The bool reports ownership; owned paths never fall back to REST after an RPC
// error.
func (r Runtime) connectRequest(method, path string, query url.Values, payload any) ([]byte, bool, error) {
	if r.Core == nil {
		return nil, false, nil
	}
	parsedPath, err := url.Parse(path)
	if err != nil {
		return nil, true, fmt.Errorf("parse request path: %w", err)
	}
	if embedded := parsedPath.Query(); len(embedded) > 0 {
		if query == nil {
			query = make(url.Values)
		}
		for key, values := range embedded {
			query[key] = append(query[key], values...)
		}
	}
	path = parsedPath.Path

	httpClient, baseURL := cliapp.NewConnectHTTPClient(r.Core)
	ctx := context.Background()
	segments := pathSegments(path)
	if len(segments) == 0 {
		return nil, false, nil
	}
	if len(segments) == 3 && segments[0] == "skills" && (segments[2] == "test" || segments[2] == "test-history") {
		client := testingconnect.NewTestingServiceClient(httpClient, baseURL)
		return callTesting(ctx, client, method, segments, query, payload)
	}

	switch segments[0] {
	case "experiments":
		client := experimentsconnect.NewExperimentsServiceClient(httpClient, baseURL)
		return callExperiments(ctx, client, method, segments, payload)
	case "graph":
		client := graphconnect.NewGraphServiceClient(httpClient, baseURL)
		return callGraph(ctx, client, method, segments, query, payload)
	case "skills":
		if len(segments) == 3 && segments[2] == "experiments" {
			client := experimentsconnect.NewExperimentsServiceClient(httpClient, baseURL)
			return callExperiments(ctx, client, method, segments, payload)
		}
		client := skillsconnect.NewSkillsServiceClient(httpClient, baseURL)
		return callSkills(ctx, client, method, segments, query, payload)
	case "actions":
		client := actionsconnect.NewActionsServiceClient(httpClient, baseURL)
		return callActions(ctx, client, method, segments, query, payload)
	case "tags":
		client := tagsconnect.NewTagsServiceClient(httpClient, baseURL)
		return callTags(ctx, client, method, segments, payload)
	case "agents":
		client := agentsconnect.NewAgentsServiceClient(httpClient, baseURL)
		return callAgents(ctx, client, method, segments, payload)
	case "members":
		client := agentsconnect.NewAgentsServiceClient(httpClient, baseURL)
		return callLegacyMembers(ctx, client, method, segments, payload)
	case "teams":
		client := teamsconnect.NewTeamsServiceClient(httpClient, baseURL)
		body, handled, callErr := callTeams(ctx, client, method, segments, query, payload)
		if handled {
			return body, handled, callErr
		}
		hb := heartbeatconnect.NewHeartbeatServiceClient(httpClient, baseURL)
		body, handled, callErr = callHeartbeat(ctx, hb, method, segments, query, payload)
		if handled {
			return body, handled, callErr
		}
		mf := memberflowconnect.NewMemberflowServiceClient(httpClient, baseURL)
		return callMemberflow(ctx, mf, method, segments, payload)
	case "topics":
		if len(segments) > 1 && (segments[1] == "graph" || segments[1] == "rules" || segments[1] == "drain-status") {
			client := memberflowconnect.NewMemberflowServiceClient(httpClient, baseURL)
			return callMemberflow(ctx, client, method, segments, payload)
		}
		client := topicsconnect.NewTopicsServiceClient(httpClient, baseURL)
		return callTopics(ctx, client, method, segments, payload)
	case "og-metadata":
		client := metadataconnect.NewMetadataServiceClient(httpClient, baseURL)
		return callMetadata(ctx, client, method, query)
	case "search":
		searchClient := searchconnect.NewSearchServiceClient(httpClient, baseURL)
		aiClient := aisearchconnect.NewAISearchServiceClient(httpClient, baseURL)
		return callSearch(ctx, searchClient, aiClient, method, segments, query, payload)
	case "discover", "discovery-gaps", "discovery-metrics", "skill-usage":
		client := discoveryconnect.NewDiscoveryServiceClient(httpClient, baseURL)
		return callDiscovery(ctx, client, method, segments, query, payload)
	case "objectives", "orientation-cost", "instruments", "operating-models":
		client := memberflowconnect.NewMemberflowServiceClient(httpClient, baseURL)
		return callMemberflow(ctx, client, method, segments, payload)
	case "tasks", "runs", "heartbeat-attempts", "heartbeats", "prompt-preview", "prompt-preview-structured":
		client := heartbeatconnect.NewHeartbeatServiceClient(httpClient, baseURL)
		return callHeartbeat(ctx, client, method, segments, query, payload)
	default:
		return nil, false, nil
	}
}

func callExperiments(ctx context.Context, client experimentsconnect.ExperimentsServiceClient, method string, s []string, payload any) ([]byte, bool, error) {
	if method == "GET" && len(s) == 3 && s[0] == "skills" && s[2] == "experiments" {
		id := s[1]
		resp, err := client.ListExperiments(ctx, connect.NewRequest(&experimentsv1.ListExperimentsRequest{SkillId: &id}))
		return rpcBody(resp, "experiments", true, err)
	}
	if len(s) == 0 || s[0] != "experiments" {
		return nil, false, nil
	}
	switch {
	case method == "GET" && len(s) == 1:
		resp, err := client.ListExperiments(ctx, connect.NewRequest(&experimentsv1.ListExperimentsRequest{}))
		return rpcBody(resp, "experiments", true, err)
	case method == "POST" && len(s) == 1:
		req := &experimentsv1.CreateExperimentRequest{}
		if err := unmarshalPayload(payload, req); err != nil {
			return nil, true, err
		}
		resp, err := client.CreateExperiment(ctx, connect.NewRequest(req))
		return rpcBody(resp, "", true, err)
	case method == "GET" && len(s) == 2:
		resp, err := client.GetExperiment(ctx, connect.NewRequest(&experimentsv1.GetExperimentRequest{ExperimentId: s[1]}))
		return rpcBody(resp, "", true, err)
	case method == "PUT" && len(s) == 2:
		req := &experimentsv1.UpdateExperimentRequest{ExperimentId: s[1]}
		if err := unmarshalPayload(payload, req); err != nil {
			return nil, true, err
		}
		req.ExperimentId = s[1]
		resp, err := client.UpdateExperiment(ctx, connect.NewRequest(req))
		return rpcBody(resp, "", true, err)
	case method == "DELETE" && len(s) == 2:
		resp, err := client.DeleteExperiment(ctx, connect.NewRequest(&experimentsv1.DeleteExperimentRequest{ExperimentId: s[1]}))
		return rpcBody(resp, "", true, err)
	case method == "POST" && len(s) == 3 && s[2] == "start":
		resp, err := client.StartExperiment(ctx, connect.NewRequest(&experimentsv1.StartExperimentRequest{ExperimentId: s[1]}))
		return rpcBody(resp, "", true, err)
	case method == "POST" && len(s) == 3 && s[2] == "conclude":
		req := &experimentsv1.ConcludeExperimentRequest{ExperimentId: s[1]}
		if err := unmarshalPayload(payload, req); err != nil {
			return nil, true, err
		}
		req.ExperimentId = s[1]
		resp, err := client.ConcludeExperiment(ctx, connect.NewRequest(req))
		return rpcBody(resp, "", true, err)
	case method == "GET" && len(s) == 3 && s[2] == "outcomes":
		resp, err := client.ListOutcomes(ctx, connect.NewRequest(&experimentsv1.ListOutcomesRequest{ExperimentId: s[1]}))
		return rpcBody(resp, "outcomes", true, err)
	case method == "POST" && len(s) == 3 && s[2] == "outcomes":
		req := &experimentsv1.RecordOutcomeRequest{ExperimentId: s[1]}
		if err := unmarshalPayload(payload, req); err != nil {
			return nil, true, err
		}
		req.ExperimentId = s[1]
		resp, err := client.RecordOutcome(ctx, connect.NewRequest(req))
		return rpcBody(resp, "", true, err)
	case method == "POST" && len(s) == 3 && s[2] == "assignments":
		req := &experimentsv1.AssignExperimentRequest{ExperimentId: s[1]}
		if err := unmarshalPayload(payload, req); err != nil {
			return nil, true, err
		}
		req.ExperimentId = s[1]
		resp, err := client.AssignExperiment(ctx, connect.NewRequest(req))
		return rpcBody(resp, "", true, err)
	case method == "POST" && len(s) == 3 && s[2] == "audit-receipt":
		req := &experimentsv1.RecordAuditReceiptRequest{ExperimentId: s[1]}
		if err := unmarshalPayload(payload, req); err != nil {
			return nil, true, err
		}
		req.ExperimentId = s[1]
		resp, err := client.RecordAuditReceipt(ctx, connect.NewRequest(req))
		return rpcBody(resp, "", true, err)
	case method == "POST" && len(s) == 3 && s[2] == "holdout-receipt":
		req := &experimentsv1.RecordHoldoutReceiptRequest{ExperimentId: s[1]}
		if err := unmarshalPayload(payload, req); err != nil {
			return nil, true, err
		}
		req.ExperimentId = s[1]
		resp, err := client.RecordHoldoutReceipt(ctx, connect.NewRequest(req))
		return rpcBody(resp, "", true, err)
	case method == "POST" && len(s) == 3 && s[2] == "promote":
		req := &experimentsv1.PromoteExperimentRequest{ExperimentId: s[1]}
		if err := unmarshalPayload(payload, req); err != nil {
			return nil, true, err
		}
		req.ExperimentId = s[1]
		resp, err := client.PromoteExperiment(ctx, connect.NewRequest(req))
		return rpcBody(resp, "", true, err)
	case method == "GET" && len(s) == 3 && s[2] == "report":
		resp, err := client.GetExperimentReport(ctx, connect.NewRequest(&experimentsv1.GetExperimentReportRequest{ExperimentId: s[1]}))
		return rpcBody(resp, "", true, err)
	default:
		return nil, false, nil
	}
}

func callGraph(ctx context.Context, client graphconnect.GraphServiceClient, method string, s []string, query url.Values, payload any) ([]byte, bool, error) {
	switch {
	case method == "GET" && len(s) == 1:
		resp, err := client.GetGraph(ctx, connect.NewRequest(&graphv1.GetGraphRequest{}))
		return rpcBody(resp, "", true, err)
	case method == "POST" && len(s) == 2 && s[1] == "regenerate":
		resp, err := client.RegenerateGraph(ctx, connect.NewRequest(&graphv1.RegenerateGraphRequest{}))
		return rpcBody(resp, "", true, err)
	case method == "GET" && len(s) == 2 && s[1] == "orphans":
		resp, err := client.ListOrphanedSkills(ctx, connect.NewRequest(&graphv1.ListNodesRequest{}))
		return rpcBody(resp, "nodes", true, err)
	case method == "GET" && len(s) == 2 && s[1] == "skillless":
		resp, err := client.ListSkilllessAgents(ctx, connect.NewRequest(&graphv1.ListNodesRequest{}))
		return rpcBody(resp, "nodes", true, err)
	case method == "GET" && len(s) == 2 && s[1] == "empty-teams":
		resp, err := client.ListEmptyTeams(ctx, connect.NewRequest(&graphv1.ListNodesRequest{}))
		return rpcBody(resp, "nodes", true, err)
	case method == "GET" && len(s) == 2 && s[1] == "unaffiliated":
		resp, err := client.ListUnaffiliatedAgents(ctx, connect.NewRequest(&graphv1.ListNodesRequest{}))
		return rpcBody(resp, "nodes", true, err)
	case method == "GET" && len(s) == 2 && s[1] == "popular":
		limit, _ := strconv.ParseInt(query.Get("limit"), 10, 32)
		resp, err := client.ListPopularNodes(ctx, connect.NewRequest(&graphv1.ListPopularNodesRequest{Limit: int32(limit)}))
		return rpcBody(resp, "nodes", true, err)
	case method == "GET" && len(s) == 2 && s[1] == "cycles":
		resp, err := client.ListCycles(ctx, connect.NewRequest(&graphv1.ListCyclesRequest{}))
		raw, handled, callErr := rpcBody(resp, "cycles", true, err)
		if callErr != nil {
			return raw, handled, callErr
		}
		var cycles []struct {
			NodeIDs []string `json:"nodeIds"`
		}
		if err := json.Unmarshal(raw, &cycles); err != nil {
			return nil, true, err
		}
		out := make([][]string, len(cycles))
		for i := range cycles {
			out[i] = cycles[i].NodeIDs
		}
		encoded, err := json.Marshal(out)
		return encoded, true, err
	case method == "GET" && len(s) == 2 && s[1] == "health":
		resp, err := client.GetHealthScores(ctx, connect.NewRequest(&graphv1.GetHealthScoresRequest{}))
		return rpcBody(resp, "scores", true, err)
	case method == "GET" && len(s) == 2 && s[1] == "health-config":
		resp, err := client.GetHealthConfig(ctx, connect.NewRequest(&graphv1.GetHealthConfigRequest{}))
		return rpcBody(resp, "", true, err)
	case method == "PUT" && len(s) == 2 && s[1] == "health-config":
		cfg := &graphv1.HealthConfig{}
		if err := unmarshalPayload(payload, cfg); err != nil {
			return nil, true, err
		}
		resp, err := client.UpdateHealthConfig(ctx, connect.NewRequest(&graphv1.UpdateHealthConfigRequest{Config: cfg}))
		return rpcBody(resp, "", true, err)
	case method == "GET" && len(s) == 3 && s[1] == "nodes":
		resp, err := client.GetNode(ctx, connect.NewRequest(&graphv1.GetNodeRequest{NodeId: s[2]}))
		return rpcBody(resp, "", true, err)
	case method == "GET" && len(s) == 4 && s[1] == "nodes" && s[3] == "edges":
		resp, err := client.ListNodeEdges(ctx, connect.NewRequest(&graphv1.ListNodeEdgesRequest{NodeId: s[2]}))
		return rpcBody(resp, "edges", true, err)
	default:
		return nil, false, nil
	}
}

func callMemberflow(ctx context.Context, client memberflowconnect.MemberflowServiceClient, method string, s []string, payload any) ([]byte, bool, error) {
	var resp *connect.Response[memberflowv1.JsonResponse]
	var err error
	switch {
	case method == "GET" && len(s) == 5 && s[0] == "teams" && s[2] == "members" && s[4] == "topics":
		resp, err = client.GetMemberTopics(ctx, connect.NewRequest(&memberflowv1.MemberRequest{TeamId: s[1], AgentId: s[3]}))
	case method == "PUT" && len(s) == 5 && s[0] == "teams" && s[2] == "members" && s[4] == "topics":
		value, valueErr := jsonValue(payload)
		if valueErr != nil {
			return nil, true, valueErr
		}
		resp, err = client.UpdateMemberTopics(ctx, connect.NewRequest(&memberflowv1.UpdateMemberTopicsRequest{TeamId: s[1], AgentId: s[3], Topics: value}))
	case method == "GET" && len(s) == 3 && s[0] == "teams" && s[2] == "topics":
		resp, err = client.GetTeamTopics(ctx, connect.NewRequest(&memberflowv1.TeamRequest{TeamId: s[1]}))
	case method == "GET" && len(s) == 2 && s[0] == "topics" && s[1] == "graph":
		resp, err = client.GetTopicGraph(ctx, connect.NewRequest(&memberflowv1.EmptyRequest{}))
	case method == "GET" && len(s) == 2 && s[0] == "topics" && s[1] == "rules":
		resp, err = client.GetRules(ctx, connect.NewRequest(&memberflowv1.EmptyRequest{}))
	case method == "GET" && len(s) == 2 && s[0] == "topics" && s[1] == "drain-status":
		resp, err = client.GetDrainStatus(ctx, connect.NewRequest(&memberflowv1.EmptyRequest{}))
	case method == "GET" && len(s) == 1 && s[0] == "objectives":
		resp, err = client.GetObjectives(ctx, connect.NewRequest(&memberflowv1.EmptyRequest{}))
	case method == "GET" && len(s) == 1 && s[0] == "orientation-cost":
		resp, err = client.GetOrientationCost(ctx, connect.NewRequest(&memberflowv1.EmptyRequest{}))
	case method == "GET" && len(s) == 1 && s[0] == "instruments":
		resp, err = client.GetInstruments(ctx, connect.NewRequest(&memberflowv1.EmptyRequest{}))
	case method == "GET" && len(s) == 1 && s[0] == "operating-models":
		resp, err = client.GetOperatingModels(ctx, connect.NewRequest(&memberflowv1.EmptyRequest{}))
	case method == "GET" && len(s) == 2 && s[0] == "operating-models" && s[1] == "validate":
		resp, err = client.ValidateOperatingModels(ctx, connect.NewRequest(&memberflowv1.EmptyRequest{}))
	case method == "GET" && len(s) == 2 && s[0] == "operating-models" && s[1] == "diff":
		resp, err = client.DiffOperatingModels(ctx, connect.NewRequest(&memberflowv1.EmptyRequest{}))
	case method == "GET" && len(s) == 2 && s[0] == "operating-models" && s[1] == "coverage":
		resp, err = client.GetOperatingModelCoverage(ctx, connect.NewRequest(&memberflowv1.EmptyRequest{}))
	case method == "GET" && len(s) == 2 && s[0] == "operating-models" && s[1] == "map":
		resp, err = client.GetOperatingMap(ctx, connect.NewRequest(&memberflowv1.EmptyRequest{}))
	default:
		return nil, false, nil
	}
	return rpcBody(resp, "data", true, err)
}

func callHeartbeat(ctx context.Context, client heartbeatconnect.HeartbeatServiceClient, method string, s []string, query url.Values, payload any) ([]byte, bool, error) {
	value, valueErr := jsonValue(payload)
	if valueErr != nil {
		return nil, true, valueErr
	}
	q := queryMap(query)
	var resp *connect.Response[heartbeatv1.JsonResponse]
	var err error
	switch {
	case len(s) == 1 && s[0] == "tasks" && method == "POST":
		resp, err = client.CreateTask(ctx, connect.NewRequest(&heartbeatv1.JsonMutationRequest{Body: value}))
	case len(s) == 1 && s[0] == "runs" && method == "POST":
		resp, err = client.CreateRun(ctx, connect.NewRequest(&heartbeatv1.JsonMutationRequest{Body: value}))
	case len(s) == 1 && s[0] == "runs" && method == "GET":
		resp, err = client.ListRuns(ctx, connect.NewRequest(&heartbeatv1.QueryRequest{Query: q}))
	case len(s) == 1 && s[0] == "heartbeat-attempts" && method == "GET":
		resp, err = client.ListHeartbeatAttempts(ctx, connect.NewRequest(&heartbeatv1.QueryRequest{Query: q}))
	case len(s) == 2 && s[0] == "runs" && s[1] == "investigate" && method == "POST":
		resp, err = client.CreateInvestigationRun(ctx, connect.NewRequest(&heartbeatv1.JsonMutationRequest{Body: value}))
	case len(s) == 2 && s[0] == "runs" && s[1] == "investigation-apply" && method == "POST":
		resp, err = client.CreateInvestigationApplyRun(ctx, connect.NewRequest(&heartbeatv1.JsonMutationRequest{Body: value}))
	case len(s) == 2 && s[0] == "runs" && method == "GET":
		resp, err = client.GetRun(ctx, connect.NewRequest(&heartbeatv1.RunRequest{RunId: s[1]}))
	case len(s) == 3 && s[0] == "runs" && s[2] == "retry" && method == "POST":
		resp, err = client.RetryRun(ctx, connect.NewRequest(&heartbeatv1.RunMutationRequest{RunId: s[1], Body: value}))
	case len(s) == 3 && s[0] == "runs" && s[2] == "events" && method == "GET":
		resp, err = client.GetRunEvents(ctx, connect.NewRequest(&heartbeatv1.RunQueryRequest{RunId: s[1], Query: q}))
	case len(s) == 3 && s[0] == "runs" && s[2] == "continue" && method == "POST":
		resp, err = client.ContinueRun(ctx, connect.NewRequest(&heartbeatv1.RunMutationRequest{RunId: s[1], Body: value}))
	case len(s) == 2 && s[0] == "heartbeats" && s[1] == "control" && method == "GET":
		resp, err = client.GetHeartbeatControl(ctx, connect.NewRequest(&heartbeatv1.EmptyRequest{}))
	case len(s) == 3 && s[0] == "heartbeats" && s[1] == "control" && s[2] == "policy" && method == "PUT":
		resp, err = client.UpdateHeartbeatControlPolicy(ctx, connect.NewRequest(&heartbeatv1.JsonMutationRequest{Body: value}))
	case len(s) == 3 && s[0] == "heartbeats" && s[1] == "control" && s[2] == "pause" && method == "POST":
		resp, err = client.PauseHeartbeatControl(ctx, connect.NewRequest(&heartbeatv1.JsonMutationRequest{Body: value}))
	case len(s) == 3 && s[0] == "heartbeats" && s[1] == "control" && s[2] == "resume" && method == "POST":
		resp, err = client.ResumeHeartbeatControl(ctx, connect.NewRequest(&heartbeatv1.JsonMutationRequest{Body: value}))
	case len(s) == 2 && s[0] == "heartbeats" && s[1] == "running" && method == "GET":
		resp, err = client.ListRunning(ctx, connect.NewRequest(&heartbeatv1.EmptyRequest{}))
	case len(s) == 5 && s[0] == "heartbeats" && s[1] == "running" && s[4] == "stop" && method == "POST":
		resp, err = client.StopRunning(ctx, connect.NewRequest(&heartbeatv1.MemberMutationRequest{TeamId: s[2], AgentId: s[3], Body: value, Query: q}))
	case len(s) == 1 && s[0] == "prompt-preview" && method == "POST":
		resp, err = client.PreviewPrompt(ctx, connect.NewRequest(&heartbeatv1.JsonMutationRequest{Body: value}))
	case len(s) == 1 && s[0] == "prompt-preview-structured" && method == "POST":
		resp, err = client.PreviewPromptStructured(ctx, connect.NewRequest(&heartbeatv1.JsonMutationRequest{Body: value}))
	case len(s) >= 2 && s[0] == "teams":
		return callTeamHeartbeat(ctx, client, method, s, q, value)
	default:
		return nil, false, nil
	}
	return rpcBody(resp, "data", true, err)
}

func callTeamHeartbeat(ctx context.Context, client heartbeatconnect.HeartbeatServiceClient, method string, s []string, q map[string]string, body *structpb.Value) ([]byte, bool, error) {
	team := s[1]
	var resp *connect.Response[heartbeatv1.JsonResponse]
	var err error
	switch {
	case len(s) == 3 && s[2] == "heartbeats" && method == "GET":
		resp, err = client.ListHeartbeats(ctx, connect.NewRequest(&heartbeatv1.TeamRequest{TeamId: team}))
	case len(s) == 4 && s[2] == "heartbeats" && s[3] == "control" && method == "GET":
		resp, err = client.GetTeamHeartbeatControl(ctx, connect.NewRequest(&heartbeatv1.TeamRequest{TeamId: team}))
	case len(s) == 5 && s[2] == "heartbeats" && s[3] == "control" && s[4] == "policy" && method == "PUT":
		resp, err = client.UpdateTeamHeartbeatControlPolicy(ctx, connect.NewRequest(&heartbeatv1.TeamMutationRequest{TeamId: team, Body: body, Query: q}))
	case len(s) == 5 && s[2] == "heartbeats" && s[3] == "control" && s[4] == "pause" && method == "POST":
		resp, err = client.PauseTeamHeartbeatControl(ctx, connect.NewRequest(&heartbeatv1.TeamMutationRequest{TeamId: team, Body: body, Query: q}))
	case len(s) == 5 && s[2] == "heartbeats" && s[3] == "control" && s[4] == "resume" && method == "POST":
		resp, err = client.ResumeTeamHeartbeatControl(ctx, connect.NewRequest(&heartbeatv1.TeamMutationRequest{TeamId: team, Body: body, Query: q}))
	case len(s) == 4 && s[2] == "heartbeats":
		req := &heartbeatv1.MemberMutationRequest{TeamId: team, AgentId: s[3], Body: body, Query: q}
		if method == "GET" {
			resp, err = client.GetHeartbeat(ctx, connect.NewRequest(&heartbeatv1.MemberRequest{TeamId: team, AgentId: s[3]}))
		} else if method == "POST" {
			resp, err = client.CreateHeartbeat(ctx, connect.NewRequest(req))
		} else if method == "PUT" {
			resp, err = client.UpdateHeartbeat(ctx, connect.NewRequest(req))
		} else if method == "DELETE" {
			resp, err = client.DeleteHeartbeat(ctx, connect.NewRequest(&heartbeatv1.MemberRequest{TeamId: team, AgentId: s[3]}))
		} else {
			return nil, false, nil
		}
	case len(s) == 5 && s[2] == "heartbeats" && s[4] == "trigger" && method == "POST":
		resp, err = client.TriggerHeartbeat(ctx, connect.NewRequest(&heartbeatv1.MemberMutationRequest{TeamId: team, AgentId: s[3], Body: body, Query: q}))
	case len(s) == 5 && s[2] == "heartbeats" && s[4] == "logs" && method == "GET":
		resp, err = client.ListLogs(ctx, connect.NewRequest(&heartbeatv1.MemberQueryRequest{TeamId: team, AgentId: s[3], Query: q}))
	case len(s) == 6 && s[2] == "heartbeats" && s[4] == "logs" && method == "GET":
		resp, err = client.GetLog(ctx, connect.NewRequest(&heartbeatv1.LogRequest{TeamId: team, AgentId: s[3], LogId: s[5]}))
	case len(s) == 4 && s[2] == "heartbeats" && s[3] == "logs" && method == "GET":
		resp, err = client.ListTeamLogs(ctx, connect.NewRequest(&heartbeatv1.TeamQueryRequest{TeamId: team, Query: q}))
	case len(s) == 3 && s[2] == "trigger" && method == "POST":
		resp, err = client.TriggerTeam(ctx, connect.NewRequest(&heartbeatv1.TeamMutationRequest{TeamId: team, Body: body, Query: q}))
	case len(s) == 3 && s[2] == "execution-status" && method == "GET":
		resp, err = client.GetTeamExecutionStatus(ctx, connect.NewRequest(&heartbeatv1.TeamRequest{TeamId: team}))
	case len(s) == 5 && s[2] == "queue" && s[3] == "running" && method == "DELETE":
		resp, err = client.ClearTeamQueueRunning(ctx, connect.NewRequest(&heartbeatv1.MemberMutationRequest{TeamId: team, AgentId: s[4], Body: body, Query: q}))
	case len(s) == 5 && s[2] == "members" && (s[4] == "responsibilities" || s[4] == "heartbeat-instructions" || s[4] == "context" || s[4] == "handoff"):
		req := &heartbeatv1.MemberMutationRequest{TeamId: team, AgentId: s[3], Body: body, Query: q}
		read := &heartbeatv1.MemberRequest{TeamId: team, AgentId: s[3]}
		switch s[4] {
		case "responsibilities":
			if method == "GET" {
				resp, err = client.GetResponsibilities(ctx, connect.NewRequest(read))
			} else {
				resp, err = client.SetResponsibilities(ctx, connect.NewRequest(req))
			}
		case "heartbeat-instructions":
			if method == "GET" {
				resp, err = client.GetHeartbeatInstructions(ctx, connect.NewRequest(read))
			} else {
				resp, err = client.SetHeartbeatInstructions(ctx, connect.NewRequest(req))
			}
		case "context":
			resp, err = client.GetMemberContext(ctx, connect.NewRequest(read))
		case "handoff":
			if method == "GET" {
				resp, err = client.GetLastHandoff(ctx, connect.NewRequest(read))
			} else {
				resp, err = client.ClearLastHandoff(ctx, connect.NewRequest(req))
			}
		}
	case len(s) == 3 && s[2] == "handoff-history":
		if method == "GET" {
			resp, err = client.GetHandoffHistory(ctx, connect.NewRequest(&heartbeatv1.TeamQueryRequest{TeamId: team, Query: q}))
		} else {
			resp, err = client.ClearHandoffHistory(ctx, connect.NewRequest(&heartbeatv1.TeamMutationRequest{TeamId: team, Body: body, Query: q}))
		}
	case len(s) == 3 && s[2] == "tasks":
		if method == "GET" {
			resp, err = client.GetTaskBoard(ctx, connect.NewRequest(&heartbeatv1.TeamQueryRequest{TeamId: team, Query: q}))
		} else {
			resp, err = client.AddTask(ctx, connect.NewRequest(&heartbeatv1.TeamMutationRequest{TeamId: team, Body: body, Query: q}))
		}
	case len(s) == 4 && s[2] == "tasks":
		req := &heartbeatv1.TaskMutationRequest{TeamId: team, TaskId: s[3], Body: body}
		if method == "DELETE" {
			resp, err = client.DeleteTask(ctx, connect.NewRequest(req))
		} else {
			resp, err = client.UpdateTask(ctx, connect.NewRequest(req))
		}
	case len(s) == 4 && s[2] == "bugs" && s[3] == "capture" && method == "POST":
		resp, err = client.CaptureBug(ctx, connect.NewRequest(&heartbeatv1.TeamMutationRequest{TeamId: team, Body: body, Query: q}))
	case len(s) == 5 && s[2] == "bugs" && s[4] == "capture":
		resp, err = client.RepairBug(ctx, connect.NewRequest(&heartbeatv1.BugMutationRequest{TeamId: team, DraftId: s[3], Body: body}))
	case len(s) == 3 && s[2] == "retention" && method == "GET":
		resp, err = client.GetRetention(ctx, connect.NewRequest(&heartbeatv1.TeamRequest{TeamId: team}))
	case len(s) == 3 && s[2] == "prune" && method == "POST":
		resp, err = client.PruneSharedState(ctx, connect.NewRequest(&heartbeatv1.TeamMutationRequest{TeamId: team, Body: body, Query: q}))
	case len(s) == 3 && s[2] == "prompt-matrix" && method == "GET":
		resp, err = client.PreviewPromptMatrix(ctx, connect.NewRequest(&heartbeatv1.TeamQueryRequest{TeamId: team, Query: q}))
	default:
		return nil, false, nil
	}
	return rpcBody(resp, "data", true, err)
}

func jsonValue(payload any) (*structpb.Value, error) {
	if payload == nil {
		return nil, nil
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, err
	}
	return structpb.NewValue(decoded)
}

func queryMap(values url.Values) map[string]string {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]string, len(values))
	for key, items := range values {
		if len(items) > 0 {
			out[key] = items[0]
		}
	}
	return out
}

func callAgents(ctx context.Context, client agentsconnect.AgentsServiceClient, method string, s []string, payload any) ([]byte, bool, error) {
	switch {
	case method == "GET" && len(s) == 1:
		resp, err := client.ListAgents(ctx, connect.NewRequest(&agentsv1.ListAgentsRequest{}))
		return rpcBody(resp, "agents", true, err)
	case method == "GET" && len(s) == 2:
		resp, err := client.GetAgent(ctx, connect.NewRequest(&agentsv1.GetAgentRequest{Id: s[1]}))
		return rpcBody(resp, "", true, err)
	case method == "POST" && len(s) == 1:
		agent := &agentsv1.AgentInput{}
		if err := unmarshalPayload(payload, agent); err != nil {
			return nil, true, err
		}
		resp, err := client.CreateAgent(ctx, connect.NewRequest(&agentsv1.CreateAgentRequest{Agent: agent}))
		return rpcBody(resp, "", true, err)
	case method == "PUT" && len(s) == 2:
		agent := &agentsv1.AgentInput{}
		if err := unmarshalPayload(payload, agent); err != nil {
			return nil, true, err
		}
		resp, err := client.UpdateAgent(ctx, connect.NewRequest(&agentsv1.UpdateAgentRequest{Id: s[1], Agent: agent, UpdateMask: payloadMask(payload)}))
		return rpcBody(resp, "", true, err)
	case method == "DELETE" && len(s) == 2:
		resp, err := client.DeleteAgent(ctx, connect.NewRequest(&agentsv1.DeleteAgentRequest{Id: s[1]}))
		return rpcBody(resp, "", true, err)
	case method == "GET" && len(s) == 3 && s[2] == "teams":
		resp, err := client.ListAgentTeams(ctx, connect.NewRequest(&agentsv1.ListAgentTeamsRequest{Id: s[1]}))
		return rpcBody(resp, "", true, err)
	case method == "GET" && len(s) == 3 && s[2] == "soul":
		resp, err := client.ManageSoul(ctx, connect.NewRequest(&agentsv1.ManageSoulRequest{Id: s[1]}))
		return rpcBody(resp, "", true, err)
	case method == "PUT" && len(s) == 3 && s[2] == "soul":
		var body struct {
			Content string `json:"content"`
		}
		if err := decodePayload(payload, &body); err != nil {
			return nil, true, err
		}
		resp, err := client.ManageSoul(ctx, connect.NewRequest(&agentsv1.ManageSoulRequest{Id: s[1], Content: &body.Content}))
		return rpcBody(resp, "", true, err)
	default:
		return nil, false, nil
	}
}

// callLegacyMembers preserves the historical avatar-oriented member command
// while routing it through the canonical AgentsService. The API's old member
// REST adapter is retired; this compatibility projection now lives at the CLI
// edge where its legacy field names are consumed.
func callLegacyMembers(ctx context.Context, client agentsconnect.AgentsServiceClient, method string, s []string, payload any) ([]byte, bool, error) {
	type memberPayload struct {
		ID          string  `json:"id"`
		Name        *string `json:"name"`
		BodyColor   *string `json:"bodyColor"`
		HeadColor   *string `json:"headColor"`
		AccentColor *string `json:"accentColor"`
	}
	toLegacy := func(agent *agentsv1.Agent) map[string]any {
		appearance := agent.GetAppearance()
		return map[string]any{
			"id": agent.GetId(), "name": agent.GetDisplayName(),
			"bodyColor": appearance.GetBody(), "headColor": appearance.GetHead(), "accentColor": appearance.GetAccent(),
			"createdAt": agent.GetCreatedAt(), "updatedAt": agent.GetUpdatedAt(),
		}
	}
	switch {
	case method == "GET" && len(s) == 1:
		resp, err := client.ListAgents(ctx, connect.NewRequest(&agentsv1.ListAgentsRequest{}))
		if err != nil {
			return nil, true, err
		}
		members := make([]map[string]any, 0, len(resp.Msg.GetAgents()))
		for _, agent := range resp.Msg.GetAgents() {
			members = append(members, toLegacy(agent))
		}
		body, err := json.Marshal(members)
		return body, true, err
	case method == "GET" && len(s) == 2:
		resp, err := client.GetAgent(ctx, connect.NewRequest(&agentsv1.GetAgentRequest{Id: s[1]}))
		if err != nil {
			return nil, true, err
		}
		body, err := json.Marshal(toLegacy(resp.Msg))
		return body, true, err
	case method == "POST" && len(s) == 1:
		var legacy memberPayload
		if err := decodePayload(payload, &legacy); err != nil {
			return nil, true, err
		}
		agent := &agentsv1.AgentInput{Id: legacy.ID, Appearance: &agentsv1.Appearance{}}
		applyLegacyMemberPayload(agent, legacy.Name, legacy.BodyColor, legacy.HeadColor, legacy.AccentColor)
		resp, err := client.CreateAgent(ctx, connect.NewRequest(&agentsv1.CreateAgentRequest{Agent: agent}))
		if err != nil {
			return nil, true, err
		}
		body, err := json.Marshal(toLegacy(resp.Msg))
		return body, true, err
	case method == "PUT" && len(s) == 2:
		var legacy memberPayload
		if err := decodePayload(payload, &legacy); err != nil {
			return nil, true, err
		}
		agent := &agentsv1.AgentInput{Appearance: &agentsv1.Appearance{}}
		applyLegacyMemberPayload(agent, legacy.Name, legacy.BodyColor, legacy.HeadColor, legacy.AccentColor)
		mask := &fieldmaskpb.FieldMask{}
		if legacy.Name != nil {
			mask.Paths = append(mask.Paths, "display_name")
		}
		if legacy.BodyColor != nil {
			mask.Paths = append(mask.Paths, "appearance.body")
		}
		if legacy.HeadColor != nil {
			mask.Paths = append(mask.Paths, "appearance.head")
		}
		if legacy.AccentColor != nil {
			mask.Paths = append(mask.Paths, "appearance.accent")
		}
		resp, err := client.UpdateAgent(ctx, connect.NewRequest(&agentsv1.UpdateAgentRequest{Id: s[1], Agent: agent, UpdateMask: mask}))
		if err != nil {
			return nil, true, err
		}
		body, err := json.Marshal(toLegacy(resp.Msg))
		return body, true, err
	case method == "DELETE" && len(s) == 2:
		resp, err := client.DeleteAgent(ctx, connect.NewRequest(&agentsv1.DeleteAgentRequest{Id: s[1]}))
		return rpcBody(resp, "", true, err)
	default:
		return nil, false, nil
	}
}

func applyLegacyMemberPayload(agent *agentsv1.AgentInput, name, body, head, accent *string) {
	if name != nil {
		agent.DisplayName = *name
	}
	if body != nil {
		agent.Appearance.Body = *body
	}
	if head != nil {
		agent.Appearance.Head = *head
	}
	if accent != nil {
		agent.Appearance.Accent = *accent
	}
}

func callTeams(ctx context.Context, client teamsconnect.TeamsServiceClient, method string, s []string, query url.Values, payload any) ([]byte, bool, error) {
	switch {
	case method == "GET" && len(s) == 1:
		resp, err := client.ListTeams(ctx, connect.NewRequest(&teamsv1.ListTeamsRequest{}))
		return rpcBody(resp, "teams", true, err)
	case method == "GET" && len(s) == 2:
		resp, err := client.GetTeam(ctx, connect.NewRequest(&teamsv1.GetTeamRequest{Id: s[1]}))
		return rpcBody(resp, "", true, err)
	case method == "POST" && len(s) == 1:
		team := &teamsv1.TeamInput{}
		if err := unmarshalPayload(payload, team); err != nil {
			return nil, true, err
		}
		resp, err := client.CreateTeam(ctx, connect.NewRequest(&teamsv1.CreateTeamRequest{Team: team}))
		return rpcBody(resp, "", true, err)
	case method == "PUT" && len(s) == 2:
		team := &teamsv1.TeamInput{}
		if err := unmarshalPayload(payload, team); err != nil {
			return nil, true, err
		}
		resp, err := client.UpdateTeam(ctx, connect.NewRequest(&teamsv1.UpdateTeamRequest{Id: s[1], Team: team, UpdateMask: payloadMask(payload)}))
		return rpcBody(resp, "", true, err)
	case method == "DELETE" && len(s) == 2:
		resp, err := client.DeleteTeam(ctx, connect.NewRequest(&teamsv1.DeleteTeamRequest{Id: s[1]}))
		return rpcBody(resp, "", true, err)
	case method == "GET" && len(s) == 3 && s[2] == "knowledge":
		last := int32(0)
		if value := query.Get("last"); value != "" {
			if parsed, err := strconv.Atoi(value); err == nil {
				last = int32(parsed)
			}
		}
		resp, err := client.ListKnowledge(ctx, connect.NewRequest(&teamsv1.ListKnowledgeRequest{TeamId: s[1], Topic: query.Get("topic"), TopicPrefix: query.Get("topic_prefix"), Last: last}))
		return rpcBody(resp, "", true, err)
	case method == "POST" && len(s) == 3 && s[2] == "knowledge":
		var body struct {
			Topic      string `json:"topic"`
			Content    string `json:"content"`
			CallerNote string `json:"caller_note"`
			Source     string `json:"source"`
			Supersedes string `json:"supersedes"`
		}
		if err := decodePayload(payload, &body); err != nil {
			return nil, true, err
		}
		resp, err := client.AddKnowledge(ctx, connect.NewRequest(&teamsv1.AddKnowledgeRequest{TeamId: s[1], Topic: body.Topic, Content: body.Content, CallerNote: body.CallerNote, Source: body.Source, Supersedes: body.Supersedes}))
		return rpcBody(resp, "", true, err)
	case method == "PUT" && len(s) == 4 && s[2] == "knowledge":
		var body struct {
			Topic      *string `json:"topic"`
			Content    *string `json:"content"`
			Source     *string `json:"source"`
			Supersedes *string `json:"supersedes"`
		}
		if err := decodePayload(payload, &body); err != nil {
			return nil, true, err
		}
		resp, err := client.UpdateKnowledge(ctx, connect.NewRequest(&teamsv1.UpdateKnowledgeRequest{TeamId: s[1], KnowledgeId: s[3], Topic: body.Topic, Content: body.Content, Source: body.Source, Supersedes: body.Supersedes}))
		return rpcBody(resp, "", true, err)
	case method == "DELETE" && len(s) == 4 && s[2] == "knowledge":
		resp, err := client.DeleteKnowledge(ctx, connect.NewRequest(&teamsv1.DeleteKnowledgeRequest{TeamId: s[1], KnowledgeId: s[3]}))
		return rpcBody(resp, "", true, err)
	case method == "POST" && len(s) == 3 && s[2] == "members":
		var body struct {
			AgentID string   `json:"agentId"`
			Roles   []string `json:"roles"`
		}
		if err := decodePayload(payload, &body); err != nil {
			return nil, true, err
		}
		resp, err := client.AddMember(ctx, connect.NewRequest(&teamsv1.AddMemberRequest{TeamId: s[1], AgentId: body.AgentID, Roles: body.Roles}))
		return rpcBody(resp, "", true, err)
	case method == "PUT" && len(s) == 4 && s[2] == "members":
		var body struct {
			Roles  []string `json:"roles"`
			Status *string  `json:"status"`
		}
		if err := decodePayload(payload, &body); err != nil {
			return nil, true, err
		}
		resp, err := client.UpdateMember(ctx, connect.NewRequest(&teamsv1.UpdateMemberRequest{TeamId: s[1], AgentId: s[3], Roles: body.Roles, Status: body.Status}))
		return rpcBody(resp, "", true, err)
	case method == "DELETE" && len(s) == 4 && s[2] == "members":
		resp, err := client.RemoveMember(ctx, connect.NewRequest(&teamsv1.RemoveMemberRequest{TeamId: s[1], AgentId: s[3]}))
		return rpcBody(resp, "", true, err)
	case method == "GET" && len(s) == 3 && s[2] == "roles":
		resp, err := client.GetRoles(ctx, connect.NewRequest(&teamsv1.GetRolesRequest{TeamId: s[1]}))
		return rpcBody(resp, "roles", true, err)
	case method == "GET" && len(s) == 3 && s[2] == "org":
		resp, err := client.GetOrgChart(ctx, connect.NewRequest(&teamsv1.GetOrgChartRequest{TeamId: s[1]}))
		return rpcBody(resp, "", true, err)
	case method == "PUT" && len(s) == 5 && s[2] == "org" && s[3] == "edges":
		var body struct {
			ManagerAgentID string `json:"managerAgentId"`
		}
		if err := decodePayload(payload, &body); err != nil {
			return nil, true, err
		}
		resp, err := client.UpdateOrgChartEdge(ctx, connect.NewRequest(&teamsv1.UpdateOrgChartEdgeRequest{TeamId: s[1], ReportAgentId: s[4], ManagerAgentId: body.ManagerAgentID}))
		return rpcBody(resp, "", true, err)
	case method == "DELETE" && len(s) == 5 && s[2] == "org" && s[3] == "edges":
		resp, err := client.DeleteOrgChartEdge(ctx, connect.NewRequest(&teamsv1.DeleteOrgChartEdgeRequest{TeamId: s[1], ReportAgentId: s[4]}))
		return rpcBody(resp, "", true, err)
	case method == "GET" && len(s) == 5 && s[2] == "members" && s[4] == "messages":
		resp, err := client.ListMessages(ctx, connect.NewRequest(&teamsv1.ListMessagesRequest{TeamId: s[1], AgentId: s[3]}))
		return rpcBody(resp, "", true, err)
	case method == "POST" && len(s) == 5 && s[2] == "members" && s[4] == "messages":
		var body struct {
			FromAgentID string `json:"fromAgentId"`
			Content     string `json:"content"`
		}
		if err := decodePayload(payload, &body); err != nil {
			return nil, true, err
		}
		resp, err := client.SendMessage(ctx, connect.NewRequest(&teamsv1.SendMessageRequest{TeamId: s[1], AgentId: s[3], FromAgentId: body.FromAgentID, Content: body.Content}))
		return rpcBody(resp, "", true, err)
	case method == "DELETE" && len(s) == 6 && s[2] == "members" && s[4] == "messages":
		resp, err := client.DeleteMessage(ctx, connect.NewRequest(&teamsv1.DeleteMessageRequest{TeamId: s[1], AgentId: s[3], MessageId: s[5]}))
		return rpcBody(resp, "", true, err)
	case method == "DELETE" && len(s) == 5 && s[2] == "members" && s[4] == "messages":
		resp, err := client.ClearMessages(ctx, connect.NewRequest(&teamsv1.ClearMessagesRequest{TeamId: s[1], AgentId: s[3]}))
		return rpcBody(resp, "", true, err)
	case method == "POST" && len(s) == 3 && s[1] == "import" && s[2] == "claude-code":
		var body struct {
			TeamName string `json:"teamName"`
		}
		if err := decodePayload(payload, &body); err != nil {
			return nil, true, err
		}
		resp, err := client.ImportClaudeCodeTeam(ctx, connect.NewRequest(&teamsv1.ImportClaudeCodeTeamRequest{TeamName: body.TeamName}))
		return rpcBody(resp, "", true, err)
	case method == "GET" && len(s) == 4 && s[2] == "export" && s[3] == "claude-code":
		resp, err := client.ExportClaudeCodeTeam(ctx, connect.NewRequest(&teamsv1.ExportClaudeCodeTeamRequest{TeamId: s[1]}))
		return rpcBody(resp, "export", true, err)
	default:
		return nil, false, nil
	}
}

func callTopics(ctx context.Context, client topicsconnect.TopicsServiceClient, method string, s []string, payload any) ([]byte, bool, error) {
	switch {
	case method == "GET" && len(s) == 2 && s[1] == "tree":
		resp, err := client.ListTopicTree(ctx, connect.NewRequest(&topicsv1.ListTopicTreeRequest{}))
		return rpcBody(resp, "topics", true, err)
	case method == "GET" && len(s) == 1:
		resp, err := client.ListTopics(ctx, connect.NewRequest(&topicsv1.ListTopicsRequest{}))
		return rpcBody(resp, "topics", true, err)
	case method == "GET" && len(s) == 2:
		resp, err := client.GetTopic(ctx, connect.NewRequest(&topicsv1.GetTopicRequest{Id: s[1]}))
		return rpcBody(resp, "", true, err)
	case method == "POST" && len(s) == 1:
		topic := &topicsv1.TopicInput{}
		if err := unmarshalPayload(payload, topic); err != nil {
			return nil, true, err
		}
		resp, err := client.CreateTopic(ctx, connect.NewRequest(&topicsv1.CreateTopicRequest{Topic: topic}))
		return rpcBody(resp, "", true, err)
	case method == "PUT" && len(s) == 2:
		topic := &topicsv1.TopicInput{}
		if err := unmarshalPayload(payload, topic); err != nil {
			return nil, true, err
		}
		resp, err := client.UpdateTopic(ctx, connect.NewRequest(&topicsv1.UpdateTopicRequest{Id: s[1], Topic: topic, UpdateMask: payloadMask(payload)}))
		return rpcBody(resp, "", true, err)
	case method == "DELETE" && len(s) == 2:
		resp, err := client.DeleteTopic(ctx, connect.NewRequest(&topicsv1.DeleteTopicRequest{Id: s[1]}))
		return rpcBody(resp, "", true, err)
	case method == "GET" && len(s) == 3 && s[2] == "skills":
		resp, err := client.GetAccumulatedSkills(ctx, connect.NewRequest(&topicsv1.GetAccumulatedSkillsRequest{Id: s[1]}))
		return rpcBody(resp, "", true, err)
	case method == "POST" && len(s) == 2 && s[1] == "match":
		req := &topicsv1.MatchTopicsRequest{}
		if err := unmarshalPayload(payload, req); err != nil {
			return nil, true, err
		}
		resp, err := client.MatchTopics(ctx, connect.NewRequest(req))
		return rpcBody(resp, "", true, err)
	default:
		return nil, false, nil
	}
}

func callTesting(ctx context.Context, client testingconnect.TestingServiceClient, method string, s []string, query url.Values, payload any) ([]byte, bool, error) {
	switch {
	case method == "POST" && s[2] == "test":
		req := &testingv1.RunSkillTestRequest{SkillId: s[1]}
		if err := unmarshalPayload(payload, req); err != nil {
			return nil, true, err
		}
		req.SkillId = s[1]
		resp, err := client.RunSkillTest(ctx, connect.NewRequest(req))
		return rpcBody(resp, "", true, err)
	case method == "GET" && s[2] == "test-history":
		limit, _ := strconv.ParseInt(query.Get("limit"), 10, 32)
		resp, err := client.ListSkillTestHistory(ctx, connect.NewRequest(&testingv1.ListSkillTestHistoryRequest{SkillId: s[1], Limit: int32(limit)}))
		return rpcBody(resp, "results", true, err)
	default:
		return nil, false, nil
	}
}

func callMetadata(ctx context.Context, client metadataconnect.MetadataServiceClient, method string, query url.Values) ([]byte, bool, error) {
	if method != "GET" {
		return nil, false, nil
	}
	resp, err := client.FetchOpenGraph(ctx, connect.NewRequest(&metadatav1.FetchOpenGraphRequest{Url: query.Get("url")}))
	return rpcBody(resp, "", true, err)
}

func payloadMask(payload any) *fieldmaskpb.FieldMask {
	raw, _ := json.Marshal(payload)
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(raw, &fields)
	paths := make([]string, 0, len(fields))
	for key := range fields {
		paths = append(paths, key)
	}
	return &fieldmaskpb.FieldMask{Paths: paths}
}

func decodePayload(payload any, target any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, target)
}

func callSearch(ctx context.Context, textClient searchconnect.SearchServiceClient, aiClient aisearchconnect.AISearchServiceClient, method string, s []string, query url.Values, payload any) ([]byte, bool, error) {
	switch {
	case method == "GET" && len(s) == 2 && s[1] == "skills":
		resp, err := textClient.SearchSkills(ctx, connect.NewRequest(&searchv1.SearchSkillsRequest{Query: query.Get("q"), Tag: query.Get("tag"), Folder: query.Get("folder")}))
		return rpcBody(resp, "", true, err)
	case method == "GET" && len(s) == 3 && s[1] == "skills" && s[2] == "content":
		limit, _ := strconv.ParseInt(query.Get("limit"), 10, 32)
		resp, err := textClient.SearchSkillContent(ctx, connect.NewRequest(&searchv1.SearchSkillContentRequest{Query: query.Get("q"), Tags: query["tag"], Folders: query["folder"], CaseSensitive: query.Get("caseSensitive") == "true", WholeWord: query.Get("wholeWord") == "true", Regex: query.Get("regex") == "true", Limit: int32(limit)}))
		return rpcBody(resp, "", true, err)
	case method == "GET" && len(s) == 2 && s[1] == "agents":
		resp, err := textClient.SearchAgents(ctx, connect.NewRequest(&searchv1.SearchAgentsRequest{Query: query.Get("q"), Tag: query.Get("tag"), Status: query.Get("status")}))
		return rpcBody(resp, "", true, err)
	case method == "GET" && len(s) == 3 && s[1] == "agents" && s[2] == "content":
		limit, _ := strconv.ParseInt(query.Get("limit"), 10, 32)
		resp, err := textClient.SearchAgentContent(ctx, connect.NewRequest(&searchv1.SearchAgentContentRequest{Query: query.Get("q"), Tags: query["tag"], CaseSensitive: query.Get("caseSensitive") == "true", WholeWord: query.Get("wholeWord") == "true", Regex: query.Get("regex") == "true", Limit: int32(limit)}))
		return rpcBody(resp, "", true, err)
	case method == "GET" && len(s) == 2 && s[1] == "teams":
		req := &searchv1.SearchTeamsRequest{Query: query.Get("q")}
		if raw := query.Get("enabled"); raw != "" {
			value := raw == "true"
			req.Enabled = &value
		}
		resp, err := textClient.SearchTeams(ctx, connect.NewRequest(req))
		return rpcBody(resp, "", true, err)
	case method == "GET" && len(s) == 3 && s[1] == "teams" && s[2] == "content":
		limit, _ := strconv.ParseInt(query.Get("limit"), 10, 32)
		resp, err := textClient.SearchTeamContent(ctx, connect.NewRequest(&searchv1.SearchTeamContentRequest{Query: query.Get("q"), CaseSensitive: query.Get("caseSensitive") == "true", WholeWord: query.Get("wholeWord") == "true", Regex: query.Get("regex") == "true", Limit: int32(limit)}))
		return rpcBody(resp, "", true, err)
	case method == "POST" && len(s) == 2 && s[1] == "ai":
		req := &aisearchv1.SearchSkillsRequest{}
		if err := unmarshalPayload(payload, req); err != nil {
			return nil, true, err
		}
		resp, err := aiClient.SearchSkills(ctx, connect.NewRequest(req))
		return rpcBody(resp, "", true, err)
	case method == "POST" && len(s) == 3 && s[1] == "agents" && s[2] == "ai":
		req := &aisearchv1.SearchAgentsRequest{}
		if err := unmarshalPayload(payload, req); err != nil {
			return nil, true, err
		}
		resp, err := aiClient.SearchAgents(ctx, connect.NewRequest(req))
		return rpcBody(resp, "", true, err)
	case method == "POST" && len(s) == 3 && s[1] == "actions" && s[2] == "ai":
		req := &aisearchv1.SearchActionsRequest{}
		if err := unmarshalPayload(payload, req); err != nil {
			return nil, true, err
		}
		resp, err := aiClient.SearchActions(ctx, connect.NewRequest(req))
		return rpcBody(resp, "", true, err)
	case method == "POST" && len(s) == 3 && s[1] == "teams" && s[2] == "ai":
		req := &aisearchv1.SearchTeamsRequest{}
		if err := unmarshalPayload(payload, req); err != nil {
			return nil, true, err
		}
		resp, err := aiClient.SearchTeams(ctx, connect.NewRequest(req))
		return rpcBody(resp, "", true, err)
	case method == "GET" && len(s) == 3 && s[1] == "ai" && s[2] == "status":
		resp, err := aiClient.GetStatus(ctx, connect.NewRequest(&aisearchv1.GetStatusRequest{}))
		return rpcBody(resp, "", true, err)
	case method == "POST" && len(s) == 3 && s[1] == "ai" && s[2] == "reconcile":
		req := &aisearchv1.ReconcileRequest{Collection: query.Get("collection"), DryRun: query.Get("dry_run") == "1" || query.Get("dry_run") == "true"}
		resp, err := aiClient.Reconcile(ctx, connect.NewRequest(req))
		if req.GetDryRun() {
			body, handled, err := rpcBody(resp, "", true, err)
			if err != nil {
				return nil, handled, err
			}
			body, err = renameJSONField(body, "dryRun", "dry_run")
			return body, handled, err
		}
		return rpcBody(resp, "status", true, err)
	case method == "GET" && len(s) == 4 && s[1] == "ai" && s[2] == "reconcile" && s[3] == "status":
		resp, err := aiClient.GetReconcileStatus(ctx, connect.NewRequest(&aisearchv1.GetReconcileStatusRequest{}))
		return rpcBody(resp, "", true, err)
	case method == "POST" && len(s) == 4 && s[1] == "ai" && s[2] == "reconcile" && s[3] == "cancel":
		resp, err := aiClient.CancelReconcile(ctx, connect.NewRequest(&aisearchv1.CancelReconcileRequest{}))
		return rpcBody(resp, "", true, err)
	default:
		return nil, false, nil
	}
}

func callDiscovery(ctx context.Context, client discoveryconnect.DiscoveryServiceClient, method string, s []string, query url.Values, payload any) ([]byte, bool, error) {
	switch {
	case method == "POST" && len(s) == 1 && s[0] == "discover":
		req := &discoveryv1.DiscoverRequest{}
		if err := unmarshalPayload(payload, req); err != nil {
			return nil, true, err
		}
		resp, err := client.Discover(ctx, connect.NewRequest(req))
		return rpcBody(resp, "", true, err)
	case method == "GET" && len(s) == 1 && s[0] == "discovery-gaps":
		resp, err := client.ListDiscoveryGaps(ctx, connect.NewRequest(&discoveryv1.ListDiscoveryGapsRequest{Since: query.Get("since"), Type: query.Get("type")}))
		return rpcBody(resp, "", true, err)
	case method == "GET" && len(s) == 1 && s[0] == "discovery-metrics":
		resp, err := client.GetDiscoveryMetrics(ctx, connect.NewRequest(&discoveryv1.GetDiscoveryMetricsRequest{Since: query.Get("since"), Type: query.Get("type")}))
		return rpcBody(resp, "", true, err)
	case method == "GET" && len(s) == 1 && s[0] == "skill-usage":
		resp, err := client.GetSkillUsage(ctx, connect.NewRequest(&discoveryv1.GetSkillUsageRequest{Since: query.Get("since")}))
		return rpcBody(resp, "", true, err)
	default:
		return nil, false, nil
	}
}

func callSkills(ctx context.Context, client skillsconnect.SkillsServiceClient, method string, s []string, query url.Values, payload any) ([]byte, bool, error) {
	switch {
	case method == "GET" && len(s) == 1:
		resp, err := client.ListSkills(ctx, connect.NewRequest(&skillsv1.ListSkillsRequest{Folder: query.Get("folder"), Tag: query.Get("tag"), Modes: splitCSV(query.Get("modes")), WithoutProgrammaticHome: query.Get("withoutProgrammaticHome") == "true"}))
		return rpcBody(resp, "skills", true, err)
	case method == "GET" && len(s) == 2 && s[1] == "sync":
		resp, err := client.SyncSkills(ctx, connect.NewRequest(&skillsv1.SyncSkillsRequest{}))
		return rpcBody(resp, "", true, err)
	case method == "GET" && len(s) == 2:
		resp, err := client.GetSkill(ctx, connect.NewRequest(&skillsv1.GetSkillRequest{Id: s[1]}))
		return rpcBody(resp, "skill", true, err)
	case method == "GET" && len(s) == 3 && s[2] == "versions":
		resp, err := client.ListSkillVersions(ctx, connect.NewRequest(&skillsv1.ListSkillVersionsRequest{Id: s[1]}))
		return rpcBody(resp, "", true, err)
	case method == "GET" && len(s) == 3 && s[2] == "variants":
		resp, err := client.ListSkillVariants(ctx, connect.NewRequest(&skillsv1.ListSkillVariantsRequest{SkillId: s[1]}))
		return rpcBody(resp, "variants", true, err)
	case method == "POST" && len(s) == 2 && s[1] == "read":
		req := &skillsv1.ReadSkillsRequest{}
		if err := unmarshalPayload(payload, req); err != nil {
			return nil, true, err
		}
		resp, err := client.ReadSkills(ctx, connect.NewRequest(req))
		return rpcBody(resp, "", true, err)
	case method == "POST" && len(s) == 2 && s[1] == "import":
		req := &skillsv1.ImportSkillRequest{}
		if err := unmarshalPayload(payload, req); err != nil {
			return nil, true, err
		}
		resp, err := client.ImportSkill(ctx, connect.NewRequest(req))
		return rpcBody(resp, "", true, err)
	case method == "POST" && len(s) == 3 && s[2] == "review":
		req := &skillsv1.ReviewImportedSkillRequest{Id: s[1]}
		if err := unmarshalPayload(payload, req); err != nil {
			return nil, true, err
		}
		req.Id = s[1]
		resp, err := client.ReviewImportedSkill(ctx, connect.NewRequest(req))
		return rpcBody(resp, "", true, err)
	case method == "POST" && len(s) == 3 && s[2] == "staleness":
		req := &skillsv1.ReportImportedSkillStalenessRequest{Id: s[1]}
		if err := unmarshalPayload(payload, req); err != nil {
			return nil, true, err
		}
		req.Id = s[1]
		resp, err := client.ReportImportedSkillStaleness(ctx, connect.NewRequest(req))
		return rpcBody(resp, "", true, err)
	case method == "POST" && len(s) == 1:
		req := &skillsv1.CreateSkillRequest{}
		if err := unmarshalPayload(payload, req); err != nil {
			return nil, true, err
		}
		resp, err := client.CreateSkill(ctx, connect.NewRequest(req))
		return rpcBody(resp, "skill", true, err)
	case method == "POST" && len(s) == 4 && s[2] == "revert":
		version, err := strconv.ParseInt(s[3], 10, 32)
		if err != nil {
			return nil, true, fmt.Errorf("invalid skill version %q: %w", s[3], err)
		}
		resp, err := client.RevertSkill(ctx, connect.NewRequest(&skillsv1.RevertSkillRequest{Id: s[1], Version: int32(version)}))
		return rpcBody(resp, "", true, err)
	case method == "POST" && len(s) == 3 && s[2] == "variants":
		req := &skillsv1.CreateSkillVariantRequest{SkillId: s[1]}
		if err := unmarshalPayload(payload, req); err != nil {
			return nil, true, err
		}
		req.SkillId = s[1]
		resp, err := client.CreateSkillVariant(ctx, connect.NewRequest(req))
		return rpcBody(resp, "variant", true, err)
	case method == "POST" && len(s) == 3 && s[2] == "use":
		resp, err := client.RecordSkillUsage(ctx, connect.NewRequest(&skillsv1.RecordSkillUsageRequest{Id: s[1]}))
		return rpcBody(resp, "", true, err)
	case method == "PUT" && len(s) == 2:
		req := &skillsv1.UpdateSkillRequest{Id: s[1]}
		raw, err := json.Marshal(payload)
		if err != nil {
			return nil, true, err
		}
		var presence map[string]json.RawMessage
		if err := json.Unmarshal(raw, &presence); err != nil {
			return nil, true, err
		}
		if err := protojson.Unmarshal(raw, req); err != nil {
			return nil, true, err
		}
		req.Id = s[1]
		_, req.ReplaceModes = presence["modes"]
		_, req.ReplaceTags = presence["tags"]
		_, req.ReplaceTargetDimensions = presence["targetDimensions"]
		resp, err := client.UpdateSkill(ctx, connect.NewRequest(req))
		return rpcBody(resp, "skill", true, err)
	case method == "PUT" && len(s) == 3 && s[2] == "rating":
		req := &skillsv1.RateSkillRequest{Id: s[1]}
		if err := unmarshalPayload(payload, req); err != nil {
			return nil, true, err
		}
		req.Id = s[1]
		resp, err := client.RateSkill(ctx, connect.NewRequest(req))
		return rpcBody(resp, "", true, err)
	case method == "DELETE" && len(s) == 2:
		resp, err := client.DeleteSkill(ctx, connect.NewRequest(&skillsv1.DeleteSkillRequest{Id: s[1]}))
		return rpcBody(resp, "", true, err)
	case method == "DELETE" && len(s) == 4 && s[2] == "variants":
		resp, err := client.DeleteSkillVariant(ctx, connect.NewRequest(&skillsv1.DeleteSkillVariantRequest{SkillId: s[1], VariantId: s[3]}))
		return rpcBody(resp, "", true, err)
	default:
		return nil, false, nil
	}
}

func callActions(ctx context.Context, client actionsconnect.ActionsServiceClient, method string, s []string, query url.Values, payload any) ([]byte, bool, error) {
	switch {
	case method == "GET" && len(s) == 1:
		resp, err := client.ListActions(ctx, connect.NewRequest(&actionsv1.ListActionsRequest{Pack: query.Get("pack"), Status: query.Get("status"), Owner: query.Get("owner"), Tag: query.Get("tag")}))
		return rpcBody(resp, "actions", true, err)
	case method == "GET" && len(s) == 2:
		resp, err := client.GetAction(ctx, connect.NewRequest(&actionsv1.GetActionRequest{Id: s[1]}))
		return rpcBody(resp, "action", true, err)
	case method == "POST" && len(s) == 2 && s[1] == "preview":
		req := &actionsv1.AuthorActionRequest{}
		if err := unmarshalPayload(payload, req); err != nil {
			return nil, true, err
		}
		resp, err := client.AuthorAction(ctx, connect.NewRequest(req))
		return rpcBody(resp, "", true, err)
	case method == "POST" && len(s) == 1:
		raw, err := json.Marshal(payload)
		if err != nil {
			return nil, true, err
		}
		contract := &actionsv1.Action{}
		if err := protojson.Unmarshal(raw, contract); err != nil {
			return nil, true, err
		}
		var fields map[string]json.RawMessage
		_ = json.Unmarshal(raw, &fields)
		var pack string
		_ = json.Unmarshal(fields["pack"], &pack)
		resp, err := client.AuthorAction(ctx, connect.NewRequest(&actionsv1.AuthorActionRequest{Contract: contract, Pack: pack, Apply: true}))
		if err != nil {
			return nil, true, err
		}
		body, err := marshalFields(resp.Msg, map[string]string{"action": "rendered", "validation": "validation"})
		return body, true, err
	case method == "POST" && len(s) == 3 && s[2] == "validate":
		resp, err := client.ValidateAction(ctx, connect.NewRequest(&actionsv1.ValidateActionRequest{Id: s[1]}))
		return rpcBody(resp, "validation", true, err)
	case method == "POST" && len(s) == 3 && s[2] == "run":
		req := &actionsv1.RunActionRequest{Id: s[1]}
		if err := unmarshalPayload(payload, req); err != nil {
			return nil, true, err
		}
		req.Id = s[1]
		resp, err := client.RunAction(ctx, connect.NewRequest(req))
		body, handled, err := rpcBody(resp, "", true, err)
		if err != nil {
			return nil, handled, err
		}
		body, err = normalizeInt64JSONField(body, "durationMs")
		return body, handled, err
	case method == "PUT" && len(s) == 2:
		action := &actionsv1.Action{}
		if err := unmarshalPayload(payload, action); err != nil {
			return nil, true, err
		}
		resp, err := client.UpdateAction(ctx, connect.NewRequest(&actionsv1.UpdateActionRequest{Id: s[1], Action: action}))
		return rpcBody(resp, "", true, err)
	case method == "DELETE" && len(s) == 2:
		resp, err := client.DeleteAction(ctx, connect.NewRequest(&actionsv1.DeleteActionRequest{Id: s[1], Hard: query.Get("hard") == "true"}))
		return rpcBody(resp, "", true, err)
	default:
		return nil, false, nil
	}
}

func callTags(ctx context.Context, client tagsconnect.TagsServiceClient, method string, s []string, payload any) ([]byte, bool, error) {
	switch {
	case method == "GET" && len(s) == 1:
		resp, err := client.ListTags(ctx, connect.NewRequest(&tagsv1.ListTagsRequest{}))
		return rpcBody(resp, "tags", true, err)
	case method == "POST" && len(s) == 1:
		req := &tagsv1.CreateTagRequest{}
		if err := unmarshalPayload(payload, req); err != nil {
			return nil, true, err
		}
		resp, err := client.CreateTag(ctx, connect.NewRequest(req))
		return rpcBody(resp, "tag", true, err)
	default:
		return nil, false, nil
	}
}

func rpcBody[T any](resp *connect.Response[T], unwrap string, handled bool, err error) ([]byte, bool, error) {
	if err != nil {
		return nil, handled, cliapp.WrapAPIError("prompt-manager RPC", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, handled, fmt.Errorf("server returned an empty response")
	}
	message, ok := any(resp.Msg).(proto.Message)
	if !ok {
		return nil, handled, fmt.Errorf("generated response does not implement proto.Message")
	}
	raw, err := protojson.Marshal(message)
	if err != nil {
		return nil, handled, err
	}
	if unwrap == "" {
		return raw, handled, nil
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil {
		return nil, handled, err
	}
	value, ok := object[unwrap]
	if !ok {
		field := message.ProtoReflect().Descriptor().Fields().ByJSONName(unwrap)
		if field != nil && field.IsList() {
			return []byte("[]"), handled, nil
		}
		if field != nil && field.IsMap() {
			return []byte("{}"), handled, nil
		}
		return nil, handled, fmt.Errorf("server response omitted %s", unwrap)
	}
	return value, handled, nil
}

func unmarshalPayload(payload any, message proto.Message) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return protojson.Unmarshal(raw, message)
}

func pathSegments(path string) []string {
	path = strings.Trim(strings.SplitN(path, "?", 2)[0], "/")
	if path == "" {
		return nil
	}
	parts := strings.Split(path, "/")
	for i := range parts {
		if decoded, err := url.PathUnescape(parts[i]); err == nil {
			parts[i] = decoded
		}
	}
	return parts
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func marshalFields(message proto.Message, fields map[string]string) ([]byte, error) {
	raw, err := protojson.Marshal(message)
	if err != nil {
		return nil, err
	}
	var source map[string]json.RawMessage
	if err := json.Unmarshal(raw, &source); err != nil {
		return nil, err
	}
	target := make(map[string]json.RawMessage, len(fields))
	for targetName, sourceName := range fields {
		target[targetName] = source[sourceName]
	}
	return json.Marshal(target)
}

// normalizeInt64JSONField restores the numeric representation used by the
// pre-Connect CLI models. Protobuf JSON encodes 64-bit integers as strings to
// preserve JavaScript precision, while this bounded duration is historically
// an integer in prompt-manager's CLI contract.
func normalizeInt64JSONField(body []byte, field string) ([]byte, error) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(body, &object); err != nil {
		return nil, err
	}
	raw, ok := object[field]
	if !ok || len(raw) == 0 || raw[0] != '"' {
		return body, nil
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	if _, err := strconv.ParseInt(value, 10, 64); err != nil {
		return nil, fmt.Errorf("invalid %s: %w", field, err)
	}
	object[field] = json.RawMessage(value)
	return json.Marshal(object)
}

func renameJSONField(body []byte, source, target string) ([]byte, error) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(body, &object); err != nil {
		return nil, err
	}
	value, ok := object[source]
	if !ok {
		return body, nil
	}
	delete(object, source)
	object[target] = value
	return json.Marshal(object)
}
