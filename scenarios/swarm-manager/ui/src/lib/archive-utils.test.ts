import { describe, it, expect } from "vitest";
import { findRequirementGroup } from "./archive-utils";
import type { ArchiveRequirementGroup } from "../types/domain";

const makeGroup = (id: string, children: ArchiveRequirementGroup[] = []): ArchiveRequirementGroup => ({
  id,
  name: `Group ${id}`,
  requirements: [],
  children,
});

describe("findRequirementGroup", () => {
  it("returns undefined for empty array", () => {
    expect(findRequirementGroup([], "any")).toBeUndefined();
  });

  it("finds a flat (top-level) match", () => {
    const groups = [makeGroup("a"), makeGroup("b"), makeGroup("c")];
    expect(findRequirementGroup(groups, "b")).toBe(groups[1]);
  });

  it("finds a nested match", () => {
    const nested = makeGroup("deep");
    const groups = [makeGroup("a", [makeGroup("b", [nested])])];
    expect(findRequirementGroup(groups, "deep")).toBe(nested);
  });

  it("returns undefined when no match exists", () => {
    const groups = [makeGroup("a", [makeGroup("b")])];
    expect(findRequirementGroup(groups, "missing")).toBeUndefined();
  });
});
