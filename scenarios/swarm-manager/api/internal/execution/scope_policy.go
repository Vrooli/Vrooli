package execution

import (
	"context"
	"fmt"
	"strings"

	"swarm-manager/internal/pathutil"

	executionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/execution"
)

const (
	scopePolicyFixed            = "fixed"
	scopePolicyExtendWithRecord = "extend-with-record"
)

// readScopeExtensions reads Plan Manager's append-only boundary-extension
// ledger. Plan Manager remains authoritative; this is only the execution
// consumer's projection of the paths needed by the next workflow input and
// finalization.
func (s *Service) readScopeExtensions(ctx context.Context, record Record, item backlogItem) ([]ScopeExtension, error) {
	if strings.ToLower(strings.TrimSpace(item.ScopePolicy)) != scopePolicyExtendWithRecord || strings.TrimSpace(record.PlanManagerExecutionID) == "" {
		return nil, nil
	}
	client, ok := s.planRenderer.(interface {
		GetStatus(context.Context, *executionv1.GetStatusRequest) (*executionv1.GetStatusResponse, error)
	})
	if !ok {
		return nil, fmt.Errorf("extend-with-record scope policy requires the Plan Manager execution status read")
	}
	status, err := client.GetStatus(ctx, &executionv1.GetStatusRequest{ExecutionId: record.PlanManagerExecutionID})
	if err != nil {
		return nil, fmt.Errorf("read recorded scope extensions: %w", err)
	}
	if status == nil || status.GetExecution() == nil {
		return nil, fmt.Errorf("read recorded scope extensions: Plan Manager omitted execution")
	}

	var result []ScopeExtension
	for _, extension := range status.GetExecution().GetBoundaryExtensions() {
		if extension == nil {
			continue
		}
		paths := append([]string(nil), extension.GetAddedAllow()...)
		for _, path := range paths {
			for _, denied := range item.AcceptanceDeny {
				if scopeGlobsIntersect(path, denied) {
					return nil, fmt.Errorf("recorded scope extension %q is refused by acceptance_deny %q", path, denied)
				}
			}
		}
		if len(paths) == 0 {
			continue
		}
		result = append(result, ScopeExtension{
			Paths:      paths,
			Reason:     extension.GetReason(),
			RecordedAt: extension.GetCreatedAt(),
			Author:     extension.GetAuthor(),
		})
	}
	return result, nil
}

func effectiveWriteScope(item backlogItem, extensions []ScopeExtension) []string {
	allow := append([]string(nil), item.AcceptanceAllow...)
	if strings.ToLower(strings.TrimSpace(item.ScopePolicy)) == scopePolicyExtendWithRecord {
		for _, extension := range extensions {
			for _, path := range extension.Paths {
				forbidden := false
				for _, denied := range item.AcceptanceDeny {
					if scopeGlobsIntersect(path, denied) {
						forbidden = true
						break
					}
				}
				if !forbidden {
					allow = append(allow, path)
				}
			}
		}
	}
	return pathutil.UniqueSortedStrings(allow)
}

// scopeGlobsIntersect is deliberately conservative. A recorded extension is
// rejected when its literal roots overlap a denied root, which preserves the
// authored deny guard even when either side contains doublestar wildcards.
func scopeGlobsIntersect(left, right string) bool {
	left = strings.Trim(strings.ReplaceAll(left, "\\", "/"), "/")
	right = strings.Trim(strings.ReplaceAll(right, "\\", "/"), "/")
	if left == "" || right == "" || left == right {
		return true
	}
	lp, rp := scopeGlobLiteralPrefix(left), scopeGlobLiteralPrefix(right)
	if lp == "" || rp == "" {
		return true
	}
	return lp == rp || strings.HasPrefix(lp, rp+"/") || strings.HasPrefix(rp, lp+"/")
}

func scopeGlobLiteralPrefix(glob string) string {
	for index, char := range glob {
		switch char {
		case '*', '?', '[', '{':
			return strings.Trim(glob[:index], "/")
		}
	}
	return strings.Trim(glob, "/")
}
