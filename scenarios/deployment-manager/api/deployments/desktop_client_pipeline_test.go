package deployments

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	pipelinev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/pipeline"
	"google.golang.org/protobuf/proto"
)

func TestRunPublishPipelineConnectRequestsDeployOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request: %v", err)
		}
		envelope := &pipelinev1.PipelineRunRequest{}
		if err := proto.Unmarshal(body, envelope); err != nil {
			t.Fatalf("decode Connect request: %v", err)
		}
		if len(envelope.GetConfig().GetStages()) != 1 || envelope.GetConfig().GetStages()[0].String() != "STAGE_NAME_DEPLOY" {
			t.Fatalf("requested stages = %v, want [STAGE_NAME_DEPLOY]", envelope.GetConfig().GetStages())
		}
		deploy := envelope.GetConfig().GetDeploy()
		if deploy == nil || deploy.GetCandidateId() != "candidate-1" || deploy.GetDestinationRevisionId() != "destination-1" || deploy.GetAuthorizationEpoch() != 7 || deploy.GetReadinessReviewKey() != "review-1" {
			t.Fatalf("release identity = %+v", deploy)
		}
		w.Header().Set("Content-Type", "application/proto")
		response, err := proto.Marshal(&pipelinev1.PipelineRunResponse{PipelineId: "pipe-1"})
		if err != nil {
			t.Fatalf("encode Connect response: %v", err)
		}
		_, _ = w.Write(response)
	}))
	defer server.Close()

	client := &DesktopPackagerClient{httpClient: server.Client(), baseURL: server.URL, log: func(string, map[string]interface{}) {}}
	response, err := client.RunPublishPipelineConnect(context.Background(), &PublishPipelineRequest{
		ScenarioName:            "qualified-app",
		Platforms:               []string{"linux-x64"},
		ArtifactDigest:          "sha256:manifest",
		ExpectedArtifactDigests: map[string]string{"linux-x64": "sha256:artifact"},
		ReleaseVersion:          "1.2.3",
		CandidateID:             "candidate-1", DestinationRevisionID: "destination-1", AuthorizationEpoch: 7, ReadinessReviewKey: "review-1",
		DeployConfig: &PublishDeployConfig{AppKey: "qualified-app", ReleaseID: "release-1", Channel: "stable", CandidateID: "candidate-1", DestinationRevisionID: "destination-1", AuthorizationEpoch: 7, ReadinessReviewKey: "review-1"},
	})
	if err != nil {
		t.Fatalf("RunPublishPipelineConnect() error = %v", err)
	}
	if response.PipelineID != "pipe-1" {
		t.Fatalf("pipeline id = %q, want pipe-1", response.PipelineID)
	}
}
