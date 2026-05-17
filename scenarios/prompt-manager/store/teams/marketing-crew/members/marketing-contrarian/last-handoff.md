### Pending decisions
- 5 visible pending:
  - `dec-1778787137208717804` OSS evidence-path capability gap.
  - `dec-1778790757599330605` audience-update for OSS contributor first-run cognitive cost.
  - `dec-1778792544572466080` now-obsolete prior rejection/amendment proposal for `dec-1778790757599330605`.
  - `dec-1778873456617563354` OSS coverage-gap after stale first-publish rejection.
  - `dec-1778878881483178807` queue-hygiene proposal to reject/close obsolete `dec-1778792544572466080`.
- Owned-context pending remains 2: `dec-1778792544572466080` and `dec-1778878881483178807`. Skip new owned-context decisions if this reaches 3.

### Proposals scored
- `dec-1778787137208717804`: still relevant capability-gap; no challenge.
- `dec-1778790757599330605`: still passes after researcher response; acceptance should include the narrowed target/disposition details.
- `dec-1778792544572466080`: obsolete because the underlying hygiene challenge was resolved.
- `dec-1778873456617563354`: passes; keeps OSS coverage missing and blocks refreshed drafting until evidence-path access returns.
- `dec-1778878881483178807`: still valid queue hygiene for closing the obsolete rejection/amendment proposal.

### Challenge notes written
- None. No concrete new failure-mode hit found.

### Challenge resolution updates
- `knw-1778965248848270406`: `challenge-resolution-record/dec-1778790757599330605`, state `resolved`.
- `knw-1778965249016637218`: `challenge-resolution-record/dec-1778873456617563354`, state `resolved`.

### Aging scan
- `knw-1778965248848631416`: `aging-scan-note/dec-1778787137208717804`.
- Outcome: not stale, still relevant, no rejection/supersession/framework update.

### Rejections raised
- None this run.

### Framework updates
- None. No observed failure outside the current framework.

### Supersessions
- No direct supersession primitive exposed. Existing queue-hygiene decision `dec-1778878881483178807` remains the path to close obsolete `dec-1778792544572466080`.

### Knowledge entries written
- `knw-1778965248848631416` aging scan note.
- `knw-1778965248848270406` audience-update resolution re-check.
- `knw-1778965249016637218` OSS coverage-gap resolution re-check.