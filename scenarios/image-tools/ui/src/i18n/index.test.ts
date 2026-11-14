/**
 * i18n/index tests — the init module runs its side-effects once at import
 * (the singleton is already wired by the time any test imports it). These
 * tests cover the still-branching exported surface and the runtime side
 * effects that fire on a language change:
 *
 *   - setLocale → changeLanguage → <html lang/dir> + localStorage mirror
 *   - applyDocumentLocale for a supported (RTL) locale vs an unsupported one
 *     (falls back to the en config's ltr direction)
 *   - getCurrentLocale narrows a region-tagged / pseudo locale to a Locale
 *   - getLocaleConfig returns the native label + direction
 *   - SUPPORTED_LOCALES / LOCALE_CODES re-exports
 *
 * The setup file resets to `cimode` before every test, so each case
 * re-establishes the locale it needs.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import {
  getCurrentLocale,
  getLocaleConfig,
  i18n,
  LOCALE_CODES,
  setLocale,
  SUPPORTED_LOCALES,
} from "./index";

const STORAGE_KEY = "vrooli.locale";

describe("i18n locale config", () => {
  it("re-exports the canonical locale code tuple", () => {
    expect(SUPPORTED_LOCALES).toBe(LOCALE_CODES);
    expect([...LOCALE_CODES]).toEqual(["en", "ja", "ar"]);
  });

  it("exposes each locale's native label and direction", () => {
    expect(getLocaleConfig("en")).toEqual({ nativeLabel: "English", dir: "ltr" });
    expect(getLocaleConfig("ar")).toEqual({ nativeLabel: "العربية", dir: "rtl" });
  });
});

describe("setLocale side-effects", () => {
  afterEach(async () => {
    // Return to the test-default pseudo-locale so other suites are unaffected.
    await i18n.changeLanguage("cimode");
  });

  it("mirrors a supported LTR locale into <html> and localStorage", async () => {
    await setLocale("en");
    expect(document.documentElement.lang).toBe("en");
    expect(document.documentElement.dir).toBe("ltr");
    expect(window.localStorage.getItem(STORAGE_KEY)).toBe("en");
  });

  it("applies rtl direction for a supported RTL locale", async () => {
    await setLocale("ar");
    expect(document.documentElement.lang).toBe("ar");
    expect(document.documentElement.dir).toBe("rtl");
    expect(window.localStorage.getItem(STORAGE_KEY)).toBe("ar");
  });
});

describe("applyDocumentLocale fallback + languageChanged guard", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  afterEach(async () => {
    await i18n.changeLanguage("cimode");
  });

  it("falls back to the en (ltr) config and skips storage for an unsupported lng", async () => {
    // `cimode` is not in SUPPORTED_LOCALES → applyDocumentLocale uses the en
    // config (ltr) and the languageChanged handler's isSupported guard skips
    // the localStorage write.
    await i18n.changeLanguage("cimode");
    expect(document.documentElement.lang).toBe("cimode");
    expect(document.documentElement.dir).toBe("ltr");
    expect(window.localStorage.getItem(STORAGE_KEY)).toBeNull();
  });
});

describe("getCurrentLocale", () => {
  afterEach(async () => {
    await i18n.changeLanguage("cimode");
  });

  it("returns the active locale when it is supported", async () => {
    await setLocale("ja");
    expect(getCurrentLocale()).toBe("ja");
  });

  it("narrows an unsupported runtime language to en", async () => {
    await i18n.changeLanguage("cimode");
    expect(getCurrentLocale()).toBe("en");
  });
});

/**
 * `detectInitialLocale` runs exactly once, at module load. To exercise its
 * stored-choice / navigator-fallback branches we re-import the module fresh
 * under controlled `localStorage` + `navigator.language` so the one-shot
 * resolution takes a different path each time.
 */
describe("detectInitialLocale (fresh-import branches)", () => {
  const realLanguage = window.navigator.language;

  afterEach(() => {
    vi.resetModules();
    Object.defineProperty(window.navigator, "language", {
      value: realLanguage,
      configurable: true,
    });
    window.localStorage.clear();
  });

  const setNavigatorLanguage = (value: string) => {
    Object.defineProperty(window.navigator, "language", {
      value,
      configurable: true,
    });
  };

  it("honours a supported stored locale", async () => {
    vi.resetModules();
    window.localStorage.setItem(STORAGE_KEY, "ja");
    const mod = await import("./index");
    expect(mod.i18n.language).toBe("ja");
  });

  it("falls back to the navigator primary subtag when nothing is stored", async () => {
    vi.resetModules();
    window.localStorage.clear();
    setNavigatorLanguage("ar-EG");
    const mod = await import("./index");
    expect(mod.i18n.language).toBe("ar");
  });

  it("falls back to en when neither storage nor navigator is supported", async () => {
    vi.resetModules();
    window.localStorage.clear();
    setNavigatorLanguage("fr-FR");
    const mod = await import("./index");
    expect(mod.i18n.language).toBe("en");
  });
});
