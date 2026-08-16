package health

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"landing-page-business-suite/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	// Root /health is served by cli-core's built-in `status` command. This
	// group only wraps LPBS-specific readiness checks.
	return cliapp.CommandGroup{
		Title: "Readiness",
		Commands: []cliapp.Command{
			{Name: "service-auth-status", NeedsAPI: true, Description: "Check LPBS service-to-service auth readiness", Run: func(args []string) error { return RunServiceAuthStatus(deps, args) }},
			{Name: "deploy-readiness", NeedsAPI: true, Description: "Run LPBS readiness checks for desktop deploy handoff", Run: func(args []string) error { return RunDeployReadiness(deps, args) }},
		},
	}
}

func RunServiceAuthStatus(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("service-auth-status", flag.ContinueOnError)
	requireEnabled := fs.Bool("require-enabled", false, "Exit non-zero if service auth is not configured")
	jsonOut := cliutil.JSONFlag(fs)
	if err := support.ParseFlagSetInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("usage: service-auth-status [--require-enabled] [--json]")
	}

	resp, err := deps.ScenarioApp().Get("/usage/health", nil)
	if err != nil {
		return err
	}

	var parsed support.UsageHealthResponse
	if err := json.Unmarshal(resp, &parsed); err != nil {
		return fmt.Errorf("parse usage health response: %w", err)
	}

	status := "disabled"
	if parsed.ServiceAuthConfigured {
		status = "enabled"
	}
	mode := strings.TrimSpace(parsed.ServiceAuthMode)
	if mode == "" {
		mode = "unknown"
	}

	report := cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("Service auth: %s", status),
			fmt.Sprintf("Mode: %s", mode),
		},
	}
	if !parsed.DatabaseConnected {
		report.Triage = append(report.Triage, cliapp.TriageGroup{
			Heading: "Dependencies",
			Items:   []string{"[FAIL] database: disconnected"},
		})
	}
	if parsed.ServiceAuthConfigured {
		report.NextSteps = []string{"landing-page-business-suite service-auth-status --json", "Verify the consumer session through the credential authority before retrying"}
	} else {
		report.Triage = append(report.Triage, cliapp.TriageGroup{
			Heading: "Auth Gate",
			Items:   []string{"[FAIL] service_auth: disabled"},
		})
		report.NextSteps = []string{
			"vrooli credentials provision --identity vrooli/landing-page-business-suite --field consumer-signing-key",
			"landing-page-business-suite service-auth-status --require-enabled",
			"scenario-to-desktop deploy-target test <target-name> --require-service-auth",
		}
	}

	if *jsonOut {
		if err := cliapp.PrintReportJSON(os.Stdout, report); err != nil {
			return err
		}
		if *requireEnabled && !parsed.ServiceAuthConfigured {
			return serviceAuthNotConfiguredError()
		}
		return nil
	}

	if err := cliapp.RenderOperationalReport(os.Stdout, report); err != nil {
		return err
	}
	if *requireEnabled && !parsed.ServiceAuthConfigured {
		return serviceAuthNotConfiguredError()
	}
	return nil
}

