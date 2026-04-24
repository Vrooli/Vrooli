### Queue state
- Pending decisions: 3
- Read-only mode: no (3/12, well under ceiling)

### Proposals scored this heartbeat
- 3 scored
- Clean (no failure modes hit): 3
- Hit ≥1 mode: 0
- Hit ≥2 modes (rejection-eligible): 0

### Per-proposal scoring
- **dec-1777059142792794233** (oss-advertiser, content-publish-proposal, 5-tweet weekly dev-log): clean on all 8 modes. Feature claims commit-verifiable (cc4e99ad..bffe3f27af), builder-register voice, no metrics without `pending-telemetry` flag, OSS framed as invitation, coverage-missing being addressed not ignored, retention explicitly flagged `awareness-only`.
- **dec-1777059144293750532** (oss-advertiser, content-publish-proposal, 4-tweet initiative-agents thread): clean on all 8 modes. Same commit-verifiable claims, explicit "OSS self-host is invitation, not fallback", retention names dev-log rhythm + contributor onboarding anchor (not just awareness).
- **dec-1777062676053029079** (researcher, capability-gap for competitive-intel scanning): clean on all 8 modes. capability-gap decisions raise missing tooling; failure-mode 8 is satisfied by the paired notebook entry at `docs/marketing/notebook/AUDIENCE_OBSERVATIONS.md` (2026-04-24).

### Challenge notes written
- none (all 3 proposals clean against the eight failure modes)

### Separate observation (outside the eight modes)
- `queue-hygiene-observation/oss-advertiser-stacked-publish-proposals` (knw-1777064493675641067) — dec-...4233 and dec-...4532 are stacked publish-proposals from oss-advertiser (parallel-run artifact, 2s apart, overlapping source material). Neither trips a failure mode individually; stacking is a TEAM.md queue-discipline violation, not a drift concern. Operator pick-one recommended at vision walk.

### Aging scan
- Pending decisions >14 heartbeats: 0 (all 3 created 2026-04-24)
- Supersessions proposed: none
- Rejections proposed (aged out): none
- "Still relevant" notes written: none

### Rejections proposed this heartbeat
- none (no proposal hit ≥2 failure modes)

### Framework-update proposed
- none (parallel-run stacking is a single event affecting 2 proposals, not a recurring pattern across ≥2 independent events; defer until/unless it recurs next heartbeat)

### Supersessions
- none (zero prior pending `decision-rejection-proposed` / `framework-update` owned by marketing-contrarian)

### Forward signal for next heartbeat
- Watch for repeat parallel-run stacking across members — if it recurs, raise `framework-update` proposing a ninth mode or a codified pre-heartbeat queue-hygiene check.
- Neither oss-advertiser draft was rejected by contrarian; operator decides at vision walk.