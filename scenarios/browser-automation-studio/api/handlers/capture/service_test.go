package capture

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	capturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture"
	captureconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture/captureconnect"
)

// newTestServer wires the service through the generated Connect HTTP
// handler so tests exercise codec + header propagation in addition to
// the in-process Capture method.
func newTestServer(t *testing.T, deps Deps) (captureconnect.CaptureServiceClient, *httptest.Server) {
	t.Helper()
	if deps.Logger == nil {
		deps.Logger = logrus.New()
	}
	if deps.Now == nil {
		deps.Now = func() time.Time { return time.Unix(0, 0) }
	}
	mount := Module(deps)
	mux := http.NewServeMux()
	mux.Handle(mount.Path, mount.Handler)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	client := captureconnect.NewCaptureServiceClient(srv.Client(), srv.URL)
	return client, srv
}

func TestCapture_HappyPath_Screenshot(t *testing.T) {
	exec := &fakeExecutor{}
	client, _ := newTestServer(t, Deps{Executor: exec})

	resp, err := client.Capture(context.Background(), connect.NewRequest(&capturev1.CaptureRequest{
		Url: "https://example.com",
	}))
	require.NoError(t, err)
	require.False(t, resp.Msg.DryRun)
	require.Len(t, resp.Msg.Artifacts, 1)
	require.Equal(t, capturev1.CaptureType_CAPTURE_TYPE_SCREENSHOT, resp.Msg.Artifacts[0].Type)

	require.Equal(t, 1, exec.Calls)
	require.NotNil(t, exec.LastReq)
	require.NotNil(t, exec.LastReq.FlowDefinition)
	require.Len(t, exec.LastReq.FlowDefinition.Nodes, 1)
	nav := exec.LastReq.FlowDefinition.Nodes[0].GetAction().GetNavigate()
	require.NotNil(t, nav)
	require.Equal(t, "https://example.com", nav.Url)
}