func RunDeployReadiness(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("deploy-readiness", flag.ContinueOnError)
	profileIDFlag := fs.String("profile-id", "", "Remote profile id to test")
	profileTagFlag := fs.String("profile-tag", "", "Remote profile tag to test")
	appKeyFlag := fs.String("app-key", "", "Remote download app key to verify exists (optional)")
	domainFlag := fs.String("domain", "", "Deployment domain used for next-step guidance")
	requireServiceAuth := fs.Bool("require-service-auth", true, "Require LPBS service auth to be enabled")
	jsonOut := cliutil.JSONFlag(fs)
	if err := support.ParseFlagSetInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("usage: deploy-readiness [--profile-id <id> | --profile-tag <tag>] [--app-key <key>] [--domain <domain>] [--require-service-auth=true|false] [--json]")
	}

	profileID := strings.TrimSpace(*profileIDFlag)
	profileTag := strings.TrimSpace(*profileTagFlag)
	appKey := strings.TrimSpace(*appKeyFlag)
	domain := strings.TrimSpace(*domainFlag)
	if profileID != "" && profileTag != "" {
		return fmt.Errorf("use either --profile-id or --profile-tag, not both")
	}

	checks := make([]support.DeployReadinessCheck, 0, 4)
	nextSteps := make([]string, 0, 6)
	ready := true

	sessionCfg, sessionErr := deps.LoadAdminSession()
	adminSessionConfigured := sessionErr == nil && strings.TrimSpace(sessionCfg.Session) != ""
	adminSessionCheck := support.DeployReadinessCheck{Name: "admin_session", Required: true, Passed: adminSessionConfigured}
	if sessionErr != nil {
		adminSessionCheck.Detail = fmt.Sprintf("failed to load admin session: %v", sessionErr)
	} else if !adminSessionConfigured {
		adminSessionCheck.Detail = "admin session not configured"
	} else {
		adminSessionCheck.Detail = "admin session is configured"
	}
	checks = append(checks, adminSessionCheck)
	if !adminSessionCheck.Passed {
		ready = false
		nextSteps = append(nextSteps, "landing-page-business-suite admin-login --email <local_admin_email> --password @/path/to/local-admin-password.txt")
	}

	storageCheck := support.DeployReadinessCheck{Name: "download_storage", Required: true}
	if adminSessionConfigured {
		_, err := deps.RequestAdmin("POST", "/admin/download-storage/test", nil, nil)
		if err != nil {
			storageCheck.Detail = err.Error()
			ready = false
			nextSteps = append(nextSteps, "landing-page-business-suite admin-download-storage-test")
		} else {
			storageCheck.Passed = true
			storageCheck.Detail = "download storage test succeeded"
		}
	} else {
		storageCheck.Blocked = true
		storageCheck.Detail = "skipped: admin session is required"
		ready = false
	}
	checks = append(checks, storageCheck)

	if profileTag != "" || profileID != "" {
		resolvedProfileID := profileID
		profileCheck := support.DeployReadinessCheck{Name: "remote_profile_session", Required: true}
		if !adminSessionConfigured {
			profileCheck.Blocked = true
			profileCheck.Detail = "skipped: admin session is required"
			ready = false
		} else {
			if resolvedProfileID == "" {
				id, err := deps.ResolveRemoteProfileIDByTag(profileTag)
				if err != nil {
					profileCheck.Detail = err.Error()
					ready = false
				} else {
					resolvedProfileID = id
				}
			}
			if profileCheck.Detail == "" {
				_, err := deps.RequestAdmin("POST", "/admin/remote-profiles/"+url.PathEscape(resolvedProfileID)+"/test", nil, nil)
				if err != nil {
					profileCheck.Detail = err.Error()
					ready = false
				} else {
					profileCheck.Passed = true
					profileCheck.Detail = "remote profile session is active"
				}
			}
		}
		checks = append(checks, profileCheck)
		if !profileCheck.Passed && !profileCheck.Blocked {
			if profileTag != "" {
				nextSteps = append(nextSteps, fmt.Sprintf("landing-page-business-suite remote-profiles-login --tag %s --email <remote_admin_email> --password @/path/to/remote-admin-password.txt", profileTag))
			} else {
				nextSteps = append(nextSteps, fmt.Sprintf("landing-page-business-suite remote-profiles-login %s --email <remote_admin_email> --password @/path/to/remote-admin-password.txt", resolvedProfileID))
			}
		}
		profileID = resolvedProfileID

		remoteStorageCheck := support.DeployReadinessCheck{Name: "remote_download_storage", Required: true}
		if !adminSessionConfigured {
			remoteStorageCheck.Blocked = true
			remoteStorageCheck.Detail = "skipped: admin session is required"
			ready = false
		} else if !profileCheck.Passed {
			remoteStorageCheck.Blocked = true
			remoteStorageCheck.Detail = "skipped: remote profile session is required"
			ready = false
		} else {
			_, err := deps.RequestRemoteProxy(profileID, "POST", "/admin/download-storage/test", nil, nil, nil)
			if err != nil {
				remoteStorageCheck.Detail = err.Error()
				ready = false
				if profileTag != "" {
					nextSteps = append(nextSteps, fmt.Sprintf("landing-page-business-suite remote-profiles-proxy --profile-tag %s --method POST --path /admin/download-storage/test --json", profileTag))
				} else {
					nextSteps = append(nextSteps, fmt.Sprintf("landing-page-business-suite remote-profiles-proxy %s --method POST --path /admin/download-storage/test --json", profileID))
				}
			} else {
				remoteStorageCheck.Passed = true
				remoteStorageCheck.Detail = "remote download storage test succeeded"
			}
		}
		checks = append(checks, remoteStorageCheck)

		if appKey != "" {
			remoteAppKeyCheck := support.DeployReadinessCheck{Name: "remote_app_key", Required: true}
			if !adminSessionConfigured {
				remoteAppKeyCheck.Blocked = true
				remoteAppKeyCheck.Detail = "skipped: admin session is required"
				ready = false
			} else if !profileCheck.Passed {
				remoteAppKeyCheck.Blocked = true
				remoteAppKeyCheck.Detail = "skipped: remote profile session is required"
				ready = false
			} else {
				resp, err := deps.RequestRemoteProxy(profileID, "GET", "/admin/download-apps", nil, nil, nil)
				if err != nil {
					remoteAppKeyCheck.Detail = err.Error()
					ready = false
				} else {
					var parsed struct {
						Apps []struct {
							AppKey string `json:"app_key"`
							Name   string `json:"name"`
						} `json:"apps"`
					}
					if err := json.Unmarshal(resp, &parsed); err != nil {
						remoteAppKeyCheck.Detail = fmt.Sprintf("parse remote download apps: %v", err)
						ready = false
					} else {
						found := false
						foundName := ""
						for _, app := range parsed.Apps {
							if strings.TrimSpace(app.AppKey) == appKey {
								found = true
								foundName = strings.TrimSpace(app.Name)
								break
							}
						}
						if found {
							remoteAppKeyCheck.Passed = true
							if foundName != "" {
								remoteAppKeyCheck.Detail = fmt.Sprintf("remote app key exists (%s)", foundName)
							} else {
								remoteAppKeyCheck.Detail = "remote app key exists"
							}
						} else {
							remoteAppKeyCheck.Detail = fmt.Sprintf("remote app key missing: %s", appKey)
							ready = false
							if profileTag != "" {
								nextSteps = append(nextSteps, fmt.Sprintf("landing-page-business-suite remote-profiles-proxy --profile-tag %s --method GET --path /admin/download-apps --json", profileTag))
							} else {
								nextSteps = append(nextSteps, fmt.Sprintf("landing-page-business-suite remote-profiles-proxy %s --method GET --path /admin/download-apps --json", profileID))
							}
							nextSteps = append(nextSteps, "If you intend to onboard this app key, create it via remote proxy POST /admin/download-apps (see landing-page-desktop-upload for an example payload).")
						}
					}
				}
			}
			checks = append(checks, remoteAppKeyCheck)
		}
	}

	serviceAuthCheck := support.DeployReadinessCheck{Name: "service_auth", Required: *requireServiceAuth}
	serviceAuthResp, err := deps.ScenarioApp().Get("/usage/health", nil)
	if err != nil {
		serviceAuthCheck.Detail = err.Error()
		if serviceAuthCheck.Required {
			ready = false
		}
	} else {
		var parsed support.UsageHealthResponse
		if err := json.Unmarshal(serviceAuthResp, &parsed); err != nil {
			serviceAuthCheck.Detail = fmt.Sprintf("parse usage health response: %v", err)
			if serviceAuthCheck.Required {
				ready = false
			}
		} else if parsed.ServiceAuthConfigured {
			mode := strings.TrimSpace(parsed.ServiceAuthMode)
			if mode == "" {
				mode = "unknown"
			}
			serviceAuthCheck.Passed = true
			serviceAuthCheck.Detail = fmt.Sprintf("service auth enabled (mode=%s)", mode)
		} else {
			serviceAuthCheck.Detail = "service auth is disabled"
			if serviceAuthCheck.Required {
				ready = false
			}
		}
	}
	checks = append(checks, serviceAuthCheck)
	if !serviceAuthCheck.Passed {
		nextSteps = append(nextSteps,
			"vrooli credentials provision --identity vrooli/landing-page-business-suite --field consumer-signing-key",
			"landing-page-business-suite service-auth-status --require-enabled",
		)
		if profileTag != "" {
			nextSteps = append(nextSteps, fmt.Sprintf("scenario-to-desktop deploy-target test %s --require-service-auth", profileTag))
		}
	}
	if ready {
		nextSteps = append(nextSteps, "LPBS deploy readiness passed. Continue with scenario-to-desktop pipeline run ... --deploy-target <target> --app-key <app_key> --wait")
	}

	report := support.DeployReadinessReport{
		Ready:      ready,
		ProfileTag: profileTag,
		ProfileID:  profileID,
		Domain:     domain,
		Checks:     checks,
		NextSteps:  nextSteps,
		CheckedAt:  deps.Now().UTC().Format(time.RFC3339),
	}

	if *jsonOut {
		if err := cliapp.PrintReportJSON(os.Stdout, report); err != nil {
			return fmt.Errorf("encode readiness report: %w", err)
		}
		if !ready {
			return fmt.Errorf("deploy readiness checks failed")
		}
		return nil
	}

	opReport := cliapp.OperationalReport{
		Status:    []string{fmt.Sprintf("Status: %s", map[bool]string{true: "READY", false: "NOT READY"}[ready])},
		Triage:    []cliapp.TriageGroup{{Heading: "Checks", Items: renderReadinessChecks(checks)}},
		NextSteps: nextSteps,
	}
	if err := cliapp.RenderOperationalReport(os.Stdout, opReport); err != nil {
		return err
	}
	if !ready {
		return fmt.Errorf("deploy readiness checks failed")
	}
	return nil
}

func renderReadinessChecks(checks []support.DeployReadinessCheck) []string {
	items := make([]string, 0, len(checks))
	for _, check := range checks {
		status := "PASS"
		if check.Blocked {
			status = "BLOCKED"
		} else if !check.Passed {
			status = "FAIL"
		}
		required := ""
		if !check.Required {
			required = " (optional)"
		}
		items = append(items, fmt.Sprintf("[%s] %s%s: %s", status, check.Name, required, check.Detail))
	}
	return items
}

func serviceAuthNotConfiguredError() error {
	return support.RenderOperationalError(cliapp.OperationalReport{
		Status: []string{"Status: NOT READY"},
		Triage: []cliapp.TriageGroup{{Heading: "Auth Gate", Items: []string{"[FAIL] service_auth: service auth is not configured"}}},
		NextSteps: []string{
			"Provision the LPBS consumer signing key through the credential authority, then restart the scenario",
			"Verify LPBS runtime auth gate: landing-page-business-suite service-auth-status --require-enabled",
			"Verify desktop deploy auth gate: scenario-to-desktop deploy-target test <target-name> --require-service-auth",
		},
	})
}
