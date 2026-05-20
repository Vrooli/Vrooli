### Portfolio state
Swarm Manager overview now reports 450 total items: backlog 312, completed 125, failed 6, in_review 6, queued 1; 58 active initiatives. Stats still disagrees: dashboard backlog 107 and completed all time 52. Throughput remains poor: 58 created / 0 completed in 7d, 106 created / 27 completed in 30d; weekly velocity remains 0 for weeks starting 2026-04-28, 2026-05-05, 2026-05-12, and 2026-05-19.

### Accepted decisions applied
No newly accepted portfolio decisions found. Existing markers remain for `dec-1775254774942113200`, `dec-1776896266693850072`, `dec-1776896273747118927`, `dec-1776982737575948642`, and `dec-1777173379756603490`.

### Coverage checks
`swarm-manager initiatives context --name social-media-scheduler` still returns 404. Initiative and backlog `search-ai` returned only adjacent/non-covering results; no scheduler scenario coverage found. Existing `dec-1777312920606447957` remains non-duplicative.

### Proposed corrections
No new correction proposed because owned-context pending decisions remain at skip threshold: `dec-1777312920606447957`, `dec-1778797170919215163`, and `dec-1778883472874563187`.

### Decisions raised
None. The active correction remains `dec-1778883472874563187`: prioritize GCT QA unblockers before expanding dependent GCT work. GCT rollups remain flat.

### Knowledge entries written
Wrote `knw-1779315437946157254` on `initiative-portfolio-record/2026-05-20`.

Friction persists: logical Storage Map files are not present in the merged workspace root, so storage must be accessed via CLI; Storage Map still documents `knowledge-add --by` while live CLI requires `--caller-note`.