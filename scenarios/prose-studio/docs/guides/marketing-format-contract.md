# Marketing profile contract

Consumer-owned profiles may express each channel's writing shape without
changing Prose Studio's shared voice or inventing platform policy. The
consumer (Content Desk) owns these profile declarations; the content owner
must approve each exact draft before release.

## Use source facts as a generation boundary

For claim-grounded marketing drafts, set `constraints.require_artifacts` to
`true`. The request must include source facts in `artifacts`; without
them, generation stops before a session or candidate is persisted, with
`source_artifacts_required`. Prose Studio cannot verify that the caller's
artifacts are approved or current. `min_artifacts` gates the candidate on
how many distinct supplied artifact strings it quotes. Prose Studio's prompt
frames supplied artifacts as the only concrete specifics allowed, but neither
the prompt nor the substring gate proves entailment or prevents every stronger
rewrite. Content Desk remains responsible for claim verification, exact claim
citations and approval.

## Declare a format per consumer profile

Use distinct consumer-owned profiles for LinkedIn, r/vrooli, canonical site
articles, GitHub release notes, YouTube descriptions and X posts. Set
`required_format` according to the actual output shape (`rich_prose`,
`bullet_list`, `markdown`, etc.). A profile may set `max_code_points` as a
Unicode code-point ceiling. That count is a generic gate only: Channel Manager
remains authoritative for platform-specific counting, URL treatment, media,
disclosure and publication eligibility. A profile must not claim platform
compliance merely because a Prose Studio format gate passes.

## Keep the data boundary intact

Do not put a private voice corpus or exact personal draft in a consumer
declaration, repository, Prose Studio database or Content Desk database.
MP-DAY0's voice-permission choice remains pending until the operator answers
the existing card. Until then, use only clearly public source material and
generic style instructions; do not collect personal writing. A future
integration must prove where request text, candidate text, logs, cache and
receipts are persisted before it sends personal writing to Prose Studio.
The API's `persistence_mode: "private_ephemeral"` skips Prose Studio's
session, candidate, round and event records, requires a `local_only` profile,
rejects session resume and returns an opaque session ID. It does not prove
gateway, caller, operating-system or transport logs are safe; those paths and
host admission must be verified before personal input is sent. Exact response
text remains caller-held and must go only to the authorized private store.

The local `scripts/propose_private_style_edits.py` operation can turn a
candidate/edit pair into a **pending** exemplar or anti-pattern proposal. It
requires affirmative MP-DAY0 permission, accepts source files only from the
private corpus directory, stores exact deltas only under the MP3 private store,
and never applies them. The operator reviews each proposal before any style
file changes.

Any candidate retains machine-generated disclosure and may be rejected. The
slop detector is a reader-quality diagnostic, not an authorship verdict or a
way to evade disclosure or detector systems.
