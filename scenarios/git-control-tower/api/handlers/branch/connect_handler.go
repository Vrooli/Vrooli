// Package branch implements the BranchService Connect-RPC transport adapter.
package branch

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect"
	branchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/branch"
	branchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/branch/branch_v1connect"
)

type Server struct {
	list    func(context.Context, *branchv1.ListBranchesRequest) (*branchv1.ListBranchesResponse, error)
	create  func(context.Context, *branchv1.CreateBranchRequest) (*branchv1.CreateBranchResponse, error)
	switch_ func(context.Context, *branchv1.SwitchBranchRequest) (*branchv1.SwitchBranchResponse, error)
	publish func(context.Context, *branchv1.PublishBranchRequest) (*branchv1.PublishBranchResponse, error)
}

type Deps struct {
	List    func(context.Context, *branchv1.ListBranchesRequest) (*branchv1.ListBranchesResponse, error)
	Create  func(context.Context, *branchv1.CreateBranchRequest) (*branchv1.CreateBranchResponse, error)
	Switch  func(context.Context, *branchv1.SwitchBranchRequest) (*branchv1.SwitchBranchResponse, error)
	Publish func(context.Context, *branchv1.PublishBranchRequest) (*branchv1.PublishBranchResponse, error)
}

func NewHandler(d Deps, opts ...connect.HandlerOption) (string, http.Handler) {
	return branchconnect.NewBranchServiceHandler(&Server{list: d.List, create: d.Create, switch_: d.Switch, publish: d.Publish}, opts...)
}

func (s *Server) ListBranches(ctx context.Context, req *connect.Request[branchv1.ListBranchesRequest]) (*connect.Response[branchv1.ListBranchesResponse], error) {
	if s.list == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("branch list service is not configured"))
	}
	resp, err := s.list(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (s *Server) CreateBranch(ctx context.Context, req *connect.Request[branchv1.CreateBranchRequest]) (*connect.Response[branchv1.CreateBranchResponse], error) {
	if s.create == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("branch create service is not configured"))
	}
	resp, err := s.create(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (s *Server) SwitchBranch(ctx context.Context, req *connect.Request[branchv1.SwitchBranchRequest]) (*connect.Response[branchv1.SwitchBranchResponse], error) {
	if s.switch_ == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("branch switch service is not configured"))
	}
	resp, err := s.switch_(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (s *Server) PublishBranch(ctx context.Context, req *connect.Request[branchv1.PublishBranchRequest]) (*connect.Response[branchv1.PublishBranchResponse], error) {
	if s.publish == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("branch publish service is not configured"))
	}
	resp, err := s.publish(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

var _ branchconnect.BranchServiceHandler = (*Server)(nil)
