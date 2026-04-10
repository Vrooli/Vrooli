// @vitest-environment node
import { describe, expect, it } from "vitest";
import { selectorsManifest } from "../consts/selectors";

// [REQ:REQ-UI-012] Retention & storage settings — selector coverage
describe("retention settings selectors", () => {
  it("includes retention configuration selectors", () => {
    expect(selectorsManifest.selectors["settings.retentionTime"]).toBeDefined();
    expect(selectorsManifest.selectors["settings.retentionSize"]).toBeDefined();
    expect(selectorsManifest.selectors["settings.pruneInterval"]).toBeDefined();
  });

  it("settings selectors produce correct data-testid format", () => {
    const retention = selectorsManifest.selectors["settings.retentionTime"];
    expect(retention?.selector).toBe('[data-testid="settings-retention-time"]');
    const size = selectorsManifest.selectors["settings.retentionSize"];
    expect(size?.selector).toBe('[data-testid="settings-retention-size"]');
    const interval = selectorsManifest.selectors["settings.pruneInterval"];
    expect(interval?.selector).toBe('[data-testid="settings-prune-interval"]');
  });

  it("all three retention knobs are addressable individually", () => {
    const keys = Object.keys(selectorsManifest.selectors).filter((k) => k.startsWith("settings."));
    expect(keys.length).toBeGreaterThanOrEqual(3);
  });
});
