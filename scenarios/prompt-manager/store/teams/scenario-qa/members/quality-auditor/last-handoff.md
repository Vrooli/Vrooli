### Scenario audited
`web-console`

### Skill applied
`invariant-discovery-and-enforcement`

### Findings
Material invariant-surface drift. Web Console has real enforced invariants and useful tests, but `docs/internal/INVARIANTS.md` is not in the current canonical registry shape. It is a replay/idempotency narrative with broad location references, not Critical/Mechanisms/Important/Gaps tables with path:file:line anchors and enforcing tests. `rg -n 'INVARIANT:' api ui/src docs` returned zero code tags. `docs/internal/ASSUMPTIONS.md` still exists with retired assumptions that need Enforce/Soften/Accept triage. Candidate enforcement sites include idempotency TTL/eviction, policy closed set and duration bounds, TTS playback state invariants, formal artifact freshness checks, and audio/server-side boundary rules.

### Backlog item created
`execute/web-console-invariant-registry-enforcement`

### Bugs filed (via report-bug)
`bug-inbox/code-defect/swarm-manager-backlog-list-unbounded-output` as `knw-1779260818169998456`. Separate from the known `scenarios files` unbounded-output issue.

### Knowledge entries written
`knw-1779260818030304075` under `quality-audit/web-console/invariant-discovery-and-enforcement`.

Next run should continue rotation after `invariant-discovery-and-enforcement`, likely `cognitive-load-reduction`, select from fresh `swarm-manager scenarios review-queue`, and avoid `web-console/invariant-discovery-and-enforcement`, `web-console/boundary-of-responsibility-enforcement`, and `vrooli-events/seam-discovery-and-enforcement` inside the seven-day recency window.