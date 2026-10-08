import { afterEach, describe, expect, it } from "vitest";
import { compositions, configureEnabledPacks, createScene } from "./index";

describe("composition pack registry", () => {
  afterEach(() => configureEnabledPacks(new Set(["vrooli"])));
  it("removes Vrooli-specific compositions when the pack is disabled", () => {
    configureEnabledPacks(new Set());
    expect(compositions["meridian-arc"]).toBeUndefined();
    expect(compositions["funnel-cascade"]).toBeUndefined();
    expect(createScene("meridian-arc")).toBeDefined();
  });
  it("restores bespoke compositions when enabled", () => {
    configureEnabledPacks(new Set(["vrooli"]));
    expect(compositions["meridian-arc"]).toBeDefined();
    expect(compositions["funnel-cascade"]).toBeDefined();
  });
});
