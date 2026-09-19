package platformverdict

import (
	"context"
	"errors"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"
	platformv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/platform_verdict"
	platformconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/platform_verdict/platform_verdict_v1connect"
	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/deployment"
)

// RegisterConnectRoutes mounts SDA's derived scenario platform verdict read
// surface. No mutation or fallback computation is exposed here.
func RegisterConnectRoutes(router *gin.Engine, scenariosDir func() string) {
	path, handler := platformconnect.NewPlatformVerdictServiceHandler(&connectHandler{scenariosDir: scenariosDir})
	router.Any(path+"*path", gin.WrapH(handler))
}

type connectHandler struct {
	scenariosDir func() string
}

func (h *connectHandler) ListPlatformVerdicts(ctx context.Context, req *connect.Request[platformv1.ListPlatformVerdictsRequest]) (*connect.Response[platformv1.ListPlatformVerdictsResponse], error) {
	if h == nil || h.scenariosDir == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("platform verdict source unavailable"))
	}
	message := req.Msg
	if message == nil {
		message = &platformv1.ListPlatformVerdictsRequest{}
	}
	report, err := deployment.BuildPlatformFleet(h.scenariosDir(), strings.TrimSpace(message.GetScenario()), time.Now().UTC())
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	response := &platformv1.ListPlatformVerdictsResponse{Available: true, ComputedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	for _, scenario := range report.Scenarios {
		item := &platformv1.ScenarioPlatformVerdict{Scenario: scenario.Scenario, Overridden: scenario.Overridden, OverrideReason: scenario.OverrideReason}
		for _, platform := range scenario.Platforms {
			item.Platforms = append(item.Platforms, &platformv1.PlatformVerdict{
				HostOs:             string(platform.HostOS),
				Status:             platform.Status,
				Reason:             platform.Reason,
				ReasonCode:         platform.ReasonCode,
				BlockingDependency: platform.BlockingDependency,
				Derived:            platform.Derived,
				Overridden:         platform.Overridden,
			})
		}
		response.Scenarios = append(response.Scenarios, item)
	}
	for _, block := range report.DockerBlocked {
		response.DockerBlocked = append(response.DockerBlocked, &platformv1.FleetDependencyBlock{Scenario: block.Scenario, HostOs: string(block.HostOS), Dependency: block.Dependency, Reason: block.Reason})
	}
	for _, upgrade := range report.TierUpgrades {
		response.TierUpgrades = append(response.TierUpgrades, &platformv1.FleetTierUpgrade{Scenario: upgrade.Scenario, HostOs: string(upgrade.HostOS), CurrentTier: string(upgrade.CurrentTier), NextTier: string(upgrade.NextTier), Change: upgrade.Change, BlockingDependency: upgrade.BlockingDependency})
	}
	return connect.NewResponse(response), nil
}
