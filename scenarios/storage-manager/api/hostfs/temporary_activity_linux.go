//go:build linux

package hostfs

import (
	"context"
	"encoding/json"
	"fmt"
	platformgo "github.com/vrooli/platform-go"
	"os"
	"path/filepath"
	"storage-manager/internal/cleanup"
	"strings"
	"time"
)

type temporaryActivityObserver struct {
	exchange func(context.Context, any, any) error
}

func NewTemporaryActivityObserver() cleanup.TemporaryActivityObserver {
	transport := platformgo.PrivilegeBrokerTransport{Timeout: 35 * time.Second, MaxResponseBytes: 8 << 20}
	return &temporaryActivityObserver{exchange: transport.Exchange}
}

func (o *temporaryActivityObserver) ObserveTemporary(ctx context.Context, root string) (map[string]struct{}, error) {
	uid := os.Getuid()
	requestID := fmt.Sprintf("storage-temporary-%d", time.Now().UnixNano())
	request := struct {
		Version   string   `json:"version"`
		RequestID string   `json:"request_id"`
		Action    string   `json:"action"`
		Subject   struct{} `json:"subject"`
	}{"v1", requestID, platformgo.TemporaryActivityObserveAction, struct{}{}}
	var response temporaryActivityResponse
	err := o.exchange(ctx, request, &response)
	if err != nil {
		return nil, err
	}
	if response.Version != "v1" || response.RequestID != requestID || response.Action != request.Action || response.Status != "completed" || response.Changed || len(response.Snapshot) == 0 {
		if response.Version == "v1" && response.RequestID == requestID && response.Action == request.Action && response.Code == "temporary_activity_incomplete" {
			return nil, fmt.Errorf("temporary activity inspection incomplete: %s; deletion withheld", response.Evidence.Detail)
		}
		return nil, fmt.Errorf("temporary activity broker did not return matching complete evidence (%s); run vrooli setup --sudo-mode=ask", response.Code)
	}
	return decodeTemporaryActivity(string(response.Snapshot), root, uid, time.Now())
}

type temporaryActivityResponse struct {
	Version   string          `json:"version"`
	RequestID string          `json:"request_id"`
	Action    string          `json:"action"`
	Status    string          `json:"status"`
	Code      string          `json:"code"`
	Changed   bool            `json:"changed"`
	Snapshot  json.RawMessage `json:"temporary_activity"`
	Evidence  struct {
		Detail string `json:"detail"`
	} `json:"evidence"`
}

func decodeTemporaryActivity(raw, root string, uid int, now time.Time) (map[string]struct{}, error) {
	if len(raw) > 8<<20 {
		return nil, fmt.Errorf("temporary activity snapshot exceeds its bound")
	}
	var snapshot struct {
		UID        int       `json:"uid"`
		ObservedAt time.Time `json:"observed_at"`
		Complete   bool      `json:"complete"`
		Paths      []string  `json:"paths"`
		Roots      []string  `json:"roots"`
	}
	if err := json.Unmarshal([]byte(raw), &snapshot); err != nil {
		return nil, err
	}
	if !snapshot.Complete || snapshot.UID != uid || snapshot.ObservedAt.After(now.Add(time.Second)) || now.Sub(snapshot.ObservedAt) > 30*time.Second {
		return nil, fmt.Errorf("temporary activity snapshot is incomplete, stale, or belongs to another user")
	}
	root = filepath.Clean(root)
	covered := false
	for _, observedRoot := range snapshot.Roots {
		observedRoot = filepath.Clean(observedRoot)
		if filepath.IsAbs(observedRoot) && (root == observedRoot || strings.HasPrefix(root, observedRoot+string(filepath.Separator))) {
			covered = true
		}
	}
	if !filepath.IsAbs(root) || !covered {
		return nil, fmt.Errorf("temporary activity snapshot does not cover the requested root")
	}
	paths := map[string]struct{}{}
	for _, path := range snapshot.Paths {
		clean := filepath.Clean(path)
		if clean == root || strings.HasPrefix(clean, root+string(filepath.Separator)) {
			paths[clean] = struct{}{}
		}
	}
	return paths, nil
}
