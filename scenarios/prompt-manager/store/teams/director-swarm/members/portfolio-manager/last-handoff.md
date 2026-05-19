### Portfolio state
Swarm Manager overview now reports 448 total items: backlog 310, completed 125, failed 6, in_review 6, queued 1; 58 active initiatives. Stats summary still disagrees with overview: dashboard backlog 105 and completed all time 52. Throughput worsened again: 56 created / 0 completed in 7d, 106 created / 31 completed in 30d; weekly velocity remains 0 for weeks starting 2026-05-03, 2026-05-10, and 2026-05-17.

### Accepted decisions applied
No newly accepted portfolio decisions found. Existing application markers remain present for `dec-1775254774942113200`, `dec-1776896266693850072`, `dec-1776896273747118927`, `dec-1776982737575948642`, and `dec-1777173379756603490`.

### Coverage checks
Social-media-scheduler coverage remains non-duplicative: `swarm-manager initiatives context --name social-media-scheduler` returned 404. Initiative AI search found only adjacent/non-covering initiatives, and backlog AI search found notification/release/payment-adjacent items but no scheduler scenario coverage.

### Proposed corrections
No new correction proposed because owned-context pending decisions remain at the skip threshold: `dec-1777312920606447957`, `dec-1778797170919215163`, and `dec-1778883472874563187`.

### Decisions raised
None this heartbeat. The active correction remains `dec-1778883472874563187`: prioritize GCT QA unblockers before expanding dependent GCT work. GCT rollups are still flat, and the same QA dependencies remain central.

### Knowledge entries written
`knw-1779142631492096608` on `initiative-portfolio-record/2026-05-18`.

Friction persists: prompt-manager CLI auto-rebuild fails because `/home/matthalloran8/.vrooli/bin` is read-only, and Storage Map still documents `knowledge-add --by` while live CLI requires `--caller-note`.