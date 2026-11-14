package driver

import (
	"context"
	"encoding/json"
)

type PublicCaptureRequest struct {
	URL    string `json:"url"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
	Theme  string `json:"theme,omitempty"`
}

type FileCaptureRequest struct {
	HTML     string `json:"html"`
	Filename string `json:"filename"`
	Width    int    `json:"width,omitempty"`
	Height   int    `json:"height,omitempty"`
	Theme    string `json:"theme,omitempty"`
}

type ScenarioCaptureRequest struct {
	URL                string   `json:"url"`
	Scenario           string   `json:"scenario"`
	Route              string   `json:"route"`
	Width              int      `json:"width,omitempty"`
	Height             int      `json:"height,omitempty"`
	Steps              []string `json:"steps,omitempty"`
	WaitSelector       string   `json:"wait_selector,omitempty"`
	WaitHiddenSelector string   `json:"wait_hidden_selector,omitempty"`
	Theme              string   `json:"theme,omitempty"`
	StrictTokens       bool     `json:"strict_tokens,omitempty"`
	AllowedOrigins     []string `json:"allowed_origins,omitempty"`
	EvidenceProfile    string   `json:"evidence_profile,omitempty"`
	RegionSelector     string   `json:"region_selector,omitempty"`
	FullPagePixels     bool     `json:"full_page_pixels,omitempty"`
}

// CapturePublicPage requests one isolated EX9 public URL capture from the
// Playwright driver. The response is the versioned bundle result JSON.
func (c *Client) CapturePublicPage(ctx context.Context, request *PublicCaptureRequest) (json.RawMessage, error) {
	var response json.RawMessage
	if err := c.post(ctx, "/ux-capture/public", request, &response); err != nil {
		return nil, err
	}
	if len(response) == 0 || !json.Valid(response) {
		return nil, &Error{Op: "POST /ux-capture/public", URL: c.baseURL, Message: "driver returned an invalid capture bundle"}
	}
	return response, nil
}

// CaptureHTMLFile sends bounded file contents to the local driver. Paths never cross this API.
func (c *Client) CaptureHTMLFile(ctx context.Context, request *FileCaptureRequest) (json.RawMessage, error) {
	var response json.RawMessage
	if err := c.post(ctx, "/ux-capture/file", request, &response); err != nil {
		return nil, err
	}
	if len(response) == 0 || !json.Valid(response) {
		return nil, &Error{Op: "POST /ux-capture/file", URL: c.baseURL, Message: "driver returned an invalid capture bundle"}
	}
	return response, nil
}

// CaptureScenarioPage captures a target that the BAS API resolved from the local scenario registry.
func (c *Client) CaptureScenarioPage(ctx context.Context, request *ScenarioCaptureRequest) (json.RawMessage, error) {
	var response json.RawMessage
	if err := c.post(ctx, "/ux-capture/scenario", request, &response); err != nil {
		return nil, err
	}
	if len(response) == 0 || !json.Valid(response) {
		return nil, &Error{Op: "POST /ux-capture/scenario", URL: c.baseURL, Message: "driver returned an invalid capture bundle"}
	}
	return response, nil
}
