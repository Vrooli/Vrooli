// Package tools hosts the BAS ToolsService Connect-RPC handler.
//
// ToolsService is BAS's adapter onto the cross-scenario Tool Discovery
// Protocol: agent-inbox and AI agents call List/Get to discover what
// tools this scenario provides and Execute to invoke them.
package tools

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/connectx"

	"github.com/vrooli/browser-automation-studio/internal/toolexecution"
	agentinboxpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
	toolsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/tools/toolsconnect"
)

// Registry is the narrow seam used to surface tool manifests. The
// concrete implementation is *toolregistry.Registry; tests substitute an
// in-memory fake.
type Registry interface {
	GetManifest(ctx context.Context) *agentinboxpb.ToolManifest
	GetTool(ctx context.Context, name string) *agentinboxpb.ToolDefinition
}

// Executor is the narrow seam used to dispatch tool invocations. The
// concrete implementation is *toolexecution.ServerExecutor.
type Executor interface {
	Execute(ctx context.Context, toolName string, args map[string]interface{}) (*toolexecution.ExecutionResult, error)
}

// Deps wires the tools handler. Logger, Registry, and Executor are all
// required; Module panics if any is nil.
type Deps struct {
	Registry Registry
	Executor Executor
	Logger   *logrus.Logger
}

// Module builds the ToolsService Connect handler and returns it wrapped
// in a connectx.ServiceMount ready for connectx.RegisterChi.
func Module(d Deps) connectx.ServiceMount {
	if d.Logger == nil {
		panic("tools.Module requires Deps.Logger")
	}
	if d.Registry == nil {
		panic("tools.Module requires Deps.Registry")
	}
	if d.Executor == nil {
		panic("tools.Module requires Deps.Executor")
	}
	path, handler := toolsconnect.NewToolsServiceHandler(&service{deps: d})
	return connectx.ServiceMount{Path: path, Handler: handler}
}
