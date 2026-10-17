# M06 Review 0.3: independent visual and product-fidelity review

Reviewed 2026-10-04 UTC. This is a bounded review of proposed static design artifacts, not runtime, owner-contract or implementation acceptance. No canonical documents were changed.

## Verdict

**The core composition passes; do not scale the current pair unchanged.** It retains the owner-selected C direction and improves the revision semantics, but loses some of C's inspectability and control clarity. Repair the small evidence/provenance/phone-state gaps below, then use the composition as the dense-page exemplar. The glyph and phone-export defects found in the first inspection have already been corrected and independently rechecked.

This is not a pass based on deterministic/vector production. At a matched 1440-pixel desktop width, the actual result looks like a coherent descendant of C: ivory canvas, cobalt navigation and action hierarchy, restrained separators, sans-serif application controls and a dominant serif content artifact. The queue, comparison, evidence and decision bar give the page a useful working form. The simplified evidence panel is nevertheless a product-fidelity regression against C Review's explicit source inspection affordance.

## Exact scope and evidence

Inspected these exact artifacts and their authoring source:

- `screens/M06-review-v03-desktop.png`, including a matched 1440×900 reading-scale view and a full-resolution header crop
- `screens/M06-review-v03-phone.png`, including its current 390×844 logical reading-scale view
- `source/draw_exemplar.py`
- `brief.md`
- The four owner-selected `Marketing-C-{Review,Create,Plan,Learn}-task-led-v2.png` references under `/workspace/scratch/fc82fad4f3b1/marketing-ideation/task-led-v2/`
- Targeted surrounding contract text in `source/dossier.md`, especially revision/package semantics and responsive contracts

Current rechecked PNGs after the first two corrections:

- Desktop: 2880×1800; SHA-256 `4d6b18e56c191b125238a41930682f90e4c728362ae8903375fb9af60c6ab7e3`
- Phone: 780×1688; SHA-256 `1c807c46968d400df856d23a637896acdec4a06e80b250057c4541b2de7a859d`

The source reference is `source/draw_exemplar.py`, not the root-level path originally named in the handoff.

## What passes

**Task purpose without title.** Hiding “Review” leaves a selected decision, a clearly marked prior/current text comparison, evidence for that decision, and Accept wording / Request changes. It is recognizably a wording review, not a homepage with a review title. Phone similarly reads as an exact content reading view with a bounded editorial decision.

**The correct revision transition.** Desktop labels r1 “Previous saved wording” and r2 “Current saved draft”; the queue says “Saved draft r2.” This is materially better than the older C screenshot's current-r1/proposed-r2 wording after Create has already applied and saved r2. Accept wording assesses the existing r2 rather than creating it. The dossier describes restoring r1 as a guarded new r3 that preserves r2.

**Content fidelity.** Both current specimens preserve the new opening exactly: “Put the evidence beside the decision.” The three following paragraphs match each other and the C reference, including the explicit statement that speed improvement has not been measured. No unexpected claim or abbreviated body text was introduced. Desktop makes the sole changed paragraph visually explicit with minus/plus as well as color.

**Density and hierarchy.** The full short artifact fits comfortably without cramped text, clipping, overlapping controls or gratuitous cards. The two columns remain readable at desktop scale. The bottom action bar is visually separate and repeats r2's scope. More whitespace than C is acceptable here; it does not remove the main comparison. That whitespace should not become a reason to omit evidence identity or working controls on other dense pages.

**Release boundary.** Desktop says that no account or release permission is granted; phone says wording r2 only, no publication. Neither action bar claims to schedule or publish. The general dossier explicitly distinguishes the proposed destination-unbound assessment from the existing native ApproveDraft/type gates. An enabled-looking button in a proposed fixture is not proof that the required native contract exists; preserve that qualification in the delivered figure context and implementation acceptance.

**Phone structure and sizing after correction.** A final-reading view with Changes and Evidence routes is an appropriate adaptation of the desktop comparison. The prose remains the main object. At 390 logical pixels, the primary buttons are 44 pixels high and fit with a 14-pixel gap; all body text is visible. Bottom navigation has a clear active Review state. This is static geometry evidence only, not verified hit targets, focus, scrolling or native-device behavior.

## Corrections required before scaling unchanged

### 1. Restore specific, inspectable evidence in the specimen

Desktop currently reduces the source to “Structure supported · source r3” plus an assertion. C Review provides an “Inspect source” action; C Create names Aster review story rev 3 and the other two sources. The new specimen has a top Evidence tab, but the visible support beside the decision no longer identifies which source revision or passage can be inspected. Because draft r1/r2 and source r3 coexist, anonymous revision labels are especially weak.