func TestCapture_MultiCapture_ScreenshotConsoleNetwork(t *testing.T) {
	exec := &fakeExecutor{}
	client, _ := newTestServer(t, Deps{Executor: exec})

	resp, err := client.Capture(context.Background(), connect.NewRequest(&capturev1.CaptureRequest{
		Url: "https://example.com",
		Captures: []capturev1.CaptureType{
			capturev1.CaptureType_CAPTURE_TYPE_SCREENSHOT,
			capturev1.CaptureType_CAPTURE_TYPE_CONSOLE_LOGS,
			capturev1.CaptureType_CAPTURE_TYPE_NETWORK,
		},
	}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.Artifacts, 3)
	types := []capturev1.CaptureType{
		resp.Msg.Artifacts[0].Type, resp.Msg.Artifacts[1].Type, resp.Msg.Artifacts[2].Type,
	}
	require.ElementsMatch(t, []capturev1.CaptureType{
		capturev1.CaptureType_CAPTURE_TYPE_SCREENSHOT,
		capturev1.CaptureType_CAPTURE_TYPE_CONSOLE_LOGS,
		capturev1.CaptureType_CAPTURE_TYPE_NETWORK,
	}, types)
}

func TestCapture_DimensionsPreset_Mobile(t *testing.T) {
	exec := &fakeExecutor{}
	client, _ := newTestServer(t, Deps{Executor: exec})

	_, err := client.Capture(context.Background(), connect.NewRequest(&capturev1.CaptureRequest{
		Url: "https://example.com",
		Dimensions: &capturev1.Dimensions{
			Preset: capturev1.DimensionsPreset_DIMENSIONS_PRESET_MOBILE,
		},
	}))
	require.NoError(t, err)
	params := exec.LastReq.Parameters
	require.NotNil(t, params.ViewportWidth)
	require.NotNil(t, params.ViewportHeight)
	require.EqualValues(t, 390, *params.ViewportWidth)
	require.EqualValues(t, 844, *params.ViewportHeight)
}

func TestCapture_DimensionsExplicit_OverridesPreset(t *testing.T) {
	exec := &fakeExecutor{}
	client, _ := newTestServer(t, Deps{Executor: exec})

	w, h := int32(1200), int32(800)
	_, err := client.Capture(context.Background(), connect.NewRequest(&capturev1.CaptureRequest{
		Url: "https://example.com",
		Dimensions: &capturev1.Dimensions{
			Preset: capturev1.DimensionsPreset_DIMENSIONS_PRESET_MOBILE,
			Width:  &w,
			Height: &h,
		},
	}))
	require.NoError(t, err)
	require.EqualValues(t, 1200, *exec.LastReq.Parameters.ViewportWidth)
	require.EqualValues(t, 800, *exec.LastReq.Parameters.ViewportHeight)
}

func TestCapture_URLShorthand_ResolvesScenarioSlug(t *testing.T) {
	exec := &fakeExecutor{}
	resolver := &fakeResolver{URL: "http://127.0.0.1:9101"}
	client, _ := newTestServer(t, Deps{Executor: exec, Resolver: resolver})

	_, err := client.Capture(context.Background(), connect.NewRequest(&capturev1.CaptureRequest{
		Url: "scenario=app-monitor,path=/dashboard",
	}))
	require.NoError(t, err)
	require.NotNil(t, exec.LastReq.Parameters.StartUrl)
	require.Equal(t, "http://127.0.0.1:9101/dashboard", *exec.LastReq.Parameters.StartUrl)
	nav := exec.LastReq.FlowDefinition.Nodes[0].GetAction().GetNavigate()
	require.Equal(t, "http://127.0.0.1:9101/dashboard", nav.Url)
}

func TestCapture_DryRun_ShortCircuits(t *testing.T) {
	exec := &fakeExecutor{}
	client, _ := newTestServer(t, Deps{Executor: exec})

	req := connect.NewRequest(&capturev1.CaptureRequest{
		Url: "https://example.com",
		Captures: []capturev1.CaptureType{
			capturev1.CaptureType_CAPTURE_TYPE_SCREENSHOT,
			capturev1.CaptureType_CAPTURE_TYPE_NETWORK,
		},
		OutDir: "/tmp/dry",
	})
	req.Header().Set("X-Dry-Run", "true")
	resp, err := client.Capture(context.Background(), req)
	require.NoError(t, err)
	require.True(t, resp.Msg.DryRun)
	require.Equal(t, 0, exec.Calls, "executor must not be called for dry-run")
	require.Len(t, resp.Msg.Artifacts, 2)
	for _, a := range resp.Msg.Artifacts {
		require.Contains(t, a.Path, "/tmp/dry/")
	}
}

func TestCapture_ValidationErrors_InvalidArgument(t *testing.T) {
	type tc struct {
		name string
		req  *capturev1.CaptureRequest
		deps Deps
	}
	exec := &fakeExecutor{}
	width := int32(1200)

	cases := []tc{
		{
			name: "empty url",
			req:  &capturev1.CaptureRequest{Url: ""},
			deps: Deps{Executor: exec},
		},
		{
			name: "width without height",
			req: &capturev1.CaptureRequest{
				Url:        "https://example.com",
				Dimensions: &capturev1.Dimensions{Width: &width},
			},
			deps: Deps{Executor: exec},
		},
		{
			name: "unspecified capture type",
			req: &capturev1.CaptureRequest{
				Url:      "https://example.com",
				Captures: []capturev1.CaptureType{capturev1.CaptureType_CAPTURE_TYPE_UNSPECIFIED},
			},
			deps: Deps{Executor: exec},
		},
		{
			name: "malformed shorthand slug",
			req: &capturev1.CaptureRequest{
				Url: "scenario=BadSlug!,path=/",
			},
			deps: Deps{Executor: exec, Resolver: &fakeResolver{URL: "http://x"}},
		},
		{
			name: "shorthand without resolver",
			req:  &capturev1.CaptureRequest{Url: "scenario=app-monitor,path=/"},
			deps: Deps{Executor: exec},
		},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			client, _ := newTestServer(t, c.deps)
			_, err := client.Capture(context.Background(), connect.NewRequest(c.req))
			require.Error(t, err)
			var ce *connect.Error
			require.True(t, errors.As(err, &ce), "expected connect.Error, got %T", err)
			require.Equal(t, connect.CodeInvalidArgument, ce.Code(), "got code %s", ce.Code())
		})
	}
}
