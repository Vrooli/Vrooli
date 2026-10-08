package knowledgebase

import (
	"connectrpc.com/connect"
	"context"
	"encoding/json"
	kov1 "github.com/vrooli/vrooli/packages/proto/gen/go/knowledge-observatory/v1"
	"io"
	"os"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

// [REQ:KO-KB-001]
func TestManifestLoadsOnlyGovernedReadOperations(t *testing.T) {
	raw, err := os.ReadFile("../../manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := cliapp.ParseManifest(raw)
	if err != nil {
		t.Fatal(err)
	}
	group := manifest.FindGroup("knowledge-base")
	if group == nil || len(group.Commands) != 5 {
		t.Fatal("missing governed operations")
	}
	var data struct {
		Groups []struct {
			Commands []struct {
				Governance struct {
					Effect      string `json:"effect"`
					RunEligible bool   `json:"run_eligible"`
				} `json:"governance"`
			} `json:"commands"`
		} `json:"groups"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatal(err)
	}
	for _, g := range data.Groups {
		for _, c := range g.Commands {
			if c.Governance.Effect != "read" || !c.Governance.RunEligible {
				t.Fatal("unexpected mutation or unbound operation")
			}
		}
	}
	for _, command := range group.Commands {
		if command.Binding.Kind != "connect-rpc" {
			t.Fatal("ungoverned command", command.Name)
		}
	}
}

// [REQ:KO-KB-005] CLI booleans must reach the typed request.
func TestMaintenanceBooleanFlagsReachOwner(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "allow-missing", Bool: true}}}, BoolFlags: map[string]bool{"allow-missing": enabled}, Stdout: io.Discard})
		call := func(_ context.Context, r *connect.Request[kov1.InspectDocumentRequest]) (*connect.Response[kov1.InspectDocumentResponse], error) {
			if r.Msg.AllowMissing != enabled {
				t.Fatal("allow-missing lost")
			}
			return connect.NewResponse(&kov1.InspectDocumentResponse{}), nil
		}
		if err := invoke(call, func() *kov1.InspectDocumentRequest { return &kov1.InspectDocumentRequest{} }, []string{"allow-missing"})(ctx); err != nil {
			t.Fatal(err)
		}
		healthCtx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "skip-external-links", Bool: true}}}, BoolFlags: map[string]bool{"skip-external-links": enabled}, Stdout: io.Discard})
		health := func(_ context.Context, r *connect.Request[kov1.DocHealthRequest]) (*connect.Response[kov1.DocHealthResponse], error) {
			if r.Msg.GetSkipExternalLinks() != enabled {
				t.Fatal("skip-external-links lost")
			}
			return connect.NewResponse(&kov1.DocHealthResponse{}), nil
		}
		if err := invoke(health, func() *kov1.DocHealthRequest { return &kov1.DocHealthRequest{} }, []string{"skip-external-links"})(healthCtx); err != nil {
			t.Fatal(err)
		}
	}
}
