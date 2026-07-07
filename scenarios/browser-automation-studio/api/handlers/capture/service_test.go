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
	settings := exec.LastReq.GetFlowDefinition().GetSettings()
	require.NotNil(t, settings)
	require.EqualValues(t, 390, settings.GetViewportWidth())
	require.EqualValues(t, 844, settings.GetViewportHeight())
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
	settings := exec.LastReq.GetFlowDefinition().GetSettings()
	require.EqualValues(t, 1200, settings.GetViewportWidth())
	require.EqualValues(t, 800, settings.GetViewportHeight())
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

func TestCapture_HarvestArtifacts_ReadsExporterOutput(t *testing.T) {
	exec := &fakeExecutor{
		ExportLayout: map[string]string{
			"screenshots/step-01-nav.png": "fake-png-bytes",
			"console-logs.md":             "# console\n",
			"network-activity.md":         "# network\n",
		},
	}
	client, _ := newTestServer(t, Deps{Executor: exec})

	resp, err := client.Capture(context.Background(), connect.NewRequest(&capturev1.CaptureRequest{
		Url:    "https://example.com",
		OutDir: t.TempDir(),
		Captures: []capturev1.CaptureType{
			capturev1.CaptureType_CAPTURE_TYPE_SCREENSHOT,
			capturev1.CaptureType_CAPTURE_TYPE_CONSOLE_LOGS,
			capturev1.CaptureType_CAPTURE_TYPE_NETWORK,
		},
	}))
	require.NoError(t, err)
	require.Equal(t, 1, exec.ExportCalls, "export seam reached exactly once")
	require.Len(t, resp.Msg.Artifacts, 3)

	byType := map[capturev1.CaptureType]*capturev1.CaptureArtifact{}
	for _, a := range resp.Msg.Artifacts {
		byType[a.Type] = a
	}
	shot := byType[capturev1.CaptureType_CAPTURE_TYPE_SCREENSHOT]
	require.NotNil(t, shot)
	require.Contains(t, shot.Path, "step-01-nav.png")
	require.EqualValues(t, len("fake-png-bytes"), shot.SizeBytes)
	require.NotEqual(t, "true", shot.Metadata["unavailable"], "real screenshot must not be marked unavailable")

	console := byType[capturev1.CaptureType_CAPTURE_TYPE_CONSOLE_LOGS]
	require.NotNil(t, console)
	require.EqualValues(t, len("# console\n"), console.SizeBytes)

	network := byType[capturev1.CaptureType_CAPTURE_TYPE_NETWORK]
	require.NotNil(t, network)
	require.EqualValues(t, len("# network\n"), network.SizeBytes)
}

