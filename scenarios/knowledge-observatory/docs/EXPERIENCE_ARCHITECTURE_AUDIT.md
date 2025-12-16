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

## Opportunities for Future Loops

1. **Graph end-to-end wiring** – Implement the knowledge graph API (if missing) and add a minimal, dependency-free graph viewer (even a list/table-first “graph inspector” would surface value before full visualization).
2. **State persistence across navigation** – Preserve the last search query / last viewed collection metrics when switching tabs so returning curators can “continue where they left off”.
3. **Triage shortcuts** – Add “Go to Search” / “Go to Metrics” actions in the offline/unhealthy health panel to shorten ops troubleshooting loops.

