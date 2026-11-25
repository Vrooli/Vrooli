import { describe, expect, it } from "vitest";

import type { CatalogAsset, WorkbenchStory } from "../data";
import {
  buildFixtureBench,
  fixtureViewportHeight,
  fixtureViewportWidth,
  storyVisualKey,
} from "./fixtureBench";

const asset = {
  id: "asset-1",
  displayName: "IconButton",
  catalogId: "controls.icon-button",
  slot: "ui-primitive",
  assetKind: "ASSET_KIND_COMPONENT",
} as unknown as CatalogAsset;

function story(overrides: Partial<WorkbenchStory> & Pick<WorkbenchStory, "id" | "name" | "role">) {
  return {
    key: `1.0.0:${overrides.id}`,
    args: {},
    fieldsJson: "{}",
    version: "1.0.0",
    sourcePath: "Asset.tsx",
    contractId: "asset@1",
    ...overrides,
  };
}

describe("fixture bench model", () => {
  it("collapses frames that mount the same specimen and args, preferring the live frame", () => {
    const anatomy = story({
      id: "default",
      name: "Default",
      role: "anatomy",
      mode: "live",
      specimen: { module: "./story.tsx", export: "Default" },
    });
    const pinnedDuplicate = story({
      id: "empty",
      name: "Empty label",
      role: "boundary",
      mode: "pinned",
      specimen: { module: "./story.tsx", export: "Default" },
    });
    const distinct = story({
      id: "danger",
      name: "Danger",
      role: "boundary",
      mode: "live",
      specimen: { module: "./story.tsx", export: "Danger" },
    });

    expect(storyVisualKey(anatomy)).toBe(storyVisualKey(pinnedDuplicate));

    const model = buildFixtureBench([anatomy, pinnedDuplicate, distinct]);
    expect(model.anatomy?.id).toBe("default");
    expect(model.hidden.map((entry) => entry.id)).toEqual(["empty"]);
    expect(model.visibleCount).toBe(2);
    expect(model.groups.flatMap((group) => group.stories).map((entry) => entry.id)).toEqual([
      "danger",
    ]);
  });

  it("keeps different args and direct stories distinct", () => {
    const first = story({ id: "one", name: "One", role: "axis", axis: "tone" });
    const second = story({
      id: "two",
      name: "Two",
      role: "axis",
      axis: "tone",
      args: { tone: "warning" },
    });
    const model = buildFixtureBench([first, second]);
    expect(model.hidden).toEqual([]);
    expect(model.groups[0]?.stories.map((entry) => entry.id)).toEqual(["one", "two"]);
  });

  it("orders anatomy, axes, then boundaries", () => {
    const model = buildFixtureBench([
      story({ id: "edge", name: "Edge", role: "boundary", specimen: { module: "./story.tsx", export: "Edge" } }),
      story({ id: "axis", name: "Axis", role: "axis", axis: "surface", specimen: { module: "./story.tsx", export: "Axis" } }),
      story({ id: "base", name: "Base", role: "anatomy", specimen: { module: "./story.tsx", export: "Base" } }),
    ]);
    expect(model.anatomy?.id).toBe("base");
    expect(model.groups.map((group) => group.id)).toEqual(["axis:surface", "boundaries"]);
  });

  it("brings authored axis values into the axis row", () => {
    const model = buildFixtureBench([
      story({ id: "default", name: "Ghost, circular", role: "anatomy", specimen: { module: "./story.tsx", export: "Default" } }),
      story({
        id: "surface-soft",
        name: "Soft standing surface",
        role: "axis",
        axis: "surface",
        covers: { surface: ["ghost", "soft", "solid", "danger"] },
        specimen: { module: "./story.tsx", export: "SurfaceSoft" },
      }),
      story({ id: "surface-solid", name: "Solid", role: "boundary", specimen: { module: "./story.tsx", export: "SurfaceSolid" } }),
      story({ id: "surface-danger", name: "Danger", role: "boundary", specimen: { module: "./story.tsx", export: "SurfaceDanger" } }),
    ]);
    const axis = model.groups.find((group) => group.id === "axis:surface");
    expect(axis?.values).toEqual(["ghost", "soft", "solid", "danger"]);
    expect(axis?.stories.map((entry) => entry.id)).toEqual([
      "surface-soft",
      "surface-solid",
      "surface-danger",
    ]);
    const boundaries = model.groups.find((group) => group.id === "boundaries");
    expect(boundaries).toBeUndefined();
  });

  it("uses authored geometry and falls back to a small control viewport", () => {
    const wide = story({
      id: "wide",
      name: "Wide",
      role: "anatomy",
      geometry: { width: "390px", height: "844px" },
    });
    expect(fixtureViewportWidth(wide, asset)).toBe(390);
    expect(fixtureViewportHeight(wide)).toBe(844);

    const plain = story({ id: "plain", name: "Plain", role: "anatomy" });
    expect(fixtureViewportWidth(plain, asset)).toBe(200);
    expect(fixtureViewportHeight(plain)).toBe(900);
  });
});
