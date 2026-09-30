/**
 * Additional i18n/index.ts coverage — lines 98 (setLocale) and 109 (getCurrentLocale).
 *
 * The format.test.ts suite already covers formatters; this file covers the
 * helpers in index.ts that the existing tests miss.
 */
import { describe, it, expect, afterEach } from "vitest";

import { i18n, setLocale, getCurrentLocale, getLocaleConfig, SUPPORTED_LOCALES } from "./index";

afterEach(async () => {
  // Reset to cimode (the test pseudo-locale configured in test-setup.ts)
  await i18n.changeLanguage("cimode");
});

describe("setLocale (line 97-98)", () => {
  it("changes the active i18next language and resolves to undefined", async () => {
    const result = await setLocale("ja");
    expect(result).toBeUndefined();
    expect(i18n.language).toBe("ja");
  });

  it("changes back to 'en'", async () => {
    await setLocale("en");
    expect(i18n.language).toBe("en");
  });

  it("changes to 'ar' (RTL locale)", async () => {
    await setLocale("ar");
    expect(i18n.language).toBe("ar");
  });
});

describe("getCurrentLocale (line 108-109)", () => {
  it("returns 'en' when i18n.language is the cimode pseudo-locale (unsupported)", async () => {
    await i18n.changeLanguage("cimode");
    expect(getCurrentLocale()).toBe("en");
  });

  it("returns 'ja' when i18n.language is 'ja'", async () => {
    await i18n.changeLanguage("ja");
    expect(getCurrentLocale()).toBe("ja");
  });

  it("returns 'ar' when i18n.language is 'ar'", async () => {
    await i18n.changeLanguage("ar");
    expect(getCurrentLocale()).toBe("ar");
  });

  it("returns 'en' when i18n.language is an unsupported region tag", async () => {
    // 'en-AU' is not in SUPPORTED_LOCALES → falls back to 'en'
    await i18n.changeLanguage("en-AU");
    expect(getCurrentLocale()).toBe("en");
  });
});

describe("getLocaleConfig", () => {
  it("returns 'ltr' dir for English", () => {
    expect(getLocaleConfig("en").dir).toBe("ltr");
  });

  it("returns 'rtl' dir for Arabic", () => {
    expect(getLocaleConfig("ar").dir).toBe("rtl");
  });

  it("returns the native label for Japanese", () => {
    expect(getLocaleConfig("ja").nativeLabel).toBe("日本語");
  });
});

describe("SUPPORTED_LOCALES", () => {
  it("contains en, ja, and ar", () => {
    expect(SUPPORTED_LOCALES).toContain("en");
    expect(SUPPORTED_LOCALES).toContain("ja");
    expect(SUPPORTED_LOCALES).toContain("ar");
  });
});
