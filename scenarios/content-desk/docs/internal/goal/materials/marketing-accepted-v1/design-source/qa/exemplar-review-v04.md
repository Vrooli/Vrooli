# M06 v04 final exemplar recheck

Reviewed 2026-10-04 UTC. Independent bounded recheck of the corrections requested in `exemplar-review.md`.

## Scaling verdict: pass

**The corrected v04 exemplar is suitable for scaling the proposed visual family.** The earlier local scaling blockers are resolved. The actual screens preserve the C reference's task-led Review structure and now retain its useful evidence inspection affordance. This is a design-composition gate, not owner approval, runtime qualification, or acceptance of the entire screen/state family.

## Exact inspected artifacts

- `screens/M06-review-v04-desktop.png`: 2880×1800; SHA-256 `920c99a5035a4147c51f16861fec6214aad08632152138f593b2fe64a1982548`
- `screens/M06-review-v04-phone.png`: 780×1688; SHA-256 `a6b9e6e6b10228acd8687deb77b9dd8cf87ba6da95c719cfbaf9c97c5a47cb76`
- `screens/M06-review-v04-phone-changes.png`: 780×1688; SHA-256 `898361fae99cd58f0d8d25bab91a5e68411b664cf28c59ba6fb979d919282051`
- `qa/exemplar-composition-trial-1.png` and `qa/exemplar-composition-trial-2.png`: 850×1100 each, supplied renders for the internal document composition trial
- Targeted current `source/draw_exemplar.py` and `source/dossier.md` passages governing the corrections

## Closed findings

1. Desktop now names “Aster review story · source r3” and provides the adjacent “Inspect source” route. Phone also names the source and an Inspect route. The decision once again has inspectable evidence, rather than an anonymous revision label.
2. Phone clearly labels the saved draft as a synthetic example, says “2 decisions,” and gives the precise evidence limitation: “No timing study; speed gain unestablished.” The full post copy remains correct and unchanged.
3. Desktop Compare has a visible underline. Phone Content is a distinct button, removing the ambiguous “Content Menu” string. The previously fixed caret and correct 2× export geometry remain sound; no missing glyph, clipping or overlap was observed.
4. Phone Changes now shows exact prior-r1 and saved-r2 openings, states that the three remaining paragraphs are unchanged, and explains that restoring r1 creates a new revision, preserves r2 and requires a new review. “Preview restore as new draft” makes the next step clear. Opening the view explicitly changes no wording. This is sufficient evidence for the bounded compare/restore design gap from the first review.
5. The adjacent dossier paragraph now separates Create's pre-adoption “Keep original” from Review's saved-r2 restore flow. The stale “Keep current rejects that proposal” sentence is gone.

## Composition trial result

The desktop page works as an explicitly labeled layout overview. Its small in-screen text is not asked to carry the required decision meaning: native-sized explanation describes saved-r2 acceptance, restoration, the release boundary and the outstanding native contract. The phone page provides the complete readable short post and scoped action bar. Its 260-point figure width makes 20-pixel body text approximately 13.3 points, as the caption states. Both supplied trial renders have comfortable margins, clear reading order and no observed clipping.

This is evidence for the internal intended-size composition. It does not establish that the eventual native dossier embeds or exports identically; inspect the exact final native PDF after integration. The trial is not a canonical deliverable.

## Remaining scope

The full family still needs its explicit evidence-detail and anchored request-changes states, plus meaningful empty/error/stale/blocked states. Those were explicitly deferred and are not represented as complete here. The scoped native editorial-decision contract remains a prerequisite for operational acceptance. Static pictures do not prove actual hit targets, keyboard/focus behavior, responsive reflow or native-device behavior.

The earlier report remains the record of the initial findings and the first glyph/export corrections. Its hold on scaling the unchanged v03 pair is superseded by this v04 pass.
