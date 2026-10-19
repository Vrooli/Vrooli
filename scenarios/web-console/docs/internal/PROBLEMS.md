# Web Console — Known Problems and Risks

## 2026-10-18 — Imported GLB clearcoat/specular rejection (bounded W3 repair)

Operator reports adapted-iphone15promax.glb still failing after the authored
phone repair. Reuse scientific-debugging, scenario-work-ladder and the bounded
harness workflow from the preceding repair. Established expected behavior is
unchanged; scoped W3 implementation, no contract or readiness certification.
Recall found prior records 52fd6ce2-3d5d-45a8-b299-37bffb343248 and
48b9a6f8-db0f-4fa3-98f1-dcf3ee61ffb5; ranking/scoped recall partly degraded.

Hypotheses: (1) clearcoat/specular material extensions are rejected by preflight
despite installed GLTFLoader support; (2) embedded textures or geometry exceed
limits; (3) browser rendering fails independently. Tests: exact source preflight,
portable material preservation/security regressions, then actual narrow/desktop
rendering of both reported phones. Initial exact source fails `unsupported` in
/tmp/wc-glb-imported-exact-before.log. It is 1,112,040 bytes, 30 mesh definitions,
31 nodes, two embedded PNGs and no skins, animations, sparse accessors or
compressed primitives. Its only extensions are KHR_materials_clearcoat and
KHR_materials_specular, both implemented by installed Three.js 0.185.1.

Harness handoff: implement bounded clearcoat/specular support as specified in
ARCHITECTURE.md, preserve all resource/network/unsupported-feature guards, and
retain the previous transmission fix. Prove failing-then-passing portable tests,
exact file preflight, focused model tests, types/lint/build, scoped Test Genie
unit receipt, and served Chromium phone/desktop checks for both models. Track
findings here; finish after two fresh reviews find nothing material. No new
dependencies, shell changes, source-model edits, commit or full certification.
If interrupted, checkpoint remaining and unverified work here.

The two portable GLB/glTF preservation regressions fail before repair, as does
exact source admission. The bounded material policy now shares its allowlist
with scalar/texture validation for transmission, clearcoat and specular. Both
reported source files pass full preflight after repair; resource/network guards
remain unchanged. No new decoder, dependency or renderer plugin is needed.

Pre-close schema review caught that specularColorFactor permits HDR channels
above one in the canonical Khronos schema. Corrected the initial SDR-only
validation and added an accepted HDR case plus a retained finite numeric cap.
A normal-scale test initially lacked otherwise valid image bytes, so a later
image-reference failure could have hidden missing scale validation; supplied
a valid embedded image and positive scale controls before finalizing the test.
These are resolved in-scope findings; restart/browser proof must use the final
HDR-compatible build. Focused tests so far pass 59 in three files; types/lint
are clean. Test Genie unit run 20261019-031049-d6c1ff15 is admitted and has one
durable waiter; its mutable shared inputs may overlap these refinements.

Final focused check: 59/59 model tests in three files pass; final types and
scoped lint pass with empty diagnostics (/tmp/wc-glb-imported-{tests,types,lint}-final.log).
The previous phone preflight also passes (/tmp/wc-glb-imported-prior-phone.log).
The final HDR-compatible lifecycle restart is pending before browser checks.

Fresh review 1 after focused checks: compared each admitted extension field
against the canonical Khronos schemas and installed loader code. Reviewed
factor endpoints/defaults, HDR color shape/range, normal scale, required versus
advisory extension admission, unlit exclusions, texture indices, unknown
extensions and image decode/network limits. No material finding remains after
the HDR and test-isolation corrections. No changes to unrelated UI or files.

Served browser proof passes at 390px and 1440px: exact imported phone reports
30 rendered meshes / 8,557 triangles, screen and lens textures are visible,
zoom/fit/orbit/wireframe/background work, context loss recovers, and replacement
releases the old context. The prior authored phone also reaches ready at both
widths. No page errors, external requests or horizontal overflow. Evidence:
/tmp/wc-glb-imported-browser.{json,log}; inspected screenshots
/tmp/wc-glb-imported-{390,1440}.png and light variants. Final healthy/readiness
build: sha256:2ba56bf90e05571331b7022307438bdc9ba2965cfd9cd12f59cbabc65ec4ccab. Desktop Focus uses keyboard
activation around the previously reported approval-banner pointer overlap
(knw-1792378488243184087); that unrelated issue is unchanged, not re-reported.
Native Safari/iPhone remains unverified; mobile proof is touch/Chromium emulation.

Fresh review 2: checked final narrow/desktop rendered output, material/texture
preservation, both reported phones, error/recovery and resource ownership paths,
and portable tests without private fixture paths. No new material issue found.
No product changes after final browser proof. Remaining: terminal unit receipt
and completion journal, not additional implementation.

Final unit run 20261019-031049-d6c1ff15 is terminal FAIL after 352 seconds.
One durable waiter completed inline (AM parking token expired). Inspected the
unfiltered catalog and opaque findings.report artifact
artifact_ab1bb35fac90341f69c44eec492a40d0. The existing approvals keyboard/REST,
locale shape/interpolation failures, image-test provider-policy drift and
83.8% versus 85% function-coverage threshold remain; no model assertion failure
is reported. Existing bug records cover these families; no duplicate report,
threshold relaxation or unrelated repair. Evidence: /tmp/wc-glb-imported-testgenie-
{result,catalog,findings}.json and /tmp/wc-glb-imported-unit-diagnostics.txt.
The receipt spans shared-worktree refinements; do not claim isolated exact-input
or whole-scenario certification. All owned suite/lifecycle/browser operations
are terminal. The repair boundary is complete. Completion work-record:
91379ca5-abc6-48f8-b938-0bf8582bad78 (/tmp/wc-glb-imported-work-record.json).
No required in-scope work remains.


## 2026-10-18 — Blender GLB transmission material rejection (bounded W3 repair)

Operator reports original-iphone16pro-iter03.glb failing with the unsupported
features fallback (IMG_1753.png). Established expected behavior: inspect bounded,
self-contained static models with their authored materials. This is a localized
W3 implementation repair, not a contract or scenario readiness review.

Recall: prior 3D implementation record 48b9a6f8-db0f-4fa3-98f1-dcf3ee61ffb5
and ARCHITECTURE.md “3D model inspection”; search ranking was degraded. Prior
external-resource and allocation protections are relevant, with no evidence of
their earlier failure recurring. Source GLB is 2,583,052 bytes, 57 meshes,
no skins/animations, one embedded PNG and KHR_materials_transmission.

Hypotheses before repair:
1. Preflight rejects the transmission material extension although the installed
   GLTFLoader implements it. Test: inspect extension payloads and run the exact
   file through prepareModel; admit transmission in a controlled probe.
2. Geometry/image budgets or malformed embedded bytes cause the failure.
   Test: rerun the unchanged full preflight with only transmission admitted.
3. Browser texture/WebGL loading fails independently. Test: render the exact
   file through the served worker/renderer at narrow and desktop widths.

Harness handoff: restore the reported self-contained GLB preview with authored
transmission materials. Keep current byte/geometry/texture/network limits and
unsupported compressed/skinned-model fallbacks. Prove a failing-then-passing
portable regression, the exact source preflight, focused model tests, types/lint,
scoped Test Genie unit receipt and served narrow/desktop WebGL inspection.
Record findings here, preserve shared edits, and finish after two fresh review
passes find no material issue. No dependencies, new UI, commit, or broad
certification claim is required. If interrupted, retain remaining/unverified work.

Diagnosis: the exact source fails prepareModel with `unsupported` before repair
(/tmp/wc-glb-exact-before.log). Adding only KHR_materials_transmission to the
allowlist lets the same complete preflight pass (after log), rejecting the
budget/malformed-data hypothesis. It has 58,196 triangles and uses transmission
factor 1 on two glass materials. Installed Three.js 0.185.1 implements this
extension; the Khronos specification restricts its factor to 0–1 and excludes
combination with unlit. Repair validates those restrictions and the optional
texture index/coordinate before the unchanged embedded-image checks. Unknown
extensions, skins, sparse accessors, external assets and all budgets stay guarded.

Portable regression: two GLB/glTF material-preservation tests fail before the
repair; /tmp/wc-glb-before.log. The final model/parser/hook/UI set passes 45 tests
in three files, /tmp/wc-glb-tests-final.log. Malformed factors/payloads/references,
external transmission textures and conflicting unlit materials are covered.
Types pass; a test matcher introduced an unsafe-any lint warning, replaced with
an explicit typed URI assertion. Final scoped lint is clean. Initial one-shot
checks used an incorrect relative path for installed renderer source and did
not inspect it; corrected owner-source reads verify transmission allocation
follows the bounded viewport and the existing forced context teardown frees it.

First review: inspected preflight/loader handoff, material defaults, texture
allocation/network admission, viewport target bound, and close/context-loss
ownership. No material finding remains. No dependency or UI layout change.
Served browser proof and the unit-phase receipt are pending.

The initial browser probe successfully rendered the exact GLB but incorrectly
expected the 57 glTF mesh definitions to equal the renderer's mesh count. One
multi-primitive definition yields 58 rendered meshes; the 58,196-triangle total
is unchanged. Corrected the probe's assertion rather than changing product code.

Served proof now passes at 390px and 1440px: /tmp/wc-glb-browser.json and
/tmp/wc-glb-browser.log. The exact source renders its screen texture and glass,
reports 58 rendered meshes / 58,196 triangles, and preserves zoom/fit/orbit,
wireframe, background switching, graphics-context recovery, old-context teardown
on replacement, and the original sculpture fixture. No page errors, external
requests or horizontal overflow. Screenshots /tmp/wc-glb-{390,1440}.png and
/tmp/wc-glb-light-{390,1440}.png were inspected. Native Safari/iPhone remains
unverified; the mobile proof is Chromium with touch/390px emulation.

Desktop pointer activation of Focus was intercepted by the separate existing
approval banner (workspace-top-edge Review). The final probe uses keyboard
focus/Enter, not a forced pointer click; no claim that pointer overlap is fixed.
This adjacent shell issue is outside the material repair and was published via
report-bug as knw-1792378488243184087.
An extra lint invocation from repository root found no configuration and was
not treated as product failure; the UI-owner scoped lint already passed cleanly.

Closing review 1: freshly checked the changed parser and portable tests against
factor defaults/bounds, invalid references/payloads, unlit incompatibility,
unsupported extension and external-texture fallbacks, and unchanged allocation
limits. No material finding remains.
Closing review 2: freshly reviewed actual narrow/desktop visuals and source
lifecycle paths, including transmission texture ownership, forced context cleanup,
selected-scene sanitization and regression portability (no private absolute paths
in maintained tests). No material finding remains within the repair boundary.
Unit-phase verdict and completion journal remain pending; no certification claim.

Final scoped Test Genie unit run 20261019-025001-d1274932 is terminal FAIL,
379 seconds. One durable waiter attached; AM parking token had expired, so the
same waiter completed inline. No polling or second suite. Read the unfiltered
catalog and findings.report artifact_73e93bd498a9efb1a30cecbb0f79d589; retained
/tmp/wc-glb-testgenie-{result,catalog,findings}.json and decoded native diagnostic
strings in /tmp/wc-glb-unit-diagnostics.txt. The receipt shows the previously
recorded approvals keyboard contract, approvals REST-exception check, locale
shape/interpolation failures, 83.79% function coverage versus 85%, and the
existing useImageInspection direct-render policy finding. No model assertion
failure is reported. No threshold/guard changes or duplicate reports for these
known families. Mutable shared inputs limit exact-version attribution; receipt
is not whole-scenario certification. All owned suite/restart/browser operations
are terminal; serving build sha256:2eb5481a98c38a6ffa0591f3edf458c32db911a7d666a4958ca8ce7c111996af
is healthy/readiness true. Implementation boundary remains closed after the two
fresh review passes. Completion work-record: 52fd6ce2-3d5d-45a8-b299-37bffb343248
(/tmp/wc-glb-work-record.json). No required in-scope work remains.


## 2026-10-18 — SQLite preview (bounded W3 task, implemented)

Operator: “Let's do it!” following the read-only SQLite browser proposal. Scope:
existing preview subsystem, additive proto kind, snapshot representation of the
existing blob route, governed browser SQLite dependency, tests/fixtures/docs.
Expected behavior is established; no disputed upstream contract. This is W3
implementation, not a scenario-wide W0–W2 or maturity certification.

Harness handoff: Work until ARCHITECTURE.md “SQLite database inspection” is
implemented and proven by focused Go snapshot/security/routing regressions,
worker/query/lifecycle/UI tests, type/lint/build, scoped Test Genie unit receipt,
and actual served desktop/mobile inspection. Follow docs/mockups/sqlite-preview.svg
and the established artifact family; concept was proposed in chat and operator
authorized implementation. No new primary route or human-ratified L4 claim.
Preserve unrelated shared edits. Fix in-boundary findings; file adjacent defects.
Blocked means a decision, credential, or approval you lack. Name it. Keep going
through friction; when progress stalls, step back — write why, then park, switch
or shrink — instead of stopping or adding more of the same. Finish after two
fresh review passes find nothing material. Checkpoint changed, verified,
remaining and unverified here if interrupted. No SQL editor, mutation, new
permission flow, commit, comprehensive certification, or weakened checks.

Recall found preview architecture/history but no reusable database viewer;
some ranking/recall providers were degraded. Reuse existing modernc SQLite,
ArtifactChrome, worker byte transport, shared dialogs and clipboard. Target
limits and both media compositions are recorded before product changes.

Implementation checkpoint: database kind (enum 20), common extensions and header
sniffing are wired through Go/Connect/UI/registry/folder icons. The existing blob
capability accepts representation=sqlite-snapshot; no new path-based or SQL API.
Source mode=ro plus trusted_schema=OFF/query_only and an explicitly pinned read
transaction precede size admission and incremental backup into private memory.
Only the completed image's WAL header is normalized; tests prove committed WAL
rows, unchanged database/journal bytes, point-in-time isolation and refresh.
Native work is limited to two concurrent snapshots, 32 MiB logical/physical size,
128-page steps, 100ms busy waits and a 5s context checked between steps. Worker
queries are generated SELECTs with quoted/validated names, bound literal filters,
engine-read-only deserialization, disabled executable schema/view/extension
features, reduced SQLite limits and a 2s progress handler plus 5s host cutoff.
No custom SQL, source journal-mode change, OS sqlite command or persistence.

Budgets: 256 ordinary tables, 128 columns per table, 64 indexes, 128 foreign-key
entries, 50 rows/page and 100,000 preview rows, 16,384 characters/cell (<=64 KiB),
1 MiB schema and page payloads. Binary cells never load blob content. Integers
remain strings to preserve int64 values. CSV exports only the current page,
escapes formula-like text and marks shortened values. Each worker and export
object URL has an explicit teardown owner; late replies are ignored by identity.

Initial probes and corrections: blank-parameter binding was rejected by the
actual official engine, so parameterless statements no longer call bind([]).
The native backup retained WAL-mode header bytes; normalize only its completed
private image per SQLite cookbook, retaining source-byte assertions. Browser
review caught unsupported alpha-modified border tokens rendering white; use the
existing semantic border. Desktop harness initially omitted the drawer Focus
toggle and incorrectly awaited a grid in the narrow drawer; final harness uses
the existing focus selector. A fixture-loading URL under jsdom was HTTP rather
than file; the real-engine test now uses the maintained fixture path. A temporary
CSV refactor misplaced join outside row mapping; explicit header/rows construction
and a CSV row-count assertion guard the correction. All these failing probes
were diagnostic, not adopted passing evidence.

Validation so far: 36 database-focused browser-unit tests pass, including real
WASM parsing/querying/read-only enforcement, schema and payload bounds, worker
transport/recovery and lifecycle, and local mobile Back. Earlier preview-family
run passed 371 tests / 53 files; exact final small changes have focused checks.
Scoped types/lint clean. Go snapshot packages and HTTP capability/header tests
pass. SDA installed @sqlite.org/sqlite-wasm 3.53.4-build2 (Apache-2.0 wrapper / public
domain core), no runtime transitive dependencies or install scripts. Governance
validation passes with 10 existing warnings; no new SQLite dependency finding.
Security Health status was read; this is not a new clean vulnerability scan.

Test Genie unit run 20261019-010752-3233fe30 was server-owned and attached once with
runs wait; its terminal receipt is recorded below. It was admitted before the final pinned-read
hardening, fixture naming and schema-layout polish, so it will not be represented
as isolated exact-source certification. No check weakened or unrelated repair.

Served Chromium passes at 390px and focused 1440px: real tables, literal filters,
pagination, JSON cell details, schema/indexes/relationships, current-page CSV,
refresh, int64/text/blob cases and corrupt fallback. Additional 320px checks pass
for unavailable WASM, expired-handle recovery, searchable table chooser, AR/JA
options and constrained input-height containment. Initial "light" harness only
set an unused theme attribute; it is not light-theme proof. A final check will
apply the actual derived light palette and assert the computed background.
Native Safari, Windows runtime, and a real phone software keyboard remain
unverified. Chromium captures and mocked cutoff tests do not prove those paths.



Closing review findings and resolutions: the SQLite deserialize contract frees
FREEONCLOSE buffers on failure as well as success. Ownership is now transferred
before checking its return code, avoiding a second free, with a pointer-specific
failure regression. Untrusted images receive the documented 20 trailing padding
bytes, excluded from logical database length. An initial test counted unrelated
internal allocator frees; the adopted test checks the actual image pointer.
Review also found SQL LIKE's underscore wildcard could hide valid user table
names beginning with "sqlite"; GLOB now excludes only the reserved sqlite_ prefix,
with a real-engine regression. Desktop schema details use the available width
for columns alongside indexes and relationships. All findings were in scope.

Test Genie 20261019-010752-3233fe30 is terminal FAIL (unit, 397s). Published typed
artifact findings report lists four failed assertions: approvals overlay keyboard
contract, approvals REST exception, and approvals/terminal locale key/token
parity. Aggregate function coverage is 83.79% versus 85% (not an unchanged-baseline
claim). A separate provider policy finding points to existing useImageInspection
render imports, already filed as knw-1792296322249006933. Approvals keyboard is
already knw-1792357420384370598; other families are previously recorded. No duplicate
bug report or unrelated repair was made. No SQLite assertion failure appears in
that receipt. It predates the final allocation/prefix hardening; final focused
checks and served inspection cover those changes, not a whole-scenario certificate.
Evidence: /tmp/wc-db-testgenie-{result,findings,catalog-all}.json and
/tmp/wc-db-global-diagnostics.txt. Canonical opaque findings artifact id:
artifact_beda02ffd293a8b03614f9faa0cac857.

Final browser checks verify 390px record flow and focused 1440px grid, plus320px
unavailable/expired/AR/JA and a derived light content palette. Light harness now
leaves the shell's intentionally dark portaled dialogs pinned to their own theme;
its first attempt incorrectly overrode dialog tokens and is not adopted evidence.
Both content and options dialog pass WCAG2A/AA/2.1AA axe checks in dark and derived
light content contexts (zero violations). No claim of an app-wide light-theme
switch is made. Captures: /tmp/wc-db-{390,1440}.png, schema/cell/edge variants,
/tmp/wc-db-extra-{ar,ja,light}.png and /tmp/wc-db-keyboard.png. Browser result records:
/tmp/wc-db-browser.json and /tmp/wc-db-extra-browser.json; /tmp/wc-db-axe-*.json.


The final table-picker refinement confines search to the searchable dialog;
selecting a result clears that search and the main dropdown always retains all
tables. It cannot silently leave the primary table selector filtered. Focused UI
checks pass after that change. The final preview family passed 373 tests /53 files
(/tmp/wc-db-preview-final.log), and 36 database checks pass
(/tmp/wc-db-focused-final.log). Final types and scoped lint are clean; selectors,
typed strings and 62 database EN/AR/JA keys/interpolation tokens are in sync.
Latest affected Go packages and two HTTP snapshot tests pass
(/tmp/wc-db-go-final.log, /tmp/wc-db-http-final.log). Proto owner verification
passes, artifact 5d445b6bb3471e7e9a22ad6d4a4332449a8d3507efe5a74ee7c50411640da1ad,
source sha256:3379b11f727c0f28c1e2eef47fc8707cf5287995a742fe50672fc5b6dd777527.

