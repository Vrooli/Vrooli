package capture

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	capturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture"
)

// writeTimelineForDomNode renders a minimal timeline.json whose evaluate frame
// carries domHTML under the "result" key, mirroring what the execution writer
// persists from the driver's evaluate handler.
func writeTimelineForDomNode(t *testing.T, f *fakeExecutor, outputDir, domHTML string) {
	t.Helper()
	nodes := f.LastReq.GetFlowDefinition().GetNodes()
	require.Len(t, nodes, 2, "inline_dom flow must be navigate+evaluate")
	domNodeID := nodes[1].GetId()

	timeline := map[string]any{
		"frames": []map[string]any{
			{"node_id": nodes[0].GetId(), "step_type": "navigate"},
			{"node_id": domNodeID, "step_type": "evaluate", "extracted_data_preview": map[string]any{"result": domHTML}},
		},
	}
	raw, err := json.Marshal(timeline)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(outputDir, "timeline.json"), raw, 0o644))
}

func TestCapture_InlineDom_ReturnsRenderedHTML(t *testing.T) {
	const page = "<html><head><title>t</title></head><body>rendered body</body></html>"
	exec := &fakeExecutor{
		ExportFunc: func(f *fakeExecutor, outputDir string) error {
			writeTimelineForDomNode(t, f, outputDir, page)
			return nil
		},
	}
	client, _ := newTestServer(t, Deps{Executor: exec})

	resp, err := client.Capture(context.Background(), connect.NewRequest(&capturev1.CaptureRequest{
		Url:       "https://example.com",
		Captures:  []capturev1.CaptureType{capturev1.CaptureType_CAPTURE_TYPE_DOM},
		InlineDom: true,
	}))
	require.NoError(t, err)
	require.Equal(t, page, resp.Msg.DomHtml)

	// The flow must be navigate -> evaluate(outerHTML) connected by an edge.
	flow := exec.LastReq.GetFlowDefinition()
	require.Len(t, flow.GetNodes(), 2)
	eval := flow.GetNodes()[1].GetAction().GetEvaluate()
	require.NotNil(t, eval)
	require.Equal(t, inlineDomExpression, eval.GetExpression())
	require.Len(t, flow.GetEdges(), 1)
	require.Equal(t, flow.GetNodes()[0].GetId(), flow.GetEdges()[0].GetSource())
	require.Equal(t, flow.GetNodes()[1].GetId(), flow.GetEdges()[0].GetTarget())
}

func TestCapture_InlineDom_Disabled_NoEvaluateNode(t *testing.T) {
	exec := &fakeExecutor{}
	client, _ := newTestServer(t, Deps{Executor: exec})

	resp, err := client.Capture(context.Background(), connect.NewRequest(&capturev1.CaptureRequest{
		Url: "https://example.com",
	}))
	require.NoError(t, err)
	require.Empty(t, resp.Msg.DomHtml)
	require.Len(t, exec.LastReq.GetFlowDefinition().GetNodes(), 1)
	require.Empty(t, exec.LastReq.GetFlowDefinition().GetEdges())
}

func TestCapture_InlineDom_MissingTimelineResult_DegradesToEmpty(t *testing.T) {
	// No timeline.json written at all: the capture still succeeds, dom_html
	// is empty (the documented degraded contract).
	exec := &fakeExecutor{}
	client, _ := newTestServer(t, Deps{Executor: exec})

	resp, err := client.Capture(context.Background(), connect.NewRequest(&capturev1.CaptureRequest{
		Url:       "https://example.com",
		InlineDom: true,
	}))
	require.NoError(t, err)
	require.Empty(t, resp.Msg.DomHtml)
}

func TestCapture_InlineDom_TruncatesOversizedPayload(t *testing.T) {
	oversized := "<html>" + strings.Repeat("x", inlineDomMaxBytes) + "</html>"
	exec := &fakeExecutor{
		ExportFunc: func(f *fakeExecutor, outputDir string) error {
			writeTimelineForDomNode(t, f, outputDir, oversized)
			return nil
		},
	}
	client, _ := newTestServer(t, Deps{Executor: exec})

	resp, err := client.Capture(context.Background(), connect.NewRequest(&capturev1.CaptureRequest{
		Url:       "https://example.com",
		InlineDom: true,
	}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.DomHtml, inlineDomMaxBytes)
}

func TestCapture_InlineDom_DryRun_StaysEmpty(t *testing.T) {
	exec := &fakeExecutor{}
	client, _ := newTestServer(t, Deps{Executor: exec})

	req := connect.NewRequest(&capturev1.CaptureRequest{
		Url:       "https://example.com",
		InlineDom: true,
	})
	req.Header().Set("X-Dry-Run", "true")
	resp, err := client.Capture(context.Background(), req)
	require.NoError(t, err)
	require.True(t, resp.Msg.DryRun)
	require.Empty(t, resp.Msg.DomHtml)
	require.Zero(t, exec.Calls)
}
