package executor

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	"google.golang.org/protobuf/proto"
)

// rewriteElectronScenarioNavigation keeps a typed scenario destination inside
// the renderer admitted by the target-owned Electron session. The compiler
// resolves scenario destinations through the lifecycle registry, which is the
// right behavior for normal browser execution but points at the live scenario
// port during an Electron validation run. The typed scenario identity remains
// the source of truth for this remap; direct URL actions are never rewritten.
func rewriteElectronScenarioNavigation(instruction contracts.CompiledInstruction, target *driver.ElectronTarget) (contracts.CompiledInstruction, error) {
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

	rendererURL := strings.TrimSpace(target.RendererURL)
	if rendererURL == "" {
		return instruction, fmt.Errorf("Electron scenario navigation requires an admitted renderer URL")
	}
	base, err := url.Parse(rendererURL)
	if err != nil || base.Scheme == "" {
		if err == nil {
			err = fmt.Errorf("missing URL scheme")
		}
		return instruction, fmt.Errorf("parse admitted Electron renderer URL %q: %w", rendererURL, err)
	}

	route := strings.TrimSpace(navigate.GetScenarioPath())
	if route == "" {
		route = "/"
	}
	routeURL, err := url.Parse(route)
	if err != nil || routeURL.IsAbs() || routeURL.Host != "" {
		if err == nil {
			err = fmt.Errorf("scenario path must be relative")
		}
		return instruction, fmt.Errorf("parse Electron scenario path %q: %w", route, err)
	}

	// Preserve only the admitted renderer's scheme/host and the typed route.
	// In particular, do not carry the compiler-resolved live scenario host.
	mapped := &url.URL{
		Scheme:   base.Scheme,
		Host:     base.Host,
		Path:     routeURL.Path,
		RawPath:  routeURL.RawPath,
		RawQuery: routeURL.RawQuery,
		Fragment: routeURL.Fragment,
	}
	if mapped.Path == "" {
		mapped.Path = "/"
	}

	cloned, ok := proto.Clone(instruction.Action).(*basactions.ActionDefinition)
	if !ok {
		return instruction, fmt.Errorf("clone Electron scenario navigation action")
	}
	cloned.GetNavigate().Url = mapped.String()
	instruction.Action = cloned
	return instruction, nil
}
