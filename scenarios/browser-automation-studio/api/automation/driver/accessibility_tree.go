package driver

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// AccessibilityTreeNode is one safe role/name/state item in the normalized tree.
type AccessibilityTreeNode struct {
	Ref         string                  `json:"ref,omitempty"`
	Role        string                  `json:"role,omitempty"`
	Name        string                  `json:"name,omitempty"`
	Description string                  `json:"description,omitempty"`
	States      []string                `json:"states,omitempty"`
	Children    []AccessibilityTreeNode `json:"children"`
}

type AccessibilityTreeViewport struct {
	Width             int `json:"width"`
	Height            int `json:"height"`
	DeviceScaleFactor int `json:"deviceScaleFactor"`
}

// AccessibilityTreeLayout contains static driver-owned viewport-fit measurements.
// It exposes no page script, DOM nodes, or style values.
type AccessibilityTreeLayout struct {
	ViewportWidthPx  int `json:"viewportWidthPx"`
	DocumentWidthPx  int `json:"documentWidthPx"`
	ViewportHeightPx int `json:"viewportHeightPx"`
	DocumentHeightPx int `json:"documentHeightPx"`
}

type AccessibilityTree struct {
	Contract   string                    `json:"contract"`
	URL        string                    `json:"url"`
	Viewport   AccessibilityTreeViewport `json:"viewport"`
	Layout     *AccessibilityTreeLayout  `json:"layout,omitempty"`
	CapturedAt string                    `json:"captured_at"`
	NodeCount  int                       `json:"node_count"`
	Truncated  bool                      `json:"truncated"`
	Root       AccessibilityTreeNode     `json:"root"`
}

// AccessibilityTreeResponse is a bounded, normalized observation of the
// current active page. It deliberately carries no CDP endpoint or node handle.
type AccessibilityTreeResponse struct {
	SessionID       string             `json:"session_id"`
	PageID          string             `json:"page_id"`
	Tree            *AccessibilityTree `json:"tree"`
	EvidenceWarning string             `json:"evidence_warning,omitempty"`
}

// GetAccessibilityTree reads only the exact active page selected by the
// immutable session lease. The owner route returns the redacted tree contract.
func (c *Client) GetAccessibilityTree(ctx context.Context, sessionID, executionID, leaseID, expectedPageID string) (*AccessibilityTreeResponse, error) {
	if strings.TrimSpace(sessionID) == "" || strings.TrimSpace(expectedPageID) == "" {
		return nil, errors.New("accessibility observation requires session ID and active page ID")
	}
	if err := validateSessionOwner(executionID, leaseID); err != nil {
		return nil, err
	}
	query := url.Values{"expected_page_id": {expectedPageID}}
	path := fmt.Sprintf("/session/%s/accessibility-tree?%s", url.PathEscape(sessionID), query.Encode())
	var response AccessibilityTreeResponse
	if err := c.getWithSessionOwner(ctx, path, executionID, leaseID, &response); err != nil {
		return nil, err
	}
	if response.SessionID != sessionID || response.PageID != expectedPageID || response.Tree == nil || response.Tree.Contract != "bas-accessibility-tree-observation/v1" {
		return nil, errors.New("driver returned an accessibility observation for a different session or page")
	}
	return &response, nil
}
