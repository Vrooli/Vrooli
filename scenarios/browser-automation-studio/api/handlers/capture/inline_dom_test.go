package capture

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"github.com/vrooli/browser-automation-studio/storage"
	capturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture"
)

func TestInlineDOMDataAttributes(t *testing.T) {
	cmd := exec.Command("node", "testdata/dom_data.cjs")
	cmd.Stdin = strings.NewReader(defaultInlineDomTreeExpression)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))
}

// writeTimelineForDomNode renders a minimal timeline.json whose evaluate frame
// carries domHTML under the "result" key, mirroring what the execution writer
// persists from the driver's evaluate handler.
func writeTimelineForDomNode(t *testing.T, f *fakeExecutor, outputDir, domHTML string) {
	t.Helper()
	nodes := f.LastReq.GetFlowDefinition().GetNodes()
	require.NotNil(t, nodes[1].GetAction().GetEvaluate(), "inline DOM reads must follow navigation")
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
	require.Equal(t, defaultInlineDomExpression, eval.GetExpression())
	require.Len(t, flow.GetEdges(), 1)
	require.Equal(t, flow.GetNodes()[0].GetId(), flow.GetEdges()[0].GetSource())
	require.Equal(t, flow.GetNodes()[1].GetId(), flow.GetEdges()[0].GetTarget())
}

func TestCapture_DomArtifactDoesNotRequireInlineResponseFlag(t *testing.T) {
	const page = "<html><body>artifact body</body></html>"
	const tree = `{"tagName":"BODY","computed":{"display":"block"}}`
	exec := &fakeExecutor{
		ExportLayout: map[string]string{"result.json": "{}"},
		ExportFunc: func(f *fakeExecutor, outputDir string) error {
			frames := make([]map[string]any, 0, 2)
			for _, node := range f.LastReq.GetFlowDefinition().GetNodes() {
				evaluate := node.GetAction().GetEvaluate()
				if evaluate == nil {
					continue
				}
				result := page
				if evaluate.GetExpression() == defaultInlineDomTreeExpression {
					result = tree
				}
				frames = append(frames, map[string]any{
					"node_id": node.GetId(), "step_type": "evaluate",
					"extracted_data_preview": map[string]any{"result": result},
				})
			}
			raw, err := json.Marshal(map[string]any{"frames": frames})
			if err != nil {
				return err
			}
			return os.WriteFile(filepath.Join(outputDir, "timeline.json"), raw, 0o644)
		},
	}
	client, _ := newTestServer(t, Deps{Executor: exec, Storage: storage.NewMemoryStorage()})

	resp, err := client.Capture(context.Background(), connect.NewRequest(&capturev1.CaptureRequest{
		Url: "https://example.com",
		Captures: []capturev1.CaptureType{
			capturev1.CaptureType_CAPTURE_TYPE_DOM,
			capturev1.CaptureType_CAPTURE_TYPE_DOM_TREE,
		},
	}))
	require.NoError(t, err)

	var domNodeID, treeNodeID string
	for _, node := range exec.LastReq.GetFlowDefinition().GetNodes() {
		if evaluate := node.GetAction().GetEvaluate(); evaluate != nil {
			switch evaluate.GetExpression() {
			case defaultInlineDomExpression:
				domNodeID = node.GetId()
			case defaultInlineDomTreeExpression:
				treeNodeID = node.GetId()
			}
		}
	}
	require.NotEmpty(t, domNodeID, "a requested DOM artifact must evaluate the rendered DOM")
	require.NotEmpty(t, treeNodeID, "a requested DOM-tree artifact must evaluate its independent snapshot")
	require.NotEqual(t, domNodeID, treeNodeID)
	require.Empty(t, resp.Msg.DomHtml, "artifact capture should not implicitly expand the inline response")
	require.Empty(t, resp.Msg.DomTreeJson)
	require.Len(t, resp.Msg.Artifacts, 2)
	byType := map[capturev1.CaptureType]*capturev1.CaptureArtifact{}
	for _, artifact := range resp.Msg.Artifacts {
		byType[artifact.GetType()] = artifact
	}
	for _, expected := range []struct {
		captureType capturev1.CaptureType
		filename    string
		contents    string
		reference   string
	}{
		{capturev1.CaptureType_CAPTURE_TYPE_DOM, "dom.html", page, "dom"},
		{capturev1.CaptureType_CAPTURE_TYPE_DOM_TREE, "dom-tree.json", tree, "dom_tree"},
	} {
		artifact := byType[expected.captureType]
		require.NotNil(t, artifact)
		require.Equal(t, int64(len(expected.contents)), artifact.GetSizeBytes())
		require.NotEqual(t, "true", artifact.GetMetadata()["unavailable"])
		require.Equal(t, expected.filename, artifact.GetMetadata()["filename"])
		require.Equal(t, "bas-capture://"+resp.Msg.GetExecutionId()+"/"+expected.reference, artifact.GetReference())
		require.NotEmpty(t, artifact.GetMetadata()["view_url"], "generated DOM artifacts must be published through storage")
		require.FileExists(t, artifact.GetPath())
		contents, err := os.ReadFile(artifact.GetPath())
		require.NoError(t, err)
		require.Equal(t, expected.contents, string(contents))
	}
	result, err := os.ReadFile(filepath.Join(exec.LastExportDir, "result.json"))
	require.NoError(t, err)
	require.Contains(t, string(result), "dom.html")
	require.Contains(t, string(result), "dom-tree.json")
}

