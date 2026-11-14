# Experience Design — Light Desk

## Design decision

Light Desk is a calm operational workbench for persistent host lighting. It is closer to a compact device-control console than a gaming effects editor: operators need to know what policy is desired, which enrolled zones can carry it, and whether a command was accepted. The interface never implies that acceptance means visible light output.

The operator approved the full Light Desk design in the originating conversation on 2026-11-10: “I approve of the full light desk design”. This ratifies the concept and all six paired desktop/phone mockup boards presented immediately before that answer. Product UI implementation may proceed. The information architecture and policy vocabulary below are the approved target, not evidence that hardware is available.

## Comparison

The closest comparison is a home-automation device console: the bar is clear device identity, concise capability limits, safe controls, and explicit disconnected states. Light Desk adds operational policy provenance and freshness; it avoids the gaming-dashboard expectation of continuous animations or invented telemetry.

## Primary surface: Light Desk

Desktop uses a wide workbench. A backend/readiness strip and enrolled-controller list occupy the left side; the selected controller's stable zones and capabilities sit in the center; effective policy, source freshness, and last command outcome remain visible in a right inspector. Zone color is a swatch with a textual color value. A persistent Off action is reachable from the workbench and is marked as manual priority.

Phone uses a focused, single-column card stack rather than a compressed three-column layout. The first card summarizes backend readiness and the currently desired policy. Enrolled controllers open into a route-level detail view with zone-by-zone controls. A safe-area-aware bottom action keeps manual Off in thumb reach. Settings and history are secondary routes.

States are explicit: loading uses labelled skeletons; no enrolled controllers offers an enrollment explanation; backend absent or stopped preserves desired state and explains that reconciliation will occur after reconnect; permission failure is distinct from absence; unsupported zone effects are disabled with a reason; stale telemetry shows age and the static fallback; a failed command leaves the desired value visible alongside the failure and retry action. Hardware observation is shown only when separately measured; SDK acknowledgement is labelled as command accepted.

## Health scope and source feedback

The Health and Combined labels refer to current trusted host/substrate condition evidence from Infrastructure Manager, plus independent System Monitor memory pressure. They do not claim complete machine diagnosis or use scenario service uptime as host health. Each reading shows its own source age and reason; unavailable coverage is explicit. An intentionally disabled unrelated scenario does not itself turn the lights red, and high utilization alone is not an unhealthy machine.

Combined uses the current health color and a supported gentle pulse only while fresh active-agent evidence indicates work. Unknown activity leaves a known health color steady. Unknown health uses the configured static fallback. Activity counts exclude imported historical sessions. Brightness limits and zone capability fallback apply to the requested output. The interface must keep source degradation distinct from an actual owner-graded warning/failure.

Infrastructure Manager is an optional runtime-only reader. Opening Light Desk or selecting a preset never starts it, activates its health team, installs OpenRGB or grants hardware access. If the owner is not running, source evidence says unavailable and the rest of Light Desk remains usable.

## Shell configuration

| Setting | Value | Why |
|---|---|---|
| Kit | `vrooli-default` | Operational neutral palette, light/dark/system modes, and existing accessible status patterns fit this device workbench. |
| `density` | `sidebar` | Desktop benefits from persistent device and policy context; phone becomes route-level panels. |
| `mobileNav` | `tabs` | Keep Home, Presets, and Settings reachable without carrying desktop chrome into the phone layout. |
| `mainMode` | `scroll` | Zone lists and settings need predictable vertical reading and keyboard navigation. |

## What the design is accountable for

1. Make desired policy, backend readiness, and command/observation evidence visibly distinct.
2. Keep manual Off, stable controller identity, and per-zone capability limits understandable at both sizes.
3. Preserve truthful error, stale-source, unavailable-device, keyboard, reduced-motion, and screen-reader behavior; target WCAG 2.2 AA.

## Information architecture

| Surface | Route | The question it answers |
|---|---|---|
| Light Desk | `/` | What lighting policy is desired, and which enrolled zones can apply it? |
| Settings | `/settings` | How do backend setup, policy thresholds, source freshness, and reconciliation behave? |
| Controller detail | `/controllers/:controllerId` | What stable zones and verified capabilities belong to this controller? |
| Presets | `/presets` | Which manual or telemetry-driven preset is active, and why? |

## Evidence and refinement log

