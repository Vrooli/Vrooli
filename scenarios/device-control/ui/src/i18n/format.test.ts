import { afterEach, describe, expect, it } from "vitest";
import { i18n } from "./index";
import {
  formatCurrency,
  formatDate,
  formatList,
  formatNumber,
  formatRelativeTime,
} from "./format";

describe("locale-aware Intl formatters", () => {
  afterEach(async () => {
    // The setup file resets to cimode before each test; nothing to do here,
    // but be defensive about leaking locale state into following suites.
    await i18n.changeLanguage("cimode");
  });

  describe("formatNumber", () => {
    it("uses Intl defaults when called from cimode (pseudo-locale)", () => {
      // cimode is not a valid BCP-47 tag; the helper falls back to the
      // runtime default, so the result must at least contain the digits.
      expect(formatNumber(12345.6)).toMatch(/12.?345/);
    });

    it("respects an explicit locale override", () => {
      // German uses '.' as thousands separator and ',' as decimal.
      expect(formatNumber(12345.6, undefined, "de-DE")).toBe("12.345,6");
      // English uses the inverse.
      expect(formatNumber(12345.6, undefined, "en-US")).toBe("12,345.6");
    });

    it("switches output when i18n.language changes", async () => {
      // Using a locale pair whose Intl output actually diverges (en-US uses
      // ',' as thousands separator + '.' as decimal; de-DE inverts both) so
      // the assertion proves `i18n.language` drives the formatter — not just
      // that the call doesn't throw. en + ja would both produce "1,234.5"
      // and silently pass even if the language-switch path were broken.
      await i18n.changeLanguage("en-US");
      expect(formatNumber(1234.5)).toBe("1,234.5");
      await i18n.changeLanguage("de-DE");
      expect(formatNumber(1234.5)).toBe("1.234,5");
    });

    it("forwards Intl.NumberFormatOptions to the constructor", () => {
      expect(
        formatNumber(0.42, { style: "percent", maximumFractionDigits: 0 }, "en-US"),
      ).toBe("42%");
    });

    it("treats the dev pseudo-locale as an Intl fallback", () => {
      expect(formatNumber(1234, undefined, "dev")).toMatch(/1.?234/);
    });
  });

  describe("formatCurrency", () => {
    it("formats USD in en-US with the leading dollar sign", () => {
      expect(formatCurrency(1234.5, "USD", undefined, "en-US")).toBe("$1,234.50");
    });

    it("formats JPY without fractional digits in ja-JP", () => {
      // JPY has 0 default fraction digits; output should not include a decimal.
      const result = formatCurrency(1234, "JPY", undefined, "ja-JP");
      expect(result).not.toMatch(/\./);
      expect(result).toMatch(/1.?234/);
    });
  });

  describe("formatDate", () => {
    it("produces locale-shaped output for en vs de", () => {
      const date = new Date(Date.UTC(2026, 4, 1, 12, 0, 0));
      const en = formatDate(date, { dateStyle: "medium" }, "en-US");
      const de = formatDate(date, { dateStyle: "medium" }, "de-DE");
      // We don't assert exact strings (CLDR varies between Node versions);
      // we assert they're *different* shapes — that's the locale signal.
      expect(en).not.toBe(de);
      expect(en).toMatch(/2026/);
      expect(de).toMatch(/2026/);
    });
  });

  describe("formatRelativeTime", () => {
    it("formats negative offsets as past in en-US", () => {
      expect(formatRelativeTime(-3, "day", undefined, "en-US")).toBe("3 days ago");
    });

    it("formats positive offsets as future in en-US", () => {
      expect(formatRelativeTime(2, "hour", undefined, "en-US")).toBe("in 2 hours");
    });
  });

  describe("formatList", () => {
    it("uses 'and' in English", () => {
      expect(formatList(["a", "b", "c"], undefined, "en-US")).toBe("a, b, and c");
    });

    it("uses Japanese conjunctions in ja-JP", () => {
      const result = formatList(["a", "b", "c"], undefined, "ja-JP");
      expect(result).toContain("a");
      expect(result).toContain("b");
      expect(result).toContain("c");
      // Japanese typically uses '、' as separator.
      expect(result).toMatch(/[、,]/);
    });

    it("handles single-item and empty lists", () => {
      expect(formatList(["only"], undefined, "en-US")).toBe("only");
      expect(formatList([], undefined, "en-US")).toBe("");
    });
  });
});
