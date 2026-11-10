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

## Cross-references

- `path:../START-HERE.md` — generated onboarding and ownership gates
- `path:../guides/choosing-ui.md` — experience contract guidance
- `path:../../experience/README.md` — machine-readable experience contract
- `path:../../DESIGN.md` — design token and accessibility contract
