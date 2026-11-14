package retention

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OwnerReclaimer asks an owner to reclaim its own non-regenerable data. It is
// the storage-manager backstop for data it must never delete itself: the owner
// knows which rows and files are safe to drop, storage-manager only knows the
// budget is breached.
type OwnerReclaimer interface {
	Reclaim(ctx context.Context, owner, operation string) (json.RawMessage, error)
}

// OwnerReclaimOutcome records one backstop request and what it achieved.
type OwnerReclaimOutcome struct {
	Operation   string          `json:"operation"`
	RequestedAt time.Time       `json:"requested_at"`
	Receipt     json.RawMessage `json:"receipt,omitempty"`
	Error       string          `json:"error,omitempty"`
	BeforeBytes int64           `json:"before_bytes"`
	AfterBytes  int64           `json:"after_bytes"`
	BudgetBytes int64           `json:"budget_bytes"`
	// StillOver means the owner answered but its data is still above budget:
	// the breach is escalated rather than retried every cycle.
	StillOver bool `json:"still_over"`
}

// DefaultOwnerReclaimInterval spaces repeated backstop requests for one entry.
// An owner's reclaim can be a compaction that takes minutes and rewrites a
// whole database, so asking every retention cycle would be its own write storm.
const DefaultOwnerReclaimInterval = 6 * time.Hour

// ownerReclaimTimeout bounds one request. An owner doing long work should
// accept the request and return its receipt promptly; the next cycle measures
// the result either way.
const ownerReclaimTimeout = 15 * time.Minute

const maxOwnerReclaimReceiptBytes = 64 << 10

// HTTPOwnerReclaimer POSTs {"dry_run":false} to the owner's declared
// operation path on its discovered API URL.
type HTTPOwnerReclaimer struct {
	ResolveURL func(context.Context, string) (string, error)
	HTTPClient *http.Client
	// Header supplies owner-specific authentication; nil sends none.
	Header func(owner string) http.Header
}

func (r *HTTPOwnerReclaimer) Reclaim(ctx context.Context, owner, operation string) (json.RawMessage, error) {
	if r == nil || r.ResolveURL == nil {
		return nil, fmt.Errorf("owner reclaim client unavailable")
	}
	if !strings.HasPrefix(operation, "/api/") {
		return nil, fmt.Errorf("owner reclaim operation %q must be an /api/ path", operation)
	}
	base, err := r.ResolveURL(ctx, owner)
	if err != nil {
		return nil, fmt.Errorf("owner scenario unreachable: %w", err)
	}
	ctx, cancel := context.WithTimeout(ctx, ownerReclaimTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(base, "/")+operation, bytes.NewReader([]byte(`{"dry_run":false}`)))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	if r.Header != nil {
		for key, values := range r.Header(owner) {
			for _, value := range values {
				request.Header.Add(key, value)
			}
		}
	}
	client := r.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("owner reclaim request: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, maxOwnerReclaimReceiptBytes))
	if err != nil {
		return nil, fmt.Errorf("read owner reclaim receipt: %w", err)
	}
	receipt := json.RawMessage(body)
	if !json.Valid(body) {
		quoted, _ := json.Marshal(strings.TrimSpace(string(body)))
		receipt = quoted
	}
	if response.StatusCode == http.StatusNotFound {
		return receipt, fmt.Errorf("owner scenario does not implement %s", operation)
	}
	if response.StatusCode >= http.StatusBadRequest {
		return receipt, fmt.Errorf("owner reclaim returned HTTP %d", response.StatusCode)
	}
	return receipt, nil
}
