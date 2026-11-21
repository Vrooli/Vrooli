package optimization

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

const maxRequestBytes = 64 * 1024

// maxServiceWait is the public one-shot wait ceiling. Keep this below the
// HTTP server write deadline so a completed wait can always write its receipt.
const maxServiceWait = time.Minute

// Service is the typed boundary used by the Program Runtime Connect handler.
// It accepts one immutable request and exposes a bounded server-side wait.
type Service struct{ manager *Manager }

func NewService(manager *Manager) (*Service, error) {
	if manager == nil {
		return nil, fmt.Errorf("optimization manager is required")
	}
	return &Service{manager: manager}, nil
}

func (s *Service) Submit(ctx context.Context, requestJSON string) (Record, error) {
	if s == nil || s.manager == nil {
		return Record{}, fmt.Errorf("optimization service unavailable")
	}
	if len(requestJSON) == 0 || len(requestJSON) > maxRequestBytes {
		return Record{}, fmt.Errorf("request_json must contain 1..%d bytes: %w", maxRequestBytes, ErrInvalidRequest)
	}
	if err := rejectDuplicateJSONKeys(requestJSON); err != nil {
		return Record{}, fmt.Errorf("decode optimizer request: %v: %w", err, ErrInvalidRequest)
	}
	decoder := json.NewDecoder(bytes.NewBufferString(requestJSON))
	decoder.DisallowUnknownFields()
	var request Request
	if err := decoder.Decode(&request); err != nil {
		return Record{}, fmt.Errorf("decode optimizer request: %v: %w", err, ErrInvalidRequest)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return Record{}, fmt.Errorf("optimizer request must contain exactly one JSON value: %w", ErrInvalidRequest)
	}
	if len(request.Candidates) > 16 || len(request.CaseIDs) > 2000 || len(request.Subject) > 32 {
		return Record{}, fmt.Errorf("optimizer request exceeds candidate, case, or subject bounds: %w", ErrInvalidRequest)
	}
	return s.manager.Submit(ctx, request)
}

func rejectDuplicateJSONKeys(raw string) error {
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	var readValue func(depth int) error
	readValue = func(depth int) error {
		if depth > 1000 {
			return fmt.Errorf("optimizer request exceeds the JSON nesting limit")
		}
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		delim, composite := token.(json.Delim)
		if !composite {
			return nil
		}
		switch delim {
		case '{':
			seen := make(map[string]struct{})
			for decoder.More() {
				key, err := decoder.Token()
				if err != nil {
					return err
				}
				name, ok := key.(string)
				if !ok {
					return fmt.Errorf("optimizer request object key is not a string")
				}
				if _, duplicate := seen[name]; duplicate {
					return fmt.Errorf("optimizer request contains duplicate object key %q", name)
				}
				seen[name] = struct{}{}
				if err := readValue(depth + 1); err != nil {
					return err
				}
			}
			end, err := decoder.Token()
			if err != nil {
				return err
			}
			if end != json.Delim('}') {
				return fmt.Errorf("optimizer request has an unterminated object")
			}
		case '[':
			for decoder.More() {
				if err := readValue(depth + 1); err != nil {
					return err
				}
			}
			end, err := decoder.Token()
			if err != nil {
				return err
			}
			if end != json.Delim(']') {
				return fmt.Errorf("optimizer request has an unterminated array")
			}
		default:
			return fmt.Errorf("optimizer request contains an unexpected delimiter %q", delim)
		}
		return nil
	}
	return readValue(0)
}

func (s *Service) Wait(ctx context.Context, runID string, wait time.Duration) (Record, bool, error) {
	if s == nil || s.manager == nil {
		return Record{}, false, fmt.Errorf("optimization service unavailable")
	}
	if runID == "" || runID != strings.TrimSpace(runID) || len(runID) > 128 {
		return Record{}, false, ErrInvalidRequest
	}
	if wait < time.Millisecond || wait > maxServiceWait {
		return Record{}, false, fmt.Errorf("wait duration must be 1ms..%s: %w", maxServiceWait, ErrInvalidRequest)
	}
	return s.manager.Wait(ctx, runID, wait)
}
