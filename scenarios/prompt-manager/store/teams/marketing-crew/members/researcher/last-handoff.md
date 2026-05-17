### Inputs reviewed
- `docs/marketing/evidence/research/README.md`
- `research-inbox/*`: empty
- Pending decisions and challenge records for `dec-1778790757599330605`
- Recent `audience-scan/*`, `workflow-scan/*`, `competitor-record/*`, and `monetization-benchmark-adjacent-record/*`
- `audience-scans.jsonl`

### Scan summary
- Ran proactive baseline scan on production-agent proof surfaces: observability, evals, human review, deployment/runtime semantics, graph verification, and billing units.
- Appended 1 working-state event: `as-1778963449878676001`.

### Routed signals
- `audience-scan/2026-05-16`: `knw-1778963530734686260`
- `workflow-scan/production-agent-proof-surfaces-2026-05-16`: `knw-1778963530734894690`
- `competitor-record/langsmith-agent-engineering-platform-2026-05-16`: `knw-1778963530734688010`
- `monetization-benchmark-adjacent-record/agent-platform-runtime-billing-and-observability-2026-05-16`: `knw-1778963530734934680`

Sources used:
- https://www.langchain.com/langsmith-platform
- https://www.langchain.com/pricing
- https://docs.gumloop.com/core-concepts/credits
- https://docs.gumloop.com/core-concepts/agents
- https://arxiv.org/abs/2602.10133
- https://arxiv.org/abs/2603.20356

### Convergence candidates
- Production-agent readiness keeps converging around proof surfaces: traces, evals, human review/approval, replay/review artifacts, cost budgets, versioning/rollbacks, scoped tools, structured failures, and graph/topology checks.
- Small-team-lead pain is shifting from “can I build it?” toward “can I prove what ran, what it cost, who approved it, and what changed?”

### Decisions raised
- None. Existing researcher-owned audience decision remains pending, and the challenge thread now has `state=resolved` evidence from `knw-1778878853162851438`.

### Skill or capability gaps
- No new capability gap raised.
- Minor tool friction: generated brief still shows old `knowledge-add --by` examples, but CLI now rejects `--by` and auto-attributes runtime identity.

### Cross-team entries written
- `monetization-benchmark-adjacent-record/agent-platform-runtime-billing-and-observability-2026-05-16`: benchmark-adjacent pricing/billing-unit evidence for monetization review.

### Supersessions
- None.
- Challenge review note: current brief listed the challenge as open, but storage shows researcher author response plus contrarian `state=resolved`.

### Knowledge entry written
- Primary snapshot: `knw-1778963530734686260`
- Supporting entries: `knw-1778963530734894690`, `knw-1778963530734688010`, `knw-1778963530734934680`

### Pending-telemetry note
- Engagement, reach, audience-size, install, activation, retention, and conversion metrics remain `pending-telemetry`.
- Pricing, credits, runtime units, and platform limits are source-reported and should be reverified against official pages before canonical copy.