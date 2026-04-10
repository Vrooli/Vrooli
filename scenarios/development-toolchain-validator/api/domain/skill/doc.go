// DOC: docs/concepts/ARCHITECTURE.md#domain-modules
// DOC: docs/internal/SEAMS.md#change-axes
// DOC: PRD.md#OT-P0-002 (Skill Connection Management)
// DOC: PRD.md#OT-P0-003 (Skill Drift Detection)
//
// Package skill manages steer skill connections to reference scenarios.
//
// # Purpose
//
// Steer skills are markdown files in prompt-manager that guide AI agents.
// This package connects skills to reference scenarios, tracking:
//   - Which skills apply to which references
//   - Version/hash at connection time (for drift detection)
//   - Structural expectations and CLI assertions per connection
//
// # Why This Domain Exists
//
// The meta optimization team needs to know:
//   - Which skills affect which reference scenarios
//   - When skill content has changed since configuration (drift)
//   - What structural expectations each skill has
//
// Without this domain, there's no way to programmatically track
// which skills have been validated against which references.
//
// # Domain Boundaries
//
// This domain handles:
//   - Connecting/disconnecting skills to references
//   - Storing connection metadata (version, hash)
//   - Drift detection (comparing stored vs current version)
//   - Structural expectations per connection
//   - CLI tool assertions per connection
//
// This domain does NOT:
//   - Manage skills themselves (prompt-manager owns skills)
//   - Run validations (see: validation domain)
//   - Generate reports (see: report domain)
//
// # Key Decisions (to be implemented)
//
// 1. Version Pinning: Connections store version + content hash at connect time
// 2. Drift Detection: Compare stored hash against current prompt-manager version
// 3. Expectation Storage: Structural expectations stored as JSON per connection
//
// # Integration Points
//
//   - prompt-manager API: Fetch skill content, versions, metadata
//   - reference domain: Look up reference scenarios by ID/slug
//   - validation domain: Provide expectations for validation runs
//
// # Status: PLACEHOLDER
//
// This package is a placeholder documenting intent. Implementation will follow
// the pattern established in the reference domain:
//   - model.go: SkillConnection, Expectation, Assertion entities
//   - repository.go: Storage interface
//   - service.go: Business logic for connect/disconnect/drift
package skill
