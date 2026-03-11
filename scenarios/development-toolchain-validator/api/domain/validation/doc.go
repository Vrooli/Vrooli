// DOC: docs/concepts/ARCHITECTURE.md#domain-modules
// DOC: docs/internal/SEAMS.md#change-axes
// DOC: PRD.md#OT-P0-006 (Structural Validation Engine)
// DOC: PRD.md#OT-P0-007 (CLI Tool Validation Engine)
// DOC: PRD.md#OT-P0-008 (Skill Overlap Detection)
//
// Package validation runs structural and CLI tool validations against references.
//
// # Purpose
//
// This is the core engine of the development-toolchain-validator. It executes
// configured expectations against reference scenarios and produces pass/fail
// results per expectation, per skill connection.
//
// # Why This Domain Exists
//
// Steer skills are prose guidance. To validate them programmatically:
//   - Structural expectations check file/folder existence and content
//   - CLI assertions check tool outputs against expected values
//   - Overlap detection finds conflicting expectations between skills
//
// Without this domain, skill validation would require running expensive
// LLM-based interpretation loops instead of fast, deterministic checks.
//
// # Domain Boundaries
//
// This domain handles:
//   - Running structural validations (file exists, glob matches, content snippets)
//   - Running CLI tool validations (execute command, parse JSON, evaluate assertion)
//   - Detecting overlaps (multiple skills expecting same file/folder)
//   - Producing per-expectation, per-skill results
//
// This domain does NOT:
//   - Define expectations (see: skill domain)
//   - Modify reference scenarios (read-only validation principle)
//   - Aggregate results into reports (see: report domain)
//
// # Key Decisions (to be implemented)
//
// 1. Assertion Operators: eq, neq, gt, gte, lt, lte, exists, contains, matches, between
// 2. JSON Parsing: Use JSONPath for extracting values from CLI output
// 3. Pass/Fail Granularity: Results per assertion, not per connection
// 4. Overlap Detection: File-path level comparison of structural expectations
//
// # Integration Points
//
//   - skill domain: Fetch expectations/assertions for validation runs
//   - reference domain: Resolve scenario paths
//   - External CLIs: scenario-auditor, test-genie, scenario-completeness-scoring
//   - report domain: Provide raw results for aggregation
//
// # Validation Engine Flow
//
//	                 ┌─────────────────┐
//	                 │ Reference Path  │
//	                 └────────┬────────┘
//	                          │
//	         ┌────────────────┼────────────────┐
//	         ▼                ▼                ▼
//	┌─────────────────┐ ┌──────────────┐ ┌─────────────────┐
//	│  Structural     │ │   CLI Tool   │ │    Overlap      │
//	│  Checker        │ │   Executor   │ │    Detector     │
//	│  (files/folders)│ │  (--json)    │ │  (cross-skill)  │
//	└────────┬────────┘ └──────┬───────┘ └────────┬────────┘
//	         │                 │                  │
//	         └────────────────┬┴──────────────────┘
//	                          ▼
//	                 ┌─────────────────┐
//	                 │ ValidationResult│
//	                 │  per expectation│
//	                 └─────────────────┘
//
// # Status: PLACEHOLDER
//
// This package is a placeholder documenting intent. Implementation will include:
//   - model.go: ValidationResult, ExpectationResult, AssertionResult entities
//   - structural_checker.go: File/folder/content validation
//   - cli_executor.go: Subprocess execution with JSON parsing
//   - assertion_evaluator.go: Operator implementations
//   - overlap_detector.go: Cross-skill expectation analysis
package validation