func TestCapture_InlineDomTree_PublishesComputedSnapshotArtifact(t *testing.T) {
	const tree = `{"tagName":"BODY","computed":{"display":"block"},"rect":{"width":10}}`
	exec := &fakeExecutor{
		ExportFunc: func(f *fakeExecutor, outputDir string) error {
			writeTimelineForDomNode(t, f, outputDir, tree)
			return nil
		},
	}
	client, _ := newTestServer(t, Deps{Executor: exec})

	resp, err := client.Capture(context.Background(), connect.NewRequest(&capturev1.CaptureRequest{
		Url: "https://example.com", Captures: []capturev1.CaptureType{capturev1.CaptureType_CAPTURE_TYPE_DOM_TREE}, InlineDomTree: true,
	}))
	require.NoError(t, err)
	require.Equal(t, tree, resp.Msg.DomTreeJson)
	require.Len(t, resp.Msg.Artifacts, 1)
	require.Equal(t, capturev1.CaptureType_CAPTURE_TYPE_DOM_TREE, resp.Msg.Artifacts[0].Type)
	require.Equal(t, int64(len(tree)), resp.Msg.Artifacts[0].SizeBytes)
	require.NotContains(t, resp.Msg.Artifacts[0].GetMetadata(), "unavailable")
	require.FileExists(t, resp.Msg.Artifacts[0].Path)
}

func TestCapture_InlineDom_Disabled_NoEvaluateNode(t *testing.T) {
	exec := &fakeExecutor{}
	client, _ := newTestServer(t, Deps{Executor: exec})

	resp, err := client.Capture(context.Background(), connect.NewRequest(&capturev1.CaptureRequest{
		Url: "https://example.com",
	}))
	require.NoError(t, err)
	require.Empty(t, resp.Msg.DomHtml)
	require.Len(t, exec.LastReq.GetFlowDefinition().GetNodes(), 2)
	require.Len(t, exec.LastReq.GetFlowDefinition().GetEdges(), 1)
	for _, node := range exec.LastReq.GetFlowDefinition().GetNodes() {
		require.Nil(t, node.GetAction().GetEvaluate())
	}
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
	oversized := "<html>" + strings.Repeat("x", defaultInlineDomMaxBytes) + "</html>"
	exec := &fakeExecutor{
		ExportFunc: func(f *fakeExecutor, outputDir string) error {
			writeTimelineForDomNode(t, f, outputDir, oversized)
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
	require.Len(t, resp.Msg.DomHtml, defaultInlineDomMaxBytes)
	require.Len(t, resp.Msg.Artifacts, 1)
	require.Equal(t, "true", resp.Msg.Artifacts[0].GetMetadata()["truncated"])
	require.Equal(t, int64(defaultInlineDomMaxBytes), resp.Msg.Artifacts[0].GetSizeBytes())
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
