# Fixture: writer-skill-mismatch

A writer skill's `SKILL.md` references a topic prefix that is **not** in
its `skill.json::writes_to[]`. This is the canonical writer-skill drift
the kind-conditional skill rule exists to catch.

Setup:
- Skill `report-friction` is tagged `writer-skill` and declares
  `writes_to: ["friction-inbox/*"]` in `skill.json`.
- Its `SKILL.md` references `bug-inbox/<scope>/<slug>` — a prefix outside
  the declared `writes_to[]`.

Expected: a single `prose_topic_leak` warning rooted at
`skill:report-friction` with prefix `bug-inbox/<scope>/<slug>`, matched by
the `inferred-backtick-topic-ref` pattern.

This fixture proves that writer-skill drift is caught even when the skill
*does* declare some writes — i.e., the join is per-prefix, not per-skill.
A regression that loosened the join to "skill has any writes_to[]" would
silently accept any topic ref on any writer skill; this golden catches
that regression.
