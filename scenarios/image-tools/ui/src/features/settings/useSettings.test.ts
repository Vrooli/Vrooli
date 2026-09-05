/**
 * Unit tests for the display/accessibility settings store — the pure
 * persist + apply layer behind <SettingsProvider>. Covers the localStorage
 * adapter (incl. corrupt/partial blobs) and the `<html>` attribute writes.
 */
import { afterEach, beforeEach, describe, expect, it } from "vitest";

import {
  DEFAULT_SETTINGS,
  SETTINGS_STORAGE_KEY,
  applySettings,
  readStoredSettings,
  resolveDirection,
  writeStoredSettings,
  type SettingsState,
} from "./useSettings";
import { setLocale } from "../../i18n";

const root = () => document.documentElement;

describe("readStoredSettings", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("returns defaults when nothing is stored", () => {
    expect(readStoredSettings()).toEqual(DEFAULT_SETTINGS);
  });

  it("round-trips a persisted blob", () => {
    const custom: SettingsState = {
      fontScale: "large",
      reducedMotion: "always",
      textDirection: "rtl",
      handedness: "left",
    };
    writeStoredSettings(custom);
    expect(readStoredSettings()).toEqual(custom);
  });

  it("falls back per-field for a partial blob", () => {
    window.localStorage.setItem(SETTINGS_STORAGE_KEY, JSON.stringify({ fontScale: "xlarge" }));
    expect(readStoredSettings()).toEqual({ ...DEFAULT_SETTINGS, fontScale: "xlarge" });
  });

  it("falls back per-field for unknown values", () => {
    window.localStorage.setItem(
      SETTINGS_STORAGE_KEY,
      JSON.stringify({ fontScale: "gigantic", handedness: "left" }),
    );
    expect(readStoredSettings()).toEqual({ ...DEFAULT_SETTINGS, handedness: "left" });
  });

  it("returns defaults for a corrupt (non-JSON) blob", () => {
    window.localStorage.setItem(SETTINGS_STORAGE_KEY, "{not json");
    expect(readStoredSettings()).toEqual(DEFAULT_SETTINGS);
  });
});

describe("applySettings", () => {
  beforeEach(async () => {
    await setLocale("en");
    for (const attr of ["data-font-scale", "data-reduced-motion", "data-handedness"]) {
      root().removeAttribute(attr);
    }
    root().dir = "";
  });

  afterEach(() => {
    for (const attr of ["data-font-scale", "data-reduced-motion", "data-handedness"]) {
      root().removeAttribute(attr);
    }
    root().dir = "";
  });

  it("clears every attribute for the default settings", () => {
    applySettings(DEFAULT_SETTINGS);
    expect(root().hasAttribute("data-font-scale")).toBe(false);
    expect(root().hasAttribute("data-reduced-motion")).toBe(false);
    expect(root().hasAttribute("data-handedness")).toBe(false);
    // Auto text direction follows the (en → ltr) locale.
    expect(root().dir).toBe("ltr");
  });

  it("writes the font-scale attribute for a non-default scale", () => {
    applySettings({ ...DEFAULT_SETTINGS, fontScale: "large" });
    expect(root().getAttribute("data-font-scale")).toBe("large");
  });

  it("writes the reduced-motion attribute for always/never", () => {
    applySettings({ ...DEFAULT_SETTINGS, reducedMotion: "always" });
    expect(root().getAttribute("data-reduced-motion")).toBe("always");
    applySettings({ ...DEFAULT_SETTINGS, reducedMotion: "never" });
    expect(root().getAttribute("data-reduced-motion")).toBe("never");
  });

  it("writes the handedness attribute only for the non-default (left) hand", () => {
    applySettings({ ...DEFAULT_SETTINGS, handedness: "left" });
    expect(root().getAttribute("data-handedness")).toBe("left");
    applySettings({ ...DEFAULT_SETTINGS, handedness: "right" });
    expect(root().hasAttribute("data-handedness")).toBe(false);
  });

  it("forces dir to an explicit ltr/rtl override regardless of locale", () => {
    applySettings({ ...DEFAULT_SETTINGS, textDirection: "rtl" });
    expect(root().dir).toBe("rtl");
    applySettings({ ...DEFAULT_SETTINGS, textDirection: "ltr" });
    expect(root().dir).toBe("ltr");
  });
});

describe("resolveDirection", () => {
  afterEach(async () => {
    await setLocale("en");
  });

  it("returns the explicit choice for ltr/rtl", () => {
    expect(resolveDirection("ltr")).toBe("ltr");
    expect(resolveDirection("rtl")).toBe("rtl");
  });

  it("follows the active locale for auto", async () => {
    await setLocale("en");
    expect(resolveDirection("auto")).toBe("ltr");
    await setLocale("ar");
    expect(resolveDirection("auto")).toBe("rtl");
  });
});
