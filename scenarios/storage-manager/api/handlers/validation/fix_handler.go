package validation

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"

	"storage-manager/internal/autofix"

	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

// PreviewFix reports the deterministic storage edits the autofix registry could
// apply for the requested scenario without writing anything (shared Fix RPC).
func (h *connectHandler) PreviewFix(_ context.Context, req *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	scenario, root, err := h.resolveFixTarget(req.Msg)
	if err != nil {
		return nil, err
	}
	resp, err := autofix.PreviewFixResponse(scenario, root, req.Msg.GetRuleIds())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("preview storage autofix: %w", err))
	}
	return connect.NewResponse(resp), nil
}

// ApplyFix applies the autofix registry's deterministic storage edits for the
// requested scenario and reports what changed (shared Fix RPC). Apply is
// Preview-then-write and idempotent: a second call over an already-fixed tree
// returns no candidates.
func (h *connectHandler) ApplyFix(_ context.Context, req *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	scenario, root, err := h.resolveFixTarget(req.Msg)
	if err != nil {
		return nil, err
	}
	resp, err := autofix.ApplyFixResponse(scenario, root, req.Msg.GetRuleIds())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("apply storage autofix: %w", err))
	}
	return connect.NewResponse(resp), nil
}

// resolveFixTarget resolves a FixRequest to the scenario id and the absolute
// scenario directory the fixers operate on. A request may carry an explicit
// path (used verbatim) or a scenario id resolved to scenarios/<scenario> under
// the configured repo root.
func (h *connectHandler) resolveFixTarget(req *scenariovalidationv1.FixRequest) (string, string, error) {
	scenario := strings.TrimSpace(req.GetScenario())
	path := strings.TrimSpace(req.GetPath())
	if scenario == "" && path == "" {
		return "", "", connect.NewError(connect.CodeInvalidArgument, errors.New("scenario or path is required"))
	}

	root := path
	if root == "" {
		if h.deps.RepoRoot == "" {
			return "", "", connect.NewError(connect.CodeFailedPrecondition,
				errors.New("repo root is not configured; supply an explicit path"))
		}
		root = filepath.Join(h.deps.RepoRoot, "scenarios", scenario)
	}
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		return "", "", connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("scenario directory %q is not resolvable", root))
	}
	if scenario == "" {
		scenario = filepath.Base(root)
	}
	return scenario, root, nil
}
