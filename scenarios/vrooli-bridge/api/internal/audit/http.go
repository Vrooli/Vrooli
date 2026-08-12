package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// HTTPStore is the optional workspace-sandbox-backed audit store. The wire is
// intentionally ordinary JSON so the Bridge package does not import another
// scenario's proto module. The endpoint is unset by default and the SQLite
// store remains the local fallback.
type HTTPStore struct {
	Endpoint string
	Client   *http.Client
}

var _ Store = (*HTTPStore)(nil)

func (s *HTTPStore) Append(ctx context.Context, record Record) (Record, error) {
	if strings.TrimSpace(s.Endpoint) == "" {
		return Record{}, fmt.Errorf("audit HTTP endpoint is unset")
	}
	payload, err := json.Marshal(record)
	if err != nil {
		return Record{}, fmt.Errorf("encode audit record: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.Endpoint, bytes.NewReader(payload))
	if err != nil {
		return Record{}, fmt.Errorf("create audit request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client().Do(req)
	if err != nil {
		return Record{}, fmt.Errorf("write audit record: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode/100 != 2 {
		return Record{}, fmt.Errorf("audit sink returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if len(body) == 0 {
		return record, nil
	}
	var stored Record
	if err := json.Unmarshal(body, &stored); err != nil {
		return Record{}, fmt.Errorf("decode stored audit record: %w", err)
	}
	return stored, nil
}

func (s *HTTPStore) List(ctx context.Context, filter ListFilter) ([]Record, error) {
	if strings.TrimSpace(s.Endpoint) == "" {
		return nil, fmt.Errorf("audit HTTP endpoint is unset")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.Endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create audit list request: %w", err)
	}
	resp, err := s.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("read audit records: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("audit reader returned HTTP %d", resp.StatusCode)
	}
	var records []Record
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&records); err != nil {
		return nil, fmt.Errorf("decode audit records: %w", err)
	}
	if filter.NodeID == "" {
		return records, nil
	}
	filtered := records[:0]
	for _, record := range records {
		if record.NodeID == filter.NodeID {
			filtered = append(filtered, record)
		}
	}
	return filtered, nil
}

func (s *HTTPStore) client() *http.Client {
	if s.Client != nil {
		return s.Client
	}
	return http.DefaultClient
}
