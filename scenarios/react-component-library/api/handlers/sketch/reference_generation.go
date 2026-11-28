package sketch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
	internalSketch "react-component-library/internal/sketch"
)

var referenceTargetSegment = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

type referenceGenerationHTTP struct {
	repoRoot string
	resolve  func(context.Context) (string, error)
	client   *http.Client
	logger   *log.Logger
}

type generateReferenceRequest struct {
	Scenario string `json:"scenario"`
	Page     string `json:"page"`
	Prompt   string `json:"prompt"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Seed     int64  `json:"seed"`
}

type generateReferenceResponse struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Size     int64    `json:"size"`
	MIME     string   `json:"mime"`
	URL      string   `json:"url"`
	JobID    string   `json:"jobId"`
	ModelID  string   `json:"modelId"`
	Tier     string   `json:"tier"`
	Warnings []string `json:"warnings,omitempty"`
}

type imageToolsSubmitResponse struct {
	JobID        string   `json:"job_id"`
	JobIDCamel   string   `json:"jobId"`
	ModelID      string   `json:"model_id"`
	ModelIDCamel string   `json:"modelId"`
	Tier         string   `json:"tier"`
	Warnings     []string `json:"warnings"`
}

type imageToolsJobResponse struct {
	Job struct {
		State     string `json:"state"`
		ResultRef string `json:"resultRef"`
		Error     string `json:"error"`
		Message   string `json:"message"`
	} `json:"job"`
}

func newReferenceGenerationHTTP(repoRoot string, logger *log.Logger) *referenceGenerationHTTP {
	if logger == nil {
		logger = log.Default()
	}
	return &referenceGenerationHTTP{
		repoRoot: repoRoot,
		resolve: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "image-tools")
		},
		client: &http.Client{Timeout: 3 * time.Minute},
		logger: logger,
	}
}

func (h *referenceGenerationHTTP) generate(w http.ResponseWriter, r *http.Request) {
	var req generateReferenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid generation request")
		return
	}
	if !referenceTargetSegment.MatchString(req.Scenario) || !referenceTargetSegment.MatchString(req.Page) {
		h.writeError(w, http.StatusBadRequest, "invalid reference target")
		return
	}
	req.Prompt = strings.TrimSpace(req.Prompt)
	if len(req.Prompt) < 20 {
		h.writeError(w, http.StatusBadRequest, "prompt must contain at least 20 characters")
		return
	}
	if req.Width == 0 {
		req.Width = 512
	}
	if req.Height == 0 {
		req.Height = 512
	}
	if req.Width < 64 || req.Width > 768 || req.Height < 64 || req.Height > 768 {
		h.writeError(w, http.StatusBadRequest, "reference dimensions must be between 64 and 768 pixels")
		return
	}

	base, err := h.resolve(r.Context())
	if err != nil {
		h.writeError(w, http.StatusServiceUnavailable, "image-tools is unavailable: "+err.Error())
		return
	}
	submit, err := h.submit(r.Context(), strings.TrimRight(base, "/"), req)
	if err != nil {
		h.writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	job, err := h.wait(r.Context(), strings.TrimRight(base, "/"), submit.JobID)
	if err != nil {
		h.writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	if job.Job.State != "JOB_STATE_SUCCEEDED" || job.Job.ResultRef == "" {
		detail := strings.TrimSpace(job.Job.Error)
		if detail == "" {
			detail = strings.TrimSpace(job.Job.Message)
		}
		if detail == "" {
			detail = "image-tools did not produce a reference image"
		}
		h.writeError(w, http.StatusBadGateway, detail)
		return
	}

	asset, err := h.fetchAndStore(r.Context(), strings.TrimRight(base, "/"), req, job.Job.ResultRef)
	if err != nil {
		h.writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	asset.JobID = submit.JobID
	asset.ModelID = submit.ModelID
	asset.Tier = submit.Tier
	asset.Warnings = submit.Warnings
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(asset)
}

func (h *referenceGenerationHTTP) submit(ctx context.Context, base string, req generateReferenceRequest) (imageToolsSubmitResponse, error) {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	params := map[string]any{
		"prompt":          req.Prompt,
		"width":           req.Width,
		"height":          req.Height,
		"steps":           8,
		"seed":            req.Seed,
		"variations":      1,
		"fallback_policy": "local_only",
	}
	raw, err := json.Marshal(params)
	if err != nil {
		return imageToolsSubmitResponse{}, err
	}
	if err := mw.WriteField("params", string(raw)); err != nil {
		return imageToolsSubmitResponse{}, err
	}
	if err := mw.Close(); err != nil {
		return imageToolsSubmitResponse{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/v1/ai/text_to_image", &body)
	if err != nil {
		return imageToolsSubmitResponse{}, err
	}
	httpReq.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := h.client.Do(httpReq)
	if err != nil {
		return imageToolsSubmitResponse{}, fmt.Errorf("submit image generation: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return imageToolsSubmitResponse{}, fmt.Errorf("image-tools rejected generation (%d): %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	var out imageToolsSubmitResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return imageToolsSubmitResponse{}, fmt.Errorf("image-tools returned an incomplete generation job")
	}
	if out.JobID == "" {
		out.JobID = out.JobIDCamel
	}
	if out.ModelID == "" {
		out.ModelID = out.ModelIDCamel
	}
	if out.JobID == "" {
		return imageToolsSubmitResponse{}, fmt.Errorf("image-tools returned an incomplete generation job")
	}
	return out, nil
}

func (h *referenceGenerationHTTP) wait(ctx context.Context, base, jobID string) (imageToolsJobResponse, error) {
	payload, _ := json.Marshal(map[string]string{"id": jobID})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/vrooli.image_tools.v1.jobs.JobsService/WaitJob", bytes.NewReader(payload))
	if err != nil {
		return imageToolsJobResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.client.Do(req)
	if err != nil {
		return imageToolsJobResponse{}, fmt.Errorf("wait for image generation: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return imageToolsJobResponse{}, fmt.Errorf("image-tools wait failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	var out imageToolsJobResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return imageToolsJobResponse{}, fmt.Errorf("decode image-tools job: %w", err)
	}
	return out, nil
}

func (h *referenceGenerationHTTP) fetchAndStore(ctx context.Context, base string, req generateReferenceRequest, resultRef string) (generateReferenceResponse, error) {
	if resultRef == "" || strings.HasPrefix(resultRef, "/") || strings.Contains(resultRef, "..") {
		return generateReferenceResponse{}, fmt.Errorf("image-tools returned an invalid result reference")
	}
	resultURL := base + "/api/v1/blobs/" + strings.TrimLeft(resultRef, "/")
	getReq, err := http.NewRequestWithContext(ctx, http.MethodGet, resultURL, nil)
	if err != nil {
		return generateReferenceResponse{}, err
	}
	resp, err := h.client.Do(getReq)
	if err != nil {
		return generateReferenceResponse{}, fmt.Errorf("fetch generated reference: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return generateReferenceResponse{}, fmt.Errorf("image-tools result blob returned %d", resp.StatusCode)
	}
	name := "generated-reference.png"
	id, storedName, size, err := internalSketch.SaveReferenceAsset(h.repoRoot, req.Scenario, req.Page, name, "image/png", resp.Body)
	if err != nil {
		return generateReferenceResponse{}, fmt.Errorf("store generated reference: %w", err)
	}
	return generateReferenceResponse{
		ID:   id,
		Name: storedName,
		Size: size,
		MIME: "image/png",
		URL:  "/api/v1/reference-assets/" + url.PathEscape(req.Scenario) + "/" + url.PathEscape(req.Page) + "/" + url.PathEscape(storedName),
	}, nil
}

func (h *referenceGenerationHTTP) writeError(w http.ResponseWriter, status int, message string) {
	h.logger.Printf("reference generation: %s", message)
	http.Error(w, message, status)
}
