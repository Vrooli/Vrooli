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

### Scene support slots and compatibility

Scene support frames describe coordinates; they do not by themselves assert that a
device fits. A scene revision may expose immutable support slots that reference a
frame and declare the allowed support kind/interface, device groups, required
device pose, and any measured fit envelope or clearance. A physical setup selects
one slot explicitly. Compatibility is evaluated against the exact pinned device,
scene, and motion revisions by the canonical device support, pose, and motion
evaluators. A declared contradiction is `incompatible`; complete evidence within
the slot is `compatible`; missing or unknown fit, mount, pose, or qualification
evidence is `needs_review`. Unknown dimensions or installation evidence are never
treated as zero, a wildcard, or a match. Incompatible setups cannot compile for
rendering; needs-review setups may be inspected as candidates but are not
render-eligible.

Existing immutable `scene-candidate/v1` revisions without slots retain an explicit
`legacy_interface_only` compatibility mode: their recorded support-frame
interface match is reported as `needs_review`, never as a measured fit or mount
approval. This preserves historical candidate previews and reviewed visual
setups without silently granting compatibility to new slot-based scenes.

Fit envelopes are expressed in canonical metres and name the dimensions and
clearance they constrain. They do not replace source measurements: unknown
measurement uncertainty, required pose, joint state, mask/cutout geometry, or
mount-chain facts remain visible in the compatibility result. Family-specific
requirements remain explicit: vehicle host and panel IDs must be pinned together;
fold states must stay within declared joint limits and selected display masks;
monitor brackets require matching interface and known installation facts.

## Explicit physical qualification review

Qualification is a human-triggered, revisioned decision. Candidate creation, import,
AI output, starter initialization, and render success never qualify a device. The
review action names the exact candidate revision and SHA-256, exact source bundle
manifest/review/artifact digests and allowed uses, rights/attribution claims,
dimension states and uncertainty, display/view and mask evidence, supported poses
and motions, mount/support interfaces, and an actual native render output/job
digest that the reviewer inspected. Missing or mismatched evidence blocks approval.
Rights are recorded as reviewed claims and allowed uses; this action is not a legal
guarantee or an expansion of the source Library review.

An accepted review appends an immutable qualified device revision whose review
record binds its parent candidate revision and digest. It does not edit or replace
the candidate. A stale head, changed candidate/source digest, incompatible allowed
uses, unsupported motion/support claim, or malformed evidence refuses promotion.
Reviewers may record a rejection against the exact revision; rejection preserves
the candidate and does not create a qualified revision. Each supported scope is
explicitly listed; callers cannot infer qualification beyond the recorded scopes.
The public API, CLI, and workbench all use this same owner operation. Qualified
family counts remain zero until an operator actually performs and records these
reviews after source and native checks.

The review scope is explicit. `physical_geometry` remains strict: each included
dimension needs known uncertainty, and each included display needs qualified
physical boundary, active area, and app-safe region. `visual_setup` records one
exact verified native setup (preset, display, support, motion, and dimension
state) without certifying hardware dimensions or display masks. It may include
finite, evidence-backed derived or image-estimated dimensions whose uncertainty
is unknown; that unknown status remains unchanged and the physical qualification
compatibility check continues to report it as needing review. Design assumptions
and missing dimension evidence are refused in either scope. A visual-setup review
must never be described as physical geometry qualification.

### Portable scoped review transfer

An accepted review can be exported as a bounded version-2 package while its
source project and completed native render job remain available. The package
contains the exact candidate and scene/motion/preset revisions, every Library
manifest/review and artifact byte referenced by the candidate geometry or
scene, the scoped review, native render evidence and inspected output bytes.
Import requires those exact immutable revisions and source bytes to already
exist in the destination. It rejects missing, duplicate, extra, stale, or
changed references, then reuses the existing qualification checks and
idempotent revision append. Import creates no render job, imports no Library
review, and does not broaden the reviewed dimensions, display, motion, support,
scene, or preset scope. Package hashes check integrity only; they do not
authenticate the reviewer or source owner. Import is an explicit operator
trust decision for transferring the recorded human review.

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
