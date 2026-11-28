package workspaces

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	executionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	diffv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/diff"
	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace"
	"google.golang.org/protobuf/encoding/protojson"

	domain "react-component-library/internal/workspaces"
)

const (
	maxBASScreenshotBytes = 32 << 20
	maxImageToolsResponse = 1 << 20
)

var heatmapReferencePattern = regexp.MustCompile(`^diff/([0-9a-fA-F-]{36})\.png$`)

type VisualCompareRunner interface {
	Compare(context.Context, domain.CaptureReceipt, domain.CaptureReceipt) (*diffv1.DiffResult, error)
}

type ImageBytesCompareRunner interface {
	CompareImages(context.Context, []byte, []byte) (*diffv1.DiffResult, error)
}

func mountVisualComparison(router *mux.Router, handler *Handler) {
	router.HandleFunc("/api/workspaces/{id}/states/{stateId}/visual-comparisons/{comparisonId}/heatmap", handler.getVisualComparisonHeatmap).Methods(http.MethodGet)
}

func (h *Handler) getVisualComparisonHeatmap(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workspaceID, stateID, comparisonID := vars["id"], vars["stateId"], vars["comparisonId"]
	if !uuidLikeWorkspaceID(workspaceID) || !uuidLikeWorkspaceID(stateID) || !uuidLikeWorkspaceID(comparisonID) {
		http.Error(w, "invalid workspace visual comparison identity", http.StatusBadRequest)
		return
	}
	record, err := h.repository.GetVisualComparison(r.Context(), workspaceID, stateID, comparisonID)
	if errors.Is(err, sql.ErrNoRows) {
		sourceRecord, sourceErr := h.repository.GetSourceVisualComparison(r.Context(), workspaceID, stateID, comparisonID)
		if errors.Is(sourceErr, sql.ErrNoRows) {
			http.Error(w, "visual comparison not found", http.StatusNotFound)
			return
		}
		if sourceErr != nil {
			h.captureWorkspaceError(w, "load source visual comparison heatmap", sourceErr)
			return
		}
		record = domain.VisualComparison{HeatmapRef: sourceRecord.HeatmapRef}
		err = nil
	}
	if err != nil {
		h.captureWorkspaceError(w, "load visual comparison heatmap", err)
		return
	}
	fetcher, ok := h.comparer.(interface {
		FetchHeatmap(context.Context, string) ([]byte, error)
	})
	if !ok || record.HeatmapRef == "" {
		http.Error(w, "visual comparison heatmap is unavailable", http.StatusNotFound)
		return
	}
	data, err := fetcher.FetchHeatmap(r.Context(), record.HeatmapRef)
	if err != nil {
		h.logger.Printf("workspaces: fetch visual comparison heatmap: %v", err)
		http.Error(w, "visual comparison heatmap is unavailable", http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Length", fmt.Sprint(len(data)))
	w.Header().Set("Cache-Control", "private, max-age=300")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write(data)
}

func (h *Handler) getCaptureReceiptScreenshot(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workspaceID, stateID, receiptID := vars["id"], vars["stateId"], vars["receiptId"]
	if !uuidLikeWorkspaceID(workspaceID) || !uuidLikeWorkspaceID(stateID) || !uuidLikeWorkspaceID(receiptID) {
		http.Error(w, "invalid capture receipt identity", http.StatusBadRequest)
		return
	}
	receipt, err := h.repository.GetCaptureReceipt(r.Context(), workspaceID, stateID, receiptID)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "capture receipt not found", http.StatusNotFound)
		return
	}
	if err != nil {
		h.captureWorkspaceError(w, "load capture receipt screenshot", err)
		return
	}
	fetcher, ok := h.comparer.(interface {
		FetchScreenshot(context.Context, domain.CaptureReceipt) ([]byte, error)
	})
	if !ok {
		http.Error(w, "capture screenshot is unavailable", http.StatusServiceUnavailable)
		return
	}
	data, err := fetcher.FetchScreenshot(r.Context(), receipt)
	if err != nil {
		h.logger.Printf("workspaces: resolve capture receipt screenshot: %v", err)
		http.Error(w, "capture screenshot is unavailable", http.StatusBadGateway)
		return
	}
	contentType := http.DetectContentType(data)
	if contentType != "image/png" && contentType != "image/jpeg" {
		http.Error(w, "capture screenshot has an unsupported format", http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", fmt.Sprint(len(data)))
	w.Header().Set("Cache-Control", "private, max-age=300")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write(data)
}

func workspaceHasState(workspace interface {
	GetPages() []*workspacev1.WorkspacePage
}, stateID string,
) bool {
	for _, page := range workspace.GetPages() {
		for _, display := range page.GetDisplays() {
			for _, state := range display.GetStates() {
				if state.GetId() == stateID {
					return true
				}
			}
		}
	}
	return false
}

type screenshotBytesResolver struct {
	client  *http.Client
	resolve func(context.Context, string) (string, error)
}

type imageToolsComparator struct {
	client  *http.Client
	resolve func(context.Context, string) (string, error)
}

type BASImageToolsCompare struct {
	Screenshots screenshotBytesResolver
	ImageTools  imageToolsComparator
}

func NewBASImageToolsCompare() *BASImageToolsCompare {
	client := &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	resolver := func(ctx context.Context, scenario string) (string, error) {
		return discovery.ResolveScenarioURLDefault(ctx, scenario)
	}
	return &BASImageToolsCompare{Screenshots: screenshotBytesResolver{client: client, resolve: resolver}, ImageTools: imageToolsComparator{client: client, resolve: resolver}}
}

func (c *BASImageToolsCompare) Compare(ctx context.Context, baseline, current domain.CaptureReceipt) (*diffv1.DiffResult, error) {
	base, err := c.Screenshots.Resolve(ctx, baseline)
	if err != nil {
		return nil, fmt.Errorf("resolve baseline screenshot: %w", err)
	}
	compare, err := c.Screenshots.Resolve(ctx, current)
	if err != nil {
		return nil, fmt.Errorf("resolve current screenshot: %w", err)
	}
	return c.ImageTools.Compare(ctx, base, compare)
}

func (c *BASImageToolsCompare) CompareImages(ctx context.Context, baseline, current []byte) (*diffv1.DiffResult, error) {
	return c.ImageTools.Compare(ctx, baseline, current)
}

func (c *BASImageToolsCompare) FetchScreenshot(ctx context.Context, receipt domain.CaptureReceipt) ([]byte, error) {
	return c.Screenshots.Resolve(ctx, receipt)
}

func (r screenshotBytesResolver) Resolve(ctx context.Context, receipt domain.CaptureReceipt) ([]byte, error) {
	var evidence workspaceCaptureEvidence
	if err := json.Unmarshal([]byte(receipt.EvidenceJSON), &evidence); err != nil || evidence.SchemaVersion != workspaceCaptureEvidenceSchema {
		return nil, fmt.Errorf("capture evidence is invalid")
	}
	reference, err := url.Parse(evidence.Screenshot.Reference)
	if err != nil || reference.Scheme != "bas-capture" || reference.User != nil || reference.Host == "" || reference.Host != receipt.CaptureID || reference.RawQuery != "" || reference.Fragment != "" || reference.RawPath != "" || reference.Path != "/screenshot" {
		return nil, fmt.Errorf("capture screenshot reference is invalid")
	}
	baseURL, err := r.resolve(ctx, "browser-automation-studio")
	if err != nil {
		return nil, fmt.Errorf("resolve browser automation studio: %w", err)
	}
	owner, err := validatedServiceURL(baseURL)
	if err != nil {
		return nil, err
	}
	requestJSON, err := protojson.Marshal(&api.GetExecutionScreenshotsRequest{ExecutionId: receipt.CaptureID})
	if err != nil {
		return nil, err
	}
	metadataURL := owner.ResolveReference(&url.URL{Path: "/browser_automation_studio.v1.ExecutionsService/GetExecutionScreenshots"})
	metadataRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, metadataURL.String(), bytes.NewReader(requestJSON))
	if err != nil {
		return nil, err
	}
	metadataRequest.Header.Set("Content-Type", "application/json")
	response, err := r.client.Do(metadataRequest)
	if err != nil {
		return nil, fmt.Errorf("request screenshot metadata: %w", err)
	}
	defer response.Body.Close()
	metadata, err := io.ReadAll(io.LimitReader(response.Body, 1<<20+1))
	if err != nil || len(metadata) > 1<<20 {
		return nil, fmt.Errorf("read screenshot metadata within limit")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("BAS screenshot metadata returned HTTP %d", response.StatusCode)
	}
	var listing executionv1.GetScreenshotsResponse
	if err := protojson.Unmarshal(metadata, &listing); err != nil || listing.GetExecutionId() != receipt.CaptureID {
		return nil, fmt.Errorf("BAS screenshot metadata does not match the receipt")
	}
	var target *executionv1.ExecutionScreenshot
	for _, item := range listing.GetScreenshots() {
		if item != nil && item.GetNodeId() == "target" && item.GetScreenshot() != nil {
			if target != nil {
				return nil, fmt.Errorf("capture has ambiguous target screenshots")
			}
			target = item
		}
	}
	if target == nil || (target.GetScreenshot().GetContentType() != "image/png" && target.GetScreenshot().GetContentType() != "image/jpeg") {
		return nil, fmt.Errorf("capture target screenshot is unavailable")
	}
	location, err := url.Parse(target.GetScreenshot().GetUrl())
	allowedPrefix := "/api/v1/screenshots/" + receipt.CaptureID + "/"
	if err != nil || location.IsAbs() || location.Host != "" || location.User != nil || location.RawQuery != "" || location.Fragment != "" || location.RawPath != "" || path.Clean(location.Path) != location.Path || !strings.HasPrefix(location.Path, allowedPrefix) || len(location.Path) <= len(allowedPrefix) {
		return nil, fmt.Errorf("BAS screenshot route is outside the capture owner")
	}
	screenshotURL := owner.ResolveReference(location)
	imageRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, screenshotURL.String(), nil)
	if err != nil {
		return nil, err
	}
	imageResponse, err := r.client.Do(imageRequest)
	if err != nil {
		return nil, fmt.Errorf("fetch BAS screenshot: %w", err)
	}
	defer imageResponse.Body.Close()
	if imageResponse.StatusCode < 200 || imageResponse.StatusCode >= 300 {
		return nil, fmt.Errorf("BAS screenshot returned HTTP %d", imageResponse.StatusCode)
	}
	if media := strings.ToLower(strings.TrimSpace(strings.Split(imageResponse.Header.Get("Content-Type"), ";")[0])); media != target.GetScreenshot().GetContentType() {
		return nil, fmt.Errorf("BAS screenshot content type differs from metadata")
	}
	data, err := io.ReadAll(io.LimitReader(imageResponse.Body, maxBASScreenshotBytes+1))
	if err != nil || len(data) == 0 || len(data) > maxBASScreenshotBytes {
		return nil, fmt.Errorf("BAS screenshot exceeds the allowed byte limit")
	}
	return data, nil
}

