package facts

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProtoHandlerConformanceMatchesConcreteUnarySignature(t *testing.T) {
	root := protoHandlerFixture(t, `func (h *connectHandler) Get(ctx context.Context, req *connect.Request[widgetv1.GetRequest]) (*connect.Response[widgetv1.GetResponse], error) {
	return nil, nil
}`)
	response, err := scanProtoHandlerConformance(filepath.Join(root, "scenarios", "demo"), "demo")
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Services) != 1 || response.Services[0].Status != "MATCH" || len(response.Services[0].Methods) != 1 {
		t.Fatalf("conformance response = %+v", response)
	}
	method := response.Services[0].Methods[0]
	if method.Name != "Get" || method.Status != "MATCH" || method.Generated.SHA256 == "" || method.Implementation == nil || method.Implementation.SHA256 == "" {
		t.Fatalf("method contract = %+v", method)
	}
	if len(method.ExpectedTypes) != 4 || len(method.ActualTypes) != 4 {
		t.Fatalf("signature types expected/actual = %#v / %#v", method.ExpectedTypes, method.ActualTypes)
	}
}

func TestProtoHandlerConformanceKeepsMismatchAndMissingExplicit(t *testing.T) {
	cases := []struct {
		name           string
		implementation string
		want           string
	}{
		{
			name: "response message mismatch",
			implementation: `func (h *connectHandler) Get(ctx context.Context, req *connect.Request[widgetv1.GetRequest]) (*connect.Response[widgetv1.OtherResponse], error) {
	return nil, nil
}`,
			want: "MISMATCH",
		},
		{name: "missing explicit method", want: "MISSING_EXPLICIT_METHOD"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := protoHandlerFixture(t, tc.implementation)
			response, err := scanProtoHandlerConformance(filepath.Join(root, "scenarios", "demo"), "demo")
			if err != nil {
				t.Fatal(err)
			}
			if len(response.Services) != 1 || len(response.Services[0].Methods) != 1 || response.Services[0].Methods[0].Status != tc.want || response.Services[0].Status != "PARTIAL" {
				t.Fatalf("conformance response = %+v, want method %s", response, tc.want)
			}
		})
	}
}

func TestProtoHandlerConformanceChecksPointerMethodSetAgainstConcreteHandler(t *testing.T) {
	root := protoHandlerFixtureWithHandler(t, `func (h *connectHandler) Get(ctx context.Context, req *connect.Request[widgetv1.GetRequest]) (*connect.Response[widgetv1.GetResponse], error) {
	return nil, nil
}`, "connectHandler", "NewHandler()")
	response, err := scanProtoHandlerConformance(filepath.Join(root, "scenarios", "demo"), "demo")
	if err != nil {
		t.Fatal(err)
	}
	if got := response.Services[0].Methods[0].Status; got != "MISSING_EXPLICIT_METHOD" {
		t.Fatalf("pointer receiver must not match a value handler type, got %s", got)
	}
}

func TestProtoHandlerConformanceDoesNotJoinSameNamedHandlerFromOtherPackage(t *testing.T) {
	root := protoHandlerFixture(t, `func (h *connectHandler) Get(ctx context.Context, req *connect.Request[widgetv1.GetRequest]) (*connect.Response[widgetv1.GetResponse], error) {
	return nil, nil
}`)
	otherPackage := filepath.Join(root, "scenarios", "demo", "api", "handlers", "other")
	if err := os.MkdirAll(otherPackage, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(otherPackage, "connect_handler.go"), []byte(`package other
type connectHandler struct{}
func (h *connectHandler) Get(ctx any, req any) (any, error) { return nil, nil }
`), 0o644); err != nil {
		t.Fatal(err)
	}
	response, err := scanProtoHandlerConformance(filepath.Join(root, "scenarios", "demo"), "demo")
	if err != nil {
		t.Fatal(err)
	}
	if got := response.Services[0].Methods[0].Status; got != "MATCH" {
		t.Fatalf("same-named type in another package must not affect the scoped implementation, got %s", got)
	}
}

func protoHandlerFixture(t *testing.T, implementation string) string {
	return protoHandlerFixtureWithHandler(t, implementation, "*connectHandler", "NewHandler()")
}

func protoHandlerFixtureWithHandler(t *testing.T, implementation, handlerReturn, handlerExpr string) string {
	t.Helper()
	root := t.TempDir()
	scenario := filepath.Join(root, "scenarios", "demo")
	handlerDir := filepath.Join(scenario, "api", "handlers", "widget")
	generatedDir := filepath.Join(root, "packages", "proto", "gen", "go", "demo", "v1", "widget", "widget_v1connect")
	for _, dir := range []string{handlerDir, generatedDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write := func(path, content string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(handlerDir, "module.go"), `package widget
import widgetconnect "github.com/vrooli/vrooli/packages/proto/gen/go/demo/v1/widget/widget_v1connect"
func Module() {
	_, _ = widgetconnect.NewWidgetServiceHandler(`+handlerExpr+`)
}
`)
	constructor := `return &connectHandler{}`
	if handlerReturn == "connectHandler" {
		constructor = `return connectHandler{}`
	}
	write(filepath.Join(handlerDir, "connect_handler.go"), `package widget
import (
	"context"
	"connectrpc.com/connect"
	widgetv1 "github.com/vrooli/vrooli/packages/proto/gen/go/demo/v1/widget"
)
type connectHandler struct{}
func NewHandler() `+handlerReturn+` { `+constructor+` }
`+implementation+"\n")
	write(filepath.Join(generatedDir, "widget.connect.go"), `package widget_v1connect
import (
	"context"
	"connectrpc.com/connect"
	widgetv1 "github.com/vrooli/vrooli/packages/proto/gen/go/demo/v1/widget"
)
type WidgetServiceHandler interface {
	Get(context.Context, *connect.Request[widgetv1.GetRequest]) (*connect.Response[widgetv1.GetResponse], error)
}
func NewWidgetServiceHandler(svc WidgetServiceHandler) {}
`)
	return root
}
