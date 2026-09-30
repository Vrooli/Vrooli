// Package remote binds Treasury workflow groups directly to generated Connect
// descriptors. It contains no request structs or transport-specific DTOs.
package remote

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
	_ "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	agentService = "vrooli.treasury.v1.authorization.AgentSpend"
	adminService = "vrooli.treasury.v1.authorization.TreasuryAdmin"
)

type serviceSpec struct {
	name    protoreflect.FullName
	methods []string
}

var groups = map[string][]serviceSpec{
	"book":          {{name: adminService, methods: []string{"TreasuryAdmin.CreateBook", "TreasuryAdmin.GetBook"}}},
	"authorization": {{name: agentService, methods: []string{"AgentSpend.ProposeCharge", "AgentSpend.GetAuthorization"}}},
	"mandate": {
		{name: agentService, methods: []string{"AgentSpend.ListMandates"}},
		{name: adminService, methods: []string{"TreasuryAdmin.CreateMandate", "TreasuryAdmin.RevokeMandate", "TreasuryAdmin.CancelStandingMandate", "TreasuryAdmin.ListMandates"}},
	},
	"budget": {
		{name: agentService, methods: []string{"AgentSpend.GetBudgetHeadroom"}},
		{name: adminService, methods: []string{"TreasuryAdmin.SetBudgetCaps", "TreasuryAdmin.SetGating", "TreasuryAdmin.FreezeBudget", "TreasuryAdmin.UnfreezeBudget", "TreasuryAdmin.FreezeBook", "TreasuryAdmin.UnfreezeBook", "TreasuryAdmin.GetFreezeStatus", "TreasuryAdmin.FreezeAll", "TreasuryAdmin.UnfreezeAll"}},
	},
	"approval":   {{name: adminService, methods: []string{"TreasuryAdmin.ListApprovals", "TreasuryAdmin.ResolveApproval"}}},
	"instrument": {{name: adminService, methods: []string{"TreasuryAdmin.RegisterInstrument"}}},
	"settlement": {
		{name: agentService, methods: []string{"AgentSpend.ReportOutcome"}},
		{name: adminService, methods: []string{"TreasuryAdmin.ReportManualOutcome"}},
	},
}

var readMethods = map[string]bool{
	"AgentSpend.GetAuthorization":   true,
	"AgentSpend.GetBudgetHeadroom":  true,
	"AgentSpend.ListMandates":       true,
	"TreasuryAdmin.GetBook":         true,
	"TreasuryAdmin.ListApprovals":   true,
	"TreasuryAdmin.ListMandates":    true,
	"TreasuryAdmin.GetFreezeStatus": true,
}

func Register(core *cliapp.ScenarioApp, manifest []byte, groupName string) (cliapp.SubcommandGroup, error) {
	specs, ok := groups[groupName]
	if !ok {
		return cliapp.SubcommandGroup{}, fmt.Errorf("unknown Treasury remote group %q", groupName)
	}
	selected := make(map[string]func(cliapp.RunContext) error)
	for _, spec := range specs {
		renderers := make(map[string]cliapp.Renderer, len(spec.methods))
		for _, method := range spec.methods {
			if readMethods[method] {
				renderers[method] = renderRead(method)
			} else {
				renderers[method] = renderMutation(method)
			}
		}
		available, err := cliapp.ProtoBindings(core, spec.name, cliapp.ProtoBindingOptions{Render: renderers})
		if err != nil {
			return cliapp.SubcommandGroup{}, fmt.Errorf("%s: build proto bindings: %w", groupName, err)
		}
		for _, method := range spec.methods {
			handler, exists := available[method]
			if !exists {
				return cliapp.SubcommandGroup{}, fmt.Errorf("%s: descriptor method %s unavailable", groupName, method)
			}
			selected[method] = handler
		}
	}
	group, err := cliapp.LoadFromManifest(manifest, groupName, selected)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("%s: load manifest: %w", groupName, err)
	}
	return group, nil
}

func renderRead(method string) cliapp.Renderer {
	return func(ctx cliapp.RunContext, response proto.Message) error {
		return cliapp.RenderProtoList(ctx, response, cliapp.ListReport{
			Summary: []string{method + " completed."}, ResultsHeading: "Typed response",
			Results: []string{"Use --json for the complete generated protobuf response."},
		})
	}
}

func renderMutation(method string) cliapp.Renderer {
	return func(ctx cliapp.RunContext, response proto.Message) error {
		return cliapp.RenderProtoMutation(ctx, response, cliapp.MutationReport{
			Result:      []string{method + " completed."},
			Changes:     []string{"Treasury accepted the typed mutation and returned its authoritative record."},
			NextCommand: []string{"Use the corresponding read command with --json to inspect authoritative state."},
		})
	}
}
