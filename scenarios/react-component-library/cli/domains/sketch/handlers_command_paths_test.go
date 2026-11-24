package sketch

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
	experienceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/experience/experience_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/shared"
	sketchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/sketch"
)

type commandPathExperience struct {
	experienceconnect.UnimplementedExperienceServiceHandler
	gotPlaceholder *sketchv1.PlaceholderRequest
	gotNote        *sketchv1.AddNoteRequest
	gotUnplace     *sketchv1.UnplaceRequest
	gotTemplate    *sketchv1.SetTemplateRequest
	gotBrief       *sketchv1.BuildBriefRequest
	gotHistory     *sketchv1.GetHistoryRequest
	gotRecover     *sketchv1.RecoverSketchRequest
	templateResult *sketchv1.SetTemplateResponse
	recoverResult  *sketchv1.PutSketchResponse
	historyResult  *sketchv1.GetHistoryResponse
}

func (s *commandPathExperience) GetSketch(context.Context, *connect.Request[sketchv1.GetSketchRequest]) (*connect.Response[sketchv1.GetSketchResponse], error) {
	return connect.NewResponse(&sketchv1.GetSketchResponse{ContentHash: "hash-before"}), nil
}

func (s *commandPathExperience) Placeholder(_ context.Context, req *connect.Request[sketchv1.PlaceholderRequest]) (*connect.Response[sketchv1.PutSketchResponse], error) {
	s.gotPlaceholder = req.Msg
	return connect.NewResponse(&sketchv1.PutSketchResponse{Changed: true}), nil
}

func (s *commandPathExperience) AddNote(_ context.Context, req *connect.Request[sketchv1.AddNoteRequest]) (*connect.Response[sketchv1.PutSketchResponse], error) {
	s.gotNote = req.Msg
	return connect.NewResponse(&sketchv1.PutSketchResponse{Changed: true}), nil
}

func (s *commandPathExperience) Unplace(_ context.Context, req *connect.Request[sketchv1.UnplaceRequest]) (*connect.Response[sketchv1.PutSketchResponse], error) {
	s.gotUnplace = req.Msg
	return connect.NewResponse(&sketchv1.PutSketchResponse{Changed: true}), nil
}

func (s *commandPathExperience) SetTemplate(_ context.Context, req *connect.Request[sketchv1.SetTemplateRequest]) (*connect.Response[sketchv1.SetTemplateResponse], error) {
	s.gotTemplate = req.Msg
	response := s.templateResult
	if response == nil {
		response = &sketchv1.SetTemplateResponse{Changed: true}
	}
	return connect.NewResponse(response), nil
}

func (s *commandPathExperience) BuildBrief(_ context.Context, req *connect.Request[sketchv1.BuildBriefRequest]) (*connect.Response[sketchv1.BuildBriefResponse], error) {
	s.gotBrief = req.Msg
	return connect.NewResponse(&sketchv1.BuildBriefResponse{Markdown: "# Ordered build\n\n1. Implement\n2. Verify"}), nil
}

func (s *commandPathExperience) GetHistory(_ context.Context, req *connect.Request[sketchv1.GetHistoryRequest]) (*connect.Response[sketchv1.GetHistoryResponse], error) {
	s.gotHistory = req.Msg
	response := s.historyResult
	if response == nil {
		response = &sketchv1.GetHistoryResponse{Revisions: []*sketchv1.SketchRevision{{ContentHash: "rev-1", CreatedAt: "now", Current: true}}}
	}
	return connect.NewResponse(response), nil
}

func (s *commandPathExperience) RecoverSketch(_ context.Context, req *connect.Request[sketchv1.RecoverSketchRequest]) (*connect.Response[sketchv1.PutSketchResponse], error) {
	s.gotRecover = req.Msg
	response := s.recoverResult
	if response == nil {
		response = &sketchv1.PutSketchResponse{ContentHash: "recovered", Changed: true}
	}
	return connect.NewResponse(response), nil
}

func runSketchCommand(t *testing.T, svc *commandPathExperience, name string, args ...string) (string, error) {
	t.Helper()
	path, handler := experienceconnect.NewExperienceServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	core := cliapptest.NewTestApp(t, mux)
	manifest, err := os.ReadFile("../../manifest.json")
	require.NoError(t, err)
	group, err := Register(core, manifest)
	require.NoError(t, err)
	command := cliapptest.FindCommand(t, []cliapp.SubcommandGroup{group}, GroupName, name)
	return cliapptest.RunCommand(t, command, core, args...)
}

