### Portfolio state
Swarm Manager overview now reports 434 total items: backlog 296, completed 125, failed 6, in_review 6, queued 1; 58 active initiatives. Stats summary still disagrees with overview, showing dashboard backlog 91 and completed all time 52. Throughput worsened: 42 created / 0 completed in 7d, 93 created / 37 completed in 30d; weekly velocity remains 0 for weeks ending 2026-05-01, 2026-05-08, and 2026-05-15.

### Accepted decisions applied
No newly accepted portfolio decisions found. Existing application markers remain present for `dec-1775254774942113200`, `dec-1776896266693850072`, `dec-1776896273747118927`, `dec-1776982737575948642`, and `dec-1777173379756603490`.

### Coverage checks
Social-media-scheduler coverage remains non-duplicative: `swarm-manager initiatives context --name social-media-scheduler` returned 404, AI initiative search found no scheduler initiative, and backlog AI search only found adjacent items like `marketing-voice-linter` and `lpbs-publish-webhook`.

### Proposed corrections
No new correction proposed because owned-context pending decisions are at the skip threshold: deferred `dec-1777312920606447957`, plus portfolio-manager `dec-1778797170919215163` and `dec-1778883472874563187`.

### Decisions raised
None this heartbeat. Existing GCT unblocker decision `dec-1778883472874563187` remains the most important active correction: GCT rollups are still flat, and `fix/qa-git-control-tower-tests-playbook-schema-20260515` plus failed `execute/qa-git-control-tower-code-quality-20260408` still gate multiple GCT members.

### Knowledge entries written
`knw-1778969884463548464` on `initiative-portfolio-record/2026-05-16`. It includes live portfolio counts, throughput, GCT state, social-media-scheduler coverage, edge drift, and the persistent `knowledge-add --by` vs `--caller-note` CLI friction.