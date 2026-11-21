import { describe, expect, it } from "vitest";

import { en } from "../locales/en.js";
import { strings } from "../strings.js";
import { createTranslator, directionOf, formatLocaleOf, KEYS_LOCALE, languageOf } from "../translate.js";

describe("createTranslator", () => {
  it("renders English by default and for tags it has no catalog for", () => {
    expect(createTranslator("en")(strings.viewer.closeAriaLabel)).toBe(en.viewer.closeAriaLabel);
    expect(createTranslator("fr-CA")(strings.viewer.closeAriaLabel)).toBe(en.viewer.closeAriaLabel);
  });

  it("selects a catalog by exact tag, then by language", () => {
    const english = createTranslator("en")(strings.viewer.download);
    expect(createTranslator("ja")(strings.viewer.download)).not.toBe(english);
    expect(createTranslator("ja-JP")(strings.viewer.download)).toBe(createTranslator("ja")(strings.viewer.download));
    expect(createTranslator("ar-SA")(strings.viewer.download)).toBe(createTranslator("ar")(strings.viewer.download));
  });

  it("returns the key itself under the keys pseudo-locale, so tests assert structure rather than copy", () => {
    expect(createTranslator(KEYS_LOCALE)(strings.viewer.download)).toBe("viewer.download");
  });

  it("interpolates parameters and leaves an unsupplied placeholder visible", () => {
    const t = createTranslator("en");
    expect(t(strings.readerPreview.searchCount, { current: 2, total: 9, cap: "+" })).toBe("2 of 9+ matches");
    expect(t(strings.readerPreview.searchCount, { current: 2 })).toBe("2 of {{total}}{{cap}} matches");
  });

  it("selects a plural variant from count and falls back to the base key", () => {
    const t = createTranslator("en");
    expect(t(strings.documentPreview.results, { count: 1 })).toBe("1 match in loaded items");
    expect(t(strings.documentPreview.results, { count: 3 })).toBe("3 matches in loaded items");
    // `viewer.linesSuffix` has no plural variants: the base key serves every count.
    expect(t(strings.viewer.linesSuffix, { count: 1 })).toBe("1 lines");
  });

  it("applies overrides over the catalog, and the host adapter over both", () => {
    const overrides = { [strings.viewer.download]: "Save a copy" };
    expect(createTranslator("en", { overrides })(strings.viewer.download)).toBe("Save a copy");
    const adapter = (key: string) => (key === strings.viewer.download ? "From the host" : undefined);
    const t = createTranslator("en", { overrides, adapter });
    expect(t(strings.viewer.download)).toBe("From the host");
    // The adapter declining falls through to the catalog.
    expect(t(strings.viewer.closeAriaLabel)).toBe(en.viewer.closeAriaLabel);
  });
});

describe("locale helpers", () => {
  it("reads the language subtag and picks a text direction", () => {
    expect(languageOf("ar-SA")).toBe("ar");
    expect(languageOf("en_GB")).toBe("en");
    expect(directionOf("ar")).toBe("rtl");
    expect(directionOf("ja")).toBe("ltr");
    expect(directionOf(KEYS_LOCALE)).toBe("ltr");
  });

  it("formats the pseudo-locale as English", () => {
    expect(formatLocaleOf(KEYS_LOCALE)).toBe("en");
    expect(formatLocaleOf("ja-JP")).toBe("ja-JP");
  });
});
