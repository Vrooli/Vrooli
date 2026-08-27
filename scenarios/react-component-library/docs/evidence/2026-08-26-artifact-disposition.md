# Investigation artifact disposition

Reviewed 2026-08-26:

- `2026-08-26-library-defect-topology.html`
- `2026-08-26-versions-tab-redesign.html`

## Library defect topology

| Finding | Disposition |
| --- | --- |
| ControlBase blanket floor and six-rung mismatch | Resolved in ControlBase 1.1.0; geometry test covers all six rungs. |
| Voice glyph clipped by overlay container | Resolved in VoiceInputButton 4.3.0; toolbar regression tests pass. |
| 92 per-instance style injectors / competing style channels | Foundation and idempotent head injector are implemented; the active exported-source census now measures one implementation for each shared concern, and the injector test proves one sheet for 1, 3, and 10 instances. |
| 230 vacuous contracts | Ratchet implemented. Existing legacy contracts remain allowlisted by design; the allowlist may shrink but may not grow. |
| 20 idle claim evaluators / missing geometry reach | Kind presets and machine floors are implemented; residual corpus debt remains visible to experience-manager. |
| Foreign consumer palette classes | Gate and calibration are implemented. Historical residuals remain named for the palette-migration owner. |
| Deprecated historical imports | Active dependents migrated; immutable released sources remain visible to the deprecated-import gate and belong to version retirement. |

## Versions tab redesign

| Finding | Disposition |
| --- | --- |
| Four version projections were split across tabs | Resolved by `api/versionHistory.ts` and the joined VersionsCard row. |
| Hand-rolled row and diff viewer | Resolved by library `VersionRow`, `DataTable`, and `DiffViewer`; the page viewer was removed. |
| Missing captures, findings, adopters, and change summary | Resolved in expandable VersionRow composition and covered by 9 focused tests. |
| Required-token and zero-test evidence semantics | Resolved; zero runs render neutral `unknown`, and required tokens render as warnings. |
| Detail-page tab consolidation | Resolved as six component tabs: Preview, Overview, Files, Versions, Experience, Relationships. |
| Library composition gaps | All identified gaps are recorded as resolved in `2026-08-26-library-gaps.json`. |

## External/current-worktree findings

The malformed `resize-handle` catalog surface and the latest-source lint
errors were repaired in the same asset chain. The unrelated full-suite
failures and the known web-console locale defect remain outside this plan and
are not silently reclassified as plan success. The aggregate catalog command
now completes within the lifecycle deadline; its remaining findings are
corpus-wide debt, historical version debt, or explicitly deferred palette and
deprecated-import migration work.
