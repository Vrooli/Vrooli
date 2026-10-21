# Device definitions, acquisition and qualification

## Scope

Foundation contract, 2026-10-19. Product behavior below is specified for implementation; the generated reference scaffold does not implement it. Research demonstrates individual techniques, not a released scenario.

Treat model, physical definition and software presentation separately. Existing/imported/new devices follow one qualification path. An attractive thumbnail or accepted plate does not prove dimensional geometry.

## Representative release matrix

| Group | Required representative behavior | Additional distinctions |
|---|---|---|
| Phone | Stationary presentation and true rigid rotation | Front/rear/side fidelity, camera protrusion, status/island/home regions |
| Laptop | Desk presentation and real lid opening/closing | Separate base/lid mechanical parts, hinge, moving screen, closed envelope |
| Tablet | Landscape/portrait presentation with supported placement | Bare body versus keyboard/stand attachments |
| Foldable | Qualified declared open/closed state and correct display selection | Inner/outer screens, rigid dual-screen versus continuous flex; animation separately qualified |
| Watch | Wearable display presentation in qualified pose | Round and rectangular mask semantics; strap/contact scope |
| Monitor/TV | Desktop stand and wall-mounted configuration cases | Exact stand state, mounting interface/clearance; variants stay distinct |
| Smart display/frame | Table-supported smart display and digital-frame cases | Speaker/base versus framed panel; original printed frame requires digital adaptation |
| Vehicle host | Stationary automotive screen within credible host/trim | CarPlay/Android Auto app viewport separate from cluster/host UI |

The grouped catalog must exercise both stand/wall and smart-display/frame assembly distinctions. A family enum or unsupported asset lead is not completion. The broader 23-family research remains the extension stress set: e-readers, handhelds, POS/kiosks, appliances/industrial panels and optical targets. Do not represent HUD/projector/AR optics as ordinary physical flat panels.

## Physical contract

Each exact device records manufacturer/model/generation/size/variant and measurement source; generic designs are visibly labeled generic. Dimensions retain units, measurement state and uncertainty. Separate external envelope, part geometry, panel/glass, active display, cutouts, ports/buttons, support/contact and mounting interfaces. Never derive a hinge location or open envelope from closed dimensions alone. Manufacturer diagonal/raster does not independently establish active corners or bezel thickness.

Parts have stable names/local frames; joints declare parent/child, axis/pivot, range and rest pose. Attachments declare mating interfaces and constraints. Motion qualification examines the full swept envelope, not just endpoints. Folds can have several independently mapped displays and screen-state handoff; a deforming mesh requires UV/deformation calibration, not four-corner homography.

## Public revision projection

Device, scene and motion revisions are immutable owner snapshots exposed through typed Connect messages. Their public messages must round-trip every defined owner field, including optional display primitive/material indices, without dropping metadata or confusing an absent value with zero. The public physical-starter initializer, its preset, and every pinned component revision must remain readable through their owner APIs with the same IDs, revisions and digests. Exercise this path through the Connect clients when adding a family; direct store tests do not prove the public contract.

Placement preserves real scale. Support solves contacts including laptop feet or phone camera bump; cosmetic floating gaps cannot conceal unsupported poses. Static support geometry under an assumed center of mass is not measured dynamic stability. Wall compatibility requires an actual mount chain and clearance, not a boolean inferred from a flat back.

## Source and rights policy

Use zero-purchase assets. Prefer CC0 or properly attributed CC BY for an editable bundled library, after checking exact acquired dependency closure. A public download label is a lead, not acquisition/quality proof. Retain original bytes, source/author/license evidence, access method, dates, hashes, adaptation log and attribution text. Render-only rights are separate from source redistribution, editable export, preview reuse and external-AI upload. Never infer wallpaper/font/logo/image rights from the model's top-level license.

Reject misleading free previews, unresolved provenance and incompatible permissions for the requested use. Account-required sources require a supported acquisition path; do not extract viewer assets to bypass it. Generic alternatives are explicitly labeled and still require measured consistency and craft. Software GPL redistribution obligations and asset rights are separate inventories.

## Reference-based creation and correction

Start with authoritative dimensions and reference views; select a close device only as a declared construction reference. Lock known dimensions before silhouette/proportion fitting. Calibrate view/camera before changing geometry to fit a photograph. Compare front/side/rear and held-out oblique views; isolate geometry from lighting/material mismatch. Preserve unresolved hidden detail as unknown.

A correction iteration records input revision, observed mismatch, proposed edit, execution/tool/model identity when available, output revision, reference overlays and acceptance. Never change the reference or omit a difficult view to improve a score. No universal numerical fit threshold is invented: derive tolerance from source uncertainty and intended screen/hardware footprint, then record it before candidate evaluation. Human review judges realism, detail, material and composition separately from dimensional checks.

## Import and portability gates

Check external URIs, archive escape, script autoexec, finite transforms, extreme mesh/texture budgets, missing dependencies, normals/UVs/material extensions and active display orientation. Preserve native fidelity and qualify portable GLB separately. Native renderer shaders, light rigs and procedural materials do not necessarily survive GLB. Current file preview strips animations; use it for static inspection only. Preview simplification needs its own revision and error envelope; do not alter final geometry silently.

## Cross-References

[data](../concepts/DATA.md), [creation](creation.md), [acceptance](../internal/ACCEPTANCE.md), [research](../research/README.md).
