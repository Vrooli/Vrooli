import { describe, expect, it } from "vitest";

import {
  countSurfacesByKind,
  decodeSurfaceId,
  encodeSurfaceId,
  filterSurfaces,
  type SurfaceRecord,
} from "./inventory";

function s(overrides: Partial<SurfaceRecord> = {}): SurfaceRecord {
  return {
    scenario: "ui-health",
    slot: "X",
    kind: "component",
    displayName: "X",
    description: "",
    filePath: "ui/src/X.tsx",
    ...overrides,
  };
}

describe("inventory helpers", () => {
  it("filterSurfaces returns all when filter is 'all'", () => {
    const surfaces = [s({ slot: "A", kind: "component" }), s({ slot: "B", kind: "page" })];
    expect(filterSurfaces(surfaces, "all")).toEqual(surfaces);
  });

  it("filterSurfaces narrows by kind", () => {
    const surfaces = [
      s({ slot: "A", kind: "component" }),
      s({ slot: "B", kind: "page" }),
      s({ slot: "C", kind: "page" }),
    ];
    expect(filterSurfaces(surfaces, "page").map((x) => x.slot)).toEqual(["B", "C"]);
  });

  it("countSurfacesByKind counts per kind, ignores unspecified", () => {
    const out = countSurfacesByKind([
      s({ kind: "component" }),
      s({ kind: "component" }),
      s({ kind: "page" }),
      s({ kind: "unspecified" }),
    ]);
    expect(out.all).toBe(4);
    expect(out.component).toBe(2);
    expect(out.page).toBe(1);
    expect(out.hook).toBe(0);
  });

  it("encode/decodeSurfaceId roundtrips with a slot", () => {
    const id = encodeSurfaceId("ui-health", "Dash");
    expect(id).toBe("ui-health__Dash");
    expect(decodeSurfaceId(id)).toEqual({ scenario: "ui-health", slot: "Dash" });
  });

  it("encode/decodeSurfaceId roundtrips an empty slot via the '_' sentinel", () => {
    const id = encodeSurfaceId("ui-health", "");
    expect(id).toBe("ui-health___");
    expect(decodeSurfaceId(id)).toEqual({ scenario: "ui-health", slot: "" });
  });

  it("decodeSurfaceId returns null for malformed input", () => {
    expect(decodeSurfaceId("no-separator")).toBeNull();
    expect(decodeSurfaceId("__no-scenario")).toBeNull();
  });
});
