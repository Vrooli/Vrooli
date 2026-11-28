# Post-fix Ask latency and citation gate

Build: `sha256:aad9eda63bf907d8c8aea815aea55dcd8040a855862f076ecf0420d0b560e693`

The gateway answer decoder now removes repeated citation IDs while preserving
first-seen order. The Search service still validates that citations are present
and belong to the supplied result IDs. Both sequential 20-query gates passed.

| Run | Success | p50 | p95 | Maximum | Search/status p95 | Post-search p95 |
|---|---:|---:|---:|---:|---:|---:|
| 1 | 20/20 | 2,002.91 ms | 2,875.04 ms | 3,497.12 ms | 421.55 ms | 2,458.29 ms |
| 2 | 20/20 | 1,774.28 ms | 2,340.27 ms | 3,116.60 ms | 428.30 ms | 1,934.19 ms |

Each response passed the gate's unique citation-to-supplied-result-ID contract;
both p95 values are below the 8,000 ms target. The former `music player` 503
was the Search service rejecting a duplicate citation ID from the Gateway, not a
provider outage. The run-1 JSON is retained beside this summary as
`2026-11-22-postfix-run-1.json`; run 2's complete console JSON was not retained.
These latency runs do not establish independent answer-quality review.
