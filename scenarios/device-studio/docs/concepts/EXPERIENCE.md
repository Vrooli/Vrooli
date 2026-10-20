# Experience — Device Studio

## Purpose Of This Document

Foundation contract, 2026-10-19. Product behavior below is specified for implementation; the generated reference scaffold does not implement it. Research demonstrates individual techniques, not a released scenario.

The intended experience is specified in [DESIGN.md](../../DESIGN.md) and [experience/](../../experience/README.md). This document owns information architecture and primary surface rationale.

## The Comparison

The interaction reference is a focused product-shot studio: the visible stage and timeline of creative editing software, with a guided configuration path for users who do not know 3D modeling. This is a design analogy, not a claim of feature parity or a recommendation to buy another application. The photographic v6 outputs are the visual direction; the v5 scene finish remains below acceptance.

## E1 Project Surfaces

The first implemented functional prototype creates a durable local project, lists and opens it by stable ID, and renames it against the revision observed by the user. It does not implement the shot workflow. The operator rejected its Studio Index visual proposal on 2026-10-20 as too basic and plain; the initial mockups and runtime captures are historical prototype evidence only.

The approved Light-table studio direction in DESIGN.md presents projects through their actual shot imagery and a strong resume action, then opens an immersive editor with contextual controls. The operator approved the Projects/Shot desktop/mobile boards under `../mockups/revision-2/`; the remaining boards extend that direction. These mockups do not prove live functionality. A new project still starts empty, and no imagery, render availability or successful job state is fabricated. The expanded product route inventory below remains draft except for E1 routes proven by implementation and their scoped evidence.

## The Primary Surface

The shot builder keeps device/scene and real screen content visible while users select presentation, motion and output. Desktop uses a large stage, a library/selection region, contextual inspector and timeline. Advanced camera, lighting, calibration and numeric controls remain available. Mobile transforms into Stage/Controls/Timeline tabs with sequential fields and numeric alternatives rather than squeezing desktop panes.

Every asynchronous region distinguishes loading, empty, partial, stale, unsupported, failed and success. Preserve drafts when leaving or recovering. A preview labels its strategy and approximation; a final-look render is a separate action. AI/agent actions disclose funding, scope, budget and external data before launch. Local reuse and deterministic edits do not start hidden work.

## Shell Configuration

Adopt the existing library console archetype for project/catalog/settings navigation, with a filled creative workspace for shot editing. Do not redraw global chrome. Theme, locale, accessibility and global preferences live in Settings; project budgets/privacy remain project settings. Concrete shell constants/tokens are planned in DESIGN and will be wired with real UI implementation; current scaffold values are not evidence of adoption.

## What The Design Is Accountable For

A new user can find/import a capture, choose a compatible device/scene, understand unsupported combinations and costs, configure readable presentation, render, retrieve direct files and reopen the exact shot. Creation/review users can see reference mismatches and candidate lineage. Keyboard/screen-reader users get labeled controls, focus/error feedback and alternatives to canvas dragging. Reduced motion affects editor feedback without silently altering an authored export.

## Information Architecture

Target route groups: project list/overview; searchable/filterable asset library/detail; import; creation workbench; review; shot builder; job queue/detail; result inspection/export; optional account/provider settings; project privacy/budget settings. [experience/index.json](../../experience/index.json) is the authoritative planned/actual route registry for this foundation. Current actual template routes are `/`, `/notes`, `/settings`; future product routes stay draft until reconciled against implementation.

## Cross-References

[DESIGN.md](../../DESIGN.md), [UI-ARCHITECTURE.md](UI-ARCHITECTURE.md), [PRD](../../PRD.md), [creation](../domains/creation.md), [acceptance](../internal/ACCEPTANCE.md).
