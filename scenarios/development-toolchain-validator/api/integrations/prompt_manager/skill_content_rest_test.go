package prompt_manager

import (
	"context"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	skillsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/skills"
	skillsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/skills/skills_v1connect"
)

type contentSkillsService struct {
	skillsconnect.UnimplementedSkillsServiceHandler
	get func(string) (*skillsv1.Skill, error)
}

func (f contentSkillsService) GetSkill(_ context.Context, req *connect.Request[skillsv1.GetSkillRequest]) (*connect.Response[skillsv1.GetSkillResponse], error) {
	skill, err := f.get(req.Msg.GetId())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&skillsv1.GetSkillResponse{Skill: skill}), nil
}

func contentServer(t *testing.T, get func(string) (*skillsv1.Skill, error)) *httptest.Server {
	t.Helper()
	_, handler := skillsconnect.NewSkillsServiceHandler(contentSkillsService{get: get})
	return httptest.NewServer(handler)
}

func TestSkillContentFetchesBody(t *testing.T) {
	const id = "progress"
	const body = "# Progress\n\nAdvance the operational progress log.\n"

	srv := contentServer(t, func(got string) (*skillsv1.Skill, error) {
		if got != id {
			return nil, connect.NewError(connect.CodeNotFound, context.Canceled)
		}
		return &skillsv1.Skill{Id: id, Content: body}, nil
	})
	defer srv.Close()

	a := NewSkillContentRESTAdapter(Options{Resolver: discovery.NewStaticResolver(srv.URL)})
	got, err := a.SkillContent(context.Background(), id)
	if err != nil {
		t.Fatalf("SkillContent: %v", err)
	}
	if got != body {
		t.Fatalf("content = %q, want %q", got, body)
	}
}

func TestSkillContentEmptyIDErrors(t *testing.T) {
	a := NewSkillContentRESTAdapter(Options{Resolver: discovery.NewStaticResolver("http://unused")})
	if _, err := a.SkillContent(context.Background(), "   "); err == nil {
		t.Fatalf("empty id should error")
	}
}

func TestSkillContentNotFoundErrors(t *testing.T) {
	srv := contentServer(t, func(string) (*skillsv1.Skill, error) {
		return nil, connect.NewError(connect.CodeNotFound, context.Canceled)
	})
	defer srv.Close()

	a := NewSkillContentRESTAdapter(Options{Resolver: discovery.NewStaticResolver(srv.URL)})
	if _, err := a.SkillContent(context.Background(), "missing"); err == nil {
		t.Fatalf("404 should surface as error")
	}
}
