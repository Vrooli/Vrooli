import { describe, expect, it } from "vitest";
import { hasDetailPage } from "./detail-page-registry";
import type { GraphEntityType } from "../types";

describe("hasDetailPage", () => {
  it.each<[GraphEntityType, boolean]>([
    ["backlog", true],
    ["scenario", true],
    ["execution", true],
    ["goal", true],
    ["capture", true],
    ["agent-activity", false],
    ["agent-run", false],
  ])("returns %s for entity type '%s'", (entityType, expected) => {
    expect(hasDetailPage(entityType)).toBe(expected);
  });

  it("returns false for unknown entity types", () => {
    // Defensive: if a new entity type is added to GraphEntityType but not
    // to DetailEntityType, hasDetailPage should return false.
    expect(hasDetailPage("unknown-type" as GraphEntityType)).toBe(false);
  });
});
