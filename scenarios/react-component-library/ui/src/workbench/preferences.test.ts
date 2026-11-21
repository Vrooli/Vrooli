import { beforeEach, describe, expect, it } from "vitest";

import {
  DEFAULT_WORKBENCH_PREFERENCES,
  WORKBENCH_PREFERENCES_KEY,
  applyWorkbenchPreferences,
  readWorkbenchPreferences,
  writeWorkbenchPreferences,
} from "./preferences";

beforeEach(() => {
  window.localStorage.clear();
  delete document.documentElement.dataset.wbDensity;
  delete document.documentElement.dataset.wbReduceMotion;
});

describe("workbench preferences", () => {
  it("returns defaults when nothing is stored", () => {
    expect(readWorkbenchPreferences()).toEqual(DEFAULT_WORKBENCH_PREFERENCES);
  });

  it("round-trips a partial preference set over the defaults", () => {
    window.localStorage.setItem(
      WORKBENCH_PREFERENCES_KEY,
      JSON.stringify({ density: "compact", defaultDevice: "phone" }),
    );
    expect(readWorkbenchPreferences()).toMatchObject({
      density: "compact",
      defaultDevice: "phone",
      freshness: DEFAULT_WORKBENCH_PREFERENCES.freshness,
    });
  });

  it("survives malformed local storage", () => {
    window.localStorage.setItem(WORKBENCH_PREFERENCES_KEY, "{not json");
    expect(readWorkbenchPreferences()).toEqual(DEFAULT_WORKBENCH_PREFERENCES);
  });

  it("writes and applies density and motion to the document", () => {
    writeWorkbenchPreferences({
      ...DEFAULT_WORKBENCH_PREFERENCES,
      density: "compact",
      reduceMotion: true,
    });
    expect(readWorkbenchPreferences().density).toBe("compact");
    expect(document.documentElement.dataset.wbDensity).toBe("compact");
    expect(document.documentElement.dataset.wbReduceMotion).toBe("true");
  });

  it("applies explicit preferences without writing them", () => {
    applyWorkbenchPreferences({ ...DEFAULT_WORKBENCH_PREFERENCES, density: "comfortable" });
    expect(document.documentElement.dataset.wbDensity).toBe("comfortable");
    expect(window.localStorage.getItem(WORKBENCH_PREFERENCES_KEY)).toBeNull();
  });
});
