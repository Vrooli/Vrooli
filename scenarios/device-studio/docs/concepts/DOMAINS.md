# Domains — Device Studio

## Purpose Of This Document

Foundation contract, 2026-10-19. Product behavior below is specified for implementation; the generated reference scaffold does not implement it. Research demonstrates individual techniques, not a released scenario.

The table describes target ownership. Named API directories are planned, not existing product implementations.

## Domain Inventory

| Domain | Responsibility | Owns Data | Primary Archetype | Source Paths |
|---|---|---|---|---|
| projects | Project identity, membership, media references and project policies | Projects and policy revisions | service | docs/concepts/DATA.md|
| devices | Measured hardware, parts, displays and adaptations | Device revisions and source/dimension evidence | service | docs/domains/devices.md|
| scenes | Environments, supports, mounts, cameras and lights | Scene revisions and compatible placement slots | service | docs/domains/rendering.md|
| presentations | App viewport and simulated OS/browser chrome | Profiles, masks and source fit policy | service | docs/domains/presentations.md|
| motions | Device/joint/camera tracks and constraints | Reusable motion revisions | service | docs/domains/rendering.md|
| shots | Join immutable inputs and compile frame schedules | Shot drafts/revisions and compiler results | validation | docs/domains/rendering.md|
| jobs | Admit/execute/checkpoint/reconcile work | Jobs, attempts and owner receipt references | orchestration | docs/concepts/FLOWS.md|
| creations | Search/import/build/compare/refine candidates | Creation briefs, candidate lineage and iteration decisions | orchestration | docs/domains/creation.md|
| reviews | Physical, presentation and craft qualification | Scoped reviews and invalidation decisions | validation | docs/internal/ACCEPTANCE.md|

## Domain Details

Projects own policy revisions and media association, not secrets or wallets. Library domains own domain-specific definitions and adaptations, not a second global media registry. Shots read pinned library revisions and project policy. Jobs accept compiled work and record execution; renderer adapters evaluate it without mutating saved shots. Creations orchestrate candidate writes through library APIs and job attempts. Reviews evaluate exact revisions/artifacts and scope; generic Asset Studio publication is a separate external operation.

Avoid circular ownership: a job receipt is immutable evidence referenced by a review; a review does not update a past job's success. A candidate can be technically rendered but visually rejected. Asset metadata references shared artifact identity, while device qualification records semantic associations.

## Shared Concepts

A device definition is physical/design intent; an asset revision is acquired/authored geometry and material bytes; a scene supplies context and support; a presentation maps media/chrome to displays; a motion describes time-varying state; a shot joins them. A render is one execution of one immutable compiled shot. A review qualifies a bounded use envelope, not the entire product family. Use meters, seconds and explicit coordinate frames; preserve original source units.

## Deferred Domains

No general mesh editor, universal asset marketplace, full-film editor, credit wallet or host installer is added. Optical devices, flexible-screen dynamics, physics-driven straps, 4K/HDR and hybrid reconstruction may extend existing contracts after separate qualification.

## Non-Domains

SQLite, FFmpeg, Blender, Connect, preview components and dependency discovery are mechanisms. They must not become authority-bearing generic service buckets. The template notes domain remains a fenced worked example until a real slice replaces it.

## Cross-References

[ARCHITECTURE.md](ARCHITECTURE.md), [FLOWS.md](FLOWS.md), [DATA.md](DATA.md), [SEAMS.md](../internal/SEAMS.md).