func TestCapture_HarvestArtifacts_MarksUnsupportedTypesUnavailable(t *testing.T) {
	exec := &fakeExecutor{} // no export layout — every file is missing
	client, _ := newTestServer(t, Deps{Executor: exec})

	resp, err := client.Capture(context.Background(), connect.NewRequest(&capturev1.CaptureRequest{
		Url:    "https://example.com",
		OutDir: t.TempDir(),
		Captures: []capturev1.CaptureType{
			capturev1.CaptureType_CAPTURE_TYPE_VIDEO,
			capturev1.CaptureType_CAPTURE_TYPE_DOM,
			capturev1.CaptureType_CAPTURE_TYPE_PERFORMANCE,
		},
	}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.Artifacts, 3)
	for _, a := range resp.Msg.Artifacts {
		require.Equal(t, "true", a.Metadata["unavailable"], "type %v must be flagged unavailable", a.Type)
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

// --- Phase 1: interaction-aware perf capture --------------------------------

// scrollFlowJSON is a 2-node bas/flows-shape interaction: navigate (the flow's
// own entry) + a scroll-by-selector. The capture path prepends its own
// navigate-to-URL node and splices this flow after it.
const scrollFlowJSON = `{
  "metadata": {"name": "scroll-list", "version": "1"},
  "nodes": [
    {"id": "flow-scroll", "action": {"type": "ACTION_TYPE_SCROLL",
      "scroll": {"selector": "[data-testid='virtualized-list']", "delta_y": 2000}}}
  ]
}`

func TestCapture_InteractionFlow_SplicedAfterNavigate(t *testing.T) {
	exec := &fakeExecutor{}
	client, _ := newTestServer(t, Deps{Executor: exec})

	resp, err := client.Capture(context.Background(), connect.NewRequest(&capturev1.CaptureRequest{
		Url:                 "https://example.com/list",
		Captures:            []capturev1.CaptureType{capturev1.CaptureType_CAPTURE_TYPE_PERFORMANCE},
		InteractionFlowJson: scrollFlowJSON,
	}))
	require.NoError(t, err)
	require.False(t, resp.Msg.DryRun)

	require.NotNil(t, exec.LastReq)
	def := exec.LastReq.FlowDefinition
	require.NotNil(t, def)
	// navigate node + spliced scroll node.
	require.Len(t, def.Nodes, 2)
	nav := def.Nodes[0].GetAction().GetNavigate()
	require.NotNil(t, nav)
	require.Equal(t, "https://example.com/list", nav.Url)
	scroll := def.Nodes[1].GetAction().GetScroll()
	require.NotNil(t, scroll, "spliced node must be the scroll interaction")
	require.Equal(t, "[data-testid='virtualized-list']", scroll.GetSelector())
	// An edge wires navigate → the interaction's first node so the scroll runs
	// inside the same perf-trace window, after the navigate.
	require.Len(t, def.Edges, 1)
	require.Equal(t, def.Nodes[0].GetId(), def.Edges[0].GetSource())
	require.Equal(t, def.Nodes[1].GetId(), def.Edges[0].GetTarget())

	// Perf trace was requested.
	require.Equal(t, 1, exec.Calls)
}

// A bas/flows body with short-form metadata (execution_mode:"observer",
// viewport settings) must splice identically to one fed through
// `execute-adhoc --flow-file` — i.e. the capture path applies the same compat
// normalization, not raw protojson.
func TestCapture_InteractionFlow_ShortFormMetadataNormalized(t *testing.T) {
	exec := &fakeExecutor{}
	client, _ := newTestServer(t, Deps{Executor: exec})

	flow := `{
	  "metadata": {"name": "scroll-list", "execution_mode": "observer", "reset": "none"},
	  "settings": {"viewport_width": 1440, "viewport_height": 900},
	  "nodes": [
	    {"id": "s", "action": {"type": "ACTION_TYPE_SCROLL", "scroll": {"selector": "[data-testid='list']", "delta_y": 1000}}}
	  ]
	}`
	resp, err := client.Capture(context.Background(), connect.NewRequest(&capturev1.CaptureRequest{
		Url:                 "https://example.com/list",
		Captures:            []capturev1.CaptureType{capturev1.CaptureType_CAPTURE_TYPE_PERFORMANCE},
		InteractionFlowJson: flow,
	}))
	require.NoError(t, err)
	require.False(t, resp.Msg.DryRun)
	require.NotNil(t, exec.LastReq)
	require.Len(t, exec.LastReq.FlowDefinition.Nodes, 2)
	require.NotNil(t, exec.LastReq.FlowDefinition.Nodes[1].GetAction().GetScroll())
}

func TestCapture_EmptyInteractionFlow_PreservesNavigateOnly(t *testing.T) {
	exec := &fakeExecutor{}
	client, _ := newTestServer(t, Deps{Executor: exec})

	resp, err := client.Capture(context.Background(), connect.NewRequest(&capturev1.CaptureRequest{
		Url:                 "https://example.com",
		Captures:            []capturev1.CaptureType{capturev1.CaptureType_CAPTURE_TYPE_PERFORMANCE},
		InteractionFlowJson: "",
	}))
	require.NoError(t, err)
	require.False(t, resp.Msg.DryRun)
	require.NotNil(t, exec.LastReq)
	require.Len(t, exec.LastReq.FlowDefinition.Nodes, 1)
	require.Empty(t, exec.LastReq.FlowDefinition.Edges)
}

func TestCapture_MalformedInteractionFlow_InvalidArgument(t *testing.T) {
	exec := &fakeExecutor{}
	client, _ := newTestServer(t, Deps{Executor: exec})

	_, err := client.Capture(context.Background(), connect.NewRequest(&capturev1.CaptureRequest{
		Url:                 "https://example.com",
		InteractionFlowJson: "{ not valid json",
	}))
	require.Error(t, err)
	var ce *connect.Error
	require.True(t, errors.As(err, &ce))
	require.Equal(t, connect.CodeInvalidArgument, ce.Code())
	require.Equal(t, 0, exec.Calls, "executor must not run on a malformed flow")
}