- Proposed concept: desktop workbench with persistent device/policy context; phone-focused route stack with thumb-zone Off.
- Truth constraint: desired state, backend acknowledgement, and physical observation are separate fields and labels.
- Ratification status: concept and mockups approved together by the operator on 2026-11-10, recorded at 17:49Z in the originating supervisor conversation. Exact answer: “I approve of the full light desk design”. No further design approval gate is required for ordinary implementation refinements.
- Six approved paired desktop/phone boards and exact prompts are saved in `docs/mockups/README.md`, `CUSTOMIZE.md`, and `manifest-v1.json`: overview, controller, presets, custom preset editor, setup/settings, and recovery states. The operator explicitly requested image-first review, so boards were presented before this combined ratification.
- Preserve the implementation corrections in `docs/mockups/CUSTOMIZE.md`: visible phone command/observation evidence, source age and stale threshold, exact preview labels, controller navigation context, mobile capability/brightness parity, and unpictured error states. Save draft preserves the active policy; Save & use replaces it. Shown colors/thresholds are illustrative, not hardware facts or adopted defaults.

### ui-regression-owner-20261111

Agent-authored behavioral refinements on 2026-11-11 preserve the six ratified boards and their routes. Preset edits use a cloned policy, retain independent saved health/activity drafts, and keep selection locked throughout Save & use. Browser storage errors retain the editable draft and cannot relabel a confirmed remote selection as a failed save. A partial remote failure refreshes the policy and reports uncertainty about selection. Manual Off receives handled failure/retry feedback and remains the active preset during preview or local draft saving.

Controller detail retains backend error details with in-place query/enrollment retry, identifies each zone's apply action accessibly, and shows the editable color's hexadecimal value and configured brightness cap. The preview obeys that cap. Dashboard static writes require a ready backend and stable identity. Its desktop columns and controller status panel use grouping elements inside the page region, preserving one app main landmark and the existing phone ordering.

Local verification passed 149 tests across 26 files with 97.08% lines/statements, 89.80% branches and 91.85% functions. The committed 85% coverage requirements remain in effect. The command was `pnpm test:coverage --coverage.reportsDirectory=/tmp/lighting-control-ui-coverage-final` in `ui/`. `pnpm type-check` and scoped ESLint of the three page sources/tests also passed. Loaded Dashboard and Controller pages passed axe checks using the existing test helper. Phone-composition tests retain command and physical-evidence labels and manual Off reachability.

Disposition is locally verified implementation. The supervisor owns managed restart, fresh Test Genie receipts and served desktop/phone comparison against the approved boards. Browser contrast, focus/keyboard operation at real viewports, safe-area geometry, screenshots and physical lighting remain outside this local-test evidence.

### zone-policy-selection-20261111

This agent-authored implementation decision clarifies the approved interaction between controller edits and global presets. Save & apply establishes manual ownership for that zone, so every subsequent automatic tick preserves its settings. Explicit Use preset or Save & use releases manual ownership for enrolled zones and replaces manual Off. The shared control writer archives the previous manual/legacy state and last-command bytes before changing ownership. Automatic ticks never release that ownership. The original approved mockups remain the design references. Preset instructions and completion feedback now describe this consequence, including retained recovery data.

Older desired rows lack source ownership and remain available for review without automatic replay. Existing Save & apply explicitly adopts manual settings. An explicit global preset selection archives and adopts enrolled legacy rows for that policy. The canonical storage/precedence contract and compatibility decision are in `CONTROL.md` and `DATA.md`. Deterministic control tests cover preserved manual output on every policy tick, a stale former pulse across process restart, fresh retry after reconnect, legacy bytes and old manual Off/expiry, and transactional archival on explicit policy selection.

### phone-action-footer-20261111

The originating supervisor supplied served capture `bas-capture://2ec70dcf-666d-4c89-a0b8-1acd96b39527/screenshot`. At 390 by 844, its DOM tree placed Save actions at y739.09 with height44 while the safe-area navigation began at y741. The nested editor began at y702.91 and constrained its sticky child. The agent-authored correction composes the phone Save draft, Save & use and manual Off actions once at page level. Controller Off also occupies a page-level phone footer. Desktop keeps its existing editor/status placement.

The mobile footer uses the existing shell's main scrollport with bottom0. AppShell renders its safe-area-aware navigation as a flow sibling outside that scrollport, so the footer follows the actual navigation geometry. The former guessed 3.75rem navigation offsets are removed. Settings controls retain ordinary scroll access unless its owning surface composes the shared footer.

The new browser-local mutating case `bas/cases/experience-spec/presets-mobile-actions.json` measures all three action bounds and center hit targets against the actual navigation before and after scrolling, with a synthetic 34px safe area. It also requires visible labels and unobscured center hit targets for every navigation link. Canonical EVALUATE/SCROLL classification requires mutating execution, confirmation and routed isolation labels, despite the fixture making no backend control calls. Workflow Health owns the routed test header and lease. Assertions use the registered page selector. Strict BAS validation passed with eight nodes and seven edges; storage validation and routed eligibility passed. Local UI verification passed161 tests with 96.94% lines/statements, 88.87% branches and 89.67% functions. The first served regression was rejected before execution because its original observer label did not match EVALUATE/SCROLL. The corrected routed case awaits server-owned execution. The supervisor capture `bas-capture://5ede9b7b-3361-49d4-9d64-eb3db5e54d49/screenshot` visibly shows Save draft, Save & use, Off and all three navigation icons/labels. This raster evidence does not replace the pending center-hit assertions.

