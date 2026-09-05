# Web Search

Web-search provides live web results, cited research, and a persistent findings ledger. Search Hub federates its live and stored-finding providers. The API, CLI, and UI share generated Connect-RPC contracts.

## Start and validate

```bash
make -C scenarios/web-search start
vrooli scenario test web-search
# Use the run ID returned above and attach once:
test-genie runs wait --json web-search "<run-id>"
```

Scenario lifecycle owns builds, ports, resources, and health. Use `vrooli scenario port web-search API_PORT` to discover the API. See [live validation](docs/operations/LIVE_VALIDATION.md) for attended research checks and execution-specific evidence.

## Agent capabilities

The three-speed stack keeps judgment in skills, repeated workflows in governed programs, and authoritative evidence decisions in the scenario.

| Layer | Surface | Responsibility |
|---|---|---|
| Skills | `web-search`, `web-search-investigate`, `web-search-improve` | Choose sources and freshness, judge coverage and contradictions, verify outcomes, improve methods |
| Programs | `web-search.research`, `web-search.compare-sources`, `web-search.research-l3` | Bounded research, complete evidence handoff, idempotent start and one owner wait |
| Programs | `web-search.record-attempt`, `web-search.learning-read`, `web-search.setpoint-read`, `web-search.findings-curate` | Durable outcome capture, comparable learning cohorts, diagnostic reads and curation proposals |
| Scenario | `research answer`, findings operations, declared `web-search/research` workflow | Evidence eligibility, source-linked findings, execution ownership and structured results |

Read a skill with `prompt-manager skill read <name>`. Program contracts and source live in `.vrooli/program-runtime/` and are discovered by Program Runtime from this scenario.

## Research

```bash
# Complete raw hits, without synthesis:
program-runtime library run web-search.research --input query=Python,effort=l0

# Current page-grounded evidence:
web-search research answer "question" --effort l2 --json

# Explicit stored reuse, subject to evidence eligibility:
web-search research answer "question" --max-age-seconds 3600 --json

# A declared, bounded agent investigation:
web-search research l3 "question" --idempotency-key "<stable-key>" --json
web-search research wait "<run-id>" --timeout-seconds 60 --json
```

L0 returns raw URLs and snippets. L1 adds snippet-grounded synthesis. L2 fetches pages and synthesizes cited evidence. L3 decomposes a question, researches gaps, returns structured claims and unresolved issues, and captures supported findings with L3 provenance.

Current evidence is the default: a request with zero maximum age bypasses the live-results cache. Stored reuse requires an explicit age budget and an eligible finding from the exact originating query, or an explicitly selected `finding_id`. Active status, known retrieval dates, confidence, citations, and source requirements still apply. Semantic similarity alone does not establish answer sufficiency.

`source_domains` permits matching hosts and their subdomains. `minimum_sources` counts distinct hostnames; the skill must still judge publisher independence. Responses distinguish stored findings, cited synthesis, raw hits, and unresolved evidence. Source outages, insufficient support, capture failures, and oversized output remain explicit. L0/L1 never capture; L2 capture is opt-in.

L3 IDs identify declared workflow executions, replacing legacy raw Agent Manager run IDs. A wait timeout preserves the same execution; reattach without starting a replacement. Agent Manager owns budgets and terminal state. Successful agent completion is separate from an answered, partial, or abstained research result. Structural checks reject unsupported success-shaped output; factual correctness still requires source review.

## Learning and improvement

Findings store external facts. The `web-search-usage` Vrooli Memory scope stores research attempts, reusable advice, and method observations. Before recording verified success, check citation support, freshness, coverage, and unresolved contradictions against referenced evidence.

`web-search.record-attempt` persists the quality assessment and selected contributing findings in one idempotent Memory write. It does not increment legacy finding-use counters. Retrieval is not successful use. Failed, unavailable, and unknown attempts remain in the outcome record; test observations cannot establish an operator baseline.

`web-search.learning-read` reads fixed comparable windows. Empty or truncated cohorts remain unreliable and targets remain unknown until real observations establish them. The improvement skill promotes recurring successful work into versioned programs, repairs stable evidence rules in the scenario, and removes redundant work above those operations. More captures or less live traffic alone do not prove improvement.

## Dependencies and limits

SearXNG supplies candidates; Ollama supplies synthesis. L2 fetches HTTP content first and can use Browser Automation Studio for pages needing browser execution. Agent Manager owns L3. Search Hub owns federation, and Vrooli Memory owns outcome storage and learning measures. Each path reports unavailable dependencies or partial evidence instead of inventing answers.

Findings remain authoritative in SQLite; the regenerable vector index resolves `storage.Collection("findings")` under the lifecycle instance namespace, so live and shadow do not share a collection.

Live search-engine availability changes independently of this scenario. Aggregate cache/governor measures and comparable operator learning baselines remain explicit measurement gaps; per-call degradation signals are available. See [operational targets](PRD.md), [requirements](requirements/index.json), and [live acceptance](docs/operations/LIVE_VALIDATION.md) for the evidence required before claiming maturity.
