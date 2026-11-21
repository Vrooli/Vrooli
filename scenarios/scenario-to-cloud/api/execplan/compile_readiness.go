package execplan

import (
	"strconv"
	"strings"
)

func readinessAction(in CompileInputs, deps []string) Action {
	checks := []string{"local"}
	if strings.TrimSpace(in.Manifest.Edge.Domain) != "" {
		checks = append(checks, "https", "origin", "public")
	}
	return Action{
		ID:                 OpVerifyReadiness,
		OwnerOperation:     OpVerifyReadiness,
		Effect:             EffectNone,
		RequiredCapability: "health:observe",
		Inputs: map[string]string{
			"checks":      strings.Join(checks, ","),
			"domain":      strings.TrimSpace(in.Manifest.Edge.Domain),
			"health_path": in.Manifest.Edge.AppHealthPath(),
			"ui_port":     strconv.Itoa(in.Manifest.Ports["ui"]),
			"host":        strings.TrimSpace(in.Manifest.Target.VPS.Host),
			"release_id":  in.ReleaseArtifacts.ID,
		},
		DependsOn:    deps,
		Verification: "readiness_checks_pass",
		Recovery:     "rollback_to_predecessor",
		Retry:        RetrySafeReplay,
		CancelPoint:  true,
	}
}
