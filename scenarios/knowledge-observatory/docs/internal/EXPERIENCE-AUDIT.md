# Experience Architecture Audit – Knowledge Observatory

## Purpose Statement

Knowledge Observatory helps Vrooli operators and agents inspect what semantic knowledge exists (and how healthy it is) so they can search, diagnose, and improve the system’s “memory” with confidence.

## Personas & Key Jobs

### First-time Explorer (agent/operator)
- Confirm the scenario is online and connected
- Run a first semantic search to understand what’s in memory
- Find where to view quality/health signals

### Returning Curator (builder/maintainer)
- Jump straight to Search to answer a question quickly
- Check metrics to decide whether to prune/merge/add knowledge
- Re-run workflows without losing context

### Ops Monitor (health/infra)
- Verify API health quickly
- Spot “offline/unhealthy” states without hunting
- Access the right surface (Search/Metrics) when triaging

## Flow Insights: Current vs. Ideal

| Issue | Current State | Ideal State |
|------|---------------|------------|
| Deep linking & back button | Feature cards opened modal dialogs, so there was no URL you could share/bookmark, and browser navigation didn’t reflect where you were. | Dedicated pages with routing so Search/Metrics/Graph are first-class destinations, bookmarkable and back/forward-friendly. |
| Working space for core jobs | Search/Metrics lived inside a constrained modal, limiting scan/read space and increasing context switching. | Full-page workspaces for Search/Metrics with persistent navigation to keep the user oriented. |
| “Where am I?” clarity | Users had to remember what they clicked to open the modal; closing it dropped them back to the dashboard with no sense of state. | A simple navigation model (Dashboard + tabs) that matches user intent: “monitor → search → assess quality → explore relationships”. |

## Changes Implemented (This Loop)

- Replaced modal-driven feature cards with hash-based routing + persistent tabs: `#/` (Dashboard), `#/search`, `#/metrics`, `#/graph`.
- Kept the existing Dashboard content as the “home” surface and converted feature cards into direct links to the full pages.
- Added a “Start a Knowledge Check” quick-action row so first-time and returning users can jump to Search/Metrics/Graph immediately.
- Improved system health clarity with icon-led error messaging and responsive status cards.
- Enhanced Search with guidance copy, sample queries, a clear action, and stronger empty/error states.
- Added metrics legend and responsive grids to clarify what “good” looks like at a glance.
- Introduced selector-driven `data-testid` hooks for key UI elements to support BAS automation.

## Opportunities for Future Loops

1. **Graph end-to-end wiring** – Implement the knowledge graph API (if missing) and add a minimal, dependency-free graph viewer (even a list/table-first “graph inspector” would surface value before full visualization).
2. **State persistence across navigation** – Preserve the last search query / last viewed collection metrics when switching tabs so returning curators can “continue where they left off”.
3. **Triage shortcuts** – Add “Go to Search” / “Go to Metrics” actions in the offline/unhealthy health panel to shorten ops troubleshooting loops.
4. **Search filters** – Add collection/threshold controls to narrow results without leaving the search context.

## Phase 1 UX Hardening – 2026-02-08

### Friction Addressed

- **Mechanical:** Collection actions could be applied from a row using preview state created for a different row.
- **Discoverability:** Metrics opened with no active collection, so diagnostics workflows were hidden behind an extra decision step.
- **Cognitive:** Operators could misread the collection list as KO-owned only; ownership confidence was not surfaced.

### Implemented in This Loop

- Bound maintenance apply buttons to row-specific previews instead of global selected-collection preview state.
- Auto-selected the first available collection to open diagnostics context immediately.
- Added explicit guidance that the collections list includes all Qdrant collections.
- Added inferred ownership badges and reasons (`Likely KO-managed`, `Likely external/mixed`, `Unknown ownership`).
- Added a compact “Playbook” in Integrity tab with concrete next-step sequencing.

### Remaining High-Value Follow-ups

1. Replace inferred ownership with backend-owned provenance metadata.
2. Add a collection record browser (paginated point/document inspection).
3. Add safe collection delete with dry-run and explicit confirmation semantics.

## Phase 2 API + UX Expansion – 2026-02-08

### Implemented in This Loop

- Added collection inventory endpoint with provenance signals:
  - `GET /api/v1/knowledge/collections`
  - Includes ownership key/label, total points, ingest attempts, metadata rows, distinct namespaces, and last ingest timestamp.
- Added collection record preview endpoint:
  - `GET /api/v1/knowledge/collections/{collection}/records`
  - Supports pagination (`limit`, `offset`) and filters (`namespace`, `document_id`, `search`).
- Updated Metrics page to consume backend inventory ownership metadata (instead of pure inference).
- Added a `Records` drilldown tab for collection-level content inspection with filters and next/previous pagination.

### Resulting Flow Improvement

- Operators can now investigate “what is actually in this collection” directly from Metrics without dropping to CLI/API.
- Ownership is now sourced from backend provenance signals, reducing ambiguity when external collections coexist with KO-managed collections.

## Migration Intent Contract – 2026-02-07

1. Intent:
Replace the existing hacker/terminal visual language with a clean, modern control-tower feel that improves readability, hierarchy, and confidence for operational usage.

2. References and non-goals:
- Reference direction: `scenarios/git-control-tower/ui` visual tone (clean dark surfaces, restrained accents, stronger spacing hierarchy).
- Non-goals: changing data flow, route model, API behavior, or replacing selector contracts used by BAS automation.

3. Constraints:
- Preserve all scenario workflows and route destinations.
- Preserve existing `data-testid` selectors and automation hooks.
- Maintain keyboard focus visibility and responsive behavior for desktop/mobile.
- Keep this migration UI-only (no backend or API contract changes).

4. Scope:
Token + primitive + layout refresh.

## Phase 3 Collection Debug UX – 2026-02-08

### Friction Addressed

- **Mechanical:** Collection diagnostics and maintenance were embedded in expanding cards, forcing dense scanning inside a crowded list.
- **Cognitive:** New users had to infer a maintenance sequence from many inline controls without a clear “debug workspace.”
- **Discoverability:** Record inspection, diagnostics tabs, and document-level cleanup lived behind card expansion state instead of a stable destination.

### Implemented in This Loop

- Added route-backed collection details pages at `#/collections/<collection-name>`.
- Created a full-page tabbed collection workspace (Integrity, Chunking, Failures, Records, Maintenance).
- Updated Metrics collection cards to act as entry points (`Open Details`) rather than running heavy maintenance inline.
- Kept the details route visually scoped under Metrics (Metrics nav remains active).

### Outcome

- Operators now move through a clear flow:
  1. Metrics overview (pick collection)
  2. Dedicated collection details workspace
  3. Run diagnostics/preview/apply with contextual tabs

- This creates a stable, shareable URL for collection debugging and reduces accidental misuse of maintenance controls.
