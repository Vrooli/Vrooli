# Invariants — Vrooli Memory

## Replay / Idempotency Invariants

- Journal entries, facet text, and embeddings are append-only. Import uses the stable key `sha256(runtime + path + normalized content)` and checks that key before any inference work. Replaying an unchanged source therefore performs no classification or embeddings and creates no journal entry.
- A harness import has one active durable run per runtime. A second `RunImport` joins that run and returns its existing ID; it never starts a competing scan.
- Progress is checkpointed after every source. A process restart may leave a historical run in `running`, but the journal remains replay-safe and a later start is an independent, content-addressed reconciliation rather than a destructive resume.
- `completed_with_errors` is explicit. It is never represented as `completed`.