Served followup `20261111-021607-5f39f613` passed the experience floor after
canonical chrome classification repair. Its routed mobile workflow reached
BAS but the first page load returned SQL errors for missing lighting tables.
Execution `f405669c-d62a-42af-a3d4-d51ef7d9339d` already failed its initial
action wait, before evaluation or scrolling; the last page-selector timeout
was a downstream symptom. The API now initializes admitted test pools with
the same owned schemas and identity migration as primary bootstrap. A real
leased-repository regression verifies readable empty control state, policy
initialization, leased writes and unchanged primary state. Fresh served
geometry assertions remain pending the supervisor's managed adoption.

### customize-controls-20261111

The independent full-design review found that the editor's blanket native-effect
warning contradicted the supported per-zone policy projection. Customize now
edits the existing active and busy physical effect IDs independently, including
Steady (empty ID), with a valid busy threshold of 2–5000. Choices include exact
advertised zone effects and saved IDs even when temporarily unavailable. Each
zone still applies only its advertised effect or reports static fallback; this
UI never claims command acknowledgement or physical motion from a selection.

The browser sample has an explicit reduced-motion preference. That local preview
choice never changes physical effect IDs. No-agent output remains steady.
Native effect names select device-supported behavior; the editor does not claim
independent slow/gentle speeds where the canonical policy has no speed field.

Freshness, stale static color/brightness and collapsed Load enter/exit/debounce
controls use existing shared policy fields. They are saved with each browser
draft and affect the shared policy only on Save & use. The editor says that these
fields are shared. Cancel unsaved edits restores the saved draft or remote
configuration without a command. Source ages use the active policy's evidence
and threshold. Zero fallback brightness is described as dark static output,
because the existing backend does not project an explicit Off mode for it.
Preset kinds retain their defined source relationships: Combined uses health
color and agent activity, Health uses health, Work uses activity, and Load uses
load thresholds. This avoids promising unsupported arbitrary source combinations.

Local verification passed175 tests with97.71% lines/statements,87.91% branches
and92.47% functions. A new routed browser-local case,
`bas/cases/experience-spec/presets-customize-draft.json`, exercises independent
effect/preview choices, freshness/fallback drafts, Load thresholds and restoring
the prior selected draft through the normal editor. Strict BAS validation passed
four nodes and three edges. Served execution remains with the supervisor.

Served run `20261111-032848-2bf680b2` passed five of six browser cases,
including the complete Customize draft journey
(`17f25bd6-ba59-4e3e-823e-909bf8aa76b3`). The mobile action case initially
proved all three actions and navigation links visible and center-hittable.
At scrollTop3136 its Save buttons moved to y−42.52 and Off to y9.48, while
navigation stayed visible at y741. The screenshot confirmed the footer escaped
above the viewport: its natural flow position preceded source evidence and
policy guardrails. The same page-level sticky footer is now the final page
child, so following content cannot scroll its natural endpoint above the
viewport. The original geometry and hit-target assertions remain unchanged.
Fresh served execution is pending the supervisor's next managed adoption.

## Per-zone manual controls

Overview and controller detail use the same per-zone draft editor. Each zone
keeps its own color, brightness and mode draft. Static and native effect choices
come from that zone's current advertised capabilities; saved unavailable effects
remain visible with an explanation and cannot be applied. The color strip in
controller detail is an illustrative draft preview, never observed output.

Save & apply persists manual ownership for that zone using the existing control
API. A native effect uses its exact advertised identifier, color and brightness;
there is no unsupported speed promise. Command results remain visible per zone,
including unsupported scope or brightness and backend rejection. Manual Off
continues to take priority until explicitly resumed. A global preset selection
archives manual zone settings through the existing policy-selection path.

Both layouts show and constrain edits to the current maximum brightness. The
control service enforces the current cap at emission for UI, CLI, direct API and
reconciliation; persisted manual intent is retained so cap changes do not erase
it. Native effects that cannot honor the limit trigger a safe Off attempt;
capability failures remain explicit when that fallback cannot be delivered.
The backend retains authority over capabilities, and an advertised effect is not proof of visible
motion or measured output.

## Cross-references

- `path:../START-HERE.md` — generated onboarding and ownership gates
- `path:../guides/choosing-ui.md` — experience contract guidance
- `path:../../experience/README.md` — machine-readable experience contract
- `path:../../DESIGN.md` — design token and accessibility contract
