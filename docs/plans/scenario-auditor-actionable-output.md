# Scenario Auditor Actionable Output Plan

## Background & Pain Points
- The CLI `scenario-auditor audit` subcommand streams the raw job payload for both scans and terminates (security + standards) without any summarization (`scenarios/scenario-auditor/cli/scenario-auditor:516-724`). When a standards run reports thousands of violations, agents see megabytes of JSON that they cannot quickly triage.
- Each API job response embeds the entire findings set in-memory: `SecurityScanResult.Findings` already contains every issue emitted by the scanner (`scenarios/scenario-auditor/api/handlers_scanner.go:70-120`), and `StandardsCheckResult.Violations` mirrors that structure for compliance checks (`scenarios/scenario-auditor/api/handlers_standards.go:24-64`). Returning these verbatim to the CLI is necessary for UI/report use cases, but wastes bandwidth for loops that only need “what do I fix first?”.
- App-monitor and other UIs expect the full JSON today, so we cannot simply truncate the API payload. Instead we need a layered output contract: compact top-level summary for agent loops, plus retained access to the exhaustive dataset when humans or downstream services request it.

## Goals
1. Default CLI experience gives concise, severity-ordered guidance (counts, top violations, remediation hints) without manual jq parsing.
2. Full scan artifacts remain available (file path or explicit `--all` flag) so humans/other tooling can pull complete data.
3. API responses expose a structured summary block (counts, per-rule stats, recommended remediation order) to avoid duplicating logic in every client.
4. Introduce paging/filter knobs (limit, severity floor, grouping) so loops can request precisely the slice they need via a documented summary endpoint.
5. Document and test the new contract so ecosystem-manager prompts can rely on deterministic formatting.

## Non-Goals
- Rewriting scanner rule engines or changing how violations are detected.
- Replacing the existing storage backend for scan artifacts (reuse current JSON files on disk/logs).
- Building a UI feature in app-monitor during this iteration (but keep them unblocked).

## Proposed Architecture

### 1. Server-Side Summaries
1. **Add summary structs** for both scan types and embed them into the existing job payloads (`result.summary` for security/standards jobs, plus a combined block in the audit response). The CLI and other clients always read from these fields—no feature detection needed:
   ```go
   type ViolationSummary struct {
       Total            int                 `json:"total"`
       BySeverity       map[string]int     `json:"by_severity"`
       ByRule           []RuleCount        `json:"by_rule"`
       HighestSeverity  string             `json:"highest_severity"`
       TopViolations    []ViolationExcerpt `json:"top_violations"`
       Artifact         *ScanArtifactRef   `json:"artifact,omitempty"`
       RecommendedSteps []string           `json:"recommended_steps"`
   }
   ```
   Compute this immediately after the scan finishes (same place we currently assign `Result` in `markCompleted`; see `scenarios/scenario-auditor/api/handlers_scanner.go:249-340` and `scenarios/scenario-auditor/api/handlers_standards.go:268-420`) and attach it directly to the job payload before persistence.
2. **Persist artifacts** to disk (reuse `logs/` or a new `artifacts/scans/` folder) and record the relative path + checksum inside the summary. Store one artifact per scan (security vs. standards) so retries stay isolated while the CLI can still bundle them when a combined audit runs. Expose `GET /api/v1/scenarios/scan/jobs/{id}/artifact` (and the standards equivalent) to stream that file; when persisting a new artifact, immediately delete any older files that exceed the configured TTL so storage stays bounded without a separate janitor job.
3. **Severity ordering**: define a canonical order `critical > high > medium > low > info`, so summary generation can sort `TopViolations` deterministically. Include scenario + file snippet for each excerpt to maintain context without the full blob, and rely on existing rule metadata for remediation text—no extra BAS hooks.
4. **Top-violation buffer + paging**: keep the full violation arrays exactly as today, but also precompute a sorted buffer per severity that can satisfy arbitrarily large requests (e.g., first 1,000 entries per severity). Expose `GET /api/v1/scenarios/scan/jobs/{id}/summary` (and `/standards/jobs/{id}/summary`) with `limit`, `min_severity`, optional `cursor`, and `group_by=rule|file|scenario`. Responses mirror `ViolationSummary` plus `next_cursor`, letting CLI/app-monitor fetch more than the default 20 rows without recomputing scans.
5. **Rule grouping**: compute the most common standards/security rule IDs to help agents focus on systemic issues. If a rule exceeds a configurable threshold (e.g., >50 duplicates), collapse it into a single summary item with an aggregated count to prevent repeated spam.
6. **Recommendations**: echo each rule’s `Recommendation` text when present; if it is missing, fall back to a deterministic link in `docs/standards/<rule-id>.md` (or similar) so every summary row contains actionable guidance.

