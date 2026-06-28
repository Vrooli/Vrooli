import { afterEach, describe, expect, it } from "vitest";

import { getCurrentLocale, getLocaleConfig, i18n, setLocale, SUPPORTED_LOCALES } from ".";

describe("i18n index helpers", () => {
  afterEach(async () => {
    await setLocale("en");
    window.localStorage.clear();
  });

  it("exposes locale config and mirrors supported language changes", async () => {
    expect(SUPPORTED_LOCALES).toContain("ar");
    expect(getLocaleConfig("ar").dir).toBe("rtl");

    await setLocale("ar");

    expect(getCurrentLocale()).toBe("ar");
    expect(document.documentElement.lang).toBe("ar");
    expect(document.documentElement.dir).toBe("rtl");
    expect(window.localStorage.getItem("vrooli.locale")).toBe("ar");
  });

  it("falls back to en for unsupported runtime language values", async () => {
    await i18n.changeLanguage("cimode");
    expect(getCurrentLocale()).toBe("en");
    expect(document.documentElement.dir).toBe("ltr");
  });
});
