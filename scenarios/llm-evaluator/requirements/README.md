# Requirements Registry

This directory contains the requirements registry for the LLM Evaluator scenario, mapping PRD operational targets to technical requirements and tracking implementation/testing status.

## Module Structure

```
requirements/
├── index.json                      # Parent registry linking all modules
├── README.md                       # This file
├── 01-evaluation-jobs/
│   └── module.json                 # Job lifecycle requirements (OT-P0-001)
├── 02-core-metrics/
│   └── module.json                 # BLEU, ROUGE, semantic, LLM-judge (OT-P0-002..005)
├── 03-storage-retrieval/
│   └── module.json                 # PostgreSQL storage (OT-P0-006)
├── 04-dashboard/
│   └── module.json                 # UI visualization (OT-P0-007)
└── 05-api-integration/
    └── module.json                 # REST API layer (OT-P0-008)
```

## PRD Target Mapping

| Priority | Target ID    | Module                | Description                          |
|----------|--------------|----------------------|--------------------------------------|
| P0       | OT-P0-001    | 01-evaluation-jobs   | Job management & execution           |
| P0       | OT-P0-002    | 02-core-metrics      | BLEU score calculation               |
| P0       | OT-P0-003    | 02-core-metrics      | ROUGE score calculation              |
| P0       | OT-P0-004    | 02-core-metrics      | Semantic similarity (BERTScore)      |
| P0       | OT-P0-005    | 02-core-metrics      | LLM-as-Judge evaluation              |
| P0       | OT-P0-006    | 03-storage-retrieval | Results storage & retrieval          |
| P0       | OT-P0-007    | 04-dashboard         | Dashboard visualization              |
| P0       | OT-P0-008    | 05-api-integration   | REST API integration layer           |

## Lifecycle

1. Operational targets in PRD map to folders here (numbered for ordering, not priority)
2. `requirements/index.json` imports each module; tests auto-sync their status when they run
3. Coverage summaries live in `coverage/phase-results/` after each test phase

## Test Tagging

Tag tests with `[REQ:ID]` comments so auto-sync can update status:

```go
// [REQ:METRIC-BLEU-001] Test BLEU score calculation
func TestBLEUScore(t *testing.T) {
    // ...
}
```

```typescript
// [REQ:DASH-001] Test metric chart rendering
describe('MetricChart', () => {
    it('renders evaluation metrics', () => {
        // ...
    });
});
```

## Validation Commands

```bash
# Validate requirements structure
vrooli scenario requirements validate llm-evaluator --json

# Run tests and sync status
test-genie execute llm-evaluator --preset comprehensive

# View coverage summary
cat coverage/phase-results/summary.json
```

## Contributor Notes

- Add folders/modules that match your scenario's PRD targets (P0/P1/P2)
- Tag tests with `[REQ:ID]` so auto-sync can update status
- Never add compatibility shims during migrations—let things fail temporarily
- Keep this README under 100 lines and link to shared docs for schema details

See [Requirement Tracking Quick Start](/docs/testing/guides/requirement-tracking-quick-start.md) for full documentation.
