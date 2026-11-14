# Choosing and implementing the UI

## Purpose Of This Document

Apply the accepted guided-plus-advanced studio design while preserving library ownership and honest experience maturity.

## 1. Decide

The primary surface and responsive transformations are recorded in [EXPERIENCE.md](../concepts/EXPERIENCE.md). Read root DESIGN and draft experience states before implementation. Preserve the stage, control clarity and readable source media; avoid a general-purpose mesh editor.

## 2. Configure

Use the existing library shell declaration. Configure destinations, filled workspace, mobile navigation and design tokens when implementing the real home/shot views. The source orientation design gate remains unmet until those UI changes exist; deleting a marker alone does not qualify it.

## 3. Adopt

Use `react-component-library adoptions suggest device-studio --json`, then governed linking for useful primitives. Missing states/components are owner-managed drafts, never edits to released versions. Record unresolved gaps in [component-library-gaps.md](../reference/component-library-gaps.md).

## 4. Build

Implement one domain slice across proto/API/CLI/UI. Add planned controls/states with actual accessibility and selector bindings. Validate keyboard/reduced-motion/mobile behavior and error recovery. Promote experience claims only from real evidence. Return reusable generic components to the library when qualified.

## Cross-references

[DESIGN.md](../../DESIGN.md), [UI architecture](../concepts/UI-ARCHITECTURE.md), [experience](../../experience/README.md).
