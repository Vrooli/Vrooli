package vps

import (
	"strconv"
	"time"

	"scenario-to-cloud/domain"
)

// ComputeDrift compares manifest expectations vs actual live state.
func ComputeDrift(manifest domain.CloudManifest, liveState domain.LiveStateResult) domain.DriftReport {
	report := domain.DriftReport{
		OK:        true,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Checks:    []domain.DriftCheck{},
	}

	// Check main scenario
	scenarioRunning := isScenarioRunning(liveState.Processes, manifest.Scenario.ID)
	addCheck(&report, scenarioRunning, domain.DriftCheck{
		Category: "scenarios",
		Name:     manifest.Scenario.ID,
		Expected: "running",
	}, domain.DriftCheck{
		Message: "Main scenario is not running",
		Actions: []string{"restart_scenario"},
	})

	// Check expected resources
	for _, resID := range manifest.Dependencies.Resources {
		resourceRunning := isResourceRunning(liveState.Processes, resID)
		addCheck(&report, resourceRunning, domain.DriftCheck{
			Category: "resources",
			Name:     resID,
			Expected: "running",
		}, domain.DriftCheck{
			Message: "Expected resource is not running",
			Actions: []string{"restart_resource"},
		})
	}

	// Check for unexpected resources
	checkUnexpectedResources(&report, liveState.Processes, manifest.Dependencies.Resources)

	// Check for unexpected processes with ports
	checkUnexpectedPorts(&report, liveState.Processes)

	// Check Caddy/TLS
	checkCaddyState(&report, liveState.Caddy)

	return report
}

// addCheck adds a pass or drift check to the report based on the condition.
// passFields provides the base check info; driftFields provides additional info for failures.
func addCheck(r *domain.DriftReport, passed bool, passFields, driftFields domain.DriftCheck) {
	check := passFields
	if passed {
		check.Status = "pass"
		check.Actual = passFields.Expected
		r.Summary.Passed++
	} else {
		check.Status = "drift"
		check.Actual = "not running"
		check.Message = driftFields.Message
		check.Actions = driftFields.Actions
		r.Summary.Drifts++
	}
	r.Checks = append(r.Checks, check)
}

// isScenarioRunning checks if a scenario is running in the process list.
func isScenarioRunning(processes *domain.ProcessState, scenarioID string) bool {
	if processes == nil {
		return false
	}
	for _, s := range processes.Scenarios {
		if s.ID == scenarioID {
			return s.Status == "running"
		}
	}
	return false
}

// isResourceRunning checks if a resource is running in the process list.
func isResourceRunning(processes *domain.ProcessState, resourceID string) bool {
	if processes == nil {
		return false
	}
	for _, r := range processes.Resources {
		if r.ID == resourceID {
			return r.Status == "running"
		}
	}
	return false
}

// checkUnexpectedResources adds warnings for resources running but not in manifest.
func checkUnexpectedResources(r *domain.DriftReport, processes *domain.ProcessState, expectedIDs []string) {
	if processes == nil {
		return
	}
	expectedSet := make(map[string]bool)
	for _, res := range expectedIDs {
		expectedSet[res] = true
	}
	for _, res := range processes.Resources {
		if !expectedSet[res.ID] {
			r.Checks = append(r.Checks, domain.DriftCheck{
				Category: "resources",
				Name:     res.ID,
				Status:   "warning",
				Expected: "not specified in manifest",
				Actual:   "running",
				Message:  "Resource is running but not in manifest",
				Actions:  []string{"stop", "add_to_manifest"},
			})
			r.Summary.Warnings++
		}
	}
}

// checkUnexpectedPorts adds warnings for unexpected processes listening on ports.
func checkUnexpectedPorts(r *domain.DriftReport, processes *domain.ProcessState) {
	if processes == nil {
		return
	}
	for _, u := range processes.Unexpected {
		if u.Port > 0 {
			r.Checks = append(r.Checks, domain.DriftCheck{
				Category: "ports",
				Name:     strconv.Itoa(u.Port),
				Status:   "warning",
				Expected: "no unexpected process",
				Actual:   u.Command,
				Message:  "Unexpected process listening on port",
				Actions:  []string{"kill_pid"},
			})
			r.Summary.Warnings++
		}
	}
}

// checkCaddyState adds checks for Caddy and TLS status.
func checkCaddyState(r *domain.DriftReport, caddy *domain.CaddyState) {
	if caddy == nil {
		return
	}
	if caddy.Running {
		r.Checks = append(r.Checks, domain.DriftCheck{
			Category: "edge",
			Name:     "caddy",
			Status:   "pass",
			Expected: "running",
			Actual:   "running",
		})
		r.Summary.Passed++

		if caddy.TLS.Valid {
			r.Checks = append(r.Checks, domain.DriftCheck{
				Category: "edge",
				Name:     "tls",
				Status:   "pass",
				Expected: "valid certificate",
				Actual:   "valid certificate",
			})
			r.Summary.Passed++
		}
	} else {
		r.Checks = append(r.Checks, domain.DriftCheck{
			Category: "edge",
			Name:     "caddy",
			Status:   "drift",
			Expected: "running",
			Actual:   "not running",
			Message:  "Caddy reverse proxy is not running",
			Actions:  []string{"restart_caddy"},
		})
		r.Summary.Drifts++
	}
}