Show a named source such as “Aster review story · source r3” and an explicit “Inspect source” action beside the summary. Use the source detail state to preserve locator, scope and the rest of the three-source list. Phone should also name the supporting source and expose its inspection route. This can be repaired within the existing whitespace and need not reinstate every decorative icon from C.

### 2. Keep phone meaning as precise as desktop

“No measured speed claim” is less precise than “No timing study. This limit remains in r2.” It can be read as a statement about wording rather than evidence. Prefer a short version such as “No timing study; speed gain unestablished.” The current body itself is correct.

The phone has no synthetic/proposed marker, while desktop has both “Synthetic revision” and an explicit static-fixture footer. Add an unobtrusive specimen label in the phone or in an inseparable figure frame. An independently circulated phone PNG should not resemble a claimed runtime screenshot.

“2 waiting” is ambiguous beside a user-decision view when the desktop separately has “Waiting on others.” “2 decisions” is clearer for this specific header. This is a small copy correction, not a new workflow.

### 3. Complete the consequential phone state, rather than claim parity from routes alone

The current phone pair has a Changes (1) route but no inspected Changes state. Desktop offers “Return to r1…”; no equivalent is visible in the phone reading view. It is reasonable to put the restore proposal in Changes, but that destination must be specified and subsequently shown, with a precise “Use r1 wording in a new revision” consequence and preservation of r2. Similarly, Request changes needs a concrete anchored-request state. These need not all be added to the first phone screen.

The pair therefore passes reading/accept/request scope parity, but does not yet establish complete compare/restore/evidence parity. Carry these explicit states into the connected screen set before calling it complete.

### 4. Resolve small control-affordance inconsistencies

Desktop Compare is selected only by weight/color, while the surrounding selected modes and phone Read final use an underline. Add the same visible selected marker; this also reduces reliance on cobalt alone.

Phone renders the top-right text as “Content Menu” from a single string substitution. Make the intended separate actions visibly distinct, or use one clearly named menu entry. A static picture cannot prove touch areas, so the contract should assign adequate targets to each action rather than rely on the word shapes.

Search becoming a simple text action and removal of the explicit decision filter are acceptable simplifications for this two-item fixture, provided the larger queue and Content search states supply those capabilities. They are not reasons to copy C pixel for pixel.

## Defects found and closed during this review

1. **Missing desktop caret glyph.** The first inspected file visibly contained a square after “Vrooli · All work.” The authoring replacement used U+2304, absent from the actual Open Sans Semibold font cmap. The parent replaced it with a drawn two-segment chevron. The current full-resolution crop shows a clean chevron; closed.
2. **Phone export unexpectedly shrunk.** The first phone PNG was 361×780 because `pdftoppm -scale-to width*2` constrained the larger dimension. The intended source was 390×844, so the exported controls and text were unexpectedly reduced. The parent now supplies explicit `-scale-to-x width*2` and `-scale-to-y height*2`; the current phone is 780×1688, and its 390×844 review view is legible. Desktop remains 2880×1800; closed.

Original PNG hashes, retained as inspection evidence:

- Desktop before correction: `2f4b60cd535218e6f9935049534ae553a781e31d98d5c65bc0f71f78d8d89398`
- Phone before correction: `f3e480725c400f1d9e44817857175c2ee93f44157c4508bb58aeecca773ebb56`

No remaining missing glyphs were observed in the rechecked current pair. The minus, plus, guillemet-style navigation, ellipsis and text labels render. This is not an exhaustive multilingual font-coverage test.

## Related source consistency note

The targeted dossier text correctly explains the saved-r2 and restore-to-new-r3 model, but the adjacent action-bar paragraph still says “Keep current rejects that proposal.” That wording belongs to the older unapplied-proposal flow and should be scoped to Create's proposal context or replaced for Review. Do not allow that leftover phrase to redefine r2 as an unapplied candidate in later screens.

## Scaling and delivery gate

Proceed with the visual language and task-led composition after the small evidence, label and selection-marker fixes. Preserve the corrected export settings and vector icon approach. Carry the unresolved phone detail states as explicit work, and review them before declaring desktop/phone parity. Native dossier use should follow the parent's stated plan: a labeled desktop overview for composition, readable phone/focused detail for consequential controls, native explanatory text, and full-size atlas access. Exact embedded/exported PDF reading-scale inspection remains a later gate and is not established by this review.
