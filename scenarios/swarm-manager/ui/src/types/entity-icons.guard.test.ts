/**
 * Guard test for the entity-icon single source of truth.
 *
 * These assertions fail loudly if a future change re-introduces icon drift —
 * e.g. a goal surface hardcoding an icon or a sidebar tab whose icon diverges
 * from its entity.
 */

import { describe, expect, it } from "vitest";
import { ENTITY_TYPE_ICONS, SIDEBAR_TAB_ICONS, type EntityType, type SidebarEntityType } from "./constants";
import { ENTITY_REGISTRY, getEntityIcon, GRAPH_ENTITY_TYPES } from "../surfaces/graph/lib/entity-shapes";

/**
 * Sidebar tabs that represent an entity type. Each must resolve to the exact
 * same icon component as its entity in ENTITY_TYPE_ICONS.
 */
const SIDEBAR_ENTITY_PAIRS: Array<[SidebarEntityType, EntityType]> = [
  ["backlog", "backlog"],
  ["captures", "capture"],
  ["goals", "goal"],
  ["executions", "execution"],
];

describe("entity icon SSOT", () => {
  it("resolves every entity-backed sidebar tab to its entity's SSOT icon", () => {
    for (const [tab, entity] of SIDEBAR_ENTITY_PAIRS) {
      expect(SIDEBAR_TAB_ICONS[tab]).toBe(ENTITY_TYPE_ICONS[entity]);
    }
  });

  it("resolves graph-node icons from the same SSOT", () => {
    for (const entityType of GRAPH_ENTITY_TYPES) {
      // Every graph entity type is also a key of ENTITY_TYPE_ICONS.
      expect(getEntityIcon(entityType)).toBe(ENTITY_TYPE_ICONS[entityType as EntityType]);
      expect(ENTITY_REGISTRY[entityType].icon).toBe(ENTITY_TYPE_ICONS[entityType as EntityType]);
    }
  });
});
