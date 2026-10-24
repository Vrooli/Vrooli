package promptmanager

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	skillsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/skills"
	skillsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/skills/skills_v1connect"
)

type mockHTTPDoer struct {
	doFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockHTTPDoer) Do(req *http.Request) (*http.Response, error) {
	return m.doFunc(req)
}

func makeResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

type fakeSkillsService struct {
	skillsconnect.UnimplementedSkillsServiceHandler
	read func(*skillsv1.ReadSkillsRequest) (*skillsv1.ReadSkillsResponse, error)
}

func (f fakeSkillsService) ReadSkills(_ context.Context, req *connect.Request[skillsv1.ReadSkillsRequest]) (*connect.Response[skillsv1.ReadSkillsResponse], error) {
	response, err := f.read(req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(response), nil
}

func newSkillsClient(t *testing.T, read func(*skillsv1.ReadSkillsRequest) (*skillsv1.ReadSkillsResponse, error)) *HTTPClient {
	t.Helper()
	_, handler := skillsconnect.NewSkillsServiceHandler(fakeSkillsService{read: read})
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return NewHTTPClientWithResolver(func(context.Context) (string, error) { return server.URL, nil }, server.Client())
}

func TestNewHTTPClient_Defaults(t *testing.T) {
	client := NewHTTPClient()
	if client == nil {
		t.Fatal("expected client")
	}
	if client.baseURLResolver == nil {
		t.Fatal("expected baseURLResolver")
	}
	if client.httpClient == nil {
		t.Fatal("expected httpClient")
	}
}

func TestNewHTTPClientWithResolver_Defaults(t *testing.T) {
	client := NewHTTPClientWithResolver(nil, nil)
	if client.baseURLResolver == nil {
		t.Fatal("expected default resolver")
	}
	if client.httpClient == nil {
		t.Fatal("expected default http client")
	}
}

func TestHTTPClient_ReadSkill(t *testing.T) {
	client := newSkillsClient(t, func(req *skillsv1.ReadSkillsRequest) (*skillsv1.ReadSkillsResponse, error) {
		if len(req.GetIdentifiers()) != 1 || req.GetIdentifiers()[0] != "test-skill" {
			t.Errorf("expected identifier test-skill, got %v", req.GetIdentifiers())
		}
		if req.GetVariables()["ITEM_NAME"] != "my-item" {
			t.Errorf("expected variable ITEM_NAME=my-item, got %s", req.GetVariables()["ITEM_NAME"])
		}
		if req.GetOutput() != "combined" {
			t.Errorf("expected output combined, got %s", req.GetOutput())
		}
		return &skillsv1.ReadSkillsResponse{Combined: "rendered prompt content"}, nil
	})

	result, err := client.ReadSkill(context.Background(), "test-skill", map[string]string{"ITEM_NAME": "my-item"}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "rendered prompt content" {
		t.Errorf("expected 'rendered prompt content', got %q", result)
	}
}

func TestHTTPClientReadSkillSourceReturnsImmutableMetadata(t *testing.T) {
	client := newSkillsClient(t, func(req *skillsv1.ReadSkillsRequest) (*skillsv1.ReadSkillsResponse, error) {
		if req.GetOutput() != "both" || len(req.GetVariables()) != 0 || req.GetWithScope() {
			t.Fatalf("source request = %+v", req)
		}
		return &skillsv1.ReadSkillsResponse{Combined: "<skills>{{TARGET}} {{CONFIG}}</skills>", CombinedHash: "sha256:source", Skills: []*skillsv1.Skill{{Id: "test-skill", Revision: 7, ContentHash: "sha256:raw", Variables: []*skillsv1.Variable{{Name: "TARGET"}, {Name: "CONFIG"}}}}}, nil
	})

	source, err := client.ReadSkillSource(context.Background(), "test-skill", []string{"CONFIG", "TARGET"})
	if err != nil {
		t.Fatalf("ReadSkillSource: %v", err)
	}
	if source.SkillID != "test-skill" || source.Revision != 7 || source.SelectedVariantID != "control" || source.ContentHash != "sha256:source" {
		t.Fatalf("source = %+v", source)
	}
	if strings.Join(source.TemplateVariables, ",") != "CONFIG,TARGET" {
		t.Fatalf("variables = %v", source.TemplateVariables)
	}
}

func TestHTTPClient_ReadSkill_ServerError(t *testing.T) {
	client := newSkillsClient(t, func(*skillsv1.ReadSkillsRequest) (*skillsv1.ReadSkillsResponse, error) {
		return nil, connect.NewError(connect.CodeInternal, context.Canceled)
	})

	_, err := client.ReadSkill(context.Background(), "test-skill", nil, false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "internal") {
		t.Errorf("expected Connect internal error, got %v", err)
	}
}

func TestHTTPClient_ReadSkill_ResolverError(t *testing.T) {
	client := NewHTTPClientWithResolver(
		func(_ context.Context) (string, error) {
			return "", context.DeadlineExceeded
		},
		nil,
	)

	_, err := client.ReadSkill(context.Background(), "test-skill", nil, false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "resolve URL") {
		t.Errorf("expected resolve URL in error, got %v", err)
	}
}
