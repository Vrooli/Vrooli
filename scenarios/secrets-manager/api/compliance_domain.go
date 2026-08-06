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

// calculateComplianceMetrics computes compliance metrics from credential coverage and security data.
func calculateComplianceMetrics(credentialStatus *CredentialCoverageStatus, securityResults *SecurityScanResult) ComplianceMetrics {
	coverage := CredentialCoverageStatus{}
	if credentialStatus != nil {
		coverage = *credentialStatus
	}

	summary := ComponentScanSummary{}
	vulnerabilities := []SecurityVulnerability{}
	riskScore := 0

	if securityResults != nil {
		summary = securityResults.ComponentsSummary
		vulnerabilities = securityResults.Vulnerabilities
		riskScore = securityResults.RiskScore
	}

	credentialCoverage := calculatePercentage(coverage.ConfiguredResources, coverage.TotalResources)
	securityScore := 100 - riskScore
	if securityScore < 0 {
		securityScore = 0
	}
	overallCompliance := (credentialCoverage + securityScore) / 2

	severityCounts := tallyVulnerabilitySeverities(vulnerabilities)
	configuredComponents := coverage.ConfiguredResources
	if summary.TotalComponents > 0 {
		configuredComponents += summary.ConfiguredCount
	}

	return ComplianceMetrics{
		CredentialCoverageHealth: credentialCoverage,
		SecurityScore:            securityScore,
		OverallCompliance:        overallCompliance,
		ConfiguredComponents:     configuredComponents,
		CriticalIssues:           severityCounts.critical,
		HighIssues:               severityCounts.high,
		MediumIssues:             severityCounts.medium,
		LowIssues:                severityCounts.low,
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
func buildComplianceResponse(metrics ComplianceMetrics, credentialStatus *CredentialCoverageStatus, securityResults *SecurityScanResult) map[string]interface{} {
	coverage := CredentialCoverageStatus{}
	if credentialStatus != nil {
		coverage = *credentialStatus
	}

	componentsSummary := ComponentScanSummary{}
	totalVulnerabilities := 0
	if securityResults != nil {
		componentsSummary = securityResults.ComponentsSummary
		totalVulnerabilities = len(securityResults.Vulnerabilities)
	}

	return map[string]interface{}{
		"overall_score":              metrics.OverallCompliance,
		"credential_coverage_health": metrics.CredentialCoverageHealth,
		"vulnerability_summary": map[string]int{
			"critical": metrics.CriticalIssues,
			"high":     metrics.HighIssues,
			"medium":   metrics.MediumIssues,
			"low":      metrics.LowIssues,
		},
		"remediation_progress":  metrics,
		"total_resources":       coverage.TotalResources,
		"configured_resources":  coverage.ConfiguredResources,
		"configured_components": metrics.ConfiguredComponents,
		"total_components":      componentsSummary.TotalComponents,
		"components_summary":    componentsSummary,
		"total_vulnerabilities": totalVulnerabilities,
		"last_updated":          time.Now(),
	}
}
