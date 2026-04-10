import type { ArchiveRequirementGroup } from "../types/domain";

/**
 * Recursively searches nested requirement groups by ID.
 * Extracted to eliminate duplication across BacklogDetailsPage handlers.
 */
export function findRequirementGroup(
  groups: ArchiveRequirementGroup[],
  groupId: string,
): ArchiveRequirementGroup | undefined {
  for (const g of groups) {
    if (g.id === groupId) return g;
    const found = findRequirementGroup(g.children, groupId);
    if (found) return found;
  }
  return undefined;
}
