// DOC: docs/concepts/ARCHITECTURE.md#domain-modules
// DOC: docs/internal/SEAMS.md#change-axes
// DOC: PRD.md#OT-P0-009 (Validation Report API)
// DOC: PRD.md#OT-P1-009 (Reference Health Summary)
//
// Package report aggregates validation results into actionable reports.
//
// # Purpose
//
// Raw validation results (pass/fail per assertion) are too granular for
// decision-making. This domain aggregates them into:
//   - Comprehensive validation reports per reference
//   - Skill maturity scores
//   - Coverage maps
//   - Health summaries
//
// # Why This Domain Exists
//
// The meta optimization team and ecosystem-manager need high-level signals:
//   - "Is reference X healthy?" (not "Did assertion 47 pass?")
//   - "Which skills are poorly configured?" (not raw pass counts)
//   - "Where are coverage gaps?" (not file-by-file results)
//
// This domain transforms detailed validation data into actionable insights.
//
// # Domain Boundaries
//
// This domain handles:
//   - Aggregating validation results by reference, skill, category
//   - Calculating skill maturity scores
//   - Building coverage maps (which skills cover which files)
//   - Producing health summaries combining multiple signals
//
// This domain does NOT:
//   - Run validations (see: validation domain)
//   - Store raw results (validation domain stores those)
//   - Define what "healthy" means (configuration-driven)
//
// # Key Decisions (to be implemented)
//
// 1. Health Score Formula: Weighted combination of pass rates + tooling baselines
// 2. Maturity Score: Has config (weighted) + assertions pass (weighted) + no conflicts
// 3. Coverage Map: File-tree with skill coverage annotations
// 4. Report Caching: Cache aggregated reports, invalidate on validation run
//
// # Report Types
//
//   ValidationReport: Full breakdown for one reference
//     - All skill connections
//     - Per-skill structural/CLI results
//     - Overlaps and conflicts
//     - Unconfigured skills
//
//   HealthSummary: Single health score with breakdown
//     - Validation pass rate
//     - Tooling baseline status (auditor/test-genie/completeness)
//     - Skill maturity distribution
//     - Coverage percentage
//
//   CoverageMap: File-tree visualization data
//     - Files/folders with covering skills
//     - Uncovered areas
//     - Overlap zones
//
// # Integration Points
//
//   - validation domain: Fetch validation results
//   - skill domain: Fetch skill metadata for maturity calculation
//   - reference domain: Reference metadata for reports
//   - External tooling: Baseline results from auditor/test-genie/completeness
//
// # Status: PLACEHOLDER
//
// This package is a placeholder documenting intent. Implementation will include:
//   - model.go: ValidationReport, HealthSummary, CoverageMap, SkillMaturity
//   - service.go: Report generation and aggregation logic
//   - cache.go: Report caching with invalidation
package report
