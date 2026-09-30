/**
 * Additional format.ts coverage — line 27 (`"dev"` pseudo-locale branch).
 *
 * The existing format.test.ts covers cimode → undefined locale fallback and
 * explicit locale overrides. This file covers the `"dev"` pseudo-locale
 * that must also be stripped to avoid RangeError in older runtimes.
 */
import { describe, it, expect, afterEach } from "vitest";

import { i18n } from "./index";
import { formatNumber, formatDate, formatCurrency, formatRelativeTime, formatList } from "./format";

afterEach(async () => {
  await i18n.changeLanguage("cimode");
});

describe("format helpers — 'dev' pseudo-locale (line 27)", () => {
  it("formatNumber does not throw when i18n.language is 'dev'", async () => {
    await i18n.changeLanguage("dev");
    // resolveIntlLocale("dev") → undefined → runtime default locale
    expect(() => formatNumber(1234)).not.toThrow();
    const result = formatNumber(1234);
    expect(result).toMatch(/1.?234/);
  });

  it("formatDate does not throw with the 'dev' pseudo-locale", async () => {
    await i18n.changeLanguage("dev");
    expect(() => formatDate(new Date(Date.UTC(2026, 0, 1)))).not.toThrow();
  });

  it("formatCurrency does not throw with the 'dev' pseudo-locale", async () => {
    await i18n.changeLanguage("dev");
    expect(() => formatCurrency(100, "USD")).not.toThrow();
  });

  it("formatRelativeTime does not throw with the 'dev' pseudo-locale", async () => {
    await i18n.changeLanguage("dev");
    expect(() => formatRelativeTime(-1, "day")).not.toThrow();
  });

  it("formatList does not throw with the 'dev' pseudo-locale", async () => {
    await i18n.changeLanguage("dev");
    expect(() => formatList(["a", "b"])).not.toThrow();
  });
});
