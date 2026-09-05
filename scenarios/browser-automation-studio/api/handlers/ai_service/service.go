package ai_service

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	aihandlers "github.com/vrooli/browser-automation-studio/handlers/ai"
	"github.com/vrooli/browser-automation-studio/middleware"
	"github.com/vrooli/browser-automation-studio/services/entitlement"
	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/ai"
)

type service struct {
	deps Deps
}

// =============================================================================
// TakePreviewScreenshot
// =============================================================================

func (s *service) TakePreviewScreenshot(
	ctx context.Context,
	req *connect.Request[aiv1.TakePreviewScreenshotRequest],
) (*connect.Response[aiv1.TakePreviewScreenshotResponse], error) {
	msg := req.Msg
	if msg.GetUrl() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errMissingURL)
	}

	args := aihandlers.PreviewScreenshotArgs{URL: msg.GetUrl()}
	if vp := msg.GetViewport(); vp != nil {
		args.ViewportWidth = int(vp.GetWidth())
		args.ViewportHeight = int(vp.GetHeight())
		if vp.DeviceScaleFactor != nil {
			args.DeviceScaleFactor = vp.GetDeviceScaleFactor()
		}
	}

	result, err := s.deps.Screenshot.RunPreviewScreenshot(ctx, args)
	if err != nil {
		return nil, mapHandlerError(err)
	}

	consoleLogs := make([]*aiv1.ConsoleLog, 0, len(result.ConsoleLogs))
	for _, l := range result.ConsoleLogs {
		consoleLogs = append(consoleLogs, &aiv1.ConsoleLog{
			Level:     l.Level,
			Message:   l.Message,
			Timestamp: timestamppb.New(l.Timestamp),
		})
	}

	events := make([]*structpb.Struct, 0, len(result.Events))
	for _, ev := range result.Events {
		raw, mErr := json.Marshal(ev)
		if mErr != nil {
			continue
		}
		var asMap map[string]any
		if err := json.Unmarshal(raw, &asMap); err != nil {
			continue
		}
		pb, sErr := structpb.NewStruct(asMap)
		if sErr != nil {
			continue
		}
		events = append(events, pb)
	}

	return connect.NewResponse(&aiv1.TakePreviewScreenshotResponse{
		ScreenshotPng:  result.ScreenshotPNG,
		ContentType:    result.ContentType,
		ConsoleLogs:    consoleLogs,
		Url:            result.URL,
		CapturedAt:     timestamppb.New(result.CapturedAt),
		DurationMs:     result.DurationMS,
		ViewportWidth:  int32(result.ViewportWidth),
		ViewportHeight: int32(result.ViewportHeight),
		Events:         events,
	}), nil
}

// =============================================================================
// GetLinkPreview
// =============================================================================

func (s *service) GetLinkPreview(
	ctx context.Context,
	req *connect.Request[aiv1.GetLinkPreviewRequest],
) (*connect.Response[aiv1.GetLinkPreviewResponse], error) {
	rawURL := req.Msg.GetUrl()
	if rawURL == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errMissingURL)
	}
	if parsed, perr := url.Parse(rawURL); perr != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidURL)
	}

	preview, found, err := s.deps.LinkPreview(ctx, rawURL)
	if err != nil {
		return nil, mapHandlerError(err)
	}
	if !found || preview == nil {
		return connect.NewResponse(&aiv1.GetLinkPreviewResponse{Found: false}), nil
	}
	return connect.NewResponse(&aiv1.GetLinkPreviewResponse{
		Found:       true,
		Title:       preview.Title,
		Description: preview.Description,
		Image:       preview.Image,
		Favicon:     preview.Favicon,
		SiteName:    preview.SiteName,
	}), nil
}

// =============================================================================
// AnalyzeElements
// =============================================================================

func (s *service) AnalyzeElements(
	ctx context.Context,
	req *connect.Request[aiv1.AnalyzeElementsRequest],
) (*connect.Response[aiv1.AnalyzeElementsResponse], error) {
	if req.Msg.GetUrl() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errMissingURL)
	}

	hasBYOK := middleware.HasBYOKKey(ctx)
	userID := entitlement.UserIdentityFromContext(ctx)

	result, err := s.deps.ElementAnalysis.RunAnalyzeElements(ctx, req.Msg.GetUrl(), userID, hasBYOK)
	if err != nil {
		return nil, mapHandlerError(err)
	}

	return connect.NewResponse(&aiv1.AnalyzeElementsResponse{
		Success:       true,
		Elements:      elementInfosToProto(result.Elements),
		AiSuggestions: aiSuggestionsToProto(result.AISuggestions),
		PageContext:   pageContextToProto(result.PageContext),
		Screenshot:    result.Screenshot,
		CapturedAt:    timestamppb.New(result.CapturedAt),
	}), nil
}

// =============================================================================
// GetElementAtCoordinate
// =============================================================================

func (s *service) GetElementAtCoordinate(
	ctx context.Context,
	req *connect.Request[aiv1.GetElementAtCoordinateRequest],
) (*connect.Response[aiv1.GetElementAtCoordinateResponse], error) {
	msg := req.Msg
	if msg.GetUrl() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errMissingURL)
	}
	if msg.GetX() < 0 || msg.GetY() < 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("x and y must be non-negative"))
	}

	selection, err := s.deps.ElementAnalysis.RunGetElementAtCoordinate(ctx, msg.GetUrl(), int(msg.GetX()), int(msg.GetY()))
	if err != nil {
		return nil, mapHandlerError(err)
	}
	return connect.NewResponse(&aiv1.GetElementAtCoordinateResponse{
		Selection: elementSelectionResultToProto(selection),
	}), nil
}