func (c imageToolsComparator) Compare(ctx context.Context, baseline, current []byte) (*diffv1.DiffResult, error) {
	baseURL, err := c.resolve(ctx, "image-tools")
	if err != nil {
		return nil, fmt.Errorf("resolve image-tools: %w", err)
	}
	endpoint, err := validatedServiceURL(baseURL)
	if err != nil {
		return nil, err
	}
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/api/v1/diff/compare"
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	writePart := func(field, filename string, content []byte) error {
		mediaType := http.DetectContentType(content)
		extension := ".png"
		if mediaType == "image/jpeg" {
			extension = ".jpg"
		} else if mediaType != "image/png" {
			return fmt.Errorf("captured screenshot is not a supported PNG or JPEG")
		}
		header := make(textproto.MIMEHeader)
		header.Set("Content-Disposition", fmt.Sprintf(`form-data; name=%q; filename=%q`, field, strings.TrimSuffix(filename, ".png")+extension))
		header.Set("Content-Type", mediaType)
		part, err := writer.CreatePart(header)
		if err != nil {
			return err
		}
		_, err = part.Write(content)
		return err
	}
	if err := writePart("base", "baseline.png", baseline); err != nil {
		return nil, err
	}
	if err := writePart("compare", "current.png", current); err != nil {
		return nil, err
	}
	if err := writer.WriteField("params", `{"includeHeatmap":true}`); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), &body)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response, err := c.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("call image-tools comparison: %w", err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, maxImageToolsResponse+1))
	if err != nil || len(responseBody) > maxImageToolsResponse {
		return nil, fmt.Errorf("read image-tools response within limit")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("image-tools comparison returned HTTP %d", response.StatusCode)
	}
	var result diffv1.DiffResult
	if err := protojson.Unmarshal(responseBody, &result); err != nil || result.GetHeatmapRef() == "" {
		return nil, fmt.Errorf("image-tools returned invalid comparison metadata or no heatmap")
	}
	return &result, nil
}

