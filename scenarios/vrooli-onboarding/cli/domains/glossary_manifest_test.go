package domains

import (
	"bytes"
	"context"
	"net/http"
	"os"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
	glossaryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/glossary"
	glossaryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/glossary/glossaryv1connect"
)

func TestManifestGlossarySearchUsesTypedActiveBinding(t *testing.T) {
	manifest, err := os.ReadFile("../manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	service := &glossaryTestService{}
	path, handler := glossaryconnect.NewGlossaryServiceHandler(service)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	core := cliapptest.NewTestApp(t, mux)
	flat, _, err := LoadManifestGroups(core, manifest)
	if err != nil {
		t.Fatal(err)
	}
	var command *cliapp.Command
	for i := range flat {
		if flat[i].Title != "glossary" {
			continue
		}
		for j := range flat[i].Commands {
			if flat[i].Commands[j].Name == "glossary" {
				command = &flat[i].Commands[j]
			}
		}
	}
	if command == nil || command.RunCtx == nil {
		t.Fatal("manifest glossary search command is not executable")
	}
	cliapptest.AssertPrimitiveEvidence(t, *command, cliapp.PrimitiveProtoList)
	var stdout, stderr bytes.Buffer
	ctx, err := cliapp.NewTestRunContextFromArgs(command.Args, []string{"--query", "postgres", "--json"}, core, &stdout, &stderr)
	if err != nil {
		t.Fatalf("build manifest command context: %v", err)
	}
	if err := command.RunCtx(ctx); err != nil {
		t.Fatalf("manifest glossary search: %v (stderr %q)", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Postgres") || !strings.Contains(stdout.String(), "database") {
		t.Fatalf("manifest JSON output = %q, want typed glossary result", stdout.String())
	}
	if service.query != "postgres" {
		t.Fatalf("typed request query = %q, want postgres", service.query)
	}
}

type glossaryTestService struct{ query string }

func (s *glossaryTestService) SearchGlossary(_ context.Context, req *connect.Request[glossaryv1.SearchGlossaryRequest]) (*connect.Response[glossaryv1.SearchGlossaryResponse], error) {
	s.query = req.Msg.GetQuery()
	return connect.NewResponse(&glossaryv1.SearchGlossaryResponse{
		Count: 1, Query: s.query,
		Entries: []*glossaryv1.GlossaryEntry{{Term: "Postgres", Category: "resource", Description: "database"}},
	}), nil
}

func (s *glossaryTestService) SearchConfiguration(_ context.Context, _ *connect.Request[glossaryv1.SearchConfigurationRequest]) (*connect.Response[glossaryv1.SearchConfigurationResponse], error) {
	return connect.NewResponse(&glossaryv1.SearchConfigurationResponse{}), nil
}
