# Shared style-foundation evidence

The plan's baseline measured 67 reduced-motion, 47 forced-colors, and 32
focus-visible implementations, plus per-instance style injection.

The implementation now publishes `BaseStyles` and `StyleSheet`, and the
shared-style-ownership and style-injection gates have calibration fixtures.
The latest-source census on 2026-08-26 found:

- 14 distinct reduced-motion blocks;
- 16 distinct forced-colors blocks;
- 22 distinct focus-visible blocks;
- 3 latest source files containing `dangerouslySetInnerHTML` (these are
  markdown/code rendering paths, not `<style>` injection).

Those remaining blocks are scoped component behavior in historical or newly
added asset stylesheets, so the ownership gate does not treat every selector
as a duplicate global policy. The stronger numeric one-implementation target
is therefore not yet proven. It remains a named follow-on owned by the React
Component Library maintainers; this document prevents the foundation's
existence from being mistaken for completion of that stronger target.

Evidence: `go test ./internal/gates/...`, catalog calibration fixtures, and the
latest-source census command recorded in the work log.
