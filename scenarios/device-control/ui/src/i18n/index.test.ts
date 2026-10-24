import { afterEach, describe, expect, it } from "vitest";

import { getCurrentLocale, getLocaleConfig, i18n, setLocale } from "./index";

describe("i18n runtime helpers", () => {
  afterEach(async () => {
    await setLocale("en");
  });

  it("falls back to the canonical locale for unsupported runtime languages", async () => {
    await i18n.changeLanguage("dev");
    expect(getCurrentLocale()).toBe("en");
    expect(document.documentElement.lang).toBe("dev");
    expect(document.documentElement.dir).toBe("ltr");
    expect(getLocaleConfig("ar").dir).toBe("rtl");
  });
});
