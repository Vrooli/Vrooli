// Package main provides compliance metrics domain logic.
//
// This file contains the business rules for calculating compliance scores,
// tallying vulnerabilities by severity, and building compliance responses.
// Previously this logic was embedded in security_handlers.go.
package main

import (
	"time"
)

// vulnerabilitySeverityCounts holds counts of vulnerabilities by severity level.
type vulnerabilitySeverityCounts struct {
	critical int
	high     int
	medium   int
	low      int
}

// calculateComplianceMetrics computes compliance metrics from vault and security data.
func calculateComplianceMetrics(vaultStatus *VaultSecretsStatus, securityResults *SecurityScanResult) ComplianceMetrics {
	vault := VaultSecretsStatus{}
	if vaultStatus != nil {
		vault = *vaultStatus
	}

	summary := ComponentScanSummary{}
	vulnerabilities := []SecurityVulnerability{}
	riskScore := 0

	if securityResults != nil {
		summary = securityResults.ComponentsSummary
		vulnerabilities = securityResults.Vulnerabilities
		riskScore = securityResults.RiskScore
	}

	vaultHealth := calculatePercentage(vault.ConfiguredResources, vault.TotalResources)
	securityScore := 100 - riskScore
	if securityScore < 0 {
		securityScore = 0
	}
	overallCompliance := (vaultHealth + securityScore) / 2

	severityCounts := tallyVulnerabilitySeverities(vulnerabilities)
	configuredComponents := vault.ConfiguredResources
	if summary.TotalComponents > 0 {
		configuredComponents += summary.ConfiguredCount
	}

	return ComplianceMetrics{
		VaultSecretsHealth:   vaultHealth,
		SecurityScore:        securityScore,
		OverallCompliance:    overallCompliance,
		ConfiguredComponents: configuredComponents,
		CriticalIssues:       severityCounts.critical,
		HighIssues:           severityCounts.high,
		MediumIssues:         severityCounts.medium,
		LowIssues:            severityCounts.low,
	}
}

// calculatePercentage computes an integer percentage avoiding division by zero.
func calculatePercentage(part, total int) int {
	if total == 0 {
		return 0
	}
	return (part * 100) / total
}

// tallyVulnerabilitySeverities counts vulnerabilities by severity level.
func tallyVulnerabilitySeverities(vulnerabilities []SecurityVulnerability) vulnerabilitySeverityCounts {
	counts := vulnerabilitySeverityCounts{}
	for _, vuln := range vulnerabilities {
		switch vuln.Severity {
		case "critical":
			counts.critical++
		case "high":
			counts.high++
		case "medium":
			counts.medium++
		case "low":
			counts.low++
		}
	}
	return counts
}

// buildComplianceResponse constructs the compliance API response payload.
func buildComplianceResponse(metrics ComplianceMetrics, vaultStatus *VaultSecretsStatus, securityResults *SecurityScanResult) map[string]interface{} {
	vault := VaultSecretsStatus{}
	if vaultStatus != nil {
		vault = *vaultStatus
	}

	componentsSummary := ComponentScanSummary{}
	totalVulnerabilities := 0
	if securityResults != nil {
		componentsSummary = securityResults.ComponentsSummary
		totalVulnerabilities = len(securityResults.Vulnerabilities)
	}

	return map[string]interface{}{
		"overall_score":        metrics.OverallCompliance,
		"vault_secrets_health": metrics.VaultSecretsHealth,
		"vulnerability_summary": map[string]int{
			"critical": metrics.CriticalIssues,
			"high":     metrics.HighIssues,
			"medium":   metrics.MediumIssues,
			"low":      metrics.LowIssues,
		},
		"remediation_progress":  metrics,
		"total_resources":       vault.TotalResources,
		"configured_resources":  vault.ConfiguredResources,
		"configured_components": metrics.ConfiguredComponents,
		"total_components":      componentsSummary.TotalComponents,
		"components_summary":    componentsSummary,
		"total_vulnerabilities": totalVulnerabilities,
		"last_updated":          time.Now(),
	}
}
