package ai_service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	aihandlers "github.com/vrooli/browser-automation-studio/handlers/ai"
	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/ai"
	aiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/ai/aiconnect"
)

// newServiceForTest builds a service wired with the supplied mock automation
// runner. The runner is shared across screenshot/DOM/element analysis handlers
// so each RPC can choose its own canned outcome list.
func newServiceForTest(t *testing.T, runner aihandlers.AutomationRunner) (aiconnect.AIServiceClient, *spyLinkPreview) {
	t.Helper()
	log := logrus.New()
	log.SetOutput(io.Discard)

	screenshot := aihandlers.NewScreenshotHandler(log, aihandlers.WithScreenshotRunner(runner))
	dom := aihandlers.NewDOMHandler(log, aihandlers.WithDOMRunner(runner))
	// Default suggestion generator will fail to reach Ollama and gracefully
	// degrade to empty AI suggestions; that's the behavior we want under test.
	elem := aihandlers.NewElementAnalysisHandler(log,
		aihandlers.WithElementRunner(runner),
	)
	mockAnalyzer := &fakeAnalyzer{suggestions: []aihandlers.ElementInfo{{Text: "Search", TagName: "BUTTON", Confidence: 0.9}}}
	aiAn := aihandlers.NewAIAnalysisHandler(log, dom,
		aihandlers.WithElementAnalyzer(mockAnalyzer),
		aihandlers.WithAIAnalysisTimeout(time.Second),
	)

	spy := &spyLinkPreview{}
	mount := Module(Deps{
		Screenshot:      screenshot,
		DOM:             dom,
		ElementAnalysis: elem,
		AIAnalysis:      aiAn,
		LinkPreview:     spy.Fetch,
		Logger:          log,
	})

	mux := http.NewServeMux()
	mux.Handle(mount.Path, mount.Handler)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return aiconnect.NewAIServiceClient(srv.Client(), srv.URL), spy
}

type spyLinkPreview struct {
	calls []string
	data  *LinkPreviewData
	found bool
	err   error
}

func (s *spyLinkPreview) Fetch(_ context.Context, u string) (*LinkPreviewData, bool, error) {
	s.calls = append(s.calls, u)
	return s.data, s.found, s.err
}

type fakeAnalyzer struct {
	suggestions []aihandlers.ElementInfo
	err         error
}

func (f *fakeAnalyzer) Analyze(_ context.Context, _, _ string) ([]aihandlers.ElementInfo, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.suggestions, nil
}

// =============================================================================
// TakePreviewScreenshot
// =============================================================================

func TestService_TakePreviewScreenshot_Happy(t *testing.T) {
	runner := aihandlers.NewMockAutomationRunner()
	runner.Outcomes = []autocontracts.StepOutcome{
		{Success: true, NodeID: "preview.navigate"},
		{
			Success:    true,
			NodeID:     "preview.screenshot",
			Screenshot: &autocontracts.Screenshot{Data: []byte{0x89, 0x50, 0x4E}},
			FinalURL:   "https://example.com/",
		},
	}
	client, _ := newServiceForTest(t, runner)

	resp, err := client.TakePreviewScreenshot(context.Background(), connect.NewRequest(&aiv1.TakePreviewScreenshotRequest{
		Url: "https://example.com",
	}))
	require.NoError(t, err)
	assert.Equal(t, []byte{0x89, 0x50, 0x4E}, resp.Msg.GetScreenshotPng())
	assert.Equal(t, "image/png", resp.Msg.GetContentType())
}

