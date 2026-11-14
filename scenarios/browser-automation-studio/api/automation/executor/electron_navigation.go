package executor

import (
	"fmt"
	"strings"

	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	"google.golang.org/protobuf/proto"
)

// rewriteAppTargetScenarioNavigation keeps a typed scenario destination inside
// the renderer admitted by the target-owned app session. The compiler
// resolves scenario destinations through the lifecycle registry, which is the
// right behavior for normal browser execution but points at the live scenario
// port during an Electron validation run. The typed scenario identity remains
// the source of truth for this remap; direct URL actions are never rewritten.
func rewriteAppTargetScenarioNavigation(instruction contracts.CompiledInstruction, target *driver.AppTarget) (contracts.CompiledInstruction, error) {
	if target == nil || instruction.Action == nil || instruction.Action.GetType() != basactions.ActionType_ACTION_TYPE_NAVIGATE {
		return instruction, nil
	}
	navigate := instruction.Action.GetNavigate()
	if navigate == nil {
		return instruction, nil
	}
	if navigate.GetDestinationType() != basactions.NavigateDestinationType_NAVIGATE_DESTINATION_TYPE_SCENARIO && strings.TrimSpace(navigate.GetScenario()) == "" {
		return instruction, nil
	}
	if strings.TrimSpace(target.ScenarioName) == "" || strings.TrimSpace(navigate.GetScenario()) != strings.TrimSpace(target.ScenarioName) {
		return instruction, nil
	}

	policy, err := driver.ResolveTargetURLPolicy(target.TargetKind)
	if err != nil {
		return instruction, err
	}
	route := strings.TrimSpace(navigate.GetScenarioPath())
	if route == "" {
		route = "/"
	}
	mapped, err := policy.Resolve(target.RendererURL, route)
	if err != nil {
		return instruction, err
	}

	cloned, ok := proto.Clone(instruction.Action).(*basactions.ActionDefinition)
	if !ok {
		return instruction, fmt.Errorf("clone app-target scenario navigation action")
	}
	cloned.GetNavigate().Url = mapped
	instruction.Action = cloned
	return instruction, nil
}
