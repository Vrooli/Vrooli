package onboarding

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	setupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/setup/v1"
	selectionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/selection"
	"google.golang.org/protobuf/proto"
)

type fixedResolver struct{ url string }

func (r fixedResolver) ResolveScenarioURLDefault(context.Context, string) (string, error) {
	return r.url, nil
}

func TestClientIssuesTypedOwnerHandoffWithServiceCredential(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer onboarding-token" {
			t.Fatalf("authorization = %q", r.Header.Get("Authorization"))
		}
		response, err := proto.Marshal(&selectionv1.CreateHandoffResponse{Handoff: &selectionv1.Handoff{
			Id: "handoff-1", Reference: "vrooli-onboarding://handoffs/handoff-1", DeploymentId: "deployment-1", Target: "node-1",
			MachineId: "machine-1", NodeId: "node-1", NodeKind: "bridge", EnrollmentGeneration: 4, DesiredRevision: 9,
			SelectionDigest: "sha256:selection", State: "pending",
		}})
		if err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/proto")
		_, _ = w.Write(response)
	}))
	defer server.Close()

	client := &Client{HTTP: server.Client(), Resolver: fixedResolver{url: server.URL}, Token: "onboarding-token"}
	handoff, err := client.Issue(context.Background(), Request{
		Target: "node-1", MachineID: "machine-1", NodeID: "node-1", NodeKind: "bridge", DeploymentID: "deployment-1",
		EnrollmentGeneration: 4, DesiredRevision: 9, SelectionDigest: "sha256:selection", RequestKey: "fallback-reference", Missing: []string{"alpha:key"},
		Selection: &setupv1.Selection{SchemaVersion: "v1", Scenarios: []string{"alpha"}, Apply: true},
	})
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if handoff.GetId() != "handoff-1" || handoff.GetReference() != "vrooli-onboarding://handoffs/handoff-1" {
		t.Fatalf("handoff = %+v", handoff)
	}
}
