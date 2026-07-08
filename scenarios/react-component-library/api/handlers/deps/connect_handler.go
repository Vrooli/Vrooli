package deps

import (
	"context"
	"log"

	"connectrpc.com/connect"

	"react-component-library/internal/deps"

	depsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/deps"
)

// Deps wires the seams the Connect deps handler needs.
type Deps struct {
	Service deps.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) ListDeclarations(ctx context.Context, req *connect.Request[depsv1.ListDeclarationsRequest]) (*connect.Response[depsv1.ListDeclarationsResponse], error) {
	out, err := h.deps.Service.ListForComponent(ctx, req.Msg.ComponentId)
	if err != nil {
		h.deps.Logger.Printf("deps.ListDeclarations: %v", err)
		return nil, deps.ToConnectError(err)
	}
	resp := &depsv1.ListDeclarationsResponse{Declarations: make([]*depsv1.DepDeclaration, 0, len(out))}
	for _, d := range out {
		resp.Declarations = append(resp.Declarations, declarationToProto(d))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) ValidateAdoption(ctx context.Context, req *connect.Request[depsv1.ValidateAdoptionRequest]) (*connect.Response[depsv1.ValidateAdoptionResponse], error) {
	v, err := h.deps.Service.ValidateAdoption(ctx, req.Msg.ComponentId, req.Msg.Version, req.Msg.Scenario)
	if err != nil {
		connectErr := deps.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("deps.ValidateAdoption: %v", err)
		}
		return nil, connectErr
	}
	resp := &depsv1.ValidateAdoptionResponse{
		Kind:   verdictKindToProto(v.Kind),
		Issues: make([]*depsv1.DepIssue, 0, len(v.Issues)),
	}
	for _, iss := range v.Issues {
		resp.Issues = append(resp.Issues, issueToProto(iss))
	}
	return connect.NewResponse(resp), nil
}

func declarationToProto(d deps.Declaration) *depsv1.DepDeclaration {
	return &depsv1.DepDeclaration{
		ComponentId:  d.ComponentID,
		LibraryId:    d.LibraryID,
		Version:      d.Version,
		DepName:      d.DepName,
		VersionRange: d.VersionRange,
		Kind:         string(d.Kind),
	}
}

func issueToProto(i deps.Issue) *depsv1.DepIssue {
	return &depsv1.DepIssue{
		DepName:         i.DepName,
		DeclaredRange:   i.DeclaredRange,
		ScenarioVersion: i.ScenarioVersion,
		Version:         i.Version,
		DepKind:         string(i.DepKind),
		Kind:            issueKindToProto(i.Kind),
		Detail:          i.Detail,
	}
}

func verdictKindToProto(k deps.VerdictKind) depsv1.VerdictKind {
	switch k {
	case deps.VerdictOK:
		return depsv1.VerdictKind_VERDICT_KIND_OK
	case deps.VerdictWarn:
		return depsv1.VerdictKind_VERDICT_KIND_WARN
	case deps.VerdictBlock:
		return depsv1.VerdictKind_VERDICT_KIND_BLOCK
	}
	return depsv1.VerdictKind_VERDICT_KIND_UNSPECIFIED
}

func issueKindToProto(k deps.IssueKind) depsv1.IssueKind {
	switch k {
	case deps.IssueMissingDep:
		return depsv1.IssueKind_ISSUE_KIND_MISSING_DEP
	case deps.IssueRangeDoesNotMatch:
		return depsv1.IssueKind_ISSUE_KIND_RANGE_DOES_NOT_MATCH
	case deps.IssueIncompatibleMajor:
		return depsv1.IssueKind_ISSUE_KIND_INCOMPATIBLE_MAJOR
	case deps.IssueUnparseableRange:
		return depsv1.IssueKind_ISSUE_KIND_UNPARSEABLE_RANGE
	case deps.IssueUnparseableTarget:
		return depsv1.IssueKind_ISSUE_KIND_UNPARSEABLE_TARGET
	}
	return depsv1.IssueKind_ISSUE_KIND_UNSPECIFIED
}