An additional portability probe caught Go URL formatting a Windows drive as an
URI authority (file://C:/...). The snapshot URI builder now emits file:///C:/...
and preserves encoded punctuation; Go regressions cover drive, Unix and network
paths. These are URI portability proofs, not native Windows runtime proof.

Fresh closing review 1 (after allocation/prefix/chooser/URI fixes): inspected query
construction, read-only admission, allocation/cursor/connection/worker cleanup,
budgets, stale/cancel behavior, int64/NULL/blob representation, schema bounds,
export escaping, responsive navigation and recovery. No material finding.
No tests disabled, coverage floors changed, unrelated source repaired, or commit
created. Shared working-tree edits are preserved. Final served adoption is recorded below.


Fresh closing review 2: reviewed the adopted desktop/mobile captures against the
mockup and existing artifact family, exercised table-picker search without
leaking its filter, schema/cell/record navigation, download/CSV/recovery, payload
bounds, source snapshot consistency and teardown. No material finding. The two
fresh reviews are self-verification, not independent human acceptance or L4.

Final managed restart is healthy with readiness=true, serving build
sha256:b7cc535d7182efeb21e74a507ae2625f725aad9f1dcca1e0b90a834d8efc03f9. Adoption logs:
/tmp/wc-db-adopted-native-restart.log and /tmp/wc-db-health-adopted.json.
Final actual-browser runs /tmp/wc-db-browser-adopted.log (390/1440) and
/tmp/wc-db-extra-adopted.log (five320px variants) both pass; 7 result records,
no page errors, no horizontal overflow, and four axe contexts with zero
violations. The constrained-height check is simulated, not a real keyboard.
No product work remains in this bounded task. Native Safari/Windows runtime
and whole-scenario certification remain unverified/outside this evidence.


A final large-binary stress regression found length(CAST(blob AS BLOB)) defeats
SQLite's length-only column optimization, causing SQLITE_TOOBIG for a 2 MiB
payload despite rendering only its size. The projection now takes length(blob)
directly for binary types and never materializes their content. The real-engine
2 MiB regression failed before and passes after the correction. Prior journal
checkpoint 7faa5c42-c647-4f18-a46e-cfe1d585b09d predates this additional fix;
final completion evidence below supersedes its build/test counts. The prior
closing passes are not adopted as the final reviews after this finding.


Post-stress closing review 1: checked direct length-only BLOB access versus text
conversion, default stable ordering, row/page/schema budgets, source read-only
transactions, worker/allocation teardown, filter/search/sort state and all
responsive navigation/recovery paths. No material finding after the BLOB fix.
The 37 database-focused tests now pass, including the failed-before 2 MiB case;
final types and scoped lint remain clean. Final adoption and second post-stress
review follow the managed restart; no more product changes are planned.


Final post-stress adoption: managed restart healthy/readiness=true, build
sha256:7af7b0a93551b991dcc8445a345f4933f34308ca21ae5972b9c7d93c6f306990. /tmp/wc-db-blob-restart.log and
/tmp/wc-db-health-release.json record this identity. Both actual-browser runs
/tmp/wc-db-browser-release.log and /tmp/wc-db-extra-release.log pass: 390px,
focused1440px, and six320px variants, now including a real served 2 MiB BLOB.
Eight result records, no page errors/overflow, and four zero-violation axe
contexts. The searchable chooser retains all four tables after choosing a
filtered result. Final source has 37 passing focused database tests, clean
types/scoped lint, and unchanged passing Go/proto/selector/string checks.
The 373-test preview family receipt precedes the final BLOB projection; its
failed-before/passing-after real-engine regression and served test cover that
last change. The Test Genie failure and all platform limits above still apply.

Post-stress fresh review 2: checked the final rendered grid/records/schema/cell
flows against the mockup and existing family, query/filter and download behavior,
large-binary isolation, current build identity, lifecycle cleanup, and recovery.
No material finding. Together with post-stress review1 this closes the bounded
task; these are self-reviews, not independent human or whole-scenario acceptance.
No product work remains. All owned test runs and managed restarts are terminal.


Completion work record: 9ca5d3a0-e27a-4687-8a94-45f79626ee10.


## 2026-10-18 — Font specimens (bounded W3 task, implemented)

Operator: “Yes, let's do fonts!” Expected behavior and proposed examples are
established in the conversation. Currently TTF/OTF/WOFF/WOFF2 have no dedicated
classification or renderer. Scope is the existing preview subsystem, additive
proto kind, generated outputs, governed fontkit dependency, tests and docs.
No disputed upstream contract; this is a scoped W3 implementation addition.
Concept and responsive composition extend the approved artifact family and
prior operator direction to avoid content containers. Target: ARCHITECTURE.md
“Font specimens” and docs/mockups/font-preview.svg. No scenario-wide maturity
or human-ratified L4 claim.

Harness handoff: Work until that font specimen target is implemented and proven
by focused parser/lifecycle/UI and Go routing regressions, types/lint/build,
scoped Test Genie unit phase and served desktop/mobile inspection including
real fonts, corrupt/oversize fallbacks and cleanup. Keep going through friction;
when progress stalls, step back — write why, then park, switch or shrink — instead
of stopping or adding more of the same. Blocked means a decision, credential,
or approval you lack. Name it. Finish after two fresh review passes find no
material issue. Checkpoint changed, verified, remaining and unverified here if
interrupted. No unrelated repairs, disabled checks, commit or certification.

Recall supplied existing artifact architecture and preview history; ranking
providers were partly degraded. No reusable font preview surfaced. Fontkit
2.0.4 (MIT) was selected for WOFF2 metadata/actual coverage over opentype.js,
and recorded/installed through SDA with lifecycle scripts disabled.


Implementation checkpoint: kind `font` (enum 19) is wired through Go, Connect,
TypeScript, the renderer registry and folder glyphs. Native FontFace is scoped
to a unique family and removed on teardown; neither worker validation nor
rendering installs a font. The worker parses actual mappings without expanding
cmap ranges, bounds name-string duplication and axis/cmap counts, and admits
only native-sanitized bytes to fontkit. Review found its Brotli decoder grows
output beyond the expected length, so header bounds alone were insufficient;
worker/native validation now precedes all library decompression and has ordering
and rejection regressions. Variable axes use authored ranges; no synthetic
bold or italic. Character pages are bounded to 96 items; invisible formatting
controls are omitted from the grid. Sample length is bounded without splitting
an inserted astral character.

Owner generation and verifier pass. Initial generation overlapped the additive
source edit and verifier honestly found drift; final scoped regeneration fixed
it. UI's installed proto copy was stale, so governed lifecycle setup refreshed
it, with no consumer shim. Proto artifact
5cc548e965b75fbbc4060ac15916579cebdcdcda14b0bc97b072e9920d732cb4,
source sha256:5a4c053afe9dc72c842d2580363e7a2f2c33c7254be8fd9a5c0b82cec7ba76fe.
Focused preview/API regressions: 338 PASS / 51 files; changed Go packages PASS;
types and scoped ESLint pass. Further UI finish and two regression additions
are undergoing their final scoped check. All four real formats plus an original
variable font loaded at 390px and 1440px in Chromium, including actual mapped
character search/insertion, options, corrupt fallback and zero retained preview
faces after replacement/failure. Evidence /tmp/wc-font-browser.json and captures
/tmp/wc-font-{390,1440}.png. Phone keyboard-height capture uses constrained viewport
simulation, not a native phone keyboard. Native Safari remains unverified.

Visual review replaced unsupported currentColor opacity classes with the shared
border token, removed glyph row dividers, strengthened Unicode labels and named
the character count. First UI test failures were harness cimode labels and modal
background inaccessibility, corrected without changing product behavior. Python
WOFF2 fixture encoding was unavailable because Brotli is absent; no dependency
workaround was installed. Existing real WOFF2 is used instead, alongside original
TTF/OTF/WOFF and variable TTF fixtures. No font license was copied.

Scoped Test Genie unit run 20261018-205532-b082349c has one durable waiter;
terminal verdict pending. The last UI finish overlaps that mutable-worktree run;
its assertions will be retained without claiming an isolated exact-source input.

Final review found mapped nonbreaking spaces and variation selectors could
still appear as blank character tiles. A desired-behavior regression failed
before the fix (/tmp/wc-font-invisible-before.log); Unicode whitespace/control/
default-ignorable filtering now omits those while preserving private-use icon
mappings. This is the last product change. New sample insertion also preserves
supplementary-plane characters at its 1,000 UTF-16-unit limit. Final focused
font group is 30 passing tests; the full preview family is being rechecked.

Fresh review pass 1 after these fixes: reviewed storage and decompression
admission order, metadata bounds, native font ownership, worker/native deadlines,
source renewal and late-result suppression, render-only font scoping, Unicode
search/insertion, axis/style fidelity and download recovery. No material finding
remains on the active preview path.

Served browser proof before the last character filter: all formats and variable
TTF pass at 390/1440px; 320px long-text/AR/JA options, no FontFace API fallback,
and an actual expired blob followed by Reopen recovery pass. Axe WCAG A/AA/2.1
AA checks have zero findings for both default and explicit light specimen stages
(/tmp/wc-font-axe.json, /tmp/wc-font-axe-light.json). Color changes are scoped to
the specimen; app chrome keeps shared tokens. A final managed restart will
adopt the last character filter and repeat served checks.

The first unit run completed FAIL in 381 seconds. Retained diagnostics:
/tmp/wc-font-testgenie-result.json, /tmp/wc-font-testgenie-catalog.json,
/tmp/wc-font-testgenie-report.json, /tmp/wc-font-unit-diagnostics.txt. It reports
approvals REST policy, approvals/terminal AR/JA parity, the approvals drawer's
keyboard opt-in, function coverage 83.93% versus 85%, and the font astral-limit
assertion during the overlapping edit. The current font assertion passes in a
coverage-instrumented run (six tests), as well as the stable focused family. The
filtered coverage command deliberately retains global floors and exits 1 because
unselected app paths are unexercised; only its six assertions are claimed passing
(/tmp/wc-font-coverage-regression.log). Mixed-source cache timing is an inference,
not a proven root cause of the first receipt's font assertion. No floor was lowered.

The approvals keyboard omission reproduces independently (1 fail / 58 pass)
in /tmp/wc-font-unrelated-overlay.log, and was filed through report-bug as
knw-1792357420384370598. Recall for duplicate context had degraded providers; no
matching report was retrieved. No unrelated approvals source was edited. Existing
REST/locale/coverage failure families were already present in prior preview receipts.

Closing script/font review also identified that an icon font or a font lacking
the UI's script could show mostly fallback letters in its default specimen.
Defaults now prefer an actually covered language sample, then mapped characters;
the hero likewise uses authored mappings when Aa is unavailable. User-entered
text remains exact, with ordinary browser fallback disclosed in options. Two
regressions protect this choice and private-use glyphs. Run
20261018-210302-7c4fd5ef was explicitly aborted before this final change, so its
non-completion is not a failed test verdict. A new unit-only capture is being
admitted against the now-frozen font product source.

Final specimen check: the actual logo WOFF2 contains an 8,792-byte license
name record, which correctly exposed an overly strict 4 KiB per-record admission
limit. The per-record bound is now 32 KiB; aggregate decoded names remain 64 KiB,
512 records and 160-character returned labels. This preserves amplification
protection while accepting ordinary OFL-length licenses. A real-file regression
now protects it. Earlier browser all-format loops lacked an explicit filename
transition wait; the logo check exposed that harness race. The corrected harness
waits for the new filename before sample readiness. Early screenshots and
individual passes remain observations; final adopted-file records are authoritative.

Fresh unit capture 20261018-210533-20504782 has one durable waiter. The font
specimen defaults are stable; the final license-bound refinement occurred after
admission. Assertions will be reported with this mutable-input limitation, not
an isolated-source certification. The intermediate capture
20261018-210302-7c4fd5ef remains explicitly aborted.

Latest unit receipt 20261018-210533-20504782 is terminal FAIL (388 seconds),
with no font assertion failure. It reports the existing approvals REST exception,
AR/JA locale shape/interpolation and aggregate UI function coverage 83.94% versus
85%. The approvals keyboard contract is absent from this newer failure set;
shared approvals work can change independently. Full catalog and native bounded
diagnostics: /tmp/wc-font-testgenie-stable-{result,catalog,report}.json and
/tmp/wc-font-unit-stable-diagnostics.txt. No policy or threshold was weakened.
The final exact/short Unicode-code search refinement has its focused regression;
this receipt is retained with its actual mutable-source applicability.

Full focused preview/API family before the final license regression: 342 PASS
/ 51 files (/tmp/wc-font-final-regression.log). The final focused font group,
including the real long-license font and short U+41 lookup, is being checked
in /tmp/wc-font-release-tests.log. Unicode search prefers an exact mapped
code point (including omitted leading zeroes), then supports prefix browsing.

Fresh review 1 after the final bounds/search refinements: checked compressed
size versus actual native admission, total and per-record metadata allocation,
Unicode/private-use correctness, coverage-aware defaults, renderer and native
face ownership, timed-out/late work, source reset, variable axis fidelity and
authenticated same-origin transport. Existing focused/native evidence covers
the active paths; no material remaining code finding. No new feature was added
after this review. Awaiting final served visual/interaction check.

Final served evidence: managed restart completed healthy on API 16382 / UI
21233. Readiness true, build identity
sha256:6da0bb6109826182fc3b550e3f13a5f64532efbbb894c841dd75e2c16aa2adf4
(/tmp/wc-font-health-release.json). Final adopted-file browser harness passes
all four formats, variable TTF and the actual logo font at 390/1440px, with one
active private face, character search/insertion, size/axes, corrupt fallback,
download and zero retained preview faces on replacement. Fourteen final records:
/tmp/wc-font-browser.json and /tmp/wc-font-browser-release.log. Actual icon defaults
are mapped letters (hero Vi), not a system fallback pangram. The earlier native
probe that navigated during restart failed as a harness timing error, not a pass;
final proof waited for the lifecycle owner to return healthy.

Final extra browser checks: 320px long input, no font-loading API, expired blob
and real Reopen recovery, Arabic/Japanese layout, and actual CSS application of
letter spacing, line height, centering, kerning and ligatures PASS
(/tmp/wc-font-extra-release.log, /tmp/wc-font-extra-browser.json). Default/light
Axe WCAG A/AA/2.1 AA scans have zero violations. Desktop/mobile and light/dark
composition captures visually inspected against docs/mockups/font-preview.svg;
no extra content card, no overflow or open visual finding. Final font regressions:
33 PASS / 4 files (/tmp/wc-font-release-tests.log); TypeScript and scoped ESLint
PASS (/tmp/wc-font-release-types.log, /tmp/wc-font-release-lint.log); Go changed
packages PASS; EN/AR/JA font key/token parity and diff whitespace checks PASS.

Fresh review 2: re-read the final controls, fallback and metadata flows, inspected
the actual final text/icon specimens and responsive settings captures, compared
them with the mockup and operator feedback, and assessed all retained failures.
No material font behavior or visual issue remains. The scenario-wide unit failure
is retained separately; there is no whole-scenario certification claim. Native
Safari, actual OS font engines and a real mobile software keyboard remain
unverified (phone layout/keyboard-height proof uses Chromium and a constrained
viewport). Outcome: implemented and serving, original downloads preserved, no
unrelated fixes or commit. Refresh an already-open console tab to adopt enum 19
and the newly built client.

Completed work record: `34d282e6-e4ee-4e21-83d2-714bd724a97a`.

## 2026-10-18 — 3D model viewer (bounded W3 task, implemented)

Operator: “3d sounds interesting! Let's go for it!” Prior feedback: avoid
unnecessary content containers; match the polished artifact controls. Existing
GLB resolves to unsupported, while OBJ/STL/glTF can fall through to raw text.
No disputed upstream contract is implicated; scoped implementation addition.
Recall found the existing artifact architecture and Three.js ecosystem usage;
no reusable preview implementation. Source-ledger recall timed out, other
providers supplied the local architecture.

Harness handoff: Work until the formats and UX specified in ARCHITECTURE.md
“3D model inspection” and docs/mockups/model-preview.svg are implemented.
Proof: focused parser/loader/UI and Go classification/transport regressions,
TypeScript/lint/build, scoped Test Genie unit phase, and actual served WebGL
inspection at narrow/wide drawer widths, including fallback and teardown.
Boundary: web-console preview subsystem, its proto kind and generated outputs,
governed Three.js dependencies, samples and documentation. Keep going through
friction; when progress stalls, step back — write why, then park, switch or
shrink — instead of stopping or adding more of the same. Blocked means a
decision, credential, or approval you lack. Name it. Record findings here,
resolve user feedback, and finish with two fresh review passes finding no
material issue. No certification claim or unrelated console repair. If
interrupted, checkpoint changed, verified, remaining and unverified here.

Implementation and validation checkpoint: kind `model3d` now travels through
Go classification, Connect enum 18, the generated TypeScript map and renderer
registry. Three.js 0.185.1 and @types/three 0.185.4 were installed through SDA.
The parser runs in a terminable local worker; embedded resources and allocation
limits are checked before Three.js sees bytes. Shared raster-header inspection
was factored from publication parsing without changing its existing defaults.
Four original procedural fixtures live at fixtures/file-preview/orbital-study.*.
The canvas has no extra content card. Secondary tools share InspectionOptions.

Evidence so far: 305 focused preview/API tests in 47 files PASS; 20 parser
regressions PASS independently; affected Go filepreview and Connect packages
PASS; TypeScript, scoped ESLint and diff whitespace checks PASS. Proto generation
used the governed owner; verify-committed-gen passed. An initially stale UI proto
copy was reconciled through lifecycle setup rather than a consumer shim.
Build/restart is healthy on API 16382 / UI 21233, identity
sha256:412e78c2f4a310f9e594011c5c634e4dde3ba3cc0b4c036b5c3190dab58a6678.
EN/AR/JA 3D key/token parity passes.

Browser proof: /tmp/wc-3d-browser.json and /tmp/wc-3d-browser-verified.log.
Chromium at 390px and 1440px rendered all four real formats plus the existing
room_blockout.glb; embedded PNG texture painting, external texture rejection
with zero remote requests, corrupt-file download fallback, context-loss retry
and graphics-context disposal on navigation all passed without page errors.
The first desktop harness assumed viewport width equaled drawer width; it was
corrected to exercise the actual Focus action, not by changing product code.
A first dialog-close selector likewise expected “Close” instead of the existing
“Close options.” Those failed harness attempts are not product passes. Native
Safari remains unverified. Captures: /tmp/wc-3d-{390,1440}.png and matching
options/light/textured captures. Original sample normals were corrected before
the final build; authored normals are preserved for OBJ and glTF.

Scoped Test Genie unit run 20261018-183908-5f0a60c5 is attached once via its
durable waiter; terminal receipt pending. Baseline capture bypasses the known
mid-edit phase-cache issue reported in the previous PowerPoint repair. This
remains a unit-only run despite the CLI banner saying comprehensive.

Hardening review found two allocation-amplification paths: cloned publisher
extras and repeated texture uploads using one image. Regressions failed against
the initial parser (/tmp/wc-3d-metadata-before.log and
/tmp/wc-3d-texture-before.log). Worker JSON decoding now strips unused extras
and animations; GLB JSON is rebuilt while preserving its binary chunk, and
only the selected scene reaches the renderer. Image and per-texture pixel
budgets both apply; labels and sampler count are bounded. Index values are
checked against position counts before GPU submission. Focused regressions
now pass 308 tests / 47 files; TypeScript and scoped ESLint remain clean.

The first scoped unit run completed FAIL (388 seconds): console REST-exception
policy (api/approvals.ts), AR/JA approvals + terminal-hint locale parity, and
global UI function coverage 83.95% versus 85%. The earlier PowerPoint receipt
already contained these failure families (84.06% then); coverage includes this
change and is not claimed to be an unchanged baseline. Existing issues were
reported previously; no duplicate bug was filed. Catalog and detailed evidence:
/tmp/wc-3d-testgenie-artifacts.json, /tmp/wc-3d-testgenie-report.json, and
/tmp/wc-3d-unit-diagnostics.txt. The run overlapped the final parser hardening;
a fresh unit receipt is being captured against the final source.

Additional browser proof before hardening: actual CDP two-touch gestures
triggered rendering; simulated WebGL2 absence produced the correct download
fallback; a settled light-background capture is visually inspected. Evidence:
/tmp/wc-3d-extra-browser.json, /tmp/wc-3d-light-settled-1440.png. The final
build will repeat the served checks after the sanitizer/texture-budget change.

Closing review checkpoint: a preset chosen during loading initially could be
lost when the stage initialized. ModelReader now reapplies the selected view
on stage readiness; the focused regression passes. Final focused suite:
309 tests / 47 files PASS (/tmp/wc-3d-regression-delivery.log). Latest
TypeScript/scoped ESLint PASS (/tmp/wc-3d-types-delivery.log,
/tmp/wc-3d-lint-release.log, /tmp/wc-3d-last-lint.log). Build PASS
(/tmp/wc-3d-build-delivery.log). Source is stable for final served verification.

Fresh review 1 after hardening: parser bounds, URL authority, GLB reconstruction,
image/texture allocation, scene graph and index integrity, worker deadlines,
and abort/close/renewal ownership reviewed against actual tests. No material
findings remain on the active file-preview path. Remaining broad console
checks are tracked separately; none were weakened or hidden.

Fresh unit receipt: 20261018-184727-8136890e, one durable waiter attached;
its result is pending. The last view-readiness fix has focused proof and the
run's precise source applicability will be retained with its terminal evidence.

Final served verification: make restart completed healthy at 14:51:45 EDT;
API readiness true, build identity
sha256:abfdb9491887acc13ba3fd01fe3a99c32699167a013a658f804ecc7bcdf03f56
(/tmp/wc-3d-health-delivery.json). Repeated real-model/texture/fallback/context
checks pass on the final served build at 390px and 1440px with no page errors
or remote asset requests (/tmp/wc-3d-browser-delivery.log and
/tmp/wc-3d-browser.json). Touch, WebGL absence and light-background checks pass
again; a delayed blob response confirms a Top preset chosen during loading is
applied to the actual stage. Capture /tmp/wc-3d-loading-top.png is visually
verified against the orientation and controls. The harness was corrected to
click the visible radio label, matching the actual user action, after the
screen-reader-only input was correctly covered by its label. This is not a
product change or a claimed passing result for the failed harness attempt.

Fresh review 2: final desktop and phone captures match the immersive stage and
compact shared chrome in docs/mockups/model-preview.svg; no extra canvas card.
Checked options, light stage, keyboard/touch access, preset startup, content
framing, download preservation, translation keys, cleanup and source resets.
No material findings remain in the assigned boundary. Two consecutive fresh
closing passes are clear after the hardening fixes. Native Safari and unusual
GPU drivers remain unverified; supported browser capability is checked at runtime.

Terminal receipt: fresh unit run 20261018-184727-8136890e completed FAIL
in 394 seconds. Retained failures: api/approvals.ts REST-exception policy,
AR/JA approvals + terminal-hint locale parity, and global UI function coverage
83.98% versus 85%. No 3D test failure is reported; final focused regressions
remain 309/309. The view-readiness fix overlapped this shared-worktree run; its
focused and final served proofs establish its behavior without an exact-input
certification claim. Evidence: /tmp/wc-3d-testgenie-release-result.json,
/tmp/wc-3d-testgenie-release-artifacts.json,
/tmp/wc-3d-testgenie-release-report.json,
/tmp/wc-3d-unit-release-diagnostics.txt. No further unit rerun is justified by
these unchanged failure families. No checks or thresholds were weakened.

Outcome: operator-requested 3D viewer implemented and serving, with four original
format examples. Bounds, graceful failures, cleanup, responsive interactions
and delivered visuals have focused and browser evidence. The broader console
unit suite remains red, so this is not whole-scenario certification. No commit
was requested or created. Shared unrelated work was preserved.

Completed work record: 48b9a6f8-db0f-4fa3-98f1-dcf3ee61ffb5.

## 2026-10-18 — PowerPoint sample stalls during loading (W3, repaired)

Operator report: “Not sure if it's on my end or what, but the powerpoint example
isn't loading.” IMG_1730.png shows “Preparing slides…” with no slide metadata.
Bounded repair: identify and reproduce the startup failure, fix its owning
boundary while preserving the opaque-origin sandbox and package limits, add a
regression that fails before the repair, and verify the served sample on phone
and desktop plus the scoped unit receipt. Earlier Chromium-only evidence did
not qualify authenticated remote access or native Safari.

Recall found prior tunnel-safe HTML preview work; its existing srcDoc approach
avoids an additional anonymous document request across Access. Hypothesis A:
the presentation iframe's anonymous module request loses the parent's access
cookie, so an authenticated proxy blocks it. Test: keep a valid parent cookie
and require it only for the renderer asset; inspect the actual request and
compare unprotected/local loading. Falsified if the frame sends the cookie or
the protected asset still loads and paints. Hypothesis B: a broken fixture,
parser or browser startup feature prevents metadata. Test: render the exact
sample locally with frame errors and actual slide paint checks; independent
worker/protocol regressions cover package rejection and handshake ownership.
Local rendering already supports the fixture, but does not establish Safari
compatibility. A slow connection remains a separate hypothesis until startup
phase/network evidence distinguishes it.

Reproduction confirmed hypothesis A under a cookie-protected renderer asset:
the original frame request had Origin:null, no access cookie, and never painted
a slide; its deadline incorrectly reported a size limit. The unchanged fixture
rendered locally. Evidence: /tmp/wc-pptx-loading-repro.json and the before-fix
regression /tmp/wc-pptx-loading-before.log (external script assertion failed).
The operator's exact native Safari connection remains unmeasured; this is a
reproduced application loading defect, not a claim to have inspected that device.

The renderer is now an owned, self-contained asset built through Vite's public
build/plugin APIs in development and production. A bounded host fetch carries
same-origin credentials; a fresh-nonce srcDoc executes the trusted source only
inside the existing opaque frame. No anonymous script/module requests leave the
frame. Startup, HTTP/download and runtime failures stay in the drawer with retry
and original download, without reporting a package size limit. The renderer
fetch has a 4 MiB actual byte bound and shares the 25-second startup/processing
deadline; close, retry, renewal and terminal failure abort it and ignore late
results. Existing worker package bounds and message-source checks remain intact.

A proposed dynamic-module transport was rejected after the served failure-path
check triggered the application's stale-chunk page reload. Fetching inert source
as data avoids that global reload mechanism and permits a real retry. A standalone
library build also initially retained process.env.NODE_ENV; the owned build now
defines it explicitly. Neither failed experiment is counted as passing evidence.
A build that overlapped the localized fallback-copy edit retained the earlier
copy, so the final release build is recreated from stable source before capture.

Final focused preview/API cohort: **275 PASS / 44 files**. TypeScript and scoped
ESLint pass. EN/AR/JA presentation keys and interpolation tokens match. Tests
cover the nonce policy, script-end-tag escaping, bounded authenticated fetch,
failed downloads, aborts, late results, timeout classification and frame ownership.
Checks: /tmp/wc-pptx-loading-{regressions-final,typecheck-final,lint-verified}.log.

Validation accounting: run 20261018-170901-a9a47053 overlapped loader/test edits
and contains two own presentationFrame test failures as well as the docs orphan,
approvals/locale/policy findings and function coverage threshold. Run
20261018-171549-7b979399 then reused that failed result in zero seconds, despite
changed renderer/test source. Neither receipt proves the final source passes.
The cache observation is published through report-bug as
knw-1792343949703663260 (possible mid-run identity capture, explicitly speculative).
A fresh unit run uses the supported --capture-profile baseline cache bypass:
20261018-171732-1de16450. Its fresh terminal receipt is recorded below; the
earlier failures are not recast as baseline-only.

Stable release restart completed healthy (API 16382 / UI 21233); API readiness
is true with build identity sha256:2a438f932ee46cf9381024dc3df7c99af14c1a5998f6ef3ff49a78c47d1d52df.
The final served proof passes all five cases: normal 390px loading, authenticated
390px and 1440px loading, one denied renderer fetch followed by Reopen, and one
injected runtime exception followed by Reopen. Both failures stay in the drawer
and recover with exactly one main-page navigation. Authenticated runtime fetches
carry the test access cookie. Ready cases verify slide title, chart, notes,
thumbnail navigation, EPUB regression, no external frame scripts and inaccessible
parent document. No unexpected page errors; the injected exception is expected.
Fallback appears in 141/156ms in the local harness; these are observations, not
network performance guarantees. Evidence: /tmp/wc-pptx-loading-proof.json,
/tmp/wc-pptx-loading-proof-release.log and /tmp/wc-pptx-loading-health-release.json.
The earlier harness clicked Reopen during the drawer entrance animation; ordinary
Playwright actionability waiting resolves that capture issue without product edits.
Native iPhone Safari and the operator's actual tunnel remain unmeasured.

Closing review 1 checked the owned asset URL, actual byte bound, nonce-only CSP,
HTML script-end escaping, opaque-origin isolation, trusted versus publisher code
and active-frame message ownership. Closing review 2 checked retry/renewal/close,
worker and download cancellation, late completion guards, fallback copy and
source download, processing versus startup deadlines, and the global reload
boundary. No further material issue was found in this repair boundary. The final
phone capture after a simulated runtime failure and successful retry displays
the original slide with the existing polished controls and no extra container:
/tmp/wc-pptx-loading-runtime-failure.png.

Final fresh unit receipt **20261018-171732-1de16450 FAIL** after 369 seconds.
The before-fix/overlapping presentationFrame failures are absent. Retained
failures are TestDocsNoOrphans, approvals keyboard placement and REST policy,
locale key/token parity (approvals plus two terminal hints), and global UI
function coverage 84.06% against 85%. The existing image-inspection test helper
policy finding remains. These failures are recorded separately from the passing
275 focused tests and five served cases; the whole scenario is not certified
green. Existing QA records cover the recurring docs/approvals/locale/policy
findings; no duplicate report or unrelated repair is added.
Receipt/catalog: /tmp/wc-pptx-loading-testgenie-fresh-{result,artifacts}.json.
Retained findings: /tmp/wc-pptx-loading-unit-fresh-report.json.
Completed work record: 7023278c-19b8-4079-9430-43346670df17.

## 2026-10-18 — PowerPoint and EPUB previews (W3, implementation complete)

Operator instruction: “Ooo let's do PowerPoint!” following the recommendation
for slide thumbnails/navigation, fit/zoom and speaker notes. Existing evidence:
the kind registry has no presentation kind, so PPTX opens UnsupportedPreview.
Work shape: bounded convergence task. Work until the PowerPoint inspection
contract in ARCHITECTURE.md is implemented, the paired mockup is reflected in
served desktop/phone captures, focused parser/protocol/renderer regressions pass,
and the scoped Test Genie unit phase has a terminal receipt. This extends the
operator-approved artifact inspection design; no separate plan or Swarm goal.
Boundary: preview transport/classification, UI reader, locale/selector owners,
governed dependencies and sample fixtures. Preserve existing preview families.
Record findings and resolve them; after green run two fresh reviews covering
correctness, malicious packages, cleanup, maintainability and visual parity.
Keep going through friction; when progress stalls, step back and record why
before changing approach. Checkpoint here if interrupted. Report native-platform
or rendering qualification gaps precisely; do not claim full PowerPoint parity.

Additional operator instruction: “And when you're done, please do epub too!”
Extend this bounded task sequentially with the EPUB reading contract in
ARCHITECTURE.md, paired EPUB composition, real sample book, parser/reader
regressions and served phone/desktop proof. EPUB currently has no dedicated kind
or renderer. Share package safety code without sharing the distinct reading UX.

Implemented both kinds through the typed resolve/listing adapters and renderer
registry. The pinned browser PowerPoint renderer runs inside an opaque-origin
CSP sandbox; controls, notes and text stay in the host's established inspection
chrome. Thumbnails mount on demand, desktop can retain the rail, phones select
from a grid, and keyboard/fit/zoom controls keep navigation local. Legacy PPT,
encrypted, invalid and over-budget decks retain an original download.

EPUB follows package spine order, EPUB 3 nested nav or EPUB 2 NCX, including
nonlinear notes. The semantic reader has desktop/phone contents compositions,
fragment-aware within-book navigation, chapter search, text/measure settings,
estimated progress and owned-image cleanup. Publisher scripts/styles, active
media and remote content are omitted. Common EPUB 2 XHTML declarations are
discarded without fetching a DTD; named text entities decode safely while
literal CDATA remains intact. Fixed-layout fidelity is disclosed.

Both readers share a terminable worker's physical ZIP preflight and streamed
actual-expansion accounting, with CRC, duplicate/path, count, XML depth/tag and
raster-dimension limits. No Office/LibreOffice, host converter, extraction or
third-party upload is used. Dependencies were installed and pinned through SDA;
governance validation passes with the existing advisory findings. Fixtures:
`fixtures/file-preview/operations-review.pptx` and
`fixtures/file-preview/thoughtful-software.epub` contain original sample content.

Focused proof: **266 PASS / 43 preview/API test files**; TypeScript and scoped
ESLint pass. Focused Go kind/adapter suites pass, including both new blob kinds
and listing projections. Proto generation and freshness verification pass;
published artifact bd0c11e68874d78f21a87f8fb94e31be64c11035bc94ccef37bc43243017657b.
Checks: /tmp/wc-publication-{regressions-final,typecheck-final,lint-final,go-tests}.log.

Served proof before the final renewal/cleanup pass: 390px/1440px English and
320px Arabic/Japanese cases pass, with 24 scoped WCAG A/AA audits, zero page
errors and no preview overflow. Checks include slide thumbnails, chart/table
content, notes copying, keyboard Home, zoom, options, chapter navigation/search,
within-book fragments, direct and archive-member previews, corrupt/legacy
recovery, inaccessible parent document and zero external requests from hostile
PPTX/EPUB fixtures. An additional real chart/SmartArt deck renders without a
page exception. This is Chromium evidence, not native Safari/Office parity.
Evidence: /tmp/wc-publication-proof.json, /tmp/wc-publication-proof.log and
/tmp/wc-publication-{en,ar,ja}-*-{slide,slides,book,options}.png.

Review findings resolved: renderer capture initially preceded thumbnail paint;
waiting for committed content and paint produces the expected thumbnail grid.
Added raster area limits, literal CDATA preservation, sanitized HTML bounds and
cleanup, a fresh byte-URL/frame handshake, index bounds and quiet error chrome.
The final focused suite includes expected behavior for bombs, CRC corruption,
path/duplicate/encryption rejection, EPUB 2/3 ordering, internal links, active
content removal, owned-image revocation and stale/foreign frame messages.

Initial Test Genie unit run **20261018-162024-1dfabb71 FAIL** (360 seconds).
It overlapped parser edits and reported three publication test failures plus the
previous docs orphan and approvals keyboard/REST/locale findings. The final
focused suite now passes those publication tests. Do not attribute that first
receipt solely to unrelated baseline issues. The stable final receipt and
served build review are recorded below.

Final scoped unit receipt **20261018-163525-57b079ab FAIL** (370 seconds).
Its retained failures are the perf-doc orphan, approvals keyboard/REST/locale
checks and global UI function coverage 84.07% against 85%. No publication test
failure remains in that receipt. It is not formal baseline attribution and the
whole scenario is not certified green. A final review then corrected one EPUB
compatibility case: remote artwork/styles in the manifest are omitted while the
local spine remains readable; remote spine chapters still fail safely. The full
**266-test / 43-file** preview/API regression suite, TypeScript and scoped ESLint
pass after that change. The affected locale sections have matching keys and
interpolation tokens in English/Arabic/Japanese. The final managed restart is
healthy; final sample checksums/XML well-formedness pass. Canonical main content
type, theme style lists and reciprocal speaker-notes relation are included in
the sample PPTX. No native Office claim is made.

Closing review 1 checked byte/ZIP/XML/image/HTML limits, CRC and path ambiguity,
declared versus actual expansion, external resource isolation, untrusted metadata
and old-frame results. Closing review 2 checked keyboard/focus restoration,
responsive composition, notes/text states, cleanup/renewal, sample coherence,
original downloads and parity with the paired mockups. After the recorded
corrections, both reviews found no further material issue in this boundary.
Final receipt/artifacts: /tmp/wc-publication-testgenie-final-{result,findings}.json.
Served final proof: /tmp/wc-publication-proof-final.log (all four locale/size
cases), /tmp/wc-publication-fixture-proof.json (final 390px case including remote
manifest artwork), and /tmp/wc-publication-health.json. Existing QA records cover
the repeated broader-suite findings; no duplicate report or unrelated repair.


## 2026-10-18 — Plain-text reader finish (W3, implementation complete)

Operator feedback: “Looks great mostly! The txt files have an unnecessary container though and look a bit sloppy”
Screenshot: uploads/c0823e24-7efe-47bb-8fdb-1288000e3f46/IMG_1713.png.
Bounded polish task: remove the source reader's inset rounded frame; use compact,
wrapped plain-text rows; suppress the duplicate idle line-count footer. Preserve
source bytes, keyboard selection, search, copy, paging and existing inspector controls.
The shared source surface stays edge-to-edge on desktop and phones. Validate the
reader regressions, scoped checks and served archive-member/native text views.

Removed the shared source surface's outer margin, rounded border and inset framing.
Plain text uses 14px type, wrapping and 28px default rows; source code retains its
existing density and wrap defaults. Gutters span wrapped paragraphs continuously.
The lower status row is present only for selection, copy feedback or pagination.

Validation: 242 preview/API tests pass across 38 files; all seven focused reader
regressions pass after the final gutter alignment. TypeScript, scoped ESLint and
whitespace checks pass. Served 320px/390px phone and 1440px desktop cases cover
both archived README.txt and a local long-paragraph TXT file: zero frame border,
radius or outer margin, full width, no horizontal overflow by default, 28px rows,
exact copied content, wrap override, archive Back and zero scoped WCAG A/AA
violations. Pointer/code and touch/code density defaults remain intact.

Fresh finish reviews checked screenshot parity, wrapped gutter alignment, idle
chrome, logical line/whitespace preservation, conditional status feedback, search,
keyboard range selection, paging and original downloads. No further text-reader
finding remained. The managed restart is healthy. Browser evidence:
/tmp/wc-text-browser-proof.json and /tmp/wc-text-{320,390,1440}-{archive,local}.png.
Checks: /tmp/wc-text-{preview-regressions,reader-final-tests,typecheck,lint,restart}.log.
Scoped Test Genie unit run **20261018-144202-56595bde FAIL** (337 seconds).
Retained evidence names the previously observed perf-doc orphan, approvals
keyboard/REST/locale checks, existing image-provider policy finding and global
function coverage 84.49% against 85%. No text-reader failure appears in the retained
diagnostics. Final focused checks and served cases pass. Receipt and findings:
/tmp/wc-text-testgenie-{result,findings}.json. Existing QA reports cover these suite
issues; no duplicate defect or unrelated repair was added.



## 2026-10-18 — Portable, bounded archive preview (W3, implementation complete)

Operator instruction: “Sounds great. Please fully and properly implement this”
following the accepted ZIP/TAR/TAR.GZ portability, read-only browsing and archive
bomb protection design. Work shape: bounded convergence task. Deliver the archive
contract in ARCHITECTURE.md, focused parser/store/transport/history regressions,
preview regression checks, scoped Test Genie unit phase and served desktop/phone
inspection. Preserve established Word/folder controls and UI. Stop only after
implementation and evidence review; report any native-platform qualification gap
precisely. Checkpoint here; no separate plan or Swarm goal is needed.

Do not reuse File Tools extraction: its out-of-scope unsafe-path/unbounded-copy
finding was already filed as knw-1792319454404803274. Archive preview uses no
filesystem writes. Cross-compilation is portability evidence, not native OS QA.


Implemented the Go standard-library archive inspector and typed ListArchive /
OpenArchiveEntry RPCs with CLI parity. ZIP preflight bounds real central records
before allocation; TAR/gzip scans charge skipped bodies, padding and trailer reads.
Virtual member handles share their root's expansion budget, lifetime and pinned
source identity. Immutable capabilities publish atomically. A bounded retained-byte
store pins active blob responses so slow downloads cannot escape the eviction quota.
No extraction, shell tools, nested auto-expansion, internal link following, host
execution, or virtual-path resolution occurs. HTML/Markdown member source stays inert.

The full-width browser uses existing inspection controls. Desktop has aligned
columns; phones have sequential secondary facts. Folder/filter/sort/hidden/page
state participates in drawer Back history, cancellation aborts archive RPC work,
and stale/late results cannot replace the restored view. Current breadcrumbs share the drawer Back band and reveal
the active folder, avoiding a duplicate navigation row. Returning to the root
restores keyboard focus to that band. Unsupported/corrupt/encrypted/over-budget states preserve an
original download, including transport errors after resolution.

Fresh review 1 examined hostile ZIP/TAR structure, sparse/link handling, ordinal
identity, UTF-8 serialization, portable path rules, actual decompression accounting,
source replacement and root provenance. It found and corrected ambiguous ZIP end
records and invalid UTF-8 names. Fresh review 2 examined publication/eviction,
active-reader lifetime, nested budgets, cancellation/history, inert runtime routes
and phone composition. It corrected active HTTP byte retention and intermediate
capability publication. The resulting focused regressions pass. Closing reviews after green confirmed
the virtual-only route boundary, active-reader quota accounting, shared nested
budget and history/focus behavior. Unregistered archive MIME types use a binary
fallback on every OS, covered by a focused regression. No further archive finding
remained in those reviews.

Validation: **241 PASS / 38 preview/API/drawer UI test files**, clean scoped ESLint,
TypeScript and diff whitespace checks; Go archive/store/transport/adapter race tests
and archive CLI tests pass. Proto owner generation and freshness verification pass;
artifact 5d5504b038b7f116141282b7305ed9d7ab7086195786aadf1f8a5799030bedbe.
Endpoint generation exposed missing registrations for existing continuity RPCs and
Sessions.Reopen; their descriptors were restored, then the owner generator passed.

Served Chrome proof covers 1440px desktop, 390px phone and 320px Arabic/Japanese:
ZIP/TAR/TAR.GZ, virtual Word/Excel/HTML members, nested archives, filter/options,
Back restoration, containment and scoped WCAG A/AA (zero violations). Six served
edge cases cover corrupt/unsupported recovery, encryption, empty archives, traversal
and member-size blocks, duplicate body identity, and bounded rows/full-index search.
These browser checks are Chrome emulation, not real Safari/OS keyboard qualification.

Cross-compilation succeeds for Windows amd64 and macOS amd64/arm64. Bridge inventory
shows native minimouse/macOS online and swarminator/Linux online; no Windows node
was listed. The discovered fleet gate validates pinned Git revisions, not this dirty
workspace patch. No native macOS/Windows archive execution is claimed or host sync
performed. Local Linux race tests are native execution evidence.

Scoped Test Genie unit run **20261018-105511-83a4f066 FAIL** (369 seconds): retained
command evidence names the existing perf/2026-10-15-long-session-performance.md doc
orphan; approvals keyboard/REST/locale checks; existing useImageInspection provider
policy; and global UI function coverage 84.38% against 85%. No archive failure appears
in the retained diagnostics; no baseline attribution beyond these observed identities
is claimed. The archive-focused final checks pass. Prior Word-task QA entries already
cover those unrelated suite defects; no duplicate bug report was filed.

A separate served desktop finding: the approvals notice can cover the dock's title /
focus control. Pointer click is intercepted; keyboard Enter works. Filed through
report-bug as **knw-1792320905192230758**; archive content remains usable. No approval
was answered. See /tmp/wc-archive-initial-desktop.png.

Evidence: /tmp/wc-archive-{go-final,cli-tests,ui-regression,typecheck,lint,proto-verify,
endpoints,restart-final}.log; /tmp/wc-archive-browser-proof.json,
/tmp/wc-archive-edge-proof.json; Test Genie artifact catalog and retained findings in
/tmp/wc-archive-testgenie-{result,findings}.json. Desktop/phone captures:
/tmp/wc-archive-{en,ar,ja}-{1440,390,320}-{root,folder,options}.png (actual combinations
are listed by the proof). Openable examples live in fixtures/file-preview as
release-package.zip, release-package.tar and release-package.tar.gz.


## 2026-10-18 — Word reader and folder navigation (W3, scoped implementation complete)

Operator feedback: “Let's do word docs! Also we currently have a way to view folders, but it doesn't look great and navigating back is bugged and closes the drawer. So do that too please”

Operator clarification: “Arrow inside the preview”

Bounded convergence task: implement the Word/folder contract in ARCHITECTURE.md,
using the paired mockup and existing artifact chrome. Scope: preview transport,
additive generated kind, governed parser/sanitizer dependencies, reader, folder
composition and navigation, localized copy, focused tests and docs. No native goal.
Existing operator-approved artifact direction applies; the explicit request
provides authority to implement both surfaces without another design checkpoint.
Keep going through friction; when progress stalls, step back — write why, then
park, switch or shrink — instead of stopping or adding more of the same.

Recall: existing artifact work is relevant; no folder-Back repair retrieved.
Memory/scopes providers timed out, so prior-art completeness is unverified.
W3 evidence: .docx has no classification/reader; directory is a dense row list
with a horizontally overflowing filter/sort toolbar. Controller has cached Back
history; the dismissal boundary needs served reproduction and a red/green test.
Hypotheses: header Back overlaps the mobile dismiss grabber (test hit targets);
Back remount/focus restoration triggers overlay dismissal (trace click/close);
controller history loses open state (test restored state independently).

Implementation findings: simple phone folder-to-folder Back initially stayed
open. The old header removed/repositioned Back when returning to the root; the
new navigation keeps a separate, stable Back control, disables it at the root,
and restores focus to the navigation landmark. The red regression fails before
this change because the stable root control is absent. Cached/controller state
preserved open=true, rejecting history-state loss for the reproduced paths.
The original dismissal's exact trigger was not independently isolated; verify
its intended outcome in served folder/file, pending-load and failed-load paths.

First browser review found sort refresh unmounted the folder/options through the
shared loading branch. Keep the cached renderer mounted, mark it busy and update
its listing in place. A second check found native disabling drops focused settings
and allows Escape to reach underlying listeners. Use aria-disabled with guarded
handlers during refresh so the dialog retains focus, and keep row navigation
blocked until ready. Browser accessibility then found a non-focusable reading
scroller; make document and table regions keyboard-focusable and labeled.
Visual review also corrected mixed-direction metadata with bidi isolation, moved
breadcrumb scrolling into the portal-mounted child, and removed an alpha suffix
from the already-alpha border token (which had produced bright row rules).
Other review fixes: bound HTML element count, retain paging under filters, safely
format unknown dates, show current breadcrumb, move copying to the existing
footer menu, and renew a stale blob capability through the controller on retry.
Reuse the existing reading-range/outline hook and shared Office byte/ZIP bounds.

Validation: **231 PASS / 37 preview/API/drawer files**, including real Word
conversion, image handling/external-resource rejection, sanitizer attacks, worker
bounds/timeout/cleanup, reading interactions, cached/pending/failed Back and sort
refresh. Three scoped reader/inspection locale checks pass. Strict scoped lint,
TypeScript/string and selector freshness pass; generated protobuf verification
exits zero and production/lifecycle builds succeed. The final empty-folder touch
refinement also passes all 14 folder tests. The runtime restarted healthy.

Five served cases pass: 390/1440/320px English and 320px Japanese/Arabic. They
cover folder filtering, sort/hidden refresh, nested folder/file Back, root
retention and focus, stable hit-target geometry, current breadcrumb visibility,
Word outline/search/font options, containment and scoped WCAG A/AA. Computed row
borders match the shared token; the active theme uses a 26% alpha token, so a
fixed 10% test expectation would misstate the theme contract. Six additional
served edge cases cover hidden-only empty folders, filtered pagination and
invalid/encrypted/oversized/empty Word documents with recovery/download.
Chrome emulations do not certify native Safari or a real virtual keyboard.

Fresh review 1: after the functional/visual fixes, reviewed conversion and
sanitization boundaries, byte/element/image budgets, worker cancellation,
capability renewal and controller request guards. Existing Office helpers and
reader ranges are reused; no material correctness or maintainability finding.
Fresh review 2: reviewed desktop, phone and RTL captures against the paired
mockup, including options and scrolling. Current breadcrumb and isolated metadata
are readable; borders follow theme tokens, navigation stays separate from
dismissal, content has no decorative card, and empty actions use 44px targets.
No remaining material finding in the assigned scope. Assigned Back behavior is
verified; the original specific dismissal trigger remains unisolated as noted.
Human design ratification and broad scenario certification are not claimed.

Evidence: /tmp/wc-word-regressions-final.log, /tmp/wc-word-{go,lint,types,selectors}.log,
/tmp/wc-word-proto-verify.log, /tmp/wc-word-runtime-final.log,
/tmp/wc-word-folder-browser-{final.log,proof.json} and
/tmp/wc-word-folder-edge-{browser.log,proof.json}. Captures:
/tmp/wc-word-folder-{locale}-{width}-{folder,word,folder-options,word-options}.png.
The openable fixtures are fixtures/file-preview/operations-brief.docx and fixtures/.
Dependencies were approved/installed through SDA (Mammoth 1.13.0, DOMPurify 3.4.17);
scenario dependency validation passes with existing governance warnings.

Test Genie unit run **20261018-045829-e674a661 FAIL** (365s): known docs orphan
perf/2026-10-15-long-session-performance.md; approvals keyboard/REST/locale assertions;
untouched useImageInspection provider-policy finding; function coverage 84.51%
against 85%. Findings and bounded command output were inspected through the
opaque artifact catalog (/tmp/wc-word-{artifacts,findings}.json and unit.log).
These match existing recorded categories; no baseline attribution or complete
failure inventory is claimed. Existing report ids remain applicable; no duplicate
reports or unrelated fixes were made. Work journal receipt: cffa4bcb-e64b-49e4-bade-8f67e9fd64b9.

## 2026-10-18 — Diff content regression (W3, scoped repair complete)

Operator feedback: “Thanks! That looks better. I think the diff view content has regressed though from before you did anything to it. The controls bar is definitely great, but the actual diff content doesn't look as nice and is missing the coloring for the removed lines and just in general looks sloppy”

Follow-up operator feedback: “Looks nice but it doesn’t need to be in a container. That’s unnecessary clutter /home/matthalloran8/.vrooli/cache/vrooli/web-console/uploads/c0823e24-7efe-47bb-8fdb-1288000e3f46/IMG_1706.png”

Bounded visual refinement: remove the diff canvas's outer margin, border and
corner radius. Content should span the drawer immediately below file navigation.
Keep the functional scroll region, accepted controls, coloring and line gutters.
Validate with existing focused reader regressions and served phone/desktop bounds;
this reversible spacing-only change does not require another broad certification
run or a new test that merely snapshots class names.

Verified: eight focused diff/parser regressions and scoped lint pass. The console
restarted healthy at 00:34:44. Four served cases (390/1440px, unified/split) verify
zero outer margin/border/radius, matching drawer left/right edges, direct placement
below file navigation, preserved red/green colors and no scoped WCAG A/AA issues.
Phone capture visually reviewed: /tmp/wc-diff-edgeless-en-390-real-unified.png.
Evidence: /tmp/wc-diff-edgeless-{tests,browser,runtime}.log and
/tmp/wc-diff-edgeless-browser-proof.json. Follow-up refinement is complete.



Bounded repair: restore readable, compact red/green diff content in unified and
split views while preserving the accepted controls. Reuse the source/change
contract and paired mockup below. Finish with meaningful rendering/layout
regressions, scoped lint/types, served phone/desktop color/scroll/alignment checks,
and the relevant Test Genie phase. Preserve unrelated changes. No native goal.

Prior art: recalled current toolbar work and inspected the HEAD diff reader.
HEAD used defined red/emerald palette classes and compact ~21px lines. Current
reader uses undefined bg-wc-danger, text-wc-danger-text and text-wc-success-text;
Tailwind has error-text/error-detail and success instead. Recall providers for
agent memory/scopes were unavailable; no prior fix for this symptom was retrieved.

Hypotheses: (1) missing theme classes erase removal styling (confirm by generated
CSS and served computed colors); (2) parser misclassification drops removed rows
(reject if parsed kind/old coordinates remain correct); (3) excessive padding,
per-line overflow and split sizing explain the visual regression (confirm by
served line heights/scroll containers). Unit/parser evidence and computed browser
styles, not class-name snapshots alone, must protect the repair.

Content target: dense monospace rows; unmistakable muted red/green backgrounds,
readable change text and matching number gutters; quiet context; aligned split
pairs and a single horizontal canvas scrollbar when wrapping is disabled.
Retain exact original copying, coordinates, navigation, paging and source fallback.

Confirmed root cause: the generated-CSS regression fails against the prior reader
because its removed row has no background-color declaration. It passes after
using defined error-text/success utilities. Existing parser tests retain removed
kinds and coordinates, rejecting parser loss. Served computed rows now read
red rgb(252,165,165) / green rgb(74,222,128) with distinct 10% backgrounds.
The first browser pass also confirms the previous 37px/padded-row shape is now
24px for a short line and no code line has its own scroll container.

Focused proof: **201 PASS / 31 preview/API/drawer files**, including the failing-
before/passing-after CSS generation regression. Strict scoped lint and TypeScript/
string freshness pass. Browser proof covers representative long/asymmetric and
real repository patches, unified/split modes and wrapping on/off at 390/1440px,
plus Arabic at 320px. Check computed colors, viewport containment, equal paired
row geometry, one canvas scroller, absent split partners and exact patch copying.
Two additional unwrapped-canvas checks scroll to the rightmost edge and confirm
code remains within its own split cell (/tmp/wc-diff-scroll-proof.json). Final
browser verification additionally checks that the short Go InstallName field
stays on one physical line in the 320px unified view after gutter refinement.
All **24 final served cases pass**, with zero scoped WCAG A/AA violations/page
errors. The focused eight diff/parser cases, strict lint and types passed again
after that final adjustment. The final console restart was healthy at 00:23:51.
Work record: 4e6671c3-fa48-4150-90c1-f5a1e3f98ed2.
Evidence: /tmp/wc-diff-{color-red,focused,regressions,lint,types,browser}.log,
/tmp/wc-diff-browser-proof.json and desktop/phone captures. Native iPhone Safari
is not certified. Relevant Test Genie unit run 20261018-041846-e2cec0b2 finished FAIL: the same
previously recorded docs orphan, approvals keyboard/REST/locale assertions,
useImageInspection provider policy and aggregate function floor (84.56% vs 85%).
Durable findings were inspected through the opaque artifact catalog; local copy:
/tmp/wc-diff-findings.json. No broad certification is claimed.

Fresh review 1: the shared split grid aligns wrapped heights and column widths;
number gutters use original coordinates with enough digits, missing partners are
quiet hatched blanks, and pre elements inherit monospace styling without private
scrollbars. Unwrapped long lines expand the common canvas in either mode.
Fresh review 2: inspected representative desktop split and actual phone unified
captures against the updated paired mockup. Red/green changes remain readable,
context is neutral, heavy row rules and excessive padding are gone, and accepted
controls retain their composition. Reviewing the actual Go patch at 320px then
revealed avoidable wrapping from three-digit minimum gutters and generous insets.
Reduce the minimum to two digits and narrow gutter/code insets; retain dynamic
width for larger original coordinates. The final served 320px capture verifies that refinement.
Analogous undefined text classes in the separate source/workbook error messages
and structured-value renderer were filed as knw-1792297202457523817.



## 2026-10-17 — Source, diffs, and Excel previews (W3, toolbar refinement complete)

Latest operator feedback (implemented and scoped verification complete; human acceptance not claimed): “Why did you only link excel? You were working on 3 different file types. And I highlighted the part that looks amateur. Its amateur for the other 2 file tyoes you did recently as well /home/matthalloran8/.vrooli/cache/vrooli/web-console/uploads/c0823e24-7efe-47bb-8fdb-1288000e3f46/IMG_1705.jpeg”

Toolbar-specific refinement: the marked phone controls mix a truncated worksheet
picker with an in-flow form. Prior closed-menu captures did not prove the open
controls. Keep the toolbar a stable identity/action row, give the worksheet its
own full-width navigation row, and present settings in the existing responsive
dialog with labeled sections, explicit view choices and a composed jump field.
Use the close affordance (no grabber), support Escape/backdrop/return focus, and
keep navigation focused at its destination. Verify open settings at 320/390px,
forced phone table view, desktop and Arabic/Japanese. Link actual code/log, patch
and workbook examples in the delivery. Do not claim real Safari verification.

Boundary extension from open-settings interaction proof: nested ResponsiveDialog
Escape can cascade to FullPageDrawer window listeners when dismissing the top
surface exposes the next layer during the same event. Repair useOverlaySurface
through its governed 1.4.8 draft: the focused top surface consumes Escape before
window listeners. Keep inner controls' prevented Escape behavior. Add owner
regressions before publishing; no preview-local global listener workaround.

Package publication boundary: catalog export generation rejects 17 existing
Command Center imports of evicted ProvenanceInk 0.1.2 and selects the previous
verified artifact. To ship the overlay fix, migrate only those stale imports to
released 0.1.3 through the normal adoption contract; retain all other content.
The overlay draft explicitly pins current useFocusReturn 1.1.2 because its copied
lock referenced the retired 1.1.1. Do not count a degraded fallback as the new fix.

Toolbar validation (2026-10-18): **200 PASS / 31 preview/API/drawer files**;
scoped strict lint, TypeScript/string and selector freshness pass. The owner
Escape regression fails against 1.4.7 and passes against published 1.4.8;
**14 PASS** overlay gesture/exit tests. Command Center type-check and **27 PASS**
scene/shell tests validate the stale import migration. Catalog generation now
reports zero broken consumer imports and the owner package selects the new
verified artifact f55c7ff77448f7890b1cdb5c255059cc41ee27c8b47452773612668d24d4c9e2.
The console restarted healthy at 00:05:52.

**24 PASS** served cases cover source, logs, changes and workbook at 320/390/1440px,
Japanese and Arabic at 320px, plus a 390x480 short viewport. All open options stay
within the viewport with no scoped WCAG A/AA violations or page errors. Checks
include native view selection, forced phone table, Shift-Tab containment, invalid
and valid jumps, destination focus, Escape/backdrop dismissal, parent retention,
trigger focus and a stable toolbar. **8 additional PASS** desktop/phone captures
wait for selection transitions before visual review. These are Chrome emulations,
not native iPhone Safari or virtual-keyboard certification. Temporary proof:
/tmp/wc-toolbar-browser-proof.json, /tmp/wc-toolbar-capture-proof.json and captures
/tmp/wc-toolbar-{locale}-{width}x{height}-{kind}-{open,closed}.png.

Fresh review 1: confirmed bounded parsing/copying behavior remains covered; native
radio groups preserve keyboard semantics; settings use the existing overlay
owner and font inputs retain 16px mobile type. This review caught the Escape
cascade and led to the owner regression/fix above.
Fresh review 2: reviewed open and closed desktop/phone settings, table selection,
Japanese/RTL and short surfaces. Forced phone table no longer pre-opens an
inspector; explicit inspection has a back action and restores cell focus. A
capture timing issue showed the previous radio highlight during its transition;
settled captures and checked-state assertions confirm the selected mode.

Broader receipts: Web Console unit run 20261018-035744-88297c90 fails the known
docs orphan, approvals keyboard/REST and locale assertions, plus function coverage
84.55% against 85%. A separate direct-render policy finding identifies the
untouched useImageInspection.test.tsx (filed as knw-1792296322249006933).
Command Center unit run 20261018-040612-8f8640fc fails the UI branch floor
83.66% against 85%; no baseline attribution is claimed. Component contract
ctr_c7de7039a8830918_09a1adc97fa9c48dfbdfcbc9 passes closure, source, contract and
default/swipe stories; loading/error/review captures fail in BAS CaptureService
(HTTP 500/readiness timeout), matching the earlier reported capture limitation.
React Component Library unit run 20261018-040613-eb7683e4 also fails outside the
focused overlay regressions: API design-style fixture paths and missing catalog
fixtures, stale CLI primitive evidence, 22 UI assertions (including the ThemeSwitcher
registry fixture), UI function/branch floors 82.78%/80.04% against 85%, and catalog
conformance lint. Bounded evidence is retained in /tmp/wc-toolbar-rcl-findings.json;
no complete failure inventory or baseline attribution is claimed. Command Center
coverage finding: knw-1792296578219810150. No broad certification or human acceptance
is claimed. RCL phase findings were filed as knw-1792296640374306042. Final work
journal receipt: 33a2498d-2287-406a-b25e-b804fa11d48d; final lifecycle status is healthy.

Operator feedback: “Let's do all 3! Make sure they're really well-done and professional . Especially excel since we currently can't view that at all”

Bounded convergence task: work until the source/change/workbook contract in
ARCHITECTURE.md is implemented and proven by focused UI/parser/worker/API tests,
strict lint/types/generated freshness, build, scoped Test Genie unit phase and
served desktop/phone interaction/accessibility checks. W3 evidence: CodeLinePreview
has basic highlighting/wrap/font controls; DiffPreview has only colored patch
lines; Excel has no kind or renderer. Scope is file-preview UI/API transport,
additive proto enum through owner generation, localized copy/selectors, governed
parser dependency and existing docs. Follow the paired desktop/phone mockup and
existing artifact chrome. No editor, formula engine, macro execution or unrelated
readiness certification. Preserve concurrent workspace changes.

Review twice freshly after passing scoped checks, addressing correctness, input
fidelity, lifecycle, maintainability and visual polish; retain findings here.
Keep going through friction; when progress stalls, step back — write why, then
park, switch or shrink — instead of stopping or adding more of the same.
Blocked means a decision, credential, or approval you lack. Name it. Handoff
changed, verified, remaining and unverified evidence here; never weaken checks.
No native goal created; no separate plan. Prior limitations remain as recorded.


Operator feedback (implemented and verified; human acceptance not claimed): “Continue please. But I'm looking at them right now. They could look better. You didn't make them as clean and polished as the others yet”

Visual refinement: the first implementation has functional controls but lacks the
finished hierarchy of the approved media/readers. Replace full-width gray form
chrome with composed tool headers, clear artifact identity and compact options;
frame the reading surface, refine source density and diff typography, add desktop
worksheet tabs, and redesign cell details around the formatted value. Preserve
phone row/cell flow and accessible labels. Judge served captures against this
brief, not feature counts. Do not mark this feedback resolved until the visual
pass and desktop/phone evidence are complete.

Convergence findings resolved before final review:
- Omitted long values were indistinguishable from blank cells; the workbook model
  now records omission explicitly, disables copying unavailable values, and labels
  partial previews. Merged blank cells retain their merged range. Sheet parsing
  uses the first 30 metadata-selected sheets rather than parsing every sheet.
- Search now reveals the first matching source page. Severity recognizes explicit
  labels rather than incidental message words. Large diff hunks page at 400 rows.
- Navigation closes its options and focuses the destination. Desktop source uses
  denser rows while pointer-coarse selection targets remain 44px.
- A second review found CRLF range copies could end in a lone separator CR, and
  resizing a phone row inspector into a desktop grid could hide pagination. Both
  now have focused regressions. Runtime registry fallbacks reject inherited keys
  while retaining exhaustive enum mappings. Existing touched-file lint findings
  were cleaned without weakening unknown-kind guards or protective tests.


Boundary extension from served RTL evidence: FullPageDrawer 1.4.5 centers its
grabber using logical inset-inline-start and physical negative translation. In RTL
the handle sits at x=-128 on a 320px viewport, creating a 448px scrollable panel;
focus/copy then shifts its content right by 12px. Fix the owner through the
FullPageDrawer 1.4.6-draft.1 workflow, pairing physical left centering with physical
translation. No released source is edited, and no consumer CSS workaround is used.
Evidence: /tmp/wc-inspection-ar-probe.log and Arabic failure capture.


**Earlier implementation proof, served at 23:44:17; superseded by the toolbar feedback and refinement above.**
Code/logs now have literal search with match navigation, original-line navigation,
exact selected-line copying, wrapping/font controls and explicit severity filters.
Source highlighting is bounded at 150,000 characters/3,000 lines/10,000 characters
per line; larger sources retain plain reading and 400-line paging. Literal search
covers loaded text and the chosen severity, capped at 500 matches. A roving gutter
supports arrow/Home/End navigation and Shift range selection. Search opens with
focus and Escape closes it without dismissing the file drawer.

Changes now have file/hunk navigation, complete loaded-file addition/deletion
counts, correct old/new coordinates, unified and wide split views, wrapping,
400-row hunk paging and source fallback for malformed/combined/oversized patches.
Original copying and download retain the patch bytes. The parser accepts at most
12,000 loaded patch lines. The visual pass adds composed identity headers, compact
settings disclosures, restrained canvases and comparison labels.

Excel (.xlsx/.xls/.xlsm/.xlsb) is a dedicated blob-backed proto/API/UI kind. The
local SheetJS CE 0.20.3 worker preserves formatted values, raw values, formula
text, original coordinates, blanks/false/zero, sheet order/visibility and merged
ranges. Formulas/macros/links/HTML are never executed; saved results are not
recalculated. Limits: 15 MiB compressed/input bytes, 64 MiB per ZIP entry and
128 MiB declared ZIP expansion, ten seconds, 30 sheets, 2,000 physical rows,
100 columns, 50,000 cell positions and four million displayed/raw/formula
characters. Over-budget values are explicit omissions, not blank data. Fetches
are streamed/bounded; stale workers terminate. Desktop has sticky grids, sheet
tabs and a value-first cell inspector; phones drill from rows to cells with focus
restoration. Search covers loaded cells/formulas; address navigation, paging,
copying and explanatory error/download recovery remain available.
Demo: fixtures/file-preview/operations-workbook.xlsx (three sheets).

### UX Issues — resolved within this boundary

The operator's visual feedback led to a second composition pass rather than a
feature-only handoff: quiet icon/metadata headers, search on demand, contained
settings, dense pointer source rows with full touch targets, borderless file/sheet
selectors, framed reading surfaces, desktop sheet tabs, and formatted-value-first
cell details with compact copy actions. Sheet/page/filter changes reset scroll;
returning to a row reveals its restored focus. A phone inspector resized into a
desktop grid retains pagination. The RTL grabber fix was published as the governed
FullPageDrawer 1.4.6 release and linked through the owning adoption workflow; the
selected runtime package was rebuilt with the control-plane package owner.

Final proof: **205 PASS / 31 focused preview/drawer/API files**, including the
existing preview families and ten client-wrapper compatibility cases. **36 PASS**
new reader/parser/worker cases meet unchanged scoped coverage floors across ten
modules: **99.17% lines/statements, 98% functions, 87.22% branches**. Scoped strict
lint, TypeScript/string freshness, selector freshness, locale key/token parity,
Go file-preview/handler tests, proto owner verification, dependency governance,
production build and whitespace checks pass. Dependency governance retains its
existing unrelated warnings. The final lifecycle restart is healthy at API 16382,
UI 21233. All **28 served Chromium cases** pass with zero preview WCAG A/AA
violations and zero page exceptions: twelve English cases at 1440/390/320px,
eight Japanese/Arabic cases at 320px, and eight real/negative corpus cases.
Corpus includes source, logs, a repository patch, original XLSX, XLS, XLSB,
damaged-file recovery and byte-limit recovery. Arabic checks also assert a
centered grabber and contained panel after copying. Captures were visually read.

Two fresh final self-reviews after corrections found no material in-scope issue:
1. Correctness, bounds, transport compatibility, stale work, input fidelity,
   keyboard navigation and cleanup reviewed against passing focused evidence.
2. Served desktop/phone/localized compositions compared with the refined brief;
   long content, cell details, sheet changes, search, copying and accessibility
   rechecked. Operator feedback is implemented; these reviews do not substitute
   for independent acceptance or physical-device testing.

Broader evidence remains red: Test Genie unit **20261018-032512-5801db51 FAIL**
(about 338 seconds). API reports the known unmanifested perf documentation
(TestDocsNoOrphans; knw-1792160172206667201). The UI snapshot has seven assertions:
three intermediate preview/decoder mismatches (CRLF source selection and default
warnings) were corrected and pass the settled scoped tests; the other failures
are the existing approvals keyboard contract, REST allowlist and locale parity.
Aggregate function coverage was **84.47%** against the unchanged 85% floor. This
mutable snapshot is retained, not rewritten as a whole-scenario passing verdict.
No full scenario certification is claimed.

The FullPageDrawer component-owner test report
ctr_700380e95fda57ca_09a1adc97fa9c48dfbdfcbc9 passes closure/source/story-contract
checks but fails BAS screenshot/readiness capture stages. Direct served Chromium
coverage is separate evidence, not a replacement passing component verdict.
Reported capture failures: knw-1792294952927270620. Adoption also inserted a duplicate
selector import and generated an empty drawer selector map; the editable registry
import was corrected after the selector owner check, while the generated map was
left owner-produced. Report: knw-1792294952078088054. No protective checks or tests
were weakened. These owner-workflow defects remain outside this preview task.

Evidence: /tmp/wc-inspection-{final-regressions,coverage,final-lint,final-types,
final-go,locales,selector,dependency-validation,build,final-runtime}.log;
/tmp/wc-inspection-{browser,localized,corpus}-proof.json;
/tmp/wc-inspection-{320,390,1440,ja,ar,real}-*.png;
/tmp/wc-inspection-{testgenie-result,native-findings,drawer-test,drawer-publish}.json.
Temporary browser inputs are removed after verification; the workbook demo and
proof captures remain. Work record: 9d039464-2b10-44ce-b57e-0013b295a252. Physical iPhone Safari and virtual-keyboard geometry on
real devices are unmeasured. ZIP bounds use declared central-directory sizes;
worker time/input/output limits do not claim a browser-process memory sandbox.


## 2026-10-17 — Markdown and JSON/YAML readers (W3)

Bounded task: work until the approved Markdown/configuration reader contract in
ARCHITECTURE.md is implemented and verified. Existing renderer inspection names
W3: Markdown has rendered content without reading/search/outline/source controls;
JSON/YAML are highlighted source without structure or value inspection.
Scope: file-preview UI/helpers, localized strings/selectors, governed parser and
existing docs. Reuse the existing Markdown renderer and artifact chrome. Proof:
focused behavioral/parser/lifecycle regressions, strict lint/type/build, scoped
Test Genie unit phase, served desktop/mobile captures and accessibility checks.
Two fresh review passes must find no material issue in the assigned boundary;
retain each finding and resolution here. Keep going through friction; when
progress stalls, step back and record why before switching or shrinking.
Blocked means a decision, credential, or approval you lack. Name it. No unrelated
readiness certification, source editing, private APIs or weakened tests. Existing
preview-family limitations remain evidence, not claims this task will repair them.
**Scoped implementation complete and served.** Markdown now has Read/Source,
rendered-heading contents, literal search across inline formatting with match
navigation, text sizes, safe document-relative file navigation and source copying.
Wide drawers compose an outline beside the reading column; narrow drawers open
contents on demand and restore focus to the selected heading. Search uses DOM
ranges and the browser Highlight API without rewriting React-owned content;
unsupported highlight APIs retain match counts and navigation. Search ignores
hidden/interactive chrome and releases ranges, observers and scheduled work.

JSON/YAML now has a local worker-backed Structure/Source inspector, nested
collections, filtering across collapsed descendants, stable property paths,
full-value inspection and original-value/path/source copying. Wide drawers keep
details beside the tree; phones drill into details with a focused Back control.
YAML streams expose document selection; aliases remain references. Unsupported
tags, complex/colliding keys and unrepresentable scalars use source fallback.
JSON formatting re-indents tokens without rounding large integers or decimals.
Sources are never edited. Oversized, deep, invalid or truncated input retains
source/download access. Parsing is capped at 300,000 characters, 5,000 nodes,
60 levels and four seconds; termination releases stale workers. Limited source
uses one keyboard-focusable text region rather than per-line syntax rendering.
The parser is governed **yaml 2.9.1**; scoped dependency validation passes.

Final verification: **159 PASS** across 22 focused preview/drawer test files,
including existing HTML/audio/image/PDF/video/table cases. **27 PASS** new reader,
parser and worker cases also pass the unchanged scoped coverage thresholds:
**100% lines/statements, 88.23% functions, 89.89% branches**. Strict scoped lint,
TypeScript/string freshness, selector freshness, reader locale key/token parity,
production build and whitespace checks pass. Served Chrome checks cover 320×740,
390×844 and 1440×1000, Japanese/Arabic at 320px, and real README/service JSON/CI
YAML plus invalid and 300,005-character source fallback. All **20 browser cases**
have zero preview WCAG A/AA automated violations and zero page exceptions; narrow
panels contain variable-length text. Exact clipboard spelling, Markdown search
across emphasis, heading focus, local-file navigation, formatting, YAML selection,
and reader/inspector composition were exercised. The final lifecycle restart
reported healthy on API 16382/UI 21233 at **23:00:53**.

Convergence review findings and resolutions:
- Review 1: source-mode highlights retained DOM ranges; source switching now
  clears them. Numeric spelling, YAML scalar/tag fidelity, and empty key/string
  visibility were checked and covered. Implicit-null source copies preserve the
  actual empty bytes. Copy feedback is announced and phone detail focus restores.
- Review 2: mobile contents/search competed for space; contents now dismisses
  mobile search. Large fallback could invoke expensive per-line syntax rendering;
  it now uses one text node. The real large-file accessibility check then found
  a missing keyboard scroll target; the source region now has a name and tab stop.
- Fresh review 3 after fixes: scoped code/correctness/lifecycle and input-integrity
  checks found no material issue. Worker and Highlight API ownership have explicit
  cleanup; unsupported shapes have truthful source access. Scoped tests pass.
- Fresh review 4 after fixes: served captures against the reader mockup, desktop
  and phone interactions, localized composition, real files and negative inputs
  found no remaining material issue within this boundary. These are self-reviews,
  not independent acceptance or physical-device certification.

Test Genie unit run **20261018-024908-9f1a8b9c FAIL** is retained as observed.
API `TestDocsNoOrphans` reports the existing unmanifested
`perf/2026-10-15-long-session-performance.md` (known report
`knw-1792160172206667201`). The UI snapshot contains seven failing assertions,
including a reader test/markup mismatch captured while files changed and
subsequently covered by the settled focused pass, and aggregate function
coverage **84.07%** against the 85% floor. The native snapshot is not rewritten
into a passing verdict; neither whole-UI coverage nor exact-input/full-scenario
certification is claimed. Native evidence was fetched through the run's opaque
artifact catalog and retained at `/tmp/wc-reader-native-findings.json`.
Physical iPhone Safari gesture/keyboard behavior is unmeasured. Existing API text
bounds still apply; search is capped at 500 rendered matches. No editor, new API
kind or authorization change was added.

Evidence: `/tmp/wc-reader-{final-regressions,coverage,final-types,final-lint,
locales,dependency-validation,build,runtime-final}.log`,
`/tmp/wc-reader-coverage/coverage-summary.json`,
`/tmp/wc-reader-{browser,localized,corpus}-proof.json`,
`/tmp/wc-reader-testgenie-result.json`, and
`/tmp/wc-reader-{320,390,1440,ja,ar,real}-{markdown,json,yaml,invalid,large}.png`.
Temporary browser fixture files are removed after verification; evidence captures
remain. Work record: `e6592e44-4ae3-494f-95aa-1cc985974e2e`.
Prior unrelated problem entries remain historical evidence.

## 2026-10-17 — Image/PDF/video/table preview family (W3)

Work shape: bounded task. Work until all four operator-approved preview upgrades
in ARCHITECTURE.md's focused artifact inspection contract are implemented and
verified. Proof: focused Vitest regressions for controls, parsing, math and
teardown; strict scoped lint/type/build; relevant Test Genie unit phase; served
mobile/desktop browser interactions and accessibility captures. Read the existing
renderers, audio transport and docs/mockups/artifact-preview-family.svg first.
Boundary: Web Console file-preview UI/helpers, locale/selector owners, PDF viewer
dependency/build assets and existing documentation. Validation posture is scoped;
fix understood blocking issues, retain unrelated failures without weakening tests.
Blocked means a decision, credential, or approval you lack. Name it. Checkpoint
changes, verified outcomes and remaining/unverified behavior here if interrupted.
Do not turn previews into editors, change authorization, or claim physical Safari
qualification from desktop browser fixtures.

**Scoped implementation complete and served.** Images/SVGs now have Fit/100%,
focal wheel/pinch zoom, bounded drag/keyboard panning, rotation, three backgrounds
and intrinsic dimensions. PDFs now use locally bundled PDF.js 6.4.299 with a
version-matched worker, fonts, CMaps, color profiles and WASM, with page navigation,
fit/zoom, selectable/searchable text, password retry and recovery actions. Video
shares the audio transport, adds a dedicated stage, custom seek/skip/rate/repeat,
volume/mute and fullscreen with native controls available on demand. CSV/TSV now
has stable numeric-aware sorting, filtering, bounded pagination, keyboard/pointer
column resizing, original-value copy, header/source options and mobile record
list/detail composition. Ragged fields stay visible; malformed or overly wide
input shows source rather than silently dropping values. New controls use shared
library components, semantic tokens, 44px primary controls and all three locale catalogs.

Final focused preview regressions: **128 PASS** across 17 files, including existing
HTML/audio coverage and stale asynchronous PDF-session disposal. Artifact-copy
locale/key/token parity passes. Strict scoped lint, TypeScript/string freshness,
selector freshness, production build and whitespace checks pass. PDF.js was
installed/approved through Scenario Dependency Analyzer; scoped governance passes.

Served Chrome checks at **320×740, 390×844 and 1440×1000** cover all four previews,
full-width desktop layouts, native pointer pinch (2× without scaling app chrome),
PDF search/navigation and selectable text layers, actual MP4 playback/fullscreen,
numeric table sort, keyboard resizing, mobile details and exact clipboard values.
All **12 English browser checks** and **8 Japanese/Arabic 320px checks** have zero
preview WCAG A/AA automated violations, zero page exceptions and no panel overflow.
The real tagged PDF exposed broken annotation ownership in PDF.js 5.6; upgrading
to 6.4 resolved it without changing vendor output or removing document semantics.
Actual encrypted-PDF retry/cleared-password/unlock and malformed-PDF recovery also
pass. Public document reset cancels pending render/search work before worker
teardown. Temporary encryption/error fixtures were removed after verification.

Evidence: `/tmp/wc-artifacts-final-{focused,types,lint}.log`,
`/tmp/wc-artifacts-{lint,locales,dependency,build,runtime}.log`,
`/tmp/wc-artifacts-browser-final-proof.json`,
`/tmp/wc-artifacts-localized-proof.json`,
`/tmp/wc-artifacts-pdf-recovery-final.log` and
`/tmp/wc-artifacts-{320,390,1440,ar,ja}-{image,pdf,video,table}.png`.
The final lifecycle restart reported healthy on **16382/21233** at 11:31:04.
Physical iPhone Safari gesture/fullscreen quality remains unmeasured; its native
fullscreen fallback is covered by a typed interface fixture, not a device claim.
Test Genie unit run **20261017-152805-a3a5ece0 FAIL** is retained as observed:
API `TestDocsNoOrphans`; UI approvals keyboard-avoidance contract, workspace voice
status-bar color, approvals REST allowlist and approvals locale parity (five UI
assertions). The snapshot also reports aggregate function coverage **83.57%**
against 85%. After that snapshot, 11 focused helper regressions added real
cancellation, stale clipboard ordering, pinch/pan/focal-wheel and PDF adapter
lifecycle/search/password coverage. Their scoped coverage run passes the unchanged
thresholds: **98% lines/statements, 96.15% functions, 86.9% branches** across the
three helpers. This does not establish a passing whole-UI coverage floor. No
full-suite or exact-input certificate is claimed in this concurrently edited tree.
Known docs/REST/locale reports remain `knw-1792160172206667201`,
`knw-1792243709042628594`, and `knw-1792243443164563388`. New unrelated failures
were captured through [report-bug](/home/matthalloran8/.codex/skills/report-bug/SKILL.md):
keyboard contract **knw-1792251518018795539**; voice status fill
**knw-1792251519359547967**. Record recall was degraded; prompt-manager discovery
was used once as its fallback. Native Test Genie evidence was retrieved through
the run's opaque artifact catalog, retained in `/tmp/wc-artifacts-owner-findings.json`.
Additional evidence: `/tmp/wc-artifacts-helper-{coverage.log,lint.log,types.log}`,
`/tmp/wc-artifacts-helper-coverage/coverage-summary.json` and
`/tmp/wc-artifacts-testgenie-{result.json,findings.json}`. The required
completed-work journal append was attempted with trigger/approach/evidence/outcome,
but Vrooli Memory returned `append memory: unavailable: unexpected EOF` without
an entry ID. Commitment is unknown; no duplicate append was attempted. This entry
retains the work record and validation evidence independently of that outage,
reported as **knw-1792251659715601635**.

## 2026-10-17 — Audio preview player (W3)

Work shape: bounded task, authorized by the operator's approved audio-player
proposal. Work until the audio drawer presents track identity and real waveform
seeking, transport, repeat/speed/volume, waveform zoom, A/B looping and tidy file
actions across narrow/wide drawers; optional analysis must not block streaming.
Sources: ARCHITECTURE.md's audio preview contract and audio-preview-player.svg.
Boundary: audio renderer/helpers, locale/selector owners, dependency governance,
focused tests and existing tracking docs. Validate focused Vitest regressions,
type/lint/build, scoped Test Genie unit phase, and browser captures/interactions.
Fix understood blocking defects; retain unrelated failures with their evidence.
Blocked means a decision, credential, or approval you lack. Name it. Checkpoint
partial work here after an interruption; report changes, verified and unverified
behavior. Do not broaden this into a media editor or claim native Safari proof
from Chromium.

**Scoped implementation complete.** The player now presents embedded raster
artwork/title/artist when available, a real seekable waveform, play/pause, ±10s,
whole-track repeat, speed, volume/mute, 1/2/4/8× waveform inspection and movable
time windows. A/B markers show tenths of a second, loop a selected passage, and
clear without modifying the file. Seeking outside a passage exits its loop;
whole-track and passage repeat are exclusive. The header retains Copy path;
Download remains visible and secondary actions disclose native system controls.
Native streaming is independent of bounded optional analysis. Teardown stops
playback, aborts fetches and revokes artwork URLs; StrictMode setup restores the
source after cleanup replay. Changing VBR duration estimates do not trigger
repeated decodes. The installed music-metadata dependency is pinned at 11.12.3
through Scenario Dependency Analyzer; its scoped governance validation passes.

Final focused Vitest evidence: **49 PASS** across renderer, transport, analysis,
resource lifecycle, zoom/loop, cancellation and fallback cases; the new audio-copy
locale parity check also passes. Type-check (including generated string freshness),
scoped strict ESLint, production build and whitespace checks pass. Browser checks
against the lifecycle-served production bundle exercise real M4A/AAC streaming,
embedded artwork, pointer/keyboard seeking, playback/pause, zoom and A/B repeat at
320×740, 390×844 and 1440×1000. Every viewport has zero player WCAG A/AA automated
accessibility violations, zero page errors and no horizontal card overflow. MP3
also played in bundled Chromium; that Chromium's missing AAC codec produced the
expected M4A fallback, while installed Chrome decoded it. A light-token fixture
checks palette adaptation; it is not a separate product theme claim.

Test Genie unit run **20261017-131904-025ae858 FAIL** is retained as observed:
API TestDocsNoOrphans and UI checks failed. The UI snapshot includes two new audio
test-spy assertions fixed afterward (covered by the final focused pass), unrelated
approvals REST-exception and locale failures, and concurrently changing inputs.
No passing full-suite or exact-input certificate is claimed. Locale drift was
filed as **knw-1792243443164563388**; approvals transport registry drift as
**knw-1792243709042628594**. The docs-orphan failure is already known under
**knw-1792160172206667201**. Physical iPhone Safari, system routing destinations
and background A/B timing have not been measured; A/B repeat is a listening tool,
not a sample-accurate audio editor.

Evidence: `/tmp/wc-audio-{focused.log,locales.log,lint.log,final-typecheck.log,
build.log,browser-proof.json,browser.log,testgenie.json,unit-findings.json}` and
`/tmp/wc-audio-{320,390,1440}{,-tools}.png`. The final lifecycle restart completed
healthy on 16382/21233; a subsequent status read confirmed both serving processes.

## 2026-10-16 — Replace fragile HTML body pinch scaling (W3)

**Implementation replaced; native iPhone Safari qualification remains open.**
Operator feedback: "very laggy and glitchy and easy easily breaks".
Screenshot `IMG_1682.png` shows large white gutters above/beside the report.
This evidence supersedes the native-quality implication of the earlier pinch
completion and work record `25616e66-b6b7-4ee8-887c-074c1ecbe313`.
The prior Chromium checks measured successful magnification, not Safari quality.

Bounded task: replace the document-mutating gesture path; verify stable focal
zoom, bounded panning, repeated pinch-out, unchanged authored styles, and existing
controls using focused regressions and browser fixtures. Do not relax the sandbox.
Hypotheses: body scaling/read/write/scroll work causes expensive updates and
exposes the iframe canvas; alternatively app-wide visual viewport changes cause
chrome/layout movement. Source and screenshot support the first mechanism;
app-wide zoom was unchanged in browser observations. Native Safari was not
available to reproduce its exact white-gutter behavior.

Measured the old live bundle over 240 touch moves: **248 body style mutations,
488 body measurements, 240 scroll calls**, and **24px accumulated scroll drift**
after four in/out cycles. The replacement transports gestures from the opaque
iframe and applies one clipped iframe transform per animation frame in the host.
It never transforms/measures the authored body during a gesture. Screen-space
contacts remain independent of iframe transforms. Offsets clamp at every edge;
residual panning continues through the document only after reaching an edge.
Frame identity and numeric payload checks restrict the message channel to that
preview's inspection/scrolling. No host operation or same-origin privilege is added.

Evidence: **36 focused tests PASS** across compositor/transport, renderers and
workspace wrapper. TypeScript, standalone-runtime strict JS checking, scoped
TypeScript lint, whitespace checks, production/profile builds PASS. Browser
fixtures in both bundles exercise eight pinch strokes (240 moves), pan through
1,298px of document scroll, pinch-out, source switching, and viewport/zoom controls.
During pinch: **0 body mutations, 0 body measurements, 0 scroll calls, 0 scroll
drift**. Coverage checks find no exposed edge beyond subpixel rounding (0.002px).
Authored body scale remains unset; the document viewport and opaque origin remain.

These browser fixtures construct explicit touch coordinates: Chromium's
[documented CDP coordinate defect](https://issues.chromium.org/issues/40280325)
returns identical screen coordinates for different contacts. This limitation was
confirmed locally. An attempted client-coordinate fallback fed back through the
changing iframe transform and was removed; no such workaround ships. Fixtures
exercise the DOM/rendering/message path but do not certify physical Safari input
or device frame rate. Local iOS simulator tooling is unsupported on Linux; the
bridge inventory probe is unauthenticated, so no device-disconnection claim is
made. Native iPhone Safari remains unverified.

Evidence: `/tmp/wc-html-pinch-old-probe.json`,
`/tmp/wc-pinch-compositor-{unit.log,browser-proof.json,profile-proof.json,
types.log,runtime-types.log,build.log}`, and
`/tmp/wc-pinch-compositor-{magnified,panned}.png`.
Scoped Test Genie unit run **20261016-151253-40c6b1f2 FAIL** reports only the API
Go command failure (existing **knw-1792160172206667201**), with no UI command
failure. Source changed during validation; final focused/browser checks cover
the final transport and compositor, without claiming an exact-version full-suite
certificate. The final production bundle is served by the existing UI runtime.

## 2026-10-16 — Independent HTML preview pinch magnification (W3)

**Scoped repair complete.** Follow-up to toolbar work record
`1e62ba1b-0321-424e-836c-e55629ec3356`. The operator requested browser-like pinch
magnification independent of rendering zoom. Bounded task goal: work until
preview-local pinch, focal anchoring, panning and pinch-out hold without changing
the toolbar zoom, CSS viewport/media queries, app chrome or sandbox authority;
prove with focused tests, browser touch input, type/lint/build and scoped unit phase.

Root-cause hypotheses: (1) app viewport directives intentionally disable native
browser zoom; (2) the iframe has no local gesture handler and its touch events do
not bubble into the parent; (3) sandbox policy prevents gesture scripting.
Source inspection supported (1)/(2); real-touch browser checks reject (3) because
the handler runs under the existing script-only opaque-origin sandbox. Related
Messages/terminal pinch hooks change typography and do not supply this separate
inspection behavior. The regression first failed because the HTML document had
no installed pinch handler. With the handler removed from the same production
srcdoc, real pinch leaves painting at 1×; restoring it produces 2× magnification.

Implemented a self-contained inline module inside srcdoc: distance-based 1–5×
paint scaling around the moving finger midpoint; one-finger panning when
magnified, including continuation after lifting one finger; pinch-out restores
author styles. Single-finger scrolling/taps at 1× and ordinary wheel scrolling
remain native; Ctrl+wheel supports trackpad pinch. Cancelled/degenerate gestures
are bounded. The runtime is embedded from a raw JavaScript asset rather than
serializing a bundled function, avoiding missing helper references in profiling
builds. It needs no parent bridge, external dependency, or sandbox relaxation.

Evidence: **44 focused tests PASS** across pinch, renderers, workspace wrapper and
locale catalogs. TypeScript, strict checking of the standalone JS runtime,
scoped TypeScript lint, production build and profiling build PASS. Chromium real
touch dispatch verifies 1440px layout/desktop media queries remain constant while
paint width doubles to 2880px, one-finger panning moves both axes, pinch-out
returns to 1440px, toolbar remains Fit and outer visualViewport.scale stays 1.
Actual browser checks also verify restoration of authored !important scale/origin
styles, which the jsdom iframe style implementation cannot represent faithfully.
Both production and profiling bundles pass. Native iPhone Safari remains
unverified. Evidence: `/tmp/wc-html-pinch-{focused-final.log,typecheck.log,build-final.log,
runtime-typecheck.log,browser-proof.json,profile-browser-proof.json}` and
`/tmp/wc-html-pinch-{magnified,restored}.png`.

Scoped Test Genie unit run **20261016-142915-1ef25621 FAIL**, with one execution
error: the API Go command. No UI command failure is reported. This remains the
recorded API issue **knw-1792160172206667201**; no duplicate report was filed.
The shared worktree changed during the run, so it is not an exact-version
certificate for the final module extraction; final focused and browser checks
above explicitly cover that extraction. No W0–W2 or full-suite pass is claimed.

## 2026-10-16 — HTML drawer viewport and zoom controls (W3)

**Scoped repair complete.** Work shape: bounded task. The operator screenshot and
`TextRenderers.tsx` identified W3: a dark Preview/Source button row and a frame
fixed to drawer width. Target and composition: [Architecture](../concepts/ARCHITECTURE.md)
and [desktop/mobile mockup](../mockups/html-preview-toolbar.svg).

Bounded handoff goal: Work until the documented HTML toolbar behavior holds,
proven by focused renderer/locale tests, type-check/lint/build, and desktop/mobile
browser assertions. Keep adjacent failures outside the UI repair boundary.

Implemented: shared morphing ghost icon toggle, transparent toolbar, Responsive/
Phone (390px)/Tablet (768px)/Desktop (1440px), Fit and 10–200% zoom. Presets fit
immediately; manual zoom preserves CSS viewport width and permits horizontal
panning. Resize observation updates Fit; source switches preserve choices and a
new preview resets them. English/Arabic/Japanese labels are aligned. The existing
sandbox, asset restrictions, and source-first line/truncation behavior remain.

Evidence: **35 focused tests PASS** (renderer, workspace wrapper, locale parity).
Scoped ESLint, TypeScript, production build, and whitespace checks PASS. Chromium
at desktop and 390px/320px touch widths verifies native iframe width=1440,
desktop media-query matching while scaled to the available width, horizontal
panning at 100%, source/zoom preservation, no toolbar overflow, and 44px touch
targets. Native Safari is unverified. Evidence: `/tmp/wc-html-preview-focused-final.log`,
`/tmp/wc-html-preview-typecheck.log`, `/tmp/wc-html-preview-build-final.log`,
`/tmp/wc-html-preview-browser-extra-proof.json`, and
`/tmp/wc-html-preview-{mobile,mobile-narrow,desktop}.png`.

Scoped Test Genie unit run **20261016-140952-25784903 FAIL**: API Go command
failure and initial missing locale keys. The locale omission is repaired and
all seven catalog checks pass in the final focused cohort. The API root cause
is unknown: retained evidence contains a truncated passing-test tail plus package
FAIL. Reported as **knw-1792160172206667201**. The actual HTML report also depends
on unavailable external Mermaid code (**knw-1792159967422613968**). This is not
a scenario-wide unit certificate or W0–W2 audit. The final build is served by
the existing UI runtime. Documentation append through Knowledge Observatory
returned HTTP 400, so this record was added directly to the existing file.

## 2026-10-16 — Large video blocked inside HTML previews (W3)

**RESOLVED (bounded implementation repair):** The operator could see the new
Aquila film's poster but could not play its 33,949,998-byte MP4 in the report.
The HTML runtime applied its 16 MiB presentation-resource cap to media. A HEAD
request through the live UI proxy returned **413**; the direct blob transport
already supported large files. This localized mismatch justified W3 repair;
no missing product contract or traceability audit was inferred.

The competing proxy-buffering and codec hypotheses did not explain the 413:
the proxy pipes responses and the H264/AAC film plays after the runtime cap is
removed for allowed audio/video. Non-media resources keep the 16 MiB limit.
Both preview byte routes stream from disk with native HEAD/ranges. Their writes
renew the 150-second response deadline per chunk, so ongoing large transfers
are not cut off by a total-response budget. Session binding, directory/symlink
confinement, stale-file checks, sandbox policy, and preview expiry remain enforced.

Evidence: regressions first failed with 413 for every embedded media fixture
while the same direct blob tests passed. All 18 focused file-preview tests now
pass, including 5 GiB files, suffix/late ranges beyond 4 GiB, 416 responses,
non-media rejection, and a real HTTP stream spanning multiple write budgets.
The internal resolver/store suite also passed. Live Chrome verification used
the production HTML document helper, its `allow-scripts` sandbox, and the UI
proxy: the actual Aquila film played and sought to 40 seconds. A valid sparse
5,368,773,026-byte H264 MP4 played and sought to 4 seconds; Chrome fetched its
metadata at byte 5,368,741,888 with **206**. The sparse fixture contains a large
`free` box, not five gigabytes of encoded frames; only 69,632 disk bytes were
allocated. This proves browser/transport handling of large lengths and offsets.

Regression lesson: standalone report hosting bypassed this runtime limit.
Future report QA must include playback through web-console's preview route.
Retained limitation: preview IDs expire after 30 minutes; reopening resolves a
new ID. This repair removes the media size cap, not that separate expiry policy.

Evidence directory:
`~/.vrooli/plan-artifacts/efforts/product-video-research-2026-10-15/web-console-large-video-2026-10-16/`.
The broader API run found unrelated catalogue/documentation failures and a
server-construction fixture that accesses real host sessions/legacy storage.
Filed through `report-bug` as **knw-1792125249934722814**,
**knw-1792125250578899496**, and **knw-1792125249221022524**, respectively.
The current canonical live database is separate from the legacy path mentioned
by that test; no claim of live data loss is made.
Scoped Test Genie unit run **20261016-042955-006b9b41** returned **FAIL**, with
test-command failures and UI render-policy findings; it is not a
passing scenario-wide certificate. Final changed-code regressions and live
browser assertions above passed independently.

## 2026-10-15–16 — Long-session UI performance (W3)

Implemented and measured against the real 574-message heartbeat investigation.
Collapsed previews are bounded without losing full reader/copy/export text; row
measurement consumes delivered resize boxes; find/search avoids duplicate work;
one event-aware snippet handler replaces a per-event callback cache. RCL canvas
styles now use the existing LayerManager lifecycle flag instead of root `:has`
queries. Governed releases: LayerManager 2.0.2, BottomSheet 1.3.5, FullPageDrawer
1.4.5, ResponsiveDialog 1.3.5; adopted through Scenario Dependency Analyzer.

Desktop MessagesPane average commit: **3.1 → 2.4ms**. Mobile long tasks:
**1,186 → 581 / 612ms**; mounted transcript nodes **2,186 → 1,249**. Captures
match the session, final scroll position and 23,998-character reader content.
Final focused checks: **216 Web Console + 5 library tests PASS**. Native mobile
search/account canvas colors agree across html/body/panel; guarded navigation,
long-reader inspection and restored unsent draft produce no prohibited attempts
or page errors. Ordinary production runtime is restored and healthy.

Retained debt: scoped Performance phase **20261016-040746-e5211c49 FAIL** on
Lighthouse **49 < 75**. Cold loading remains expensive; the owner also ignores
the configured desktop measurement settings (**knw-1792123967704779496**).
Existing machine/desktop fixture type-check errors remain. Desktop voice toolbar
overlap filed as **knw-1792123204122572156**. No sessions were created, messages
sent, or conversations archived/deleted. See
[full evidence and limits](../perf/2026-10-15-long-session-performance.md).
Work record: **67b8f5d5-e3be-420c-861a-626815e05689**.

## 2026-10-15 — Keyboard spacing and launch continuity (W3)

Operator confirms the bottom-overlap repair is fixed on the phone and GCT
looks perfect. This follow-up is a bounded finish pass: remove surplus space
above the keyboard and make the initial launch, health check, lazy workspace,
and session hydration share the existing Aquila identity and canvas. No
artificial loading delay or new viewport detector belongs in Web Console.

Hypotheses: a keyboard that contracts layout and visual viewports together
escapes the shared detector's layout-minus-visual comparison; retained safe
padding and toolbar gutters waste space. During launch, viewport projection
starts only in lazy Workspace, while two independent plain loading views
paint slate-950 over a slate-900 body. The screenshot's transient bottom band
is consistent with that timing and color mismatch.

Target: the shared owner detects settled, focus-qualified layout contraction
as well as visual occlusion; keyboard-open chrome removes bottom safe padding
and tightens outer gutters without reducing key hit areas. One branded launch
composition is visible before JS and throughout hydration, fills the measured
canvas, respects reduced motion, and hands off immediately when ready. Native
iPhone launch images reuse the existing icon. Validate focused regressions,
slowed-loading captures, keyboard/rotation/hardware-focus cases, and scoped
UI-health. Native OS launch timing remains device-qualified evidence.

Implemented: published **useViewportEnvironment 1.1.2** through draft ownership
and `content-set`; refreshed Web Console through Scenario Dependency Analyzer.
`App` now publishes viewport state before asynchronous loading. The initial
HTML and both React loading boundaries use one shared launch HTML/CSS source,
with the existing Aquila icon, name, and indeterminate dots on **#0f172a**.
Native portrait/landscape startup images cover the 11 declared phone profiles
(22 PNGs, about 1.1 MiB). No artificial delay or brand replacement was added.

Evidence:

- The new resizes-content regression failed against 1.1.1 and passes against
  1.1.2. Shared viewport cohort: **20 PASS**, including unchanged drawable
  frame, visual-only keyboard occlusion, ordinary resize, rotation, desktop
  hardware focus, and layout contraction without editable focus.
- Web Console: **190 PASS** across eight focused files; focused new/changed
  launch/layout/App tests and sources pass ESLint. Production UI build and
  strict shared-package build PASS. Initial strict artifact:
  **1fee8a8ea21a564ae136b15374788b9954dfa7bcdcfdfa2e9019567ae330d8b9**.
- Browser staging held the entry script, health response, workspace chunk,
  and session list independently. Every stage retained three **852px** roots,
  the same **rgb(15,23,42)** html/body/launch canvas, a loaded mark, and no
  visible plain loading label. Captures: `/tmp/wc-launch-{before-javascript,
  health,workspace-chunk,session-hydration}.png`; measurements:
  `/tmp/wc-polish-browser-proof.json`.
- Mobile toolbar: **159px → 147px → 159px** across keyboard open/dismissal
  with unchanged key heights and all **16** control centers reachable. The
  keyboard-open safe-bottom variable is **0px**. The synthetic browser uses
  zero native insets; the previously retained device home inset is removed
  in addition to the measured **12px** gutter reduction.
- Desktop 1440×900 launch has one uniform canvas, all three progress
  animations disabled under reduced motion, and **22** startup links. Handoff
  removes every launch node immediately and retains a **900px** document.
  Screenshot: `/tmp/wc-launch-desktop.png`; measurements:
  `/tmp/wc-polish-desktop-proof.json`.
- Two fresh final review passes found no further material implementation
  findings: source/ownership/state restoration, then staged mobile/desktop
  rendering and accessibility/motion. Earlier finish findings (a possible
  4px adaptive-tint sliver and asymmetric native/HTML centering) were resolved.

Limits: native iPhone keyboard spacing and OS startup-image selection await
device feedback. Shared story run **ctr_4a0ba55b2ead3e46 FAIL**: default behavior,
source/closure/contract and default browser evidence pass, but inherited
boundary-story readiness and review-sheet screenshot failures remain, related
to **knw-1792082382843370162**. RCL preview UI rebuild, token check, all 180 pin
checks, and 12-route registration check pass. Test Genie UI-health
**20261015-174648-b421e805 FAIL** retains missing workspace/shell contract debt
and terminal visual findings; new findings filed as **knw-1792086641427937775**.
The previously reported unrelated machine/desktop fixture type failures remain.
Work record: **9f823026-98fa-419b-9e0d-19533bc242af**. Documentation CLI append
failure was separately captured as **knw-1792086733379798376**; canonical files
were updated directly to complete this work.

## 2026-10-15 — Mobile toolbar and full-page drawer bottom clipping

**Scoped W3 root-sizing repair built and served; native confirmation open.**
Operator screenshots `IMG_1656.jpeg` and
`IMG_1657.jpeg` show the toolbar's second button row and the account drawer
cut off at the same bottom edge after today's shared viewport changes.

Related prior work: Git Control Tower's 2026-10-15 drawable-frame repair in
its `docs/internal/PROBLEMS.md`, work record
`bcaf815a-11a4-4af9-8dd7-61fd0ccc5227`. That change published shared viewport
1.1.1 and overlay 1.4.5 and made Web Console consume `frameHeight`.

Hypotheses and checks:

- A foreground tint strip covers controls: inspect strip geometry and layer.
  The only local strip is 4px at z-index 0, inconsistent with the large band
  and the same clipping inside a modal.
- Web Console expanded its content without expanding the document canvas:
  measure all roots in the running browser. Before repair, at 393×852,
  `html` measures **0px** with `overflow:hidden`; body and React root measure
  **852px**. GCT explicitly sizes all three. The new root-height contract
  fails for html, body fallback, and React root before the CSS repair.
- Keyboard-dismissal state retained a short frame: exercise the measured
  793px layout / 852px frame through keyboard open and dismissal. The existing
  projection correctly restores 852px even with a stale visual height; this
  hypothesis does not explain the missing root sizing.

Target: html, body, React root, and workspace fill the same shared measured
frame; modal portals use the library's geometry and layer; bottom safe-area
padding is reserved inside the toolbar or drawer once. Validate focused root,
viewport, settings, and toolbar regressions plus browser geometry/hit tests
and the scoped Test Genie UI phase. Native iPhone pixels remain unverified
until actual-device confirmation; Chromium geometry is narrower evidence.

Bounded task handoff: finish this target inside Web Console's UI and existing
documentation, retain failing/passing proof here, and report any unrelated
failures without broadening the repair. No new viewport detector or overlay
stack is needed.

Repair: size html, body, and React root to `--wc-app-height` together, with
the same `100dvh` fallback as the workspace. Preserve the shared detector,
overlay layering, keyboard projection, internal safe-area padding, and tint
strip behavior. No library release or copied viewport detector was needed.

Verification:

- Focused viewport/overlay/settings/toolbar cohort: **92 PASS** across five
  files; changed-test ESLint PASS; production build PASS and served directly
  by the lifecycle-managed UI server.
- Live Chromium at 393×852: all roots **852px**; all **16** toolbar control
  centers pass hit testing, with the lowest controls ending at **846px**.
  Account drawer and scroller end at **852px**; the lower OpenRouter input
  scrolls into view and passes hit testing.
- Synthetic installed-app geometry: reported layout **793px**, drawable
  frame **852px**, all roots and drawer **852px**. Established keyboard
  shrinks all roots to **500px**; dismissal restores **852px** even while
  the visual measurement stays **500px**. Evidence retained during the run
  at `/tmp/wc-bottom-frame-browser.json`; this is Chromium, not native pixels.
- Test Genie **20261015-172628-8f2c9d62 FAIL**: missing declared
  `workspace-shell` experience surface and shell manifest; shared adoption
  warnings; native-screen/safe-area qualification absent. These findings
  do not establish a browser clipping regression. Contract debt filed as
  **knw-1792085363990345979** through report-bug.
- Type-check still fails in unrelated MachineList and desktopTransport test
  fixtures, already filed as **knw-1792081948624493753** by the prior work.

The root-height mismatch is measured and repaired. Whether this fully removes
the photographed iPhone band remains unverified until actual-device feedback.
Work record: **13d7339b-3136-4157-968b-68257548802f**.

## Work ladder

- Rung: W1
- Evidence: `business-health validate scenario web-console` reports `business_status_unearned` for eight requirements declared complete while no requirements-sync snapshot exists.
- Blocker: None; the unearned declarations were demoted to `in_progress` while their validation references remain intact so a future comprehensive run can earn completion.
- Measured: 2026-09-04

## 2026-09-16 — Stale keyboard-shrunken viewport after dismissal

**RESOLVED, scoped W3 UI repair:** On mobile WebKit, the terminal shell could
remain constrained to the upper portion of the screen after the virtual
keyboard was dismissed. The reported screenshots showed the terminal and its
toolbar ending halfway down the device, with the remaining screen painted
black.

**Root cause:** `useAppViewport` projected `visualViewport.height` into the
shell height even when `useViewportEnvironment` had not positively established
that a keyboard was visible. WebKit can retain a keyboard-shrunken visual
viewport during dismissal or until another interaction, so the shell treated a
stale transient measurement as its permanent height.

**Fix and prevention:** The shell now uses the stable layout viewport height
unless `keyboardVisible` is true, and uses the visual height only for an
established keyboard. A regression test covers a shrunken visual viewport with
the keyboard closed. Focused viewport tests, type-check, and production build
pass.

## 2026-09-12 — Session get manifest positional mismatch

**RESOLVED, scoped W3 CLI repair:** `web-console session get
e29964c2-1c16-4ee1-af25-1eb7620c7854 --json` panicked before the owner read because
the embedded canonical manifest named its required argument `sessionId`, while
`domains/session/handlers.go::get` requested `session-id`. The handler-only test
constructed its own matching `ArgSchema`, and the registration test checked only
that commands existed, so neither exercised the failing production route.

Changed only the `session get` positional name in `cli/manifest.json` to
`session-id`; its required status, read governance and protobuf binding to `id`
remain intact. No handler fallback, generated schema or shared parser change.
The CLI reference now identifies manifest versus registration ownership.

`TestSessionGetRoutesEmbeddedManifest` uses `NewApp`, the actual embedded
manifest/parser/dispatcher, and a test-only Connect server. It reproduced the
exact panic before the manifest change. Afterward, its five cases cover get JSON,
the show alias, human output, a missing required ID without an owner Get, and an
owner not-found response without success output. Exact request identity and call
counts are checked. `go test -race . ./domains/session -count=1 -timeout=60s`
passed in `cli/` (2.567s and 1.034s); scoped `git diff --check` passed.

The exact live command then exited zero with the same session ID, persistent
backend, owner `agent-manager`, and label `swarm-execution-a91b2b02a04c3bbb`.
The standard launcher rebuilt the CLI from the updated embedded manifest before
retrying the read. No terminal input, session mutation, recovery, service restart
or broad test admission occurred. Metadata observation does not establish that
the interactive run is fresh, progressing or complete.

## 1. Interactive CLI Fidelity Edge Cases

PTY handling for complex interactive CLIs (Claude Code, Codex) may have edge cases around resize during active output, cursor reporting conflicts, and reconnect during mid-escape-sequence. Requires dedicated e2e validation.

## 2. Mobile Browser Variability

Floating keyboard toolbar behavior varies across mobile browsers (Safari iOS, Chrome Android, Firefox Mobile). Virtual keyboard interaction with xterm.js focus management needs testing on real devices.

## 3. SQLite Concurrency Under Load

While single-user bounds concurrency, rapid terminal output from multiple panes could create write contention on the SQLite transcript table. WAL mode mitigates but may need monitoring.

## 4. AI Provider Timeout and Summarize Model Selection

**PARTIALLY RESOLVED** (2026-05-17): The TTS summarize path now separates
audio-tools unreachable, deadline exceeded, and selected Ollama model missing.
Model selection is exposed in Voice Output settings through a catalog returned
by audio-tools. The catalog marks recommended, installed, missing,
default-eligible, and reasoning models, and reasoning models are excluded from
default selection.

Current default: `llama3.2:3b`, because it is installed locally and was
validated as a fast non-reasoning fallback. Research-backed candidates
(`gemma3:4b`, `gemma3n:e2b`, `phi4-mini:3.8b`) remain uninstalled on the
target machine as of 2026-05-17, so they are visible as recommended missing
models with `ollama pull` commands instead of being silently selected.

Remaining gap: run a local benchmark after explicitly installing newer
candidates. The default should change only if a candidate beats
`llama3.2:3b` on latency and preserves summary quality without visible
reasoning/prompt leakage.

## 5. Standards: Setup Steps Configuration (MEDIUM)

**RESOLVED** (2026-02-19): Simplified setup steps in service.json (removed conditional wrappers). Auditor now reports 0 standards violations.

## 6. Go Lint: Remaining errcheck Warnings

**RESOLVED** (2026-02-19): Fixed the 2 remaining errcheck warnings in `session_test.go` by wrapping deferred Delete calls in `func() { _ = sm.Delete(id) }()`.

## 7. Lighthouse SEO: 82% (Below 90% Threshold)

**RESOLVED** (2026-02-19): Added meta description to index.html and robots.txt. Lighthouse SEO now 100%.

## 8. Offline Output Buffer Growth

Sessions with default never-expire policy could accumulate unbounded transcript data in SQLite. May need eventual transcript rotation or archival strategy (deferred — not MVP).

## 8a. Web Mic Lifecycle Wedge and iOS/WebKit Residual (2026-07-02)

**RESOLVED for web-console-owned JS lifecycle drift:** server-final, user stop,
auto-stop, error, and page-lifecycle stop paths now converge on the same capture
teardown path, and production `getUserMedia` calls are forced through the mic
ownership registry. Persisted `lowLatencyVoice` / `wakeWordEnabled` flags no
longer acquire the mic on mount or tab-visible; prewarm/passive arming requires
explicit mic-control intent. Tests cover the old server-final-before-client-stop
wedge, stale-lease self-heal, no eager acquisition, and the structural
`getUserMedia` boundary.

**Residual platform limitation we cannot fix from web JS:** if iOS/WebKit fully
freezes or kills the page's JavaScript before `visibilitychange: hidden`,
`pagehide`, or `freeze` is delivered, no application code can run
`MediaStreamTrack.stop()`. In that narrow case the OS mic indicator may remain
active until the OS reclaims the track. The mitigation is to release on the
earliest delivered lifecycle signal and cancel in-flight starts; a native shell
would only matter for this hard-freeze class, not for the now-fixed web-console
state-machine wedge.

## 8b. Idle audio-session activation (background-audio interruption + wedge prevention) (2026-07-07)

**Symptom:** on iOS, opening web-console (HTTPS PWA) stopped other apps' audio
(Spotify/YouTube) — but only after the first touch/scroll, with low-latency/keep-
alive off. Separately, the mic could get wedged until a device reboot (survives
PWA force-quit — the platform residual in §8a).

**Root cause (the preventable part):** we activated the iOS `AVAudioSession` when
we did not need to, and never released it.
- `ensureAudioContextOnGesture()` installed a document-wide capture-phase
  `pointerdown`/`keydown` listener on mount and, on the **first interaction of any
  kind**, created+`resume()`d the shared AudioContext — purely to shave ~20-50ms
  off a *possible future* mic press. Resuming a WebAudio context activates the iOS
  audio session and interrupts other apps' non-mixable audio, even when the user
  never used voice.
- The context was app-lifetime and **never suspended**, so it held the session
  indefinitely.
- Low-latency voice mode (`lowLatencyVoice` + `micReadiness` pre-warm) similarly
  held a `getUserMedia` stream open while idle.

**Fix (prevention, not recovery):**
- Removed eager `ensureAudioContextOnGesture`; the AudioContext is resumed lazily
  inside the real voice/cue gesture and **suspended when idle** — on background
  (`visibilitychange:hidden`) and ~1.5s after a capture turn ends
  (`armIdleSuspend`/`keepAudioContextAwake`/`suspendSharedAudioContext` in
  `sharedAudioContext.ts`).
- Removed low-latency voice mode entirely (setting, `micReadiness` module,
  `retainStream`, injected pre-warmed streams). Providers always acquire and own a
  fresh mic stream per turn and release it fully on stop.
- Result: the audio session is active only while actually capturing or cueing, so
  background audio is no longer interrupted, and the unnecessary session churn
  that plausibly fed the §8a wedge is gone.

**Second offender (found after the first fix shipped): TTS "unlock on any gesture".**
`useTextToSpeechCore` installed persistent global `pointerdown`/`keydown`/
`touchstart` listeners that called `KokoroProvider.unlock()` — a **silent
`HTMLAudioElement.play()`** — whenever the provider reported not-yet-unlocked. On
iOS, playing an audio element (even a muted silent one) activates AVAudioSession
and ducks other apps' audio. Because the provider is rebuilt whenever the TTS
backend re-resolves (`unlocked` resets to `false`), the persistent listener
re-ran the silent play on essentially every interaction — the "after a while,
web-console pauses my music on any tap, and only a force-quit fixes it" report
(force-quit destroys the primed audio element). **Fix:** removed the preemptive
unlock from the gesture handler (it still flips the no-audio `browserAudioReady`
flag). The media element is now unlocked lazily — a real speak in a user gesture
unlocks it naturally, and a programmatic speak blocked by autoplay surfaces
`needsUnlock` + the Enable-Audio affordance (a deliberate tap → `unlock(true)`).

Together: the app now activates the iOS audio session **only while actually
capturing or playing speech**, never on incidental interaction.

This cannot *recover* an already-wedged mic (still a platform limitation, §8a),
but it removes the app behaviours that put the session into that state. See
[VOICE-LATENCY.md §3 (AudioContext lifecycle) + §5 (low-latency removed)].

## 8c. Mic auto-stops mid-sentence under kyutai — client RMS VAD is sole authority (2026-07-08, under triage)

**Symptom:** during *continuous* dictation the microphone abruptly stops ~10 s in
(before any real timeout) and the rest of the utterance is lost. This surfaced
right after the streaming backpressure wedge was fixed (audio-tools `PROBLEMS.md`
2026-07-08): the fix synced mic-state to actual transcription, so a premature
turn-end is now *visible* instead of masked by a frozen-but-still-green mic.

**Root cause (diagnosed):** the live STT engine is **kyutai/passthrough**, which
emits **no `vad-state`** events (unlike whisper-VADSegment). So on the client the
auto-stop SSOT `decideAutoStop` (`hooks/voice/autoStopDecision.ts`) has no server
tick (`serverVad.receivedAt === 0`) and falls to **§2: the client RMS VAD is the
SOLE turn-ending authority**. Under whisper the server VAD was authoritative and
vetoed the client VAD's known "stops while I'm still talking" false positives
(both signals had to agree); under kyutai that veto is gone. A client-VAD silence
verdict (`vadTick` → `stop` after `silenceTimeoutMs` of RMS below `speechThreshold`)
then ends the *whole turn* whenever the analyser reads silence — which happens when
the audio source stops delivering samples even though the user is still speaking:
  1. **Muted mic track** — the OS/browser mutes the track on sleep/wake, a
     default-input-device change, or another app seizing the mic. The `ended`
     event does NOT fire for muting (`streamHealth.isTrackUsable`), so no samples
     flow yet the track stays "live" and the analyser reads pure silence.
  2. **Suspended AudioContext** — if the shared context suspends mid-capture
     (`ctx.state !== "running"`), the analyser returns flat/silent data.
  3. **Threshold miscalibration** — the adaptive `speechThreshold` (= noise floor
     × 3) rising above the user's real speech RMS in a noisy room.

Relay evidence: sessions close `graceful=true tailFinalDelivered=true` in a
regular ~10–12 s cadence — i.e. **client-initiated** stops, not backend kills. The
frequent kyutai `reaping wedged prior streaming session` warnings are a *separate*
benign-but-noisy lock-release-latency issue (the relay dials kyutai and takes
`MODEL.lock` the instant the browser WS connects — including the mount-time
pre-connect — so overlapping/rapid sessions reap already-delivered holders; NOT
the cause of the mic stop; tracked as a follow-up).

**First changes shipped (2026-07-08):**
- **Instrumentation to make the next repro conclusive** (`useVoiceCore.ts`): every
  VAD/auto-stop log now carries `rms / trackAlive / trackMuted / ctx.state`; an
  `audio-source WENT SILENT / RECOVERED` line fires the instant the source stops
  delivering; `provider.onError` logs its triggering reason; `endCapture` logs its
  `reason`.
- **Honest mic-source handling** (`VoiceStreamProvider.attachCaptureLifecycleHandlers`
  + `MediaRecorder.onerror`): a mid-capture `mute` surfaces a `mic_muted` status
  (transient, turn kept open); a terminal `ended` recovers the retained audio via
  HTTP fallback instead of drifting into a silent VAD stop; an encoder error is
  surfaced. Covered by `VoiceStreamProvider.trackLifecycle.test.ts`.
- **Guard**: a client-VAD (`client-fallback`) auto-stop is **suppressed when the
  audio source is not delivering samples** (muted / ctx not running) — a silence
  verdict on a non-delivering source is an artifact, not a real pause; a suspended
  context is resumed. Terminal loss is still owned by the track handlers +
  no-audio watchdog.

**To close (needs one real-device repro with DevTools console open):** dictate
continuously ≥30 s until the mic stops, then read the console. The decisive line
is the `VAD client-stop` / `auto-stop` at the stop instant:
  - `trackMuted=true` or `ctx=suspended` → the source went silent (mute/suspend);
    the guard now keeps the turn alive — confirm dictation resumes on unmute.
  - `trackAlive=true trackMuted=false ctx=running` with `rms` *below* `speechThresh`
    → threshold miscalibration; the durable fix is to give kyutai a server-VAD (so
    the client isn't the sole authority) or make the sole-authority client VAD more
    conservative (longer sustained-silence requirement / segment-boundary-continue
    instead of hard turn-stop). Do NOT blind-tune thresholds without this line.

See audio-tools `docs/internal/PROBLEMS.md` (2026-07-08 wedge entry + follow-up).

## 8d. Mic won't start — "could not determine audio codec" (2026-07-08, FIXED + live-proven)

**Symptom:** the mic fails to start with `provider.onError → ending turn:
audio-tools/audioformat: could not determine audio codec (declare input_format)`.
Intermittent at first, then persistent (every session `segments=0` in the relay
log).

**Root cause:** the streaming path relied on the backend **sniffing** the codec
from the first audio frame (`buildVoiceStreamWsUrl` declared no `format`). The
backend's `audioformat.Detect` needs the first chunk's leading bytes to be the
WebM/EBML header (`0x1A45DFA3`) — the header MediaRecorder emits in its FIRST
`ondataavailable` chunk. But `VoiceStreamProvider.ondataavailable` sent each
chunk via `e.data.arrayBuffer().then(send)` — **independent async promises that
can resolve out of order**. The header chunk is the largest, so its
`arrayBuffer()` can consistently resolve *after* the next chunk once timing
shifts (GC/load) — putting a headerless frame on the wire first → sniff fails.
Because it's timing-driven it presents as "worked, then persistently broke."

**Fix (two parts, both shipped + live-proven against the running backend):**
1. **Ordered delivery** (`VoiceStreamProvider`): send the Blob directly and
   synchronously in `ondataavailable` order (`enqueueOutbound`/`flushPending`,
   `pendingChunks: Blob[]`) — no async `arrayBuffer()` that can reorder. The
   header chunk is now always the first frame on the wire. Regression test in
   `VoiceStreamProvider.trackLifecycle.test.ts` ("sends audio chunks in recording
   order as raw Blobs").
2. **Declare-first** (`buildVoiceStreamWsUrl`): declare `format=webm` (the
   web-console always records WebM/Opus) so the backend skips the fragile sniff
   entirely — defense-in-depth if any frame is ever reordered/dropped.

**Live proof (synthetic client → audio-tools :19630):** no-format + headerless
first chunk → `could not determine audio codec` (reproduces the bug); `format=webm`
+ headerless first chunk → no codec error (declare-first bypasses sniff);
`format=webm` + header-first → clean `status`, stream processes.

## 8e. Mic stays live but transcription stalls mid-session (2026-07-08, instrumented, under triage)

**Symptom:** the original wedge shape — after a while (tens of seconds) the mic
stays active and audio keeps streaming, but new transcription stops appearing.
Confirmed from a real console log: a ~110s session delivered ~17 segments then
`Final received: 0 chars`; the WS closed only at the *end* (after the final), so
it is **not** a kyutai reap (a reap closes the connection mid-session → client
reconnect, which did not happen).

**What it is NOT:** the send-side backpressure wedge (fixed 2026-07-08 — kyutai's
decode enqueues to a non-blocking send worker; verified) and not the codec bug
(§8d, fixed). Kyutai's reap warnings are a *separate* lock-release-latency issue
that hits short overlapping sessions (0-segment closes), not this stall.

**Leading hypothesis:** kyutai's streaming LM context saturating over a long
continuous session. `server.py` force-commits at `MAX_SEGMENT_FRAMES`, so
continuous speech normally keeps committing; a stall where partials AND segments
both stop, correlated with session *duration* (not audio content), points to the
streaming transformer degrading to padding-only output once its context window
fills. To be confirmed by the new instrumentation before any server-side change.

**Instrumentation shipped to make the next repro conclusive:**
- `VoiceStreamProvider` transcription-stall watchdog: logs `⚠ transcription STALL:
  no partial/segment for <gap>ms while still recording (chunks/bytes/segments)`
  whenever the backend stops emitting for >4s while audio keeps streaming.
- `segment-final #N at +Tms (gap=…): "…"` logged on every committed segment (so
  the exact time segments stop is visible), plus per-partial event tracking.
- **Also fixed:** all voice console logs used printf `%.4f` — NOT a browser
  console specifier, so every such line rendered with its args shifted by one
  (e.g. `trackAlive=running, trackMuted=true` was really ctx/trackAlive). Now
  template literals. Prior logs from before this fix must be read de-shifted.

**To close:** dictate continuously until it stalls; the console will show the last
`segment-final` time and the `⚠ transcription STALL` gap. If segments stop at a
consistent *elapsed time* regardless of speech, it confirms context saturation →
fix in kyutai `server.py` (reset/rotate the LM streaming state on a bounded
window). If it correlates with audio, look at the VAD/commit path instead.

## 8f. Mid-recording empty final masked backend loss (2026-07-09, fixed in working tree)

**Symptom:** the user could keep recording while the backend stream died, then the
client accepted a `final` with empty text as successful completion. `onclose`
then skipped reconnect/fallback because `finalReceived=true`, silently dropping
retained audio.

**Fix:** `VoiceStreamProvider` now treats any `final` received while
`MediaRecorder.state === "recording"` and the stop was not intentional as backend
loss: it does not set `finalReceived`, does not call `onResult`, closes the WS,
and lets the existing reconnect-then-HTTP-fallback path recover retained audio.
The kyutai relay also preserves typed busy as `stt_busy`, which the client
surfaces as a visible busy status.

## 9. E2E Issues

**PARTIALLY RESOLVED** (2026-02-19): Added BAS workflows for terminal command execution, route-level session persistence, reconnect replay, and multi-pane independence:
- `bas/cases/01-foundation/01-terminal/launch-custom-command-executes.json`
- `bas/cases/01-foundation/01-terminal/session-metadata-persists-across-route-navigation.json`
- `bas/cases/01-foundation/01-terminal/reconnect-offline-buffer-replay.json`
- `bas/cases/01-foundation/01-terminal/multi-pane-independent-io.json`
- `bas/cases/01-foundation/01-terminal/interactive-stdin-roundtrip.json`
- `bas/cases/01-foundation/01-terminal/session-persists-across-full-reload.json`

**Remaining gaps:**
- No BAS mobile viewport workflow yet for floating toolbar key/chord behavior.
- Playbooks phase currently blocks on BAS startup when browser-automation-studio dependencies are unavailable (e2e workflows present but cannot execute).

**Added recovery coverage (2026-07-12):**
`bas/cases/02-messages/02-voice/deterministic-incomplete-coverage.json`
executed successfully as `06696480-3902-47e3-8db6-8e7971ef42e0`. It grants the
WAV-backed fake microphone to the real Web Console UI, crosses its same-origin
voice WebSocket proxy, injects a server close only after the first PCM chunk,
and asserts the visible `incomplete_coverage` error plus metadata-only
diagnostic export. The Audio Tools fault gate was enabled only for this run and
restored to false afterward. This is deterministic desktop-browser evidence,
not a substitute for the remaining iOS/Android device matrix.

## 9a. File Preview — Intentionally Deferred (2026-06-30)

The file-preview subsystem (`api/internal/filepreview`, `FilePreviewService`, blob/range route, UI renderer registry) ships these deliberate deferrals — all are working-as-intended, not bugs:

- **PDF.js deferred.** PDFs render via the native browser viewer (`<iframe>` at the blob href) with a download/open fallback. PDF.js is only worth adopting if native rendering proves inadequate in validation across target browsers.
- **No media transcoding.** Audio/video play through native browser codecs only. Unsupported codecs (`.mov`, `.flac`, …) show a clear "your browser may not support this format" hint plus download — web-console does not transcode.
- **CLI is metadata/text only.** `web-console file-preview resolve` and `file-preview text` cover programmatic metadata + bounded text. Blob streaming/download has no CLI command (no clear operator workflow yet); it stays UI/browser-only because the blob route is consumed by native media elements.
- **In-memory preview-id store.** Preview ids live in a process-local store with a 30m TTL; they do not survive an API restart. That is acceptable for the single-operator, reopen-on-demand UX. A persistent store would only matter for long-lived shareable preview links, which are out of scope.

## 9b. File Upload — Policy and Deferred (2026-09-18)

The session upload endpoint (`POST /api/v1/sessions/{id}/upload`, `api/upload_handler.go`) accepts any non-executable file, not just images. The composer exposes a **Camera / Photos / Files** source menu; the terminal pane accepts image paste, drag-drop of any non-executable file, and a general file input.

**Policy:**

- The file part is streamed straight to disk via `r.MultipartReader()`, not buffered by `ParseMultipartForm`, so large video does not sit in memory.
- Native executables are refused twice: by extension (`.exe`, `.dll`, `.so`, `.dylib`, `.msi`, `.jar`, …) and by leading magic bytes (PE/`MZ`, ELF, Mach-O, `cafebabe`), so renaming does not launder a binary.
- Size caps are per category: images 25 MiB, video 512 MiB, audio 128 MiB, PDF 64 MiB, archives 128 MiB, text/code 25 MiB, other 64 MiB. A 513 MiB request-body ceiling backstops the lot.
- Bytes land in the cache-class `uploads/<sessionID>/` root and are removed when the PTY exits or the session is terminated.

**Deferred (working-as-intended, not bugs):**

- **Ephemeral storage.** Uploads are cache-class and deleted with the session; they are not durable evidence and have no metadata row. Durable attachments would move to a data-class root or MinIO and add an `AttachmentMeta` row.
- **No resumable/chunked upload.** One `multipart/form-data` request, no retry/resume, so a large video depends on the connection holding. A chunked/`tus` path or presigned direct-to-object-storage upload is the upgrade.
- **No media understanding.** Uploading video/audio only puts a path in the workspace; transcription or keyframe extraction are separate resource-level capabilities (whisper/ffmpeg), not upload concerns.

## 10. Audio Extraction Prep — Deferred Sub-Phases (2026-05-16)

Active initiative: `swarm-manager/initiatives/continuous-audio-platform`. The
plan at `/home/matthalloran8/.vrooli/plans/web-console-audio-extraction-prep-for-audio-tools.md`
defines 8 phases. The 2026-05-16 implementation pass completed:

- **Phase 0** (baseline) — focused voice/TTS tests green.
- **Phase 2 (partial)** — `NormalizeTextForSpeech`, `SplitIntoSpeechParagraphs`,
  `TTSMaxChunkLength` and their tests now live in `api/internal/tts`. The
  `inttts.*` qualifier is the new call shape for `package main` consumers
  (`conversation_router.go`, `conversation_adapter.go`, `conversation_store.go`,
  `tts_summarization_service.go`).
- **Phase 3** — voice domain dependency direction inverted. Handler types,
  Service interface, Backend interface, and the production Adapter all live in
  `api/internal/voice` (`types.go`, `handler_adapter.go`). `handlers/voice/module.go`
  is now a thin re-export shim; `handlers/voice/adapter.go` deleted. Exit
  criterion verified: `rg 'web-console/handlers/voice|voiceH\\.' internal/voice
  --type go` → zero hits.
- **Phase 5 (partial)** — `TestGreenfield_TTSReusableCoreNotInPackageMain` and
  `TestGreenfield_InternalAudioDomainsDoNotImportHandlers` added to
  `api/greenfield_assertions_test.go`. They lock the boundary against
  regression.

The following sub-phases are deferred and **must be completed before the
`scenarios/audio-tools` greenfield item can land cleanly**:

- **Phase 2 (remaining) — DONE 2026-05-16.** All reusable TTS core types and
  functions have moved into `api/internal/tts`:
  - `Cache`, `CacheKey`, `CacheEntry`, `CacheStats`, `NewCache`,
    `cacheKeyHash` live in `internal/tts/cache.go`.
  - `Summarizer`, `SummarizerResponse`, `NewSummarizer`, `StripThinkTags`,
    `summarizeSystemPrompts`, `summarizeTokenBudget` live in
    `internal/tts/summarizer.go`.
  - `SummarizationService`, `SummarizeRequest`, `SummarizeResult`,
    `NewSummarizationService`, sentinel errors `ErrSummarizeCoolingDown`,
    `ErrSummarizeBudgetInThink`, `ErrSummarizeTruncated`,
    `ErrSummarizeEmptyAfterStrip`, `ErrSummarizeTrulyEmpty`,
    `SummarizeErrorMessage`, `classifyEmptySummary` live in
    `internal/tts/summarization_service.go`. The previous
    `audioports.SpeechTextProcessor` indirection inside the service is gone;
    the service calls `NormalizeTextForSpeech` / `SplitIntoSpeechParagraphs`
    directly (same package — no port needed). The port abstraction at the
    consumer (conversation/store) layer is unchanged.
  - `DefaultConfig`, `ConfigPatch.Apply`, `LoadConfig`, `SaveConfig` live in
    `internal/tts/config.go` (the `Config` and `ConfigPatch` types
    pre-existed in `types.go`).
  - `DefaultSummarizeConfig`, `SummarizeConfigPatch.Apply`,
    `LoadSummarizeConfig`, `SaveSummarizeConfig`,
    `MinSummarizeTimeoutSeconds`, `DefaultSummarizeTimeoutSeconds`,
    `MaxSummarizeTimeoutSeconds` live in `internal/tts/summarize_config.go`.
  - Pure-logic tests moved alongside: `cache_test.go`, `config_test.go`,
    `summarize_config_test.go`, `summarizer_test.go`,
    `summarization_service_test.go` all under `internal/tts`.
  - Server-tied glue stays in `package main`: `tts_cache.go` keeps
    `invalidateTTSCacheForEvent`, `preSynthesizeTTS`,
    `synthesizeParagraphs`; `tts_config.go` keeps `getTTSConfig`/
    `setTTSConfig`/`getTTSSummarizeConfig`/`setTTSSummarizeConfig`/
    `resolveTTSSummarizeConfigPath`. The `Server` fields now type-hold
    `inttts.Config`, `inttts.SummarizeConfig`, `inttts.Cache`,
    `inttts.Summarizer`, `inttts.SummarizationService` directly — no
    aliases.
  - `greenfield_assertions_test.go::TestGreenfield_TTSReusableCoreNotInPackageMain`
    locks the boundary against regression for the moved files and the
    type/function definitions.
- **Phase 4 (extension) — DONE 2026-05-16.** `LocalSpeechToText` and
  `LocalTextToSpeech` adapters now live under `internal/audioports/`
  (`local_stt.go`, `local_tts.go`) and are wired by `NewServer` into
  `Server.sttPort` / `Server.ttsPort`. `SetSpeechToText` / `SetTextToSpeech`
  setters let tests substitute fakes. Orchestration callsites in
  `tts_adapter.go` (Synthesize / ListVoices / GetCache) and
  `tts_cache.go::synthesizeParagraphs` / `preSynthesizeTTS` route through
  `s.ttsPort`; cache writes in the on-demand path still go via
  `s.ttsCache.Put` because the conversation-event-keyed cache is web-console
  glue (per the Phase 8 dossier). The greenfield assertion
  `TestGreenfield_OrchestrationGoesThroughAudioPorts` in
  `greenfield_assertions_test.go` locks the boundary: package-main files
  (other than `main.go` which constructs the local adapters) must not
  reference `s.voiceService.Transcribe`, `s.ttsSynthesizer.Synthesize`,
  `s.ttsVoiceLister.ListVoices`, `internal/voice.Service.Transcribe`,
  `internal/tts.KokoroSynthesizer`, or `internal/tts.KokoroVoiceLister`. The
  summarize pipeline (`SummarizationService`, `SummarizeRequest`,
  `SummarizeResult`, `SummarizeErrorMessage`, `ErrSummarizeCoolingDown`) is
  explicitly NOT routed through audioports in this pass — that port shape
  remains an open audio-tools-shared-contract question (see Phase 8
  dossier). Also resolved the orchestration-level double-normalize wart:
  `conversation_router.go::asyncSummarizeAndNotify` and
  `conversation_adapter.go::SummarizeEvent` now pass `event.Text` directly
  to `ttsSummarization.Summarize`, since normalization is a property of the
  summarize pipeline (handled inside `SummarizationService.run`).
- **Phase 6 (extension) — DONE 2026-05-16.** All UI consumers now import
  audio capability surface through `ui/src/domains/audio` rather than
  reaching directly into `hooks/voice/**` / `hooks/tts/**`. 16 consumer
  files migrated (Workspace, TerminalPane, WorkspacePaneShell, MessagesPane,
  MobileToolbar, VoiceRejectionBanner, VoiceCommandSuggestion, AudioPlayerBar,
  tts/AudioSettingsContent, settings/VoiceInputSection, api/voice.ts,
  domains/tts-playback/types.ts, and five `__tests__/*` consumer tests).
  Added re-exports to `ui/src/domains/audio/index.ts`: `CommandSuggestion`,
  `VoiceRejection` (voice types), `TTSPlaybackState`, `TTSPlaybackCapabilities`
  (tts types), and the reusable wake-word surface (`createWakeWordEngine`,
  `MIN_ENROLLMENT_SAMPLES`, `MAX_ENROLLMENT_SAMPLES`, `AudioFeatures`,
  `WakeWordTemplate`, `useWakeWordTest`). Web-console-specific paths
  (`hooks/voice/commands`, `commandParser`, `audioCues`, `activity`,
  `useVoiceInput`, `useTextToSpeech`) deliberately stay direct per README
  classification. Boundary locked by
  `ui/src/__tests__/audio-boundary.test.ts`, which walks `src/**/*.{ts,tsx}`,
  skips the adoption surface / underlying hooks / test files, and flags any
  `from "...hooks/voice/...|hooks/tts/..."` import outside the documented
  web-console-specific allowlist. Verification: `pnpm type-check`,
  `pnpm test -- --run` (116/116 batches pass, including the new boundary
  test), and `pnpm build` all clean.
- **Phase 7 — DONE 2026-05-16.** `ARCHITECTURE.md` now has an "Audio
  Ownership Map" section with backend/frontend ownership tables, the
  Connected Scenarios Registry contract, and the capability-port adoption
  seam description. DOMAINS.md and SEAMS.md already covered earlier in the
  pass.
- **Connected scenarios registry — DONE 2026-05-16.** `audio-tools` is now
  declared in `capabilities.Known` with `DependencyKind: DependencyScenario`;
  `ScenarioChecker` probes via `vrooli scenario status <slug>`; the
  Integrations settings tab renders a "Connected Scenarios" subsection
  separate from "Local Resources". When `scenarios/audio-tools` ships, no
  registry change is required — the existing checker will start returning
  `available` automatically.
- **Phase 8 — DOSSIER BELOW 2026-05-16.** Source dossier captured inline (next
  subsection) since in-repo edits to
  `swarm-manager/execute/audio-tools-greenfield-scenario/plan.md` were not
  authorized in this pass. Promote into that plan when the audio-tools
  greenfield item is picked up.

### Phase 8 Dossier — input for `audio-tools-greenfield-scenario`

**Source inventory — files to port into audio-tools (reusable capability core):**

Backend:
- `api/internal/voice/transcribe.go` — Whisper HTTP proxy, language passthrough, hallucination filter (deduplicate identical repeated outputs).
- `api/internal/voice/stream_ws.go` — streaming transcription, partials, segment finals, final retranscription, cancellation, VAD boundaries.
- `api/internal/voice/speaker.go` + `speaker_client.go` + `speaker_config.go` — speaker verification accept/reject/advisory/fallback decisions.
- `api/internal/voice/wakeword.go`, `config.go` — wake-word config and voice config schema.
- `api/internal/audio/transcode.go` — ffmpeg-backed transcode primitives.
- `api/internal/tts/` — entire package: `kokoro_synthesize.go`, `kokoro_voices.go`, `normalizer.go`, `chunker.go`, `service.go`, `types.go`.
- `api/tts_cache.go` — `TTSCache`, `TTSCacheKey`, `cacheKeyHash`, LRU + eviction (event-keyed `Evict()` is a thin convenience over key prefix — provider-level cache is the reusable part).
- `api/tts_summarizer.go` — Ollama-backed `TTSSummarizer`, `summarizeSystemPrompts`, `summarizeTokenBudget`, `stripThinkTags`.
- `api/tts_summarization_service.go` — `TTSSummarizationService` with cooldown, inflight dedupe, empty-summary classification, timeout/error sentinels.
- `api/tts_config.go`, `api/tts_summarize_config.go` — config schemas + load/save (atomic JSON write).

Frontend:
- `ui/src/hooks/voice/**` — capture readiness, VAD utilities, provider mechanics (Whisper/VoiceStream/WebSpeech).
- `ui/src/hooks/tts/**` — Kokoro provider, browser TTS provider, cache playback controls, unlock primitive.
- `ui/src/hooks/useVoiceInput.ts`, `useTextToSpeech.ts` — split: lifecycle/readiness portion is reusable; terminal-input-gate wiring stays.
- See [`ui/src/domains/audio/README.md`](../../ui/src/domains/audio/README.md) for the authoritative per-file classification.

**Files that stay in web-console (consumer integration glue — do NOT port):**

- `api/conversation_router.go`, `api/conversation_store.go`, `api/conversation_adapter.go` — auto-TTS trigger policy, listened-cursor, fan-out.
- `api/tts_hook_handler.go` — Claude Stop hook attribution.
- `api/codex_tailer.go` — Codex rollout tailer.
- `api/tts_playback.go` — playback ack / status snapshots tied to conversation cursor.
- `api/terminal_ws_input.go` — terminal-input-gate routing.
- `api/internal/audioports/` — adoption boundary lives in the consumer; audio-tools provides an implementation.
- `ui/src/domains/tts-playback/**` — conversation-cursor state machine.
- `ui/src/components/VoiceMicButton.tsx`, `VoiceCommandSuggestion.tsx`, `VoiceRejectionBanner.tsx` — terminal-target UX (the underlying capture primitive is reusable; the targeting policy is not).

**P0 behavior currently proven by web-console tests (must be replicated in audio-tools):**

- Transcribe: language passthrough, hallucination filter, speaker verification bypass via flag, gating by speaker verification result.
- Stream WS: partials emitted while listening, segment-final on VAD boundary, final retranscription on stop, cancellation, speaker advisory/reject flows.
- Transcode: passthrough when no resample needed, ffmpeg invocation arg shape.
- TTS config & summarize config: load returns defaults when file missing, atomic save, patch-apply semantics, timeout clamp on load.
- TTS service: synthesize via Kokoro, voice list, Kokoro availability gating, invalid-args rejection, cache hit/miss, cache eviction by `eventID`.
- Normalization/chunking: `NormalizeTextForSpeech`, `SplitIntoSpeechParagraphs` for code blocks, tables, diagrams, lists.
- Summarizer + service: cooldown after timeout, inflight dedupe by `EventID`, `<think>` stripping, four-way empty-summary classification (budget-in-think / truncated / empty-after-strip / truly-empty), timeout clamp.

**Contract decisions already settled by `audio-provider-routing-contract`:**

- Provider precedence is BYOK → Vrooli/LPBS → Local. Web-console does not implement this — audio-tools owns it.
- `ErrInsufficientCredits` short-circuits without falling through providers (BYOK/LPBS fail-fast).
- Canonical voice mapping is owned by audio-tools (provider-specific voice IDs map to canonical names).
- LPBS usage reporting and credit metering are not web-console concerns.

**Tests that should be ported or rewritten in audio-tools:**

- `api/voice_transcribe_test.go` → audio-tools STT service tests (drop the speaker-verification bypass cases that exercise web-console env-flag wiring).
- `api/internal/voice/stream_ws_test.go` (if present) → audio-tools streaming-STT tests.
- `api/internal/audio/transcode_test.go` → audio-tools transcoder tests.
- `api/internal/tts/service_test.go`, `chunker_test.go`, `normalizer_test.go` → audio-tools TTS tests.
- `api/tts_cache_test.go` → split: provider-level cache tests to audio-tools; conversation-eviction tests stay in web-console.
- `api/tts_summarizer_test.go`, `tts_summarization_service_test.go` → audio-tools summarization tests.
- `api/tts_config_test.go`, `api/tts_summarize_config_test.go` → audio-tools config tests.

Tests that **stay** in web-console: `tts_hook_handler_test.go`, `tts_router_test.go` (conversation routing), `conversation_*_test.go` (auto-TTS trigger, listened cursor).

**Open questions for `audio-tools-shared-scenario-contract`:**

- Should `EventID` survive across the audio-tools boundary, or does audio-tools take an opaque cache key from the consumer? (Web-console invalidates by `EventID` today on conversation summarization.)
- Is the `SpeechTextProcessor` port worth exposing remotely, or do consumers just call `Synthesize` and let audio-tools normalize/split internally?
- Streaming STT transport: WebSocket-only, or Connect-streaming as the canonical surface with WebSocket as a browser-friendly mapping?
- Speaker verification: separate Connect service or a `Transcribe` request flag?
- Does audio-tools own the `qwen3:4b` summarization model choice, or does the consumer pass model preference?
- Whose responsibility is the unlock-on-user-gesture primitive for browser TTS — audio-tools-provided component, or web-console-owned?

**Adoption checklist for `web-console-adopt-audio-tools` (depends on this dossier):**

1. Replace `audioports.LocalSpeechTextProcessor` with `audioports.AudioToolsProcessor` (HTTP/Connect client).
2. Replace `Server.ttsSynthesizer` wiring (currently `internal/tts.NewKokoroSynthesizer`) with an audio-tools-backed `TextToSpeech` port.
3. Replace `Server.voiceService` wiring with an audio-tools-backed `SpeechToText` port.
4. Delete `api/internal/voice/`, `api/internal/audio/`, `api/internal/tts/` (or shrink to thin pass-through adapters if any web-console-specific behavior remains).
5. Update `capabilities.Known["audio-tools"]` checker — already wired; ScenarioChecker starts returning `available` automatically.
6. UI: change `ui/src/domains/audio/index.ts` re-exports from `hooks/voice/**` / `hooks/tts/**` to the audio-tools client; delete `ui/src/hooks/voice/**` and `ui/src/hooks/tts/**`.
7. Migrate remaining UI consumers (Workspace, TerminalPane, settings) to import from `domains/audio` (deferred ratchet — see Phase 6 extension below).

Preexisting failing tests not caused by this work:
- `TestDocsManifestResolves` and `TestDocsNoStaleOldPaths` reference removed
  doc paths (`DESIGN.md`, `concepts/FLOWS.md`, and the internal progress log).
Stale `docs/manifest.json` and `docs/START-HERE.md`; unrelated to audio
extraction prep.

## Archive browse and transcript regression (2026-08-19, fixed)

- **Symptoms:** Recent archive cards in the sidebar were non-interactive; the
  archive drawer showed a search prompt instead of sessions for an empty query;
  and opening an archived result produced "No conversation events yet for this
  session."
- **Root cause:** Sidebar archive cards were plain `div` elements without an
  activation callback. The drawer deliberately cleared results and selection
  for an empty query. On the API side, transcript `Get`, `Search`, and
  `GetRange` rejected any session absent from the live PTY manager even though
  archive metadata and conversation events remained durably persisted.
- **Repair:** Archive cards are now keyboard-accessible buttons that open the
  selected session in the drawer; empty search browses the filtered archive;
  and read-only conversation operations authorize explicitly archived,
  dismissed, or recovery-pending metadata without recreating a live PTY.
  Mutation operations retain the live-session requirement.
- **Regression evidence:** `TestGetConversationSession_AllowsArchivedTranscript`
  proves an archived transcript remains readable without restoring a live
  session. Archive drawer and sidebar component suites cover initial browsing,
  direct card activation, read-only transcript selection, and coexistence with
  crash-recovery rows. Focused results: Go archive tests pass; 20 UI tests pass;
  UI type-check and the full 151-batch UI suite pass.
- **E2E note:** Browser Automation Studio execution
  `1d23329e-22b2-40c4-9bdf-1a8c05d59874` captured only the application's
  initial loading surface at
  `/home/matthalloran8/.vrooli/data/vrooli/browser-automation-studio/captures/web-console-archive-visual/1d23329e-22b2-40c4-9bdf-1a8c05d59874/screenshots/step-01-27a9e373-3266-469b-bb33-74cd1bf48496.png`.
  Live archive clicks were not forced through an observer workflow because that
  would mutate browser state against user-owned sessions; deterministic
  component and API regressions are the authoritative evidence for this repair.

## Terminal fidelity plan closeout status (2026-08-25)

**Closed in the current implementation:** cumulative UTF-8 offset reconciliation
replaces timer-based stdin retries; prediction is an overlay rather than a write to
the xterm buffer; mouse-tracking bytes use the single control-lane predicate;
terminal status is rendered in pane chrome; the terminal pane is split into focused
session, lifecycle, selection, attachment, speech, and follower seams; local and
bridged sessions use the same session manager and terminal WebSocket; and the
integration catalogue now has live checkers for audio-tools, vrooli-bridge, Ollama,
OpenRouter, and both session backends.

**Still open:** the immutable GCT baseline collection is terminally unavailable because
the source changed during both capture attempts. The latest server-owned Test Genie run
`20260825-124448-3feddfc4` remains usable for comparison: it is 14/23 passed, and the
comparison against `20260825-120927-25345d78` reports 14 clean, 9 pre-existing, and 0
regressed common phases; six newer descriptor phases are non-comparable. Portability
passes after the provider learned the canonical `platform_status` manifest. The Linux
host still cannot supply macOS or Windows runtime measurements, so those rows retain
explicit source/cross-build reasons. Quantitative gates show 75.7% Go service and 75.4%
Go CLI. The six API and six CLI cross-build cells and the three shared package suites
are green. The lifecycle owner starts web-console healthy and the live integration
catalogue contains audio-tools, Ollama, OpenRouter, both session backends, and Remote
Terminals; Remote Terminals is correctly unavailable without a bridge URL. The current
UI suite shards all pass (188 files / 1,728 tests), and an Istanbul-merged aggregate
reports 88.70% lines/statements, 86.53% functions, and 84.41% branches. Root
`make cross-compile` passes on the rerun after a transient proto-health termination.
Remaining scenario findings are named in the comparison rather than silently treated as
fixed. The input protocol is now offset-only: the old sequence/ack and paste-gate
compatibility symbols are absent, and programmatic viewport correction uses the shared
terminal scroll seam.

The required measured-behaviour ledger is explicit about unavailable evidence:

The closing validation attempt on 2026-08-25 recorded runs
`20260825-174447-0df374df` and `20260825-175107-7ad93721`. The first stopped at
provider startup while the per-user `vrooli` binary was being rebuilt. The
second executed 22 of 23 phases: 9 passed, 13 failed, and 1 was skipped. Its
comparison with `20260825-124448-3feddfc4` marked UI health, docs, soak,
performance, unit, workflow, business, experience, security, and proto as
pre-existing; portability remained unavailable because
`scenario-dependency-analyzer` could not start its unhealthy Ollama
dependency. Storage also found the existing control-plane key tree outside the
scenario's declared roots. The cache-class `uploads` root is now declared in
`.vrooli/service.json`, matching the upload writer; the control-plane key
finding remains outside this scenario's ownership boundary. The Phase 1 run
`20260825-034457-c7651d3d` is not present in the Test Genie index, so a direct
comparison to that immutable baseline remains unavailable.

Follow-up validation on 2026-08-25 recorded run `20260825-181025-07fdd7d5`: 13 phases
passed, 9 failed, and 1 was skipped. Portability, dependencies, API, unit, quality,
architecture, and the terminal protocol checks remained green. The UI-health failure
still reports its standing `visual_status_bar_color_mismatch`; this is the browser
chrome declaration check, not terminal ANSI output. The terminal color report was
traced to the host `NO_COLOR=1` environment leaking into PTY and tmux children while
`TERM` was already `xterm-256color`. `filterServiceEnv` now removes `NO_COLOR` for both
backends, a regression test covers the complete child environment, and the live tmux
server was cleared so new sessions inherit color-capable defaults.

| Row | Before | After/current | Disposition |
|---|---|---|---|
| tmux keystroke-to-echo latency | blocked: no pre-change harness | blocked: no repeatable latency instrument | No honest before/after number can be produced on this host. |
| tmux subprocesses per typed character | blocked: no pre-change process capture | blocked: no process-count instrument | The plan's required `strace`-class capture was not available as a durable baseline. |
| fixed-flick scroll distance at 60/120 Hz | blocked: no pre-change device/BAS trace | blocked: no 60/120 Hz device capture | Injectable-clock tests cover logic; they are not device measurements. |
| full-scrollback snapshot bytes/duration | blocked: no pre-change benchmark | blocked: protocol tests emit no benchmark | Snapshot ordering/completeness is tested, but benchmark numbers are not inferred. |
| bridgePTY resident memory per session | blocked: no pre-change remote allocation capture | blocked: no controlled remote-node run | The local bridge fixture proves protocol behavior, not production memory cost. |
| fixed-window grid on Linux/macOS/Windows | blocked: no comparable pre-change grids | Linux: source/font-gated only; macOS/Windows: blocked | Cross-build/source evidence is marked reasoned-from-source, not runtime qualification. |

## Work ladder

- Rung: W3 / R0 (overlay consumer migration execution)
- Evidence: W0 remains aligned with goals `hosted-cloud-tier-foundation` and `portal-front-door`; the PRD's P0 controls-surface target continues to cover the drawer/sidebar surface. `business-health validate scenario web-console` and `vrooli scenario requirements validate web-console` both pass. Baseline run `20260827-043743-f8d9edda` is terminal and records the pre-existing red implementation baseline before the overlay migration.
- Blocker: None. The plan must preserve terminal workflows while replacing local and portable overlay surfaces, and final evidence must be compared with this baseline.
- Measured: 2026-08-27.

- Rung: W3 (implementation)
- Evidence: W0 comparison remains aligned: goals `hosted-cloud-tier-foundation` and
  `portal-front-door` do not contradict this work, and the PRD continues to name
  the relevant P0 capabilities. W1 passes with `business-health validate scenario
  web-console`; the stale `REQ-P0-008a` test reference was repaired to the existing
  drawer test. W2 passes with `vrooli scenario requirements validate web-console`.
  The mobile toolbar now supplies RCL's semantic color contract from the live
  terminal-derived `--wc-*` palette at the toolbar boundary, so adopted arrows,
  voice, and icon/text controls share one theme path. Focused UI evidence: 36
  `MobileToolbar` tests pass, TypeScript/string checks pass, and BAS mobile
  capture `web-console-mobile-toolbar-theme-bridge` shows named, centered controls
  with matching surfaces and borders. Prior implementation evidence includes the
  platform matrix, dedicated route and
  migration packages, the three-lock session seam, cached echo probing, the pure
  terminal protocol reducer, typed hook commands, storage declarations, focused
  Go/UI tests, race-tested Go suites, and six-target build/vet validation.
- Remaining validation caveat: the latest server-owned Test Genie run
  `20260826-104603-fd2ab54a` passed 15/23 phases. The terminal plan's unit,
  performance-build, and CLI-contract execution failures are cleared: the UI
  build and coverage phase pass, the `hooks dispatch` command is declared, and
  the local UI suite reports 194 files / 1,826 tests with 90.66% lines,
  85.00% branches, and 85.76% functions. The remaining eight failed phases
  are standing UI-health, documentation, soak configuration, Lighthouse,
  storage isolation, workflow, experience, and security/proto findings; none
  is a terminal transport or portable-PTY failure. A full scenario pass
  therefore remains open. The plan validation operation
  `0928e81e-8eac-4686-b7db-af8ad46da03c` completed with a clean baseline
  comparison (`web-console:ready:preexisting`); it establishes no regression
  in the plan-owned scenario, but does not waive the standing failures. The
  post-run findings projection is currently unavailable because the rebuilt
  Test Genie CLI panics while loading another scenario's unsupported `asset`
  target kind; the completed run itself remains durable.
- Blocker: none. The remaining findings are diagnosable repository or environment
  debt, not missing authority.
- Measured: 2026-08-26.

- Rung: W3 (implementation/evidence)
- Evidence: The named-goal W0 search returns only archived `hosted-cloud-tier-foundation` and `portal-front-door`; neither contradicts this speech-stack work. The provider-feature capability contract, generic TTS gate, focused UI tests, live capability RPC, and production build pass; the full scenario suite remains outside the bounded validation scope.
- Blocker: None for targeted implementation; full-suite comparison remains unproven.
- Measured: 2026-08-31.

- Rung: W3 (implementation)
- Evidence: The named-goal W0 search found no goal explicitly naming web-console; the
  active Plan Manager execution `9790f7b1-de4e-4b83-9bd1-ad7716ca02ad` is the
  authoritative work directive. Existing implementation had process-exit,
  watchdog, and expiry paths deleting session metadata, and archive undo depended
  on an in-memory timer. Targeted changes preserve exited metadata and add a
  provider-neutral lifecycle transition table plus durable receipt repository.
- Blocker: Requirements validation is currently blocked by a missing `go.sum`
  entry in the unrelated `business-health` CLI; Web Console targeted tests pass.
- Measured: 2026-09-05.

- Rung: W3 (implementation/evidence)
- Evidence: Web Console starts through the managed lifecycle and exposes honest
  degraded health for the live orphan corpus. The typed continuity CLI returned
  a read-only inventory of 332 sessions, 428 conversation identities, 142,623
  events, 338 checkpoints, and 1,431 panes, with 123/104/1,352 orphan classes.
  A live dry-run observed 1,685 records with zero mutations and manifest
  `sha256:ee7ee3e9cdd22c3d57f106d26c4dd11c4c7d32d3bc7b4ac42091d3a41e89cb9f`.
  The incident thread identity query returned a match. The governed audit
  program ran successfully in both hash and hashless modes; program phase
  validation passed. `cli-health validate scenario web-console` passed, and
  targeted API/CLI/UI checks passed.
- Remaining validation caveat: comprehensive Test Genie run
  `20260906-001445-071e7267` is terminally failed on broad existing CLI/UI,
  dependency, docs, storage, security, and other maturity findings. The
  follow-up scoped run `20260906-004339-0f6fc58d` remained queued past its wait
  window with no terminal verdict. Cross-scenario Agent Manager/Search Hub
  publication and real BAS restart/recovery evidence remain open; no live
  reconciliation apply was attempted because the disposable-copy backup and
  restore gate is not yet proven.
- Blocker: None for further implementation; the remaining gaps are evidence or
  diagnosable repository debt, not missing authority.
- Measured: 2026-09-06.

## Work ladder

- Rung: W3 (implementation/evidence)
- Evidence: `vrooli scenario requirements validate web-console --json` remains structurally passing, while targeted Test Genie experience run `20260906-100500-81708c63` still reports three `experience.capture_bindings_unjoined` errors and seven deterministic-setup warnings. The authored `workspace-shell` readiness region now has an explicit required binding and the Experience Manager profile returns it, but direct BAS readiness diagnostics still fall back to generic navigation and capture a transient five-node `Loading...` tree. Focused continuity, CLI, and race tests remain passing; storage validation still reports scenario-wide filesystem ownership findings and an unrelated autoheal path.
- Blocker: The remaining experience errors are at the BAS/Experience Manager capture boundary, not proven Web Console selector failures; broad storage/readiness debt remains open. Continue with targeted implementation and evidence repair; do not weaken active bindings or claim the experience gate is green.
- Measured: 2026-09-06.

## Work ladder

- Rung: W3 (implementation/evidence)
- Evidence: Test Genie run `20260906-040729-4fd8c864` reached terminal status but failed solely on `test_genie.phase_timeout` after its durable provider run exceeded the 15-minute workflow deadline; the workflow artifact reported no scenario blockers/errors/warnings apart from that timeout observation. Requirements and business-health validation pass; the governed continuity audit succeeds with typed projection-drift evidence. Managed restart is healthy, and focused/race lifecycle tests plus vet pass after degraded-manager archive hardening.
- Blocker: Broad completion evidence remains open because the workflow provider timed out and storage validation reports scenario-wide ownership findings; BAS, baseline, generation-promotion, and final DoD evidence are not terminal.
- Measured: 2026-09-06.

## Work ladder

- Rung: W3 (implementation/evidence)
- Evidence: The named-goal W0 comparison remains aligned: `hosted-cloud-tier-foundation`
  and `portal-front-door` mention Web Console without directing this continuity
  capability to be removed or contradicted. `business-health validate scenario
  web-console` and `vrooli scenario requirements validate web-console --json`
  both pass; the latter reports only the known stale comprehensive-suite
  advisory. The implementation has fresh focused Go and UI evidence, managed
  restart evidence, and a durable failed recovery-receipt regression. The
  authoritative Test Genie run `20260906-040729-4fd8c864` reached terminal
  status but failed on the workflow provider timeout, so it is not passing
  broad W3 evidence.
- Blocker: None for further implementation. Full plan completion remains open
  pending Test Genie, baseline collection, cross-scenario generation promotion,
  BAS lifecycle/restart interaction evidence, and the remaining DoD audit.
- Measured: 2026-09-06.

## HTML file rendering and formatted-link colors (2026-09-07, scoped W3 repair)

- Scope: HTML file previews and markdown link-label colors. W0–W2 were not re-audited.
- Expected/actual: Opening an HTML artifact should offer a rendered view; the code/text renderer previously showed only source.
- Cause/fix: HTML now specializes the existing bounded text transport with Preview/Source controls. An opaque-origin iframe renders styles and scripts. A leading CSP blocks API requests, embedded pages, and form submissions; an `about:srcdoc` base preserves fragment links without resolving local assets against web-console. Line targets and truncated content initially show source. Local supporting files remain unavailable.
- Link hypotheses: A missing accent token would affect plain links too; descendant color overrides would affect formatted labels only. Chromium showed cyan plain links (`rgb(34, 211, 238)`) but white bold/code labels (`rgb(248, 250, 252)`), confirming the latter. Scenario CSS now makes formatted labels inherit the link color. No RCL release was edited.
- Evidence: 48 focused renderer/markdown tests pass, including failing-before/passing-after regressions. Scoped ESLint, TypeScript, and the production build pass. Chromium checks confirm inline CSS, an interactive counter, standards mode, parent-document isolation, and matching plain/bold/code link colors. Browser reproduction and results: `/tmp/wc-preview-browser-check.cjs`, `/tmp/wc-browser-before.html`, `/tmp/wc-browser-after.html`.
- Broader limitation: Test Genie unit run `20260907-235545-1ab76aee` failed with API/CLI/UI command failures and configuration/architecture findings. The shared worktree changed during that run; it is not exact-final-input evidence. Final focused checks supersede intermediate local assertions, not the broader failed verdict.
- Separate defect: Locale-parity tests fail on unrelated catalog keys/interpolation. Reported as Scenario QA `knw-1788825510125479292`. The new HTML strings exist in English, Arabic, and Japanese.

## Experience follow-ups

- Add a keyboard shortcut or visible help for toggling AI input.
- Add short explanatory copy for session expiration policies and countdowns.
- Consider swipe gestures for mobile pane switching after the current
  single-column flow has stronger device evidence.

### 2026-09-10 — Fleet configuration exposes target route incompatibility

- Rung: W3 (implementation/evidence)
- Evidence: `ConfigurationTab` now renders the remote failure detail and warns
  that re-apply may not fix a missing backend route. Focused Vitest coverage
  passes for the Bridge 502/target 404 case. Live `GetConfiguration` for
  `bridge-node:25c7e426-c76c-421a-8351-aaf964589802` now returns Web Console
  Connect `failed_precondition` with `target_onboarding_incompatible`; the
  target still returns HTTP 404 for `ListOperatorInputs`, while local Web
  Console and Bridge services remain healthy.
- Blocker: The target-side onboarding deployment requires an authorized
  refresh/redeployment before the semantic configuration journey can pass.
- Measured: 2026-09-10.

### 2026-09-10 — Target onboarding contract and proxy errors are typed

- Rung: W3 (implementation/evidence)
- Evidence: `vrooli-onboarding` now returns `contract_version=1` with
  `ListOperatorInputs`; Web Console rejects an absent or unsupported marker as
  `target_onboarding_incompatible`. `vrooli-bridge` maps target HTTP 404,
  timeout, authorization, contract, and 5xx outcomes to distinct Connect
  classifications. The live minimouse request now returns Connect
  `failed_precondition` with an actionable onboarding incompatibility message,
  rather than a generic HTTP 502/503.
- Result: Reachability and API compatibility are now separate signals. The
  target still needs an authorized onboarding refresh because its current
  route is absent; no remote mutation was performed.
- Measured: 2026-09-10.

### 2026-09-10 — Bridge endpoint selection now uses API Core discovery

- Rung: W3 (implementation/evidence)
- Evidence: Web Console previously allowed its `nodereach` clients to fall
  back to the CLI port detector when `WC_BRIDGE_URL` was empty. The Bridge
  client, remote PTY client, and Bridge capability probe now use
  `api-core/discovery.ResolveScenarioURLDefault("vrooli-bridge")`; focused Go
  regressions pass. Normal remote PTY sessions no longer pin the discovered
  URL into their launch spec; only an explicit `WC_BRIDGE_URL` is pinned. The
  managed Web Console restart is healthy.
- Result: The Web Console-to-Bridge seam no longer captures a sibling API port
  through a separate resolver. The live minimouse request still returns 503
  because the target-side onboarding route returns 404, so this repair does
  not yet close the remote target compatibility defect above.
- Measured: 2026-09-10.

## Work ladder

- Rung: W3 (misleading remedy on a correctly classified failure)
- Evidence: the Configuration tab told the operator to "refresh or redeploy
  vrooli-onboarding on the target" and offered Re-apply for minimouse, whose
  onboarding API is missing `ListOperatorInputs` because the machine runs
  `8ec19147c85f+dirty` (2026-09-01), 108 commits behind the control plane.
  Neither action can add a procedure to an older build.
- Repair: `targetSkewMessage` renders Bridge's `X-Vrooli-Target-Revision`,
  `X-Vrooli-Control-Plane-Revision` and `X-Vrooli-Update-Command`; the tab
  titles the state "running an older version of Vrooli", shows the update
  command, and withholds Re-apply. api-core `targetmodel` provisioning readiness
  now requires the node's `bridge-provisioner` report and a non-working-tree
  revision instead of mirroring headless readiness.
- Evidence of repair: `TestControlPlaneErrorExplainsTargetVersionSkew`,
  `TestProvisioningReadinessRequiresTheNodesOwnReport`, ConfigurationTab vitest
  "names the update command for an older machine and withholds re-apply".
  Unrelated pre-existing failures in the same runs: `TestDocsNoStaleOldPaths`,
  `TestOpenCodeWatcher_*`, and `locales.test.ts` (ar.json missing 67 keys).
- Measured: 2026-09-15

- Rung: W3 (every remote-machine action broken by one lookup; errors misnamed)
- Evidence: after the version fix minimouse still could not start a session
  ("remote machine authorization has expired"), install an agent ("Install
  failed", 400 in 20ms), or clear drift (every Fix ran the same re-apply; the
  only feedback was a status line below the credentials). Bridge logged
  `GET /api/v1/channel/session status=401` with no credential and 84
  `local owner session rejected` lines per boot.
- Root causes: (1) `targetByID` returned the selected `Target` without its
  connection, so sessions opened with no Authorization header and installs saw
  an empty Bridge URL; (2) the Bridge capability checker and the install relay
  client held a 15-minute LocalSession minted at startup, and nodereach replayed
  the Open header on reconnect; (3) nodereach mapped every 401 to
  `missing_reauth`; (4) the install HTTP client timed out at 30s inside a 90s
  relay window and every action error was `invalid_argument`; (5) the
  `bridge-provisioner` node fact reached the browser under `capability:` and
  became an installable "agent"; (6) re-apply polling stopped at 3 minutes
  while runs take ~4.5.
- Repair: `targetByID` keeps the connection; per-call credential sources
  (`bridgeCredentialSource`, `freshOwnerAuthorization`, nodereach session
  `authorize`); nodereach `ErrUnauthenticated`; typed `RunAction` codes;
  `node_capability:` browser key; Configuration tab rows carry their own action
  (Install / Re-apply / "Not fixable from here") and a top-of-panel apply panel
  that polls to the end and names each failed step's cause; the launcher lists
  install-failure reasons under the grid instead of in a tooltip.
- Evidence of repair: `TestTargetByIDKeepsTheConnection`,
  `TestNodeCapabilitiesAreNotPresentedAsAgents`, nodereach
  `TestOpenClassifiesHandshakeHTTPStatuses` and
  `TestSessionReconnectMintsAFreshCredential`, vitest
  `ConfigurationTab.actions.test.tsx` and `applyRunSummary.test.ts`. Live:
  remote session Create on minimouse succeeded; Bridge logged 0 LocalSession
  rejections after restart; minimouse drift is empty.
- Measured: 2026-09-15

## Work ladder

- Rung: W3 (scoped repair) — remote machine health had no surface
- Evidence: the Machines view showed drift and missing agents but nothing about
  disk, memory, swap or leftover onboarding folders, so minimouse's 6.8 GiB of
  stale artifacts and swap use were found only by SSH.
- Repair: `browserFactKey` maps agent observations `node-health.<reading>` to
  `node_health:<reading>` (never `node_capability:`, which would read as a
  missing feature). `machineHealth` orders readings problems-first;
  `machineIssues.healthWarnings` counts toward the card's "needs attention"
  total while `configurationCount` keeps the Drift section to configuration
  items. ConfigurationTab has a Health section (measurement + remedy, code
  stripped, OK / Needs attention / Not measured) and says plainly when the
  machine's agent is too old to report health. New keys `machines.health*` in
  en/ar/ja.
- Evidence of repair: `go test -run TestNodeCapabilitiesAreNotPresentedAsAgents .`;
  vitest `ConfigurationTab.health.test.tsx` plus the existing ConfigurationTab
  and fleet tests (51 passing); `tsc --noEmit` clean; web-console restarted.
- Pre-existing, not fixed here: `src/i18n/locales/locales.test.ts` fails —
  ar.json and ja.json lack 87 keys that en.json has (93 before this change
  added its keys to all three), including every machine key added earlier on
  2026-09-15.
- Measured: 2026-09-15

## 2026-09-16 — Maintained unit gate still has UI execution debt

**PARTIALLY RESOLVED (scoped W3 test-infrastructure repair):** Test Genie run
`20260916-185137-bcbbdf26` initially reported direct Testing Library render
imports and a test-only `remoteCreateErrorService` that duplicated the shape
of `databasetest.SliceRepo`. The three actual component offenders now use the
scenario-owned `renderWithProviders`; the remote fake embeds the existing
`RemoteService` seam; and `authStore` has explicit success, expiry, and error
regressions.

Focused evidence is green: 24 UI tests for the render-policy repairs, 6
auth-store tests, the sessions Go package, the REST-exception contract (2
tests), locale parity (7 tests), and `pnpm run type-check`. The canonical UI
coverage command still fails honestly across remaining pre-existing
UI cohorts. The overlay keyboard contract is green (50 tests), AlertDialog
and SettingsModal focused contracts are green (23 tests), and the terminal
reliability cohort is green (21 tests) after completing xterm and capability
fixtures. The shared audio transport setup gap, REST-exception registry drift,
locale catalog drift, overlay contract drift, and terminal fixture drift are
now repaired in
`src/test-utils/setup.ts`, `src/api/__tests__/no-rest-exceptions.test.ts`, and
`src/i18n/locales/{en,ar,ja}.json`;
The canonical UI suite now has zero assertion failures (278 files, 2,701
tests); its only remaining exit failure is function coverage at 82.44% against
the configured 85% floor. The threshold was not weakened. The next repair
should add meaningful coverage for the largest uncovered production branches,
then re-run the canonical command.

- Measured: 2026-09-16


## 2026-10-14T03:10:31.675440+00:00 — Interactive stable-create has no authenticated creator owner

Original Nooch E30 and Desktop E3 native admissions failed before worker prompt/session Create with `web-console stable-create identity unsupported; no Create submitted`. Read-only SessionsService/List on live WC reports supportsStableCreateIdentity=true and stableCreateOwner empty, matching AM wiring NewClient(baseURL,nil). WC List derives creator owner from verified PrincipalFromContext; the transport supplies none. No session-store capability absence is established. Preserve stable UUID/owner/fingerprint checks and do not substitute an execution mode, invent a principal, drop an offered credential or restart as proof. Required owner boundary: carry a verified authorized session-creator principal through the registered native route; qualify stable create, uncertain-result get-by-ID owner/fingerprint readback, conflict refusal, original interactive prompt delivery and existing-session preservation. Existing reports: Desktop knw-1791945836169396257, Nooch knw-1791946783491615210. Evidence: docs/internal/delivery-operations/2026-10-12/sources/execution-evidence/2026-10-13/web-console-stable-create-capability.json. Shared AM identity correction is source/serving/consumer-qualified and final unit 20261014-024937-9072cd54 PASS; this is a separate remaining boundary.


## 2026-10-16 — Authorized performance follow-up repairs (W3)

Follow-up to the long-session audit above: corrected the Lighthouse owner device
settings; prioritized/shared history loading with two background workers; moved
conversation subscriptions into navigation and selected-session playback; deferred
closed settings/appearance/fleet/archive surfaces; reserved navigation clearance
for floating controls; repaired machine/desktop fixture types. No thresholds or
real-session mutation permissions changed. The original audit's retained-debt
paragraph is historical; current evidence and remaining limits are in the
[follow-up audit](../perf/2026-10-15-long-session-performance.md#authorized-follow-up-implementation--2026-10-16).

Final validation: all 2,862 UI tests pass with every configured coverage floor
met; full type-check and touched-file lint pass. Original-code regressions establish
the root-render, hydration and reorder-lifecycle defects. Production startup
JavaScript is 15.95% smaller; peak history requests fall from 31 to 3 while total
startup reads remain 32. Live rendering traces are mixed and do not establish a
rendering-speed improvement. Performance Health unit run `20261016-131741-30ff3215`
and final deployed Web Console performance run `20261016-134657-6b914169` pass.
The broader Web Console unit run has an API failure of unverified cause, filed as
`knw-1792157470039179090`; no whole-scenario certification is claimed. Final Console
and API health checks pass. Physical iPhone behavior remains unverified; the user
accepted available browser checks after bridge inventory could not authenticate.
Full artifacts and limitations are retained in the audit.

## 2026-10-16 — Missed first presses and interrupted drawer swipes (W3)

Rung: **W3**, scoped implementation repair; work shape: bounded task.
Recorded harness acceptance goal: Work until deliberate presses reach their
controls after overlay dismissal and drawer swipes survive ordinary content
updates, proven by the focused library Vitest cohort exiting 0 and the served
native-input browser assertions exiting 0, with outputs retained. Read the RCL
gesture contract and `docs/TESTING.md`; limit implementation to shared overlay
assets and their tests, docs, and consumer lifecycle. File adjacent defects.
Run scoped unit phases and retain their actual verdicts; no broad certification
is implied. Checkpoint changed, verified, remaining and unverified here. No
missing decision, credential or approval prevents this bounded repair.

The shared RCL investigation and hypotheses are
recorded in [RCL's problems document](../../../react-component-library/docs/internal/PROBLEMS.md#2026-10-16--missed-presses-and-cancelled-drawer-gestures-w3).
Original native touch events demonstrate that a closed Machines drawer still
receives the first press intended for the view toggle during its exit. A second
press reaches the toggle. The shared swipe cancellation effect also depended on
a freshly allocated style object, so ordinary rendering aborted the gesture;
a cancelled drag could additionally leave suppression armed for the next tap.

Governed releases: `useOverlaySurface` 1.4.6/1.4.7, Dialog 1.3.9, AlertDialog
2.0.11 and CommandPalette 1.1.11. Closed roots relinquish input immediately while
retaining exit painting. Content rerenders preserve swipe ownership; geometry
changes and closure cancel safely; suppression stays with the originating drag.
Web Console's installed dependency was initially stale despite rebuilding the
package. The scenario lifecycle refreshed the installed copy, rebuilt and
restarted Web Console; it reported healthy on ports 16382/21233.

Final served browser evidence: **ten immediate view-toggle activations pass**,
five at 390px touch and five at 1440px desktop; navigation reopens the drawer on
its first press during exit. A native touch swipe remains at `translateY(50px)`
through a fleet refresh, closes only after release, restores position on cancel,
and permits the next deliberate grabber tap. No page errors. Physical iPhone
Safari has not been measured. Browser evidence:
`/tmp/wc-interaction-final-proof.json`, `/tmp/wc-interaction-final-browser.log`,
`/tmp/wc-interaction-{mobile,desktop}.png`. Original miss:
`/tmp/wc-dismiss-click-red.json`.

Final focused library cohort: **18 PASS**. Consumer cohort: **119 PASS, 1 FAIL**;
the remaining failure is ApprovalsDrawer's missing explicit virtual-keyboard
policy, in a concurrently changing feature outside this repair. Filed as
`knw-1792204786873961363`. Type-check and production build pass. Shared package
and declarations verify artifact
`sha256:2a3754fa7c334becf715138158914f592be91eb855afffd818d41818ea3d6d65`.
Library story captures retain unrelated placeholder/review-sheet failures
(`knw-1792204422313129557`); Dialog and CommandPalette story contracts pass.
Scoped unit receipts are terminal and **not passing**. Web Console run
`20261017-023854-61a3e4d0` ended `provider_unavailable` after 120 seconds waiting
for Unit Health ownership; it measured no assertions. RCL run
`20261017-023847-37d16d18` failed after 184 seconds: API Go tests, CLI Go tests,
UI coverage, UI type-check, and the render-helper policy. The returned unit log
contains findings without command diagnostic excerpts, so attribution of those
broad command failures remains unverified. Opaque artifact
`artifact_201495a033faabad007c4af9cd6c1721` and its retained bytes
`/tmp/wc-rcl-unit.log` preserve this limitation. Aggregate wait receipt:
`/tmp/wc-interaction-unit-verdicts.json`. Shared generated-owner preflight
confirmed the proto artifact fresh before these runs. No scenario-wide or native-
iPhone certificate is claimed.

Unit ownership timeout filed as `knw-1792205023824743132`; broad RCL unit
failures filed as `knw-1792205024296433128`. The scoped implementation behavior
is repaired and verified by the focused fixtures and served browser checks above.

Work record: `48bdf6ca-8689-4f37-9bf8-39dc965bb64a`.

## 2026-10-17 — Remaining intermittent taps: sidebar, markdown, view toggle (W3)

Follow-up bounded investigation: the operator still sees missed sidebar backdrop,
markdown-link, and terminal/messages-toggle presses on iPhone. Previous overlay
exit and swipe regressions remain relevant prior art, not proof that every missed
press has that cause. Source-ledger recall timed out; the earlier section and its
retained evidence are reused. Acceptance: preserve the actual press target across
ordinary content/callback updates; verify native touch delivery for all three
surfaces and retain any conditions that cannot yet be reproduced.

Hypotheses under test:

- An overlay intercepts the press: trace native hit targets, default prevention,
  capture, and cancellation before/after dismissal. Clean native touch sequences
  currently pass eight view toggles and eight sidebar backdrop dismissals.
- Content updates replace the pressed DOM node: hold a link during a callback or
  message update and assert that its identity, focus, and activation survive.
  Markdown currently creates new React component types when callbacks change.
- Native panning/focus changes cancel a touch: compare stationary and moving
  presses and keyboard/focus transitions. The sidebar uses a separate dismissal
  implementation from `useOverlaySurface`; this needs independent evidence.

Initial event trace: `/tmp/wc-remaining-trace.json`. Physical Safari is not yet
measured; clean Chromium taps alone do not certify this intermittent report.

Confirmed follow-up defects and repair:

- Markdown renderer callbacks recreated React component types and replaced
  links during a press. Stable module-level renderers plus callback context
  preserve DOM identity and focus while using current callbacks. Callback and
  streamed-content regressions pass.
- Message-row recognizers still owned long presses and native context menus
  beginning on nested links. The row now declines those controls before gesture
  recognition. Three new regressions failed before repair; all twelve row tests
  now pass.
- SidebarShell waited for a compatibility click. A primary press now dismisses
  immediately, matching the shared overlay policy, while keyboard clicks remain
  available and secondary pointer presses are ignored.
- Shared Presence only disabled pointer events on its wrapper; explicitly
  interactive descendants could still intercept the first press behind an exit.
  It now marks logically absent content inert. Exit-completion callback identity
  no longer restarts the animation's lifetime. Native input hit the old retained
  surface and produced zero activations before this fix; it now hits and
  activates the underlying control while exit content is still mounted.

These are distinct causes with similar symptoms. A repeatable clean-state toggle
miss has not been demonstrated beyond the earlier closed-overlay interception.
The toggle remains in the served-browser regression cohort; native iPhone and
keyboard-dependent recurrence are not certified by these local runs.

Additional operator requirement: opening the mobile sidebar must release the
current editable field so the virtual keyboard closes. SidebarShell previously
never took focus, and mobile button activation alone does not establish that
focus transition. The shared component now focuses its non-editable dialog once
when opening, including edge-swipe opening. It leaves persistent navigation and
subsequent focus inside the drawer alone. The focused regression failed with the
original component and passes with this change. Actual OS keyboard animation
still requires physical-device confirmation.

The served-app check exposed a further opening-frame gap: the sidebar had
`tabindex=-1`, but calling focus while its closed visibility transition was
starting left focus on the trigger. Manual focus after the opening frame worked.
SidebarShell 2.7.5 waits for that frame and cancels deferred focus on close.

The shared focus-return hook also focused explicit return targets on initially
closed mounts and reran on return-ref identity changes. A closed popover can thus
steal focus without ever opening. useFocusReturn 1.1.2 captures the opener once
and restores only after a real close, with preventScroll. Three regression cases
fail against 1.1.1 and pass against 1.1.2. A native Chromium fixture confirms the
old closed popover steals focus from a text field; the repaired one preserves it.

An intermediate shared build used SidebarShell 2.7.5 alongside Presence 1.0.6
and useFocusReturn 1.1.2; the final artifact and SidebarShell 2.7.6 receipts
are recorded below. The control plane detected each shared-package change and
rebuilt the consumer.

Final refinements: SidebarShell 2.7.6 waits until computed visibility is visible
(the first RAF can still be hidden), and closed sliding panels become inert
immediately. Responsive desktop panels retain input and complementary semantics.
Native hit testing against 2.7.5 showed the departing sidebar intercepted the
menu trigger's position. The final focused RCL cohort is 28/28; Web Console's
workspace, pane-shell, row, and markdown cohort is 97/97.

Scoped unit receipts are terminal FAIL: RCL `20261017-132539-b05b24db`
(224.3 seconds: API, CLI, UI coverage/type-check, render-helper policy) and
Web Console `20261017-132752-eec15c57` (472.5 seconds: API and UI coverage).
They preceded the last focus/exit refinements and do not certify the final
artifact. Returned logs expose findings without command diagnostic excerpts;
attribution remains unverified. Receipts: `/tmp/wc-followup-unit-verdicts.json`,
`/tmp/wc-followup-{rcl,wc}-unit.log`. The new Web Console observation is filed as
`knw-1792244510179589499`; prior RCL command/policy and capture reports are reused.

Story evidence remains non-passing where pinned captures or the review sheet
time out waiting for the passed readiness marker. Sidebar live cases pass;
useFocusReturn's story report `ctr_3665e38b537bc30f` passes. Relevant retained
failed reports include Presence `ctr_a940d1a0332a66ca` and Sidebar
`ctr_2e041cf5c048cee7`; these are not waived or counted as successful checks.

Native browser fixtures also demonstrate both markdown press loss and unsolicited
focus movement in Chromium and WebKitGTK. The old markdown link disconnects
during a held mouse press and receives zero activations; the fixed link remains
connected and receives one. A mounted closed popover previously focused its
trigger; the repaired hook leaves the current field focused. Retained evidence:
`/tmp/wc-markdown-{native,webkit}-proof.json`,
`/tmp/wc-focus-{native,webkit}-proof.json`,
`/tmp/wc-presence-{native,webkit}-proof.json`. WebKitGTK input is mouse input,
not iPhone touch certification.

Final served verification: `/tmp/wc-followup-served-proof.json` records eleven
first-press view toggles (eight after sidebar dismissal, three during another
overlay exit), primary-press backdrop dismissal, focus release after button and
edge-swipe opening, no editable-field refocus on release, and zero page errors.
`/tmp/wc-sidebar-exit-hit-proof.json` controls only the closed panel's inertness
within the same native frame: the old policy hits the departing panel, while the
new policy hits the underlying menu and permits its first native tap. This
controlled comparison removes the ambiguity of a nested `button-label` test ID.

Final artifact: `sha256:f691c4a0afd09bdbbdb8764161880a20c337dbbbddfec27494e8212994c0df73`.
Installed Web Console root/major exports resolve SidebarShell 2.7.6, Presence
1.0.6 and useFocusReturn 1.1.2. Governed restart finished healthy on ports
16382/21233. Final installed-consumer cohort: 97 passing tests; final shared
cohort: 28 passing tests. Consumer type-check and lifecycle production build
pass. Current SidebarShell story report `ctr_e9e9687a4d7f6875` passes its live
cases but retains the failed review-sheet readiness capture. The capture failure
is not waived; the broader failed unit receipts above are not recast as passes.

Native iPhone keyboard animation and the operator's precise intermittent state
remain unmeasured. The repaired application focus, hit-testing and activation
contracts are verified locally; no blanket native-device certification is claimed.

Follow-up work record: `7484dde7-c00d-4054-af17-819e82ea70de`.

### 2026-10-17 — listening-page link and embedded AAC audio

Bounded repair target: opening the supplied listening-page reference must show
the existing voice A/B and serve its local audio after a healthy lifecycle restart.
The screenshots showed percent-encoded U+FF0F full-width slashes in an agent's
Markdown href. The missing-artifact hypothesis was falsified: the ASCII path
resolved successfully and the HTML remained on disk. The malformed-reference
hypothesis reproduced `not_found` against the same session. Independently,
the HTML runtime returned 403 for a real A/B `.m4a` file: its separate media
allowlist omitted formats already supported by standalone previews.

The resolver now tries one full-width absolute-path recovery after literal
candidates fail, returning a warning and preserving genuine Unicode filenames.
The HTML runtime reuses the preview package's media extension registry. Existing
directory confinement, symlink rejection, session-bound IDs, non-media size
limits, streaming deadlines and range semantics remain covered by regressions.
New regressions failed before these repairs; the complete filepreview package
and focused `TestFilePreview` API cohort pass afterward.

Governed restart `startop-86c23d58929f4b43e818d2a5278537c0` completed healthy.
API and UI health endpoints returned 200 with readiness=true and build identity
`sha256:069a092fdf23d1f6a967d35658fe38ca22521e2560155ea268b8977cb2e21fe3`.
The exact encoded malformed path resolved after restart; the served HTML matched
the original and all 17 in-directory audio sources returned matching 206 byte
ranges with audio MIME types, including all five voice-conversion players.
The historical rejected control outside the HTML directory was excluded from
this runtime check; its reference does not widen the preview capability.
Evidence: effort `video-toolchain-reliability-2026-10-16`,
`qa/web-console-link-preview-v1/served-verification.json`.
The BAS browser reports no AAC support (`canPlayType` empty,
`DEMUXER_ERROR_NO_SUPPORTED_STREAMS`), while WAV decodes and advances normally.
The existing listening page now offers its existing matched WAV files as
fallback `<source>` elements after AAC; no audio was regenerated or changed.
Final HTML and all 34 AAC/WAV sources pass byte-range/content checks in
`served-with-fallback-verification.json`. BAS execution
`cf2ed099-c46b-448d-987a-721f1d56eee8` independently verifies that all five voice
players decode, report their expected 16.4/102.56-second durations, and advance
playback without media errors. Evidence is retained in
`browser-playback-proof.json`; initial failed codec execution and the diagnostic
probe remain available beside it. This does not certify native iPhone playback.
Scoped unit run `20261017-182249-ec2937f2` finished FAIL after 336 seconds:
`TestDocsNoOrphans` identifies `perf/2026-10-15-long-session-performance.md`;
UI REST policy and locale key/token checks also fail, and function coverage is
83.99% against 85%. These are separate from the passing focused preview
regressions and served checks; no broad certification or baseline attribution
is claimed. The immutable snapshot is retained in `unit-run-snapshot.json`.
Out-of-scope findings were published through the report-bug workflow as
`knw-1792261760670502021`.

Discovery limitation: prompt-manager was unavailable during this repair;
`skill read` and the required `discover --type all` fallback both failed.
Available local debugging/work-ladder guidance and BAS's canonical skill were
read directly. Program preparation `prog_2681b156-6c22-433c-997d-25050b071cac`
exhausted its 15-second budget; the BAS skill's direct capture leaf was used.
