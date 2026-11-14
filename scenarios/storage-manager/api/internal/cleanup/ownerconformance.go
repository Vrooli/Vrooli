package cleanup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ConformanceT is the small testing seam used by RunOwnerConformance. It is
// intentionally compatible with testing.T and testing.B without importing the
// testing package into the runtime contract.
type ConformanceT interface {
	Helper()
	Errorf(format string, args ...any)
}

// RunOwnerConformance exercises the owner cleanup contract against an already
// running owner base URL. It is safe to run against a test server because the
// apply checks use unique idempotency keys and the final check sends an empty
// preview.
func RunOwnerConformance(t ConformanceT, base string) {
	t.Helper()
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		t.Errorf("owner cleanup base URL is empty")
		return
	}
	client := &http.Client{Timeout: 30 * time.Second}
	large, err := ownerConformanceRequest[OwnerEstimateResponse](client, base, http.MethodGet, "/api/v1/cleanup/estimate", map[string]string{"min_age_seconds": strconv.FormatInt(int64((30*24*time.Hour)/time.Second), 10)})
	if err != nil {
		t.Errorf("large-age estimate failed: %v", err)
		return
	}
	small, err := ownerConformanceRequest[OwnerEstimateResponse](client, base, http.MethodGet, "/api/v1/cleanup/estimate", map[string]string{"min_age_seconds": "3600"})
	if err != nil {
		t.Errorf("small-age estimate failed: %v", err)
		return
	}
	if large.EstimatedBytes >= small.EstimatedBytes && small.EstimatedBytes > 0 {
		t.Errorf("age filter is not monotonic: 30d=%d, 1h=%d", large.EstimatedBytes, small.EstimatedBytes)
	}

	preview, err := ownerConformanceRequest[OwnerPreviewResponse](client, base, http.MethodPost, "/api/v1/cleanup/preview", nil, OwnerPreviewRequest{Estimate: small})
	if err != nil {
		t.Errorf("preview failed: %v", err)
		return
	}
	for _, item := range preview.Items {
		if item.Protected {
			t.Errorf("preview returned protected item %q", item.ID)
		}
		if item.AgeSeconds < small.MinAgeSeconds {
			t.Errorf("preview returned item %q at age %d below requested %d", item.ID, item.AgeSeconds, small.MinAgeSeconds)
		}
	}

	applyRequest := OwnerApplyRequest{ProviderID: preview.ProviderID, Preview: preview, IdempotencyKey: "owner-conformance-" + strconv.FormatInt(time.Now().UnixNano(), 10), ApprovalMode: ApprovalModeOwner}
	first, err := ownerConformanceRequest[OwnerApplyResponse](client, base, http.MethodPost, "/api/v1/cleanup/apply", nil, applyRequest)
	if err != nil {
		t.Errorf("apply failed: %v", err)
		return
	}
	second, err := ownerConformanceRequest[OwnerApplyResponse](client, base, http.MethodPost, "/api/v1/cleanup/apply", nil, applyRequest)
	if err != nil {
		t.Errorf("idempotent repeat failed: %v", err)
		return
	}
	if first.ReclaimedBytes != second.ReclaimedBytes || !sameStrings(first.RemovedItemIDs, second.RemovedItemIDs) {
		t.Errorf("repeated idempotency key changed result: first=%#v second=%#v", first, second)
	}

	emptyRequest := OwnerApplyRequest{ProviderID: preview.ProviderID, Preview: OwnerPreviewResponse{ProviderID: preview.ProviderID}, IdempotencyKey: applyRequest.IdempotencyKey + "-empty", ApprovalMode: ApprovalModeOwner}
	empty, err := ownerConformanceRequest[OwnerApplyResponse](client, base, http.MethodPost, "/api/v1/cleanup/apply", nil, emptyRequest)
	if err != nil {
		t.Errorf("empty preview apply failed: %v", err)
		return
	}
	if empty.ReclaimedBytes != 0 || len(empty.RemovedItemIDs) != 0 {
		t.Errorf("empty preview reclaimed data: %#v", empty)
	}
}

func ownerConformanceRequest[T any](client *http.Client, base, method, path string, query map[string]string, body ...any) (T, error) {
	var zero T
	endpoint, err := url.Parse(base + path)
	if err != nil {
		return zero, err
	}
	values := endpoint.Query()
	for key, value := range query {
		values.Set(key, value)
	}
	endpoint.RawQuery = values.Encode()
	var requestBody io.Reader
	if len(body) > 0 {
		payload, marshalErr := json.Marshal(body[0])
		if marshalErr != nil {
			return zero, marshalErr
		}
		requestBody = bytes.NewReader(payload)
	}
	req, err := http.NewRequest(method, endpoint.String(), requestBody)
	if err != nil {
		return zero, err
	}
	if requestBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		return zero, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return zero, fmt.Errorf("%s %s returned HTTP %d: %s", method, path, resp.StatusCode, strings.TrimSpace(string(payload)))
	}
	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return zero, err
	}
	return result, nil
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
