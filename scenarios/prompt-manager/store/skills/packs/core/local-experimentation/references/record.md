# Experiment record

Keep this evidence in the existing effort record. Use structured JSON/CSV only when a consumer needs it; include only fields relevant to the selected route. Reuse owner receipts instead of copying their state. Keep live effects separate from retained files and shared CLI freshness artifacts.

The investigator maintains its own run's navigable guide, concise current state, evidence links and owned-effect inventory throughout the work and at handoff; organization is part of the research, not a separate delivery project. Coordinate shared topic indexes with their designated writer.

## Reading path

Follow `path:docs/internal/research-experimentation/README.md` to the canonical runtime research home and register the work under its enduring topic. Use that topic's `runs/<dated-run-id>` for a new commission; old experiment/scratch paths are compatibility aliases. Read `path:docs/internal/research-experimentation/STORAGE.md` for placement and retention. Reuse the topic for related runs; a new commission, an in-run phase and a changed workspace have different meanings. Keep one run identity when its workspace changes, with a pointer from the old location.

Give the workspace one short reading guide linking its topic, brief, current result, feedback/decisions and evidence/cleanup. An existing report can be that guide; create no empty parallel files. Identify phase reports as historical and keep immutable evidence paths intact.

Use progressive disclosure on entry and resume: read the topic guide, current brief and short current view first. A run whose existing checkpoint is a long, externally linked historical record may use a short `CURRENT.md` as that view without breaking historical links; otherwise use the short checkpoint itself. Consult a targeted prior result, feedback or receipt only when it can change the next decision, prevent a repeated failure, or verify a consequential claim. Do not read cumulative histories or full logs by default. Summaries must retain material negative results, uncertainty, user feedback and exact links so a later investigator can inspect the underlying evidence without losing context.

At a material finding or closure, update the run guide and propose the topic's changed understanding, decision state and next question. Use one topic-index writer when researchers run concurrently. Register the run relationship in the existing catalog; include the topic and run-guide locators in its normal work-record/ledger handoff. Topic summaries are dated synthesis, not live goal state or scientific acceptance receipts.

## Current state

Keep one short replace-in-place checkpoint: outcome verdict, changed finding/counterevidence, ranked next decision, cumulative consumed/remaining allowance, actual native state and live effect owners/cancellation routes. Link receipts and one current synthesis; keep chronology in linked history rather than appending another apparent current state. Preserve original phase results. A fresh agent should continue from the topic guide and this view without reconstructing historical finals. Store feedback once and link its disposition.

If a checkpoint has become a history, distill its current actionable state in place and retain the previous material in the existing evidence or history location with links. Do not discard unreconciled effects or competing findings. Keep the summary proportional to the decisions it serves; no arbitrary length cap should hide important context.

For retained Codex usage, use `web-console session native-usage <session-id> --threads <comma-separated-thread-ids> --start <RFC3339> --end <RFC3339> --json`. Select coordinator and worker/follow-up threads explicitly within that session; child membership is not inferred. The bounded typed owner deduplicates responses and separates cached input, uncached input and output. It requires the read capability or a verified exact own-session binding; a path or session label is not authority. Missing/partial records and pricing remain unknown. This read measures retained usage, not live continuation or deadline enforcement.

## Persist and discover

At closure, write one `vrooli-memory journal note --kind work-record` and append one row to `path:docs/internal/research-experimentation/LEDGER.md`. Include `research-experiment:<family>`, absolute workspace, subject keywords and disposition (`adopt`, `investigate`, `reject`, `narrow`, `partial`). Make one bounded `search-hub query "<subject>" --type record,doc` readback. Retain its exact handle or failure; durable persistence and retrieval are separate. An unavailable index means visibility unverified, not empirical failure. Reuse the existing owner issue and end without repeated logging or indexing retries.

## Evidence as applicable

- **Commission:** owner, destination, authorization/exclusions, actual executor permissions, repository revision and dirty-path snapshot, overlapping work.
- **Question:** hypothesis, baseline, same input hashes/briefs, acceptance and rejection criteria fixed before output review.
- **Budget:** route count, exact trial count/seeds, per-trial and total wall time, peak GPU/RAM, headroom, acquisition/expanded/cache/staging/output/retained disk, stop conditions, cancellation trial.
- **Acquisition ledger:** route, source URL, immutable revision and artifact hashes, license/terms evidence and unresolved conditions, dependency-owner preview/receipt, download and expanded sizes, install/cache paths, acquired or not acquired.
- **Source access:** effective period, identifiers, field meanings, pagination/quota/retry limits, and documentation versus verified retrieval. For credentialed routes, include only opaque owner/item references, destination, authentication placement, temporary-use identity/expiry, and closure evidence. Separate retained provider keys from task-owned access; omit values and capability tokens.
- **Admission:** observation time, current availability and claims, peak requirement across all stages, policy, owner receipt, allowed/refused/unknown and reason. Distinguish vendor estimates from measurements.
- **Trial ledger:** id, route, input hashes, command/typed arguments, cwd, sanitized environment, seed/settings, version/closure, job/Program Runtime ids and digests, start/end, outcome, wall time, peak metrics and sampling limitations, disk, outputs/hashes, failures/interventions.
- **Review:** usable portion, motion/artifacts, reference/identity consistency, reproducibility limits, quality versus baseline, cost including setup/selection/recovery, reviewer and exact artifact.
- **Recovery/cleanup:** cancellation target and receipt, owned process/lease state, partial/cache inventory, reconciled retry, cleanup preview/action/readback, remaining effects and recovery owner.
- **Decision:** adopt/investigate/reject per route, evidence, maintained owner/change/tests, acceptance asset, unresolved gap and next check.
- **Retained inventory:** path, bytes, hash, purpose, source/license, owner and retention/removal decision. Include failed receipts and unapplied patches.
- **External source map:** physical workspace path, immutable source and copied paths/hashes, inclusion reasons, dependency closure limits, editable surface, frozen evaluator, unchanged-copy fidelity, representative transfer checks, and current-source integration differences.
- **Adaptive search:** candidate selection rationale, competing explanations, protocol amendments, search versus confirmation results, discarded warmups, failed attempts, and agent-token cost.
- **Unattended supervision:** worker/session identity, absolute deadline, independent stop receipt, check-in schedule, active effect owner/IDs, feedback, remaining effects, and any requested notification receipt. A requested, accepted, and delivered notification are distinct states.
- **Research disposition:** retained finding/prototype, proposed maintained integration, and actual integration state. A useful result does not require building a new scenario.