// =============================================================================
// AIAnalyzeElements
// =============================================================================

func (s *service) AIAnalyzeElements(
	ctx context.Context,
	req *connect.Request[aiv1.AIAnalyzeElementsRequest],
) (*connect.Response[aiv1.AIAnalyzeElementsResponse], error) {
	msg := req.Msg
	if msg.GetUrl() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errMissingURL)
	}
	if msg.GetIntent() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errMissingIntent)
	}

	hasBYOK := middleware.HasBYOKKey(ctx)
	userID := entitlement.UserIdentityFromContext(ctx)

	suggestions, err := s.deps.AIAnalysis.RunAIAnalyze(ctx, msg.GetUrl(), msg.GetIntent(), userID, hasBYOK)
	if err != nil {
		return nil, mapHandlerError(err)
	}
	return connect.NewResponse(&aiv1.AIAnalyzeElementsResponse{
		Suggestions: elementInfosToProto(suggestions),
	}), nil
}

// =============================================================================
// GetDOMTree
// =============================================================================

func (s *service) GetDOMTree(
	ctx context.Context,
	req *connect.Request[aiv1.GetDOMTreeRequest],
) (*connect.Response[aiv1.GetDOMTreeResponse], error) {
	if req.Msg.GetUrl() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errMissingURL)
	}

	raw, err := s.deps.DOM.GetDOMTreeJSON(ctx, req.Msg.GetUrl())
	if err != nil {
		return nil, mapHandlerError(err)
	}

	var asMap map[string]any
	if err := json.Unmarshal([]byte(raw), &asMap); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errDOMNotObject)
	}
	pb, err := structpb.NewStruct(asMap)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&aiv1.GetDOMTreeResponse{Tree: pb}), nil
}

// =============================================================================
// Error mapping
// =============================================================================

func mapHandlerError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, aihandlers.ErrMissingURL):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, aihandlers.ErrMissingIntent):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, aihandlers.ErrAutomationRunnerNotReady):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}
	var creditErr *aihandlers.CreditCheckError
	if errors.As(err, &creditErr) {
		if creditErr.IsInsufficientCredits() {
			return connect.NewError(connect.CodeResourceExhausted, creditErr)
		}
		return connect.NewError(connect.CodePermissionDenied, creditErr)
	}
	return connect.NewError(connect.CodeInternal, err)
}

// =============================================================================
// Proto conversions
// =============================================================================

func elementInfosToProto(in []aihandlers.ElementInfo) []*aiv1.ElementInfo {
	out := make([]*aiv1.ElementInfo, 0, len(in))
	for i := range in {
		out = append(out, elementInfoToProto(&in[i]))
	}
	return out
}

func elementInfoToProto(in *aihandlers.ElementInfo) *aiv1.ElementInfo {
	if in == nil {
		return nil
	}
	return &aiv1.ElementInfo{
		Text:        in.Text,
		TagName:     in.TagName,
		Type:        in.Type,
		Selectors:   selectorsToProto(in.Selectors),
		BoundingBox: rectangleToProto(in.BoundingBox),
		Confidence:  in.Confidence,
		Category:    in.Category,
		Attributes:  copyStringMap(in.Attributes),
	}
}

func selectorsToProto(in []aihandlers.SelectorOption) []*aiv1.SelectorOption {
	out := make([]*aiv1.SelectorOption, 0, len(in))
	for _, s := range in {
		out = append(out, &aiv1.SelectorOption{
			Selector:   s.Selector,
			Type:       s.Type,
			Robustness: s.Robustness,
			Fallback:   s.Fallback,
		})
	}
	return out
}

func rectangleToProto(r aihandlers.Rectangle) *aiv1.Rectangle {
	return &aiv1.Rectangle{X: r.X, Y: r.Y, Width: r.Width, Height: r.Height}
}

func aiSuggestionsToProto(in []aihandlers.AISuggestion) []*aiv1.AISuggestion {
	out := make([]*aiv1.AISuggestion, 0, len(in))
	for _, s := range in {
		out = append(out, &aiv1.AISuggestion{
			Action:      s.Action,
			Description: s.Description,
			ElementText: s.ElementText,
			Selector:    s.Selector,
			Confidence:  s.Confidence,
			Category:    s.Category,
			Reasoning:   s.Reasoning,
		})
	}
	return out
}

func pageContextToProto(p aihandlers.PageContext) *aiv1.PageContext {
	return &aiv1.PageContext{
		Title:       p.Title,
		Url:         p.URL,
		HasLogin:    p.HasLogin,
		HasSearch:   p.HasSearch,
		FormCount:   int32(p.FormCount),
		ButtonCount: int32(p.ButtonCount),
		LinkCount:   int32(p.LinkCount),
	}
}

func elementSelectionResultToProto(s *aihandlers.ElementSelectionResult) *aiv1.ElementSelectionResult {
	if s == nil {
		return nil
	}
	candidates := make([]*aiv1.ElementHierarchyEntry, 0, len(s.Candidates))
	for _, c := range s.Candidates {
		candidates = append(candidates, &aiv1.ElementHierarchyEntry{
			Element:     elementInfoToProto(c.Element),
			Selector:    c.Selector,
			Depth:       int32(c.Depth),
			Path:        append([]string(nil), c.Path...),
			PathSummary: c.PathSummary,
		})
	}
	return &aiv1.ElementSelectionResult{
		Element:       elementInfoToProto(s.Element),
		Candidates:    candidates,
		SelectedIndex: int32(s.SelectedIndex),
	}
}

func copyStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
