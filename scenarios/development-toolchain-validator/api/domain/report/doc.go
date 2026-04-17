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
//   - Conflicts: Cross-skill contradictions on references
//   - Drift: Skills whose content hash changed since connection
//   - Maturity: Skill maturity scoring based on expectation coverage
//   - Tool Baselines: Tool accuracy regression checks
//
// # Report Types
//
//	ConflictsReport: Cross-skill contradictions
//	  - Structural: incompatible expectations from different skills
//	  - CLI: overlapping assertions with different expected values
//
//	DriftReport: Aggregated drift across all connections
//	  - Compares stored hashes to current hashes
//	  - Unlike per-connection drift in skill domain, aggregates all at once
//
//	MaturityReport: Skill expectation coverage scoring
//	  - Low: no expectations
//	  - Medium: only structural OR only CLI
//	  - High: both structural AND CLI expectations
//
//	ToolBaselinesReport: Tool accuracy regression checks
//	  - Groups CLI results by tool name
//	  - Pass/fail/error counts per tool per reference
//
// # Integration Points
//
//   - skill.Repository: Fetch connections (bypasses service pagination)
//   - expectation repos: Fetch expectations per connection
//   - validation_runs/cli_results: Latest CLI results per reference
//
// # Domain Boundaries
//
// This domain handles:
//   - Aggregating validation results by reference, skill, category
//   - Calculating skill maturity scores
//   - Detecting cross-skill conflicts
//   - Aggregating drift across connections
//
// This domain does NOT:
//   - Run validations (see: validation domain)
//   - Store raw results (validation domain stores those)
//   - Manage connections (see: skill domain)
package report
