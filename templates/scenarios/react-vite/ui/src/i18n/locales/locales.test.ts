/**
 * Locale parity contract test.
 *
 * Catches "added a key to en, forgot every other locale" drift before it
 * ships. We compare flattened key shapes between every locale catalog —
 * with two wrinkles:
 *
 *   1. CLDR plural forms differ between languages (English has `_one` +
 *      base, Japanese has only the base), so we strip plural suffixes
 *      before comparing. Each *logical* key must exist in every locale;
 *      a locale is free to declare extra plural variants its language
 *      needs.
 *   2. Catalog keys whose final segment starts with `_` are sentinels
 *      (e.g., `_comment` documenting the file). They aren't user-facing
 *      strings, don't go through `t()`, and a locale is free to omit
 *      them. We skip them during flattening.
 *
 * The same sentinel skip lives in `scripts/gen-strings.mjs` (so they
 * don't leak into `strings.generated.ts`) and `eslint-rules/no-unused-
 * keys.js` (so they don't get flagged as orphans). The convention is
 * duplicated by intent — each consumer documents its own contract.
 */
import { describe, it, expect } from "vitest";
import en from "./en.json";
import ja from "./ja.json";

const PLURAL_SUFFIX = /_(zero|one|two|few|many|other)$/;

const stripPluralSuffix = (key: string) => key.replace(PLURAL_SUFFIX, "");

const isSentinelKey = (key: string) => key.startsWith("_");

const flatten = (
  obj: Record<string, unknown>,
  prefix = "",
): Record<string, string> => {
  const out: Record<string, string> = {};
  for (const [k, v] of Object.entries(obj)) {
    if (isSentinelKey(k)) continue;
    const path = prefix ? `${prefix}.${k}` : k;
    if (typeof v === "string") {
      out[path] = v;
    } else if (v && typeof v === "object") {
      Object.assign(out, flatten(v as Record<string, unknown>, path));
    }
  }
  return out;
};

const logicalKeys = (catalog: Record<string, unknown>) => {
  const flat = flatten(catalog);
  return new Set(Object.keys(flat).map(stripPluralSuffix));
};

describe("locale catalogs", () => {
  const catalogs: Record<string, Record<string, unknown>> = {
    en,
    ja,
  };

  it("every locale shares the same logical key shape (plural variants stripped)", () => {
    const reference = logicalKeys(en);
    for (const [locale, catalog] of Object.entries(catalogs)) {
      if (locale === "en") continue;
      const candidate = logicalKeys(catalog);

      const missing = [...reference].filter((k) => !candidate.has(k));
      const extra = [...candidate].filter((k) => !reference.has(k));

      expect(missing, `${locale}.json is missing keys present in en.json`).toEqual([]);
      expect(extra, `${locale}.json has keys not in en.json`).toEqual([]);
    }
  });

  it("no string is empty in any locale", () => {
    for (const [locale, catalog] of Object.entries(catalogs)) {
      const flat = flatten(catalog);
      for (const [key, value] of Object.entries(flat)) {
        expect(value.trim(), `${locale}.json key '${key}' is empty`).not.toBe("");
      }
    }
  });

  it("every locale's strings are actual strings (not arrays/objects/null)", () => {
    for (const [locale, catalog] of Object.entries(catalogs)) {
      const flat = flatten(catalog);
      for (const [key, value] of Object.entries(flat)) {
        expect(typeof value, `${locale}.json key '${key}' is not a string`).toBe(
          "string",
        );
      }
    }
  });
});