func TestService_TakePreviewScreenshot_RejectsEmptyURL(t *testing.T) {
	client, _ := newServiceForTest(t, aihandlers.NewMockAutomationRunner())

	_, err := client.TakePreviewScreenshot(context.Background(), connect.NewRequest(&aiv1.TakePreviewScreenshotRequest{Url: ""}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

// =============================================================================
// GetLinkPreview
// =============================================================================

func TestService_GetLinkPreview_Happy(t *testing.T) {
	client, spy := newServiceForTest(t, aihandlers.NewMockAutomationRunner())
	spy.data = &LinkPreviewData{Title: "Example", SiteName: "example.com"}
	spy.found = true

	resp, err := client.GetLinkPreview(context.Background(), connect.NewRequest(&aiv1.GetLinkPreviewRequest{
		Url: "https://example.com",
	}))
	require.NoError(t, err)
	assert.True(t, resp.Msg.GetFound())
	assert.Equal(t, "Example", resp.Msg.GetTitle())
	assert.Equal(t, []string{"https://example.com"}, spy.calls)
}

func TestService_GetLinkPreview_RejectsInvalidScheme(t *testing.T) {
	client, _ := newServiceForTest(t, aihandlers.NewMockAutomationRunner())

	_, err := client.GetLinkPreview(context.Background(), connect.NewRequest(&aiv1.GetLinkPreviewRequest{
		Url: "ftp://example.com",
	}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

// =============================================================================
// AnalyzeElements
// =============================================================================

func TestService_AnalyzeElements_RejectsEmptyURL(t *testing.T) {
	client, _ := newServiceForTest(t, aihandlers.NewMockAutomationRunner())
	_, err := client.AnalyzeElements(context.Background(), connect.NewRequest(&aiv1.AnalyzeElementsRequest{Url: ""}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestService_AnalyzeElements_HappyPath(t *testing.T) {
	runner := aihandlers.NewMockAutomationRunner()
	runner.Outcomes = []autocontracts.StepOutcome{
		{Success: true, NodeID: "analysis.navigate"},
		{Success: true, NodeID: "analysis.wait"},
		{
			Success: true,
			NodeID:  "analysis.evaluate",
			ExtractedData: map[string]any{
				"value": map[string]any{
					"elements": []any{
						map[string]any{
							"text": "Login", "tagName": "BUTTON", "type": "button",
							"selectors":   []any{map[string]any{"selector": "#login", "type": "id", "robustness": 0.9}},
							"boundingBox": map[string]any{"x": 1.0, "y": 2.0, "width": 100.0, "height": 40.0},
							"confidence":  0.85,
							"category":    "authentication",
							"attributes":  map[string]any{},
						},
					},
					"pageContext": map[string]any{"title": "Login", "url": "https://example.com"},
				},
			},
		},
		{
			Success:    true,
			NodeID:     "analysis.screenshot",
			Screenshot: &autocontracts.Screenshot{Data: []byte{0x01, 0x02}, MediaType: "image/png"},
		},
	}
	client, _ := newServiceForTest(t, runner)

	resp, err := client.AnalyzeElements(context.Background(), connect.NewRequest(&aiv1.AnalyzeElementsRequest{
		Url: "https://example.com",
	}))
	require.NoError(t, err)
	assert.True(t, resp.Msg.GetSuccess())
	require.Len(t, resp.Msg.GetElements(), 1)
	assert.Equal(t, "Login", resp.Msg.GetElements()[0].GetText())
	assert.Equal(t, "Login", resp.Msg.GetPageContext().GetTitle())
}

// =============================================================================
// GetElementAtCoordinate
// =============================================================================

func TestService_GetElementAtCoordinate_RejectsNegative(t *testing.T) {
	client, _ := newServiceForTest(t, aihandlers.NewMockAutomationRunner())
	_, err := client.GetElementAtCoordinate(context.Background(), connect.NewRequest(&aiv1.GetElementAtCoordinateRequest{
		Url: "https://example.com",
		X:   -1, Y: 0,
	}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestService_GetElementAtCoordinate_RejectsEmptyURL(t *testing.T) {
	client, _ := newServiceForTest(t, aihandlers.NewMockAutomationRunner())
	_, err := client.GetElementAtCoordinate(context.Background(), connect.NewRequest(&aiv1.GetElementAtCoordinateRequest{Url: ""}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

// =============================================================================
// AIAnalyzeElements
// =============================================================================

func TestService_AIAnalyzeElements_Happy(t *testing.T) {
	client, _ := newServiceForTest(t, aihandlers.NewMockAutomationRunner())
	resp, err := client.AIAnalyzeElements(context.Background(), connect.NewRequest(&aiv1.AIAnalyzeElementsRequest{
		Url:    "https://example.com",
		Intent: "find the search button",
	}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.GetSuggestions(), 1)
	assert.Equal(t, "Search", resp.Msg.GetSuggestions()[0].GetText())
}

func TestService_AIAnalyzeElements_RejectsEmptyIntent(t *testing.T) {
	client, _ := newServiceForTest(t, aihandlers.NewMockAutomationRunner())
	_, err := client.AIAnalyzeElements(context.Background(), connect.NewRequest(&aiv1.AIAnalyzeElementsRequest{
		Url: "https://example.com", Intent: "",
	}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

// =============================================================================
// GetDOMTree
// =============================================================================

func TestService_GetDOMTree_Happy(t *testing.T) {
	runner := aihandlers.NewMockAutomationRunner()
	runner.Outcomes = []autocontracts.StepOutcome{
		{Success: true, NodeID: "dom.navigate"},
		{Success: true, NodeID: "dom.wait"},
		{
			Success: true,
			NodeID:  "dom.extract",
			ExtractedData: map[string]any{
				"value": map[string]any{"tagName": "BODY", "id": "root"},
			},
		},
	}
	client, _ := newServiceForTest(t, runner)

	resp, err := client.GetDOMTree(context.Background(), connect.NewRequest(&aiv1.GetDOMTreeRequest{
		Url: "https://example.com",
	}))
	require.NoError(t, err)
	require.NotNil(t, resp.Msg.GetTree())
	fields := resp.Msg.GetTree().AsMap()
	assert.Equal(t, "BODY", fields["tagName"])
}

func TestService_GetDOMTree_RejectsEmptyURL(t *testing.T) {
	client, _ := newServiceForTest(t, aihandlers.NewMockAutomationRunner())
	_, err := client.GetDOMTree(context.Background(), connect.NewRequest(&aiv1.GetDOMTreeRequest{Url: ""}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

// Compile-time guard that the unused errors var doesn't escape.
var _ = errors.New
