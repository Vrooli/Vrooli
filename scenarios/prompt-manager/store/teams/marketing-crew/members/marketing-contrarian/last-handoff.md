### Pending decisions
- 5 visible pending after this run:
  - `dec-1778787137208717804` OSS evidence-path capability gap.
  - `dec-1778790757599330605` audience-update for OSS contributor first-run cognitive cost.
  - `dec-1778792544572466080` prior rejection/amendment proposal for `dec-1778790757599330605`.
  - `dec-1778873456617563354` OSS coverage-gap after stale first-publish rejection.
  - `dec-1778878881483178807` new queue-hygiene proposal to reject/close obsolete `dec-1778792544572466080`.
- Owned-context pending is 2: `dec-1778792544572466080` and `dec-1778878881483178807`. Skip new owned-context decisions if this reaches 3.

### Proposals scored
- `dec-1778787137208717804`: still relevant capability-gap; no challenge.
- `dec-1778790757599330605`: prior challenge now resolved by researcher response `knw-1778877137629609823`; acceptance should include those narrowed details.
- `dec-1778792544572466080`: now obsolete because the author answered the hygiene gap.
- `dec-1778873456617563354`: passes; it preserves coverage honesty and waits for evidence-path restoration before refreshed OSS post #1.

### Challenge notes written
- No new concrete failure challenge opened.
- Wrote `knw-1778878853316235721` as a score note for `dec-1778873456617563354`: no failure-mode hit, no challenge opened.

### Challenge resolution updates
- `knw-1778878853162851438`: `challenge-resolution-record/dec-1778790757599330605`, state `resolved`.

### Aging scan
- `knw-1778878853161060287`: `aging-scan-note/dec-1778787137208717804`.
- Outcome: not stale, still relevant, no rejection/supersession/framework update.

### Rejections raised
- `dec-1778878881483178807`: reject/close obsolete rejection/amendment proposal `dec-1778792544572466080` after author response resolved the underlying hygiene issue.

### Framework updates
- None. No new failure class observed outside the current framework and typed-observation rule.

### Supersessions
- No direct supersession primitive was exposed. Proposed queue hygiene via `dec-1778878881483178807`.

### Knowledge entries written
- `knw-1778878853161060287` aging scan note.
- `knw-1778878853162851438` challenge resolution record.
- `knw-1778878853316235721` coverage-gap score note.