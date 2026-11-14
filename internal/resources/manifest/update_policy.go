package manifest

import (
	"fmt"
	"strings"

	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

// ValidateManagedServiceUpdate checks the declaration boundary used by both
// manifest loading and frozen update-role rendering.
func ValidateManagedServiceUpdate(m ResourceManifest) error {
	if m.ManagedService == nil || m.ManagedService.UpdatePolicy == nil {
		return fmt.Errorf("managed-service update policy is required")
	}
	if err := m.ManagedService.ValidateUpdatePolicy(); err != nil {
		return err
	}
	if err := m.ManagedService.ValidateCredentialFiles(); err != nil {
		return err
	}
	return validateManagedServiceUpdateReadiness(m)
}

func validateManagedServiceUpdateReadiness(m ResourceManifest) error {
	if m.ManagedService == nil || m.ManagedService.UpdatePolicy == nil {
		return nil
	}
	p := m.ManagedService.UpdatePolicy
	if len(m.Companions) > 0 || m.RuntimeEnvCommand != nil || m.Lifecycle.ExclusiveGroup != "" {
		return fmt.Errorf("managed_service.update_policy does not support companions, runtime environment commands or exclusive groups")
	}
	if len(m.Ports) != 1 || m.Ports[0].Name != p.CandidatePort {
		return fmt.Errorf("managed_service.update_policy requires exactly one port matching candidate_port")
	}
	token := "${" + resourcedeployment.ServicePortEnvironmentName(p.CandidatePort) + "}"
	ready := false
	for _, check := range m.HealthChecks {
		if check.Kind != HealthCheckReadiness {
			continue
		}
		switch p.Strategy {
		case resourcedeployment.UpdateStrategyRollingStateless:
			prefix := "http://127.0.0.1:" + token + "/"
			if check.Type == HealthCheckHTTP && strings.HasPrefix(check.Target, prefix) && (len(check.ExpectedStatus) == 0 || (len(check.ExpectedStatus) == 1 && check.ExpectedStatus[0] == 200)) {
				ready = true
			}
		case resourcedeployment.UpdateStrategyExclusiveStopStart:
			if check.Type == HealthCheckOpenRGBV6 && check.Target == "127.0.0.1:"+token {
				ready = true
			}
		}
	}
	if !ready {
		if p.Strategy == resourcedeployment.UpdateStrategyExclusiveStopStart {
			return fmt.Errorf("exclusive managed_service.update_policy requires OpenRGB SDK v6 readiness on its loopback port")
		}
		return fmt.Errorf("rolling managed_service.update_policy requires HTTP 200 readiness on the remapped loopback candidate port")
	}
	var common []string
	for _, v := range m.ManagedService.Environment {
		common = append(common, v)
	}
	if m.ManagedService.Config != nil {
		common = append(common, m.ManagedService.Config.Content)
	}
	containsPort := func(args []string) bool {
		for _, v := range append(append([]string(nil), common...), args...) {
			if strings.Contains(v, token) {
				return true
			}
		}
		return false
	}
	if !containsPort(m.ManagedService.Arguments) {
		return fmt.Errorf("managed_service.update_policy candidate port must be explicitly rendered into the launch contract")
	}
	for platform, args := range m.ManagedService.ArgumentsByPlatform {
		if !containsPort(args) {
			return fmt.Errorf("managed_service.update_policy candidate port is not rendered by %s launch arguments", platform)
		}
	}
	return nil
}
