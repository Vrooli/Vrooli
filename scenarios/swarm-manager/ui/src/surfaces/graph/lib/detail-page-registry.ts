/**
 * Detail Page Registry
 *
 * Single source of truth for which graph entity types have detail pages.
 * This prevents drift between the node click handler (which decides whether
 * to dim/highlight) and the GraphWorkspace renderer (which renders overlays).
 *
 * HOW IT WORKS:
 * - `DetailEntityType` is the union of entity types that have detail pages.
 *   Adding a new detail page means adding to that union + creating the page.
 * - `DETAIL_ENTITY_TYPES` is derived from `DetailEntityType` at the type level.
 *   If someone adds a new DetailEntityType but forgets to add it here,
 *   TypeScript will error because the Set literal won't satisfy the constraint.
 * - `hasDetailPage()` checks membership in the Set at runtime.
 *
 * WHY THIS EXISTS:
 * When a graph node is clicked for an entity WITH a detail page, the detail
 * overlay covers the entire graph. Applying dim/highlight is wasted work that
 * leaves stale visual state when the detail page closes. We only dim/highlight
 * for entity types WITHOUT detail pages (capture, agent-activity, agent-run),
 * where the user actually sees the graph with a focused node.
 */

import type { DetailEntityType } from "../../../stores/detail-selection-store";
import type { GraphEntityType } from "../types";

/**
 * Exhaustive set of entity types that have detail pages.
 *
 * The `satisfies` clause ensures this set stays in sync with DetailEntityType:
 * - If a new DetailEntityType is added but not listed here, TypeScript errors.
 * - If a type is listed here but removed from DetailEntityType, TypeScript errors.
 */
const DETAIL_ENTITY_TYPES: ReadonlySet<DetailEntityType> = new Set<DetailEntityType>([
  "backlog",
  "scenario",
  "execution",
  "initiative",
] as const satisfies readonly DetailEntityType[]);

/**
 * Returns true if the given graph entity type has an associated detail page.
 *
 * Used by the node click handler to decide whether to apply dim/highlight
 * (only for entities without detail pages) or to open the detail overlay
 * (for entities with detail pages, where dimming would be invisible).
 */
export function hasDetailPage(entityType: GraphEntityType): boolean {
  // Cast is safe: we're checking if a GraphEntityType is in the DetailEntityType set.
  // If it's not a DetailEntityType, .has() returns false.
  return DETAIL_ENTITY_TYPES.has(entityType as DetailEntityType);
}
