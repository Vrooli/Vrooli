/**
 * Detail Page Registry
 *
 * Single source of truth for which graph entity types have detail pages.
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
 * The NodeInspectorPanel uses this to decide whether to show an "Open Details"
 * button. Entity types with detail pages get the button; those without
 * (agent-activity, agent-run) only show lens navigation.
 */

import type { DetailEntityType } from "../../../app/routes/route-paths";
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
  "capture",
] as const satisfies readonly DetailEntityType[]);

/**
 * Returns true if the given graph entity type has an associated detail page.
 *
 * Used by the NodeInspectorPanel to decide whether to show an "Open Details"
 * button for the selected node.
 */
export function hasDetailPage(entityType: GraphEntityType): boolean {
  // Cast is safe: we're checking if a GraphEntityType is in the DetailEntityType set.
  // If it's not a DetailEntityType, .has() returns false.
  return DETAIL_ENTITY_TYPES.has(entityType as DetailEntityType);
}