func (c *BASImageToolsCompare) FetchHeatmap(ctx context.Context, reference string) ([]byte, error) {
	match := heatmapReferencePattern.FindStringSubmatch(reference)
	if len(match) != 2 {
		return nil, fmt.Errorf("stored heatmap reference is invalid")
	}
	if _, err := uuid.Parse(match[1]); err != nil {
		return nil, fmt.Errorf("stored heatmap reference is invalid")
	}
	baseURL, err := c.ImageTools.resolve(ctx, "image-tools")
	if err != nil {
		return nil, fmt.Errorf("resolve image-tools: %w", err)
	}
	owner, err := validatedServiceURL(baseURL)
	if err != nil {
		return nil, err
	}
	location := &url.URL{Path: "/api/v1/blobs/" + reference}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, owner.ResolveReference(location).String(), nil)
	if err != nil {
		return nil, err
	}
	response, err := c.ImageTools.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("fetch image-tools heatmap: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("image-tools heatmap returned HTTP %d", response.StatusCode)
	}
	if media := strings.ToLower(strings.TrimSpace(strings.Split(response.Header.Get("Content-Type"), ";")[0])); media != "image/png" {
		return nil, fmt.Errorf("image-tools heatmap is not a PNG")
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, maxBASScreenshotBytes+1))
	if err != nil || len(data) == 0 || len(data) > maxBASScreenshotBytes {
		return nil, fmt.Errorf("image-tools heatmap exceeds the allowed byte limit")
	}
	return data, nil
}

func validatedServiceURL(raw string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, fmt.Errorf("service discovery returned an invalid endpoint")
	}
	return parsed, nil
}
