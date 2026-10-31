# Experience — Device Studio

## Purpose Of This Document

Foundation contract, 2026-10-19, with current implementation notes below. The E1 Projects surface and D3 saved-shot editor have scoped implementation/evidence; the broader product behavior remains design intent. Research and pure presentation examples do not establish a released or accepted renderer.

The intended experience is specified in [DESIGN.md](../../DESIGN.md) and [experience/](../../experience/README.md). This document owns information architecture and primary surface rationale.

## The Comparison

The interaction reference is a focused product-shot studio: the visible stage and timeline of creative editing software, with a guided configuration path for users who do not know 3D modeling. This is a design analogy, not a claim of feature parity or a recommendation to buy another application. The photographic v6 outputs are the visual direction; the v5 scene finish remains below acceptance.

## E1 Project Surfaces

The E1 Projects surface creates, lists, opens and revision-safely renames durable local projects. The operator rejected the initial Studio Index visual proposal on 2026-10-20 as too basic and plain; that first design and its captures remain historical evidence only.

The approved Light-table direction in DESIGN.md presents the actual project and offers a clear continuation action. The Projects/Shot boards under `../mockups/revision-2/` are implemented in the E1 surface, with scoped browser evidence. D3 adds a saved-shot editor with immutable revisions, source selection, scene calibration, presentation choices, validation, and same-context stale-revision conflict/reapply. The real plate plus calibration overlay is not a final render. New projects show an honest no-shot/no-preview state. The expanded route inventory below remains draft except for the scoped implementation annotations and evidence.

## The Primary Surface

The current saved-shot editor keeps the selected scene plate visible beside contextual controls and exposes numeric calibration and presentation fields. It does not yet compose the final display raster in the shot editor or expose render/status/export actions. Future editor work retains the stage/inspector/timeline direction; mobile-specific timeline behavior, advanced camera/lighting and final render review remain unimplemented.

Every asynchronous region distinguishes loading, empty, partial, stale, unsupported, failed and success. Preserve drafts when leaving or recovering. A preview labels its strategy and approximation; a final-look render is a separate action. AI/agent actions disclose funding, scope, budget and external data before launch. Local reuse and deterministic edits do not start hidden work.

## Shell Configuration

Adopt the existing library console archetype for project/catalog/settings navigation, with a filled creative workspace for shot editing. Do not redraw global chrome. Theme, locale, accessibility and global preferences live in Settings; project budgets/privacy remain project settings. Concrete shell constants/tokens are planned in DESIGN and will be wired with real UI implementation; current scaffold values are not evidence of adoption.

## What The Design Is Accountable For

A new user can find/import a capture, choose a compatible device/scene, understand unsupported combinations and costs, configure readable presentation, render, retrieve direct files and reopen the exact shot. Creation/review users can see reference mismatches and candidate lineage. Keyboard/screen-reader users get labeled controls, focus/error feedback and alternatives to canvas dragging. Reduced motion affects editor feedback without silently altering an authored export.

## Information Architecture

Target route groups: project list/overview; searchable/filterable asset library/detail; import; creation workbench; review; shot builder; job queue/detail; result inspection/export; optional account/provider settings; project privacy/budget settings. [experience/index.json](../../experience/index.json) is the route-contract registry. Current product routes include `/`, `/projects`, and `/projects/:projectId`, with Settings at `/settings`; route contracts remain draft and distinguish partial implementation. Notes was removed by the governed detemplate operation. Library, public render UI, results and account routes remain planned.

## Cross-References

[DESIGN.md](../../DESIGN.md), [UI-ARCHITECTURE.md](UI-ARCHITECTURE.md), [PRD](../../PRD.md), [creation](../domains/creation.md), [acceptance](../internal/ACCEPTANCE.md).