func TestSketchPlaceholderNoteAndUnplaceCommandsPreserveInputs(t *testing.T) {
	svc := &commandPathExperience{}
	out, err := runSketchCommand(t, svc, "placeholder", "shop/home", "hero", "new-filter", "--intent", "A filter for a behavior the catalog does not provide", "--note", "keep it discoverable")
	require.NoError(t, err)
	require.Contains(t, out, "Placeholder new-filter recorded in hero")
	require.Equal(t, "hash-before", svc.gotPlaceholder.GetExpectedContentHash())
	require.Equal(t, "new-filter", svc.gotPlaceholder.GetPlaceholder())
	require.Equal(t, "A filter for a behavior the catalog does not provide", svc.gotPlaceholder.GetIntent())
	require.Equal(t, "keep it discoverable", svc.gotPlaceholder.GetNote())

	out, err = runSketchCommand(t, svc, "note", "shop/home", "--scope", "hero.title", "--text", "State the customer benefit clearly")
	require.NoError(t, err)
	require.Contains(t, out, "Sketch note recorded.")
	require.Equal(t, "hero.title", svc.gotNote.GetScope())
	require.Equal(t, "State the customer benefit clearly", svc.gotNote.GetText())
	require.Equal(t, "hash-before", svc.gotNote.GetExpectedContentHash())

	out, err = runSketchCommand(t, svc, "unplace", "shop/home", "hero", "--reason", "No catalog asset fits this region")
	require.NoError(t, err)
	require.Contains(t, out, "Region hero moved to unplaced.")
	require.Equal(t, "hero", svc.gotUnplace.GetRegion())
	require.Equal(t, "No catalog asset fits this region", svc.gotUnplace.GetReason())
	require.Equal(t, "hash-before", svc.gotUnplace.GetExpectedContentHash())
}

func TestSketchTemplateBriefHistoryAndRecoverCommandPaths(t *testing.T) {
	svc := &commandPathExperience{}
	out, err := runSketchCommand(t, svc, "template", "shop/home", "page.default", "--version", "2.1", "--remap", "hero=header", "--remap", "content=body", "--confirm-unplaced", "--preview")
	require.NoError(t, err)
	require.Contains(t, out, "Template updated")
	require.Equal(t, "hash-before", svc.gotTemplate.GetExpectedContentHash())
	require.Equal(t, "page.default", svc.gotTemplate.GetAsset())
	require.Equal(t, "2.1", svc.gotTemplate.GetVersion())
	require.Equal(t, []*sketchv1.RegionRemap{{From: "hero", To: "header"}, {From: "content", To: "body"}}, svc.gotTemplate.GetRemap())
	require.True(t, svc.gotTemplate.GetConfirmUnplaced())
	require.True(t, svc.gotTemplate.GetPreview())

	_, err = runSketchCommand(t, svc, "template", "shop/home", "page.default", "--remap", "broken")
	require.ErrorContains(t, err, `invalid --remap "broken": expected old=new`)

	svc.templateResult = &sketchv1.SetTemplateResponse{}
	out, err = runSketchCommand(t, svc, "template", "shop/home", "page.default")
	require.NoError(t, err)
	require.Contains(t, out, "Template unchanged")
	out, err = runSketchCommand(t, svc, "template", "shop/home", "page.default", "--preview")
	require.NoError(t, err)
	require.Contains(t, out, "Template preview")
	svc.templateResult = &sketchv1.SetTemplateResponse{RequiresConfirmation: true}
	out, err = runSketchCommand(t, svc, "template", "shop/home", "page.default", "--preview")
	require.NoError(t, err)
	require.Contains(t, out, "confirm displaced placements before applying")

	out, err = runSketchCommand(t, svc, "brief", "shop/home")
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(out, "# Ordered build"))
	require.Equal(t, &sharedv1.SketchTarget{Scenario: "shop", Page: "home"}, svc.gotBrief.GetTarget())

	out, err = runSketchCommand(t, svc, "history", "shop/home")
	require.NoError(t, err)
	require.Contains(t, out, "Design history: shop/home")
	require.Contains(t, out, "rev-1 · now · current: true")
	require.Equal(t, &sharedv1.SketchTarget{Scenario: "shop", Page: "home"}, svc.gotHistory.GetTarget())
	svc.historyResult = &sketchv1.GetHistoryResponse{}
	out, err = runSketchCommand(t, svc, "history", "shop/home")
	require.NoError(t, err)
	require.Contains(t, out, "Design history: shop/home")

	out, err = runSketchCommand(t, svc, "recover", "shop/home")
	require.NoError(t, err)
	require.Contains(t, out, "Revision: recovered; recovered: true")
	require.Equal(t, "hash-before", svc.gotRecover.GetExpectedContentHash())
	require.Equal(t, &sharedv1.SketchTarget{Scenario: "shop", Page: "home"}, svc.gotRecover.GetTarget())
	svc.recoverResult = &sketchv1.PutSketchResponse{ContentHash: "unchanged", Changed: false}
	out, err = runSketchCommand(t, svc, "recover", "shop/home")
	require.NoError(t, err)
	require.Contains(t, out, "Revision: unchanged; recovered: false")
}
