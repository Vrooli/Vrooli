# Experience Architecture Audit

## Scenario Purpose
This scenario helps users **capture and connect thoughts in real-time** so they can **externalize their stream of consciousness into a navigable, graph-structured knowledge base**.

## Core Personas & Primary Jobs

| Persona | Primary Jobs |
|---------|-------------|
| Deep Thinker (primary) | 1. Capture thoughts as fast as they come. 2. See spatial relationships between ideas. 3. Connect related thoughts into a graph. |
| Reviewer | 1. Browse existing schemes. 2. Export thought graphs for sharing. 3. Check AI suggestions for missed connections. |

## Current vs Ideal Flows

### Deep Thinker: "Capture a thought"
- **Current**: Select scheme in sidebar -> Type in textarea -> Press Enter -> Item appears on canvas
- **Ideal**: Same (already optimized for sub-second entry via auto-focus [REQ:P0-002])
- **Friction**: None — flow is minimal

### Deep Thinker: "Connect two thoughts"
- **Current**: Switch to Graph view -> Click Link button -> Click source thought -> Click target thought
- **Ideal**: Same, but with **guidance text** explaining the 3-step process
- **Friction (fixed Phase 18)**: Link mode had no visual guidance — users had to guess what to click next. Added hint banner and aria-live announcements.

### Deep Thinker: "Navigate the canvas"
- **Current (pre-Phase 18)**: Mouse-only (drag to pan, scroll to zoom)
- **Ideal**: Mouse + keyboard (arrow keys to pan, +/- to zoom)
- **Friction (fixed Phase 18)**: Canvas was not keyboard-accessible. Added full keyboard navigation with ARIA application role.

### Reviewer: "Export a scheme"
- **Current**: Select scheme -> Click export button -> JSON downloads
- **Ideal**: Same (already a single-action flow)
- **Friction**: None

## Friction Points by Category

### Mechanical
- (None remaining for core flows)

### Cognitive
- **(Fixed)** Link mode: No guidance on what to do after activating link mode
- (Remaining) No undo for deleted thoughts/edges — destructive with no confirmation

### Discoverability
- **(Fixed Phase 18 iter 2)** Keyboard shortcuts not documented in UI — added "?" toggle overlay with all canvas shortcuts and persistent hint text
- (Remaining) AI suggestions panel (SuggestionList) is only visible when passed as props — not yet wired to a live suggestion API endpoint

## Improvements Implemented (Phase 18)
1. Link mode guidance banner with step-by-step instructions
2. Canvas keyboard navigation (arrow keys + zoom)
3. Aria-live announcements for zoom level and link mode state

## Navigation

### Intended Navigation Model
Single-page app with two navigation axes:
- **Scheme selection** (sidebar, horizontal): Choose which thought collection to work with
- **View mode** (header toggle, vertical): Canvas (spatial) vs Graph (relational)

No URL-based routing — state resets on page reload. This is appropriate for an iframe-embedded tool.

### Label-Destination Integrity
All navigation labels accurately match destinations:
- "Canvas view" / "Graph view" toggle buttons correctly switch view modes
- "Select scheme: {name}" in sidebar correctly activates the scheme
- "Export scheme" button correctly exports the active scheme
- "Link thoughts" / "Cancel linking" button accurately reflects link mode state

### Back/Forward Coherence
No back buttons exist (single-page, no routing). Browser back/forward exits the iframe context, which is expected behavior.

### Edge Cases
- **Deep links**: Not supported (no URL routing). Acceptable for iframe context.
- **Refresh**: Resets to no-scheme-selected state. Data is persisted server-side, so no data loss.
- **Multi-path navigation**: Not applicable — single entry point.

## Deferred Improvements
- Add undo for destructive actions (scheme/thought/edge delete)
- ~~Add keyboard shortcut help overlay (e.g., "?" key)~~ — implemented Phase 18 iter 2
- Wire SuggestionList to live API endpoint
- Add recent/favorite schemes for quick access