### 2. CLI Output & Flags
1. **Default view**: when `scenario-auditor audit` finishes, render a textual digest instead of dumping JSON:
   - Run status (completed/failed/timed out) with duration.
   - Summary table for security + standards (counts, highest severity, artifact path).
   - A dynamically sized "Top violations" section that scales up to 20 entries when thousands of findings exist but stays small for modest runs. Entries still include severity emoji, rule ID, file/line, and the rule-provided recommendation text.
   - `Next steps` section referencing `scenario status`, docs, or follow-up commands.
2. **Opt-in full payload**: add `--all` (stream the complete JSON, matching today’s behavior) and `--json` (print the summary JSON combining both scans). There is no bridge mode—legacy scripts must pass these flags explicitly if they still need full payloads.
3. **Filtering knobs**: add `--limit N`, `--min-severity <critical|high|...>`, and `--group-by rule|file|scenario` to adjust what shows up in the “Top violations” section. The CLI just forwards these values to the summary endpoint above so it stays a thin wrapper.
4. **Artifact retrieval + retention**: add `--artifact <path>` or `--download` to copy the referenced artifact to a user-specified file. On the server, store artifacts in a deterministic folder such as `logs/scenario-auditor/<scenario>/` with filenames `YYYYMMDD-HHMM_job-<id>.json`. Artifacts stay available until they age past the TTL; each new scan run triggers an immediate cleanup of expired files so we never need a background collector.
5. **Graceful degradation**: when a scan returns zero violations, show a celebratory message and skip artifact creation. Roll out by upgrading the API (and its docs/consumers) first, then release the CLI aligned to the new schema—no bridge mode or runtime negotiation needed.

### 3. Testing & Documentation
1. **Unit tests**: cover summary builders with representative fixtures (small, large, single-rule, multi-rule) to ensure sorting and aggregation behave as expected. Tests should live alongside existing handler tests (`scenarios/scenario-auditor/api/handlers_scanner_test.go` once created).
2. **CLI snapshot tests**: add bats or shell-based tests under `scenarios/scenario-auditor/cli/tests/` that mock the API responses and assert the formatted output for various flag combinations.
3. **Docs**: update `scenarios/scenario-auditor/README.md` and `docs/testing/reference/scenario-auditor.md` (if/when added) describing:
   - Default summary behavior and severity ordering.
   - How to fetch full results or artifacts.
   - Flag examples for filtering.
4. **Ecosystem-manager prompt**: once the plan lands, refresh `scenario-improver.md` to instruct agents to cite the summary snippet and attach artifact paths when referencing auditor results.

## Rollout Steps
1. Implement API summaries + artifact storage (behind feature flag if needed) and expose them via `result.summary` for both scan types.
2. Ship CLI changes after all API clients (including app-monitor) are updated; refuse to run if the server does not advertise the new summary schema.
3. Update docs/tests + coordinate with app-monitor so its UI consumes the new API summary fields (no legacy parsing layer).
4. Finally update ecosystem-manager prompts and any downstream automation to rely on the concise summary/flag behavior.

## Decisions from Clarification
- **Top violations sizing**: default to 20 entries but allow callers to request arbitrarily large slices via paging parameters.
- **Artifact persistence**: store per-scan artifacts with deterministic naming, purge anything older than the TTL when writing a new artifact, and reference both paths from the combined audit summary.
- **Snippet handling**: keep snippets intact (Option A); no additional redaction layer beyond existing repo access controls.
- **Remediation messaging**: surface the rule’s recommendation text, or link to rule documentation when the rule lacks inline guidance.
- **Rollout sequencing**: upgrade the API first (including UI clients), then release the CLI changes, then update ecosystem-manager/prompt consumers—never operate a dual-mode compatibility layer.
