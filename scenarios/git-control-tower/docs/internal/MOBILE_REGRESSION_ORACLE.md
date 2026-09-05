# Mobile Regression Oracle

`experience/fixtures/mobile-regression-oracle.json` is the durable, versioned
oracle for the embedded Git Control Tower mobile regression. Its
`expected_before_repair` fields preserve the red-state contract, while its
`repaired_and_guarded` status records that the final app-shell repair and a
stateful BAS capture have closed that regression.

| Case | Symptom | Owner | Required proof |
| --- | --- | --- | --- |
| GCT-MOBILE-001 | Root fails to fill the embedded viewport | ui-health | Source, layout, screenshot |
| GCT-MOBILE-002 | Status/chrome has no declared rendered contract | Brand Manager + ui-health | Source, layout, screenshot |
| GCT-MOBILE-003 | Changes scroll is not reliably restored after returning from Diff | Experience Manager | Interaction trace, DOM, layout, screenshot |
| GCT-MOBILE-004 | Selection toolbar has incorrect scroll ownership | Experience Manager | Interaction trace, layout, screenshot |
| GCT-MOBILE-005 | Diff actions/footer are misplaced | Experience Manager | Interaction trace, layout, screenshot |
| GCT-MOBILE-006 | Bottom navigation geometry and safe-area ownership are incorrect | Experience Manager + ui-health | Interaction trace, layout, screenshot |

The stable capture profile is `mobile-embedded` at 390×844. Screenshots are
supporting evidence only: a case is never considered proven by an initial-page
image alone. The interaction trace must explicitly set up the selected tab and
scroll/selection/diff state named by the case.

This oracle is deliberately separate from product implementation. It is the
fixture that validation providers, experience claims, and the final repair
share; changing a case requires updating its evidence and owner rather than
suppressing a finding.
