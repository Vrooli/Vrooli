package deploy

import (
	"context"
	"fmt"
	"strings"

	sharedenv "scenario-to-desktop-api/shared/env"
)

// Handler owns deploy-target domain behavior shared by Connect handlers and
// pipeline orchestration.
type Handler struct {
	repo *TargetRepository
}

// NewHandler creates a new deploy target handler.
func NewHandler(repo *TargetRepository) *Handler {
	return &Handler{repo: repo}
}

type deployTargetDoctorCheck struct {
	Name     string `json:"name"`
	Required bool   `json:"required"`
	Passed   bool   `json:"passed"`
	Blocked  bool   `json:"blocked,omitempty"`
	Detail   string `json:"detail"`
}

type deployTargetDoctorReport struct {
	Ready         bool                      `json:"ready"`
	Name          string                    `json:"name"`
	ScenarioName  string                    `json:"scenario_name"`
	RemoteProfile string                    `json:"remote_profile"`
	Checks        []deployTargetDoctorCheck `json:"checks"`
	NextSteps     []string                  `json:"next_steps"`
}

func runDeployTargetDoctor(ctx context.Context, name string, target *DeployTarget) deployTargetDoctorReport {
	report := deployTargetDoctorReport{
		Ready:         true,
		Name:          name,
		ScenarioName:  target.ScenarioName,
		RemoteProfile: target.RemoteProfile,
		Checks:        make([]deployTargetDoctorCheck, 0, 3),
		NextSteps:     make([]string, 0, 6),
	}

	secret := sharedenv.ResolveSecretWithSource("LPBS_SERVICE_SECRET")
	tokenCheck := deployTargetDoctorCheck{
		Name:     "scenario_to_desktop_secret",
		Required: true,
	}
	if strings.TrimSpace(secret.Value) == "" {
		tokenCheck.Passed = false
		tokenCheck.Detail = missingScenarioToDesktopSecretMessage()
		report.Ready = false
		report.NextSteps = appendUnique(report.NextSteps,
			fmt.Sprintf("scenario-to-cloud secrets get LPBS_SERVICE_SECRET --scenario %s --targets scenario", target.ScenarioName),
			"scenario-to-cloud secrets set LPBS_SERVICE_SECRET --scenario scenario-to-desktop --value <same_secret_value> --targets scenario",
		)
	} else {
		tokenCheck.Passed = true
		if secret.Source == "file" && strings.TrimSpace(secret.SourcePath) != "" {
			tokenCheck.Detail = fmt.Sprintf("LPBS_SERVICE_SECRET is set (source=file %s)", secret.SourcePath)
		} else {
			tokenCheck.Detail = fmt.Sprintf("LPBS_SERVICE_SECRET is set (source=%s)", secret.Source)
		}
	}
	report.Checks = append(report.Checks, tokenCheck)

	sessionCheck := deployTargetDoctorCheck{
		Name:     "remote_profile_session",
		Required: true,
	}
	serviceAuthCheck := deployTargetDoctorCheck{
		Name:     "lpbs_service_auth",
		Required: true,
	}

	if !tokenCheck.Passed {
		sessionCheck.Passed = false
		sessionCheck.Blocked = true
		sessionCheck.Detail = "skipped: scenario-to-desktop LPBS_SERVICE_SECRET is required"
		serviceAuthCheck.Passed = false
		serviceAuthCheck.Blocked = true
		serviceAuthCheck.Detail = "skipped: scenario-to-desktop LPBS_SERVICE_SECRET is required"
		report.Ready = false
	} else {
		client := NewLPBSClient(target.ScenarioName, secret.Value)
		if err := client.TestRemoteProfile(ctx, target.RemoteProfile); err != nil {
			sessionCheck.Passed = false
			sessionCheck.Detail = fmt.Sprintf("remote profile %q session test failed: %v", target.RemoteProfile, err)
			report.Ready = false
			report.NextSteps = appendUnique(report.NextSteps,
				fmt.Sprintf("%s remote-profiles-login --tag %s --email <remote_admin_email> --password @/path/to/remote-admin-password.txt", target.ScenarioName, target.RemoteProfile),
			)
		} else {
			sessionCheck.Passed = true
			sessionCheck.Detail = fmt.Sprintf("remote profile %q session is active", target.RemoteProfile)
		}

		status, err := client.GetServiceAuthStatus(ctx)
		switch {
		case err != nil:
			serviceAuthCheck.Passed = false
			serviceAuthCheck.Detail = fmt.Sprintf("service auth status check failed: %v", err)
			report.Ready = false
		case status == nil || !status.ServiceAuthConfigured:
			serviceAuthCheck.Passed = false
			serviceAuthCheck.Detail = fmt.Sprintf("service auth is not configured in %s runtime", target.ScenarioName)
			report.Ready = false
			report.NextSteps = appendUnique(report.NextSteps,
				fmt.Sprintf("scenario-to-cloud secrets set LPBS_SERVICE_SECRET --scenario %s --generate hex:64 --targets scenario,deployment --domain <domain> --restart", target.ScenarioName),
				fmt.Sprintf("%s service-auth-status --require-enabled", target.ScenarioName),
			)
		default:
			mode := strings.TrimSpace(status.ServiceAuthMode)
			if mode == "" {
				mode = "unknown"
			}
			serviceAuthCheck.Passed = true
			serviceAuthCheck.Detail = fmt.Sprintf("service auth enabled (mode=%s)", mode)
		}
	}

	report.Checks = append(report.Checks, sessionCheck, serviceAuthCheck)
	report.NextSteps = appendUnique(report.NextSteps, fmt.Sprintf("scenario-to-desktop deploy-target doctor %s", name))
	if report.Ready {
		report.NextSteps = []string{
			fmt.Sprintf("Deploy target %q is ready. Continue with scenario-to-desktop pipeline run ... --deploy-target %s --app-key <app_key> --wait", name, name),
		}
	}
	return report
}

func missingScenarioToDesktopSecretMessage() string {
	return "LPBS_SERVICE_SECRET is not set for scenario-to-desktop runtime (checked env and ~/.vrooli/secrets.json)"
}

func appendUnique(values []string, additions ...string) []string {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		seen[value] = struct{}{}
	}
	for _, value := range additions {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	return values
}
