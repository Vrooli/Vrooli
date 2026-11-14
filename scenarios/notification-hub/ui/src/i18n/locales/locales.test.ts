/**
 * Locale parity contract test.
 *
 * Catches "added a key to en, forgot every other locale" drift before it
 * ships, plus the related "added a `{{count}}` placeholder to one catalog
 * but not the others" silent runtime failure. Three wrinkles:
 *
 *   1. CLDR plural forms differ between languages (English has `_one` +
 *      base, Japanese has only the base), so we strip plural suffixes
 *      before comparing key shapes. Each *logical* key must exist in
 *      every locale; a locale is free to declare extra plural variants
 *      its language needs.
 *   2. Catalog keys whose final segment starts with `_` are sentinels
 *      (e.g., `_comment` documenting the file). They aren't user-facing
 *      strings, don't go through `t()`, and a locale is free to omit
 *      them. We skip them during flattening.
 *   3. Interpolation tokens (`{{count}}`, `{{name}}`, …) are aggregated
 *      *per logical (plural-stripped) base key* before comparison —
 *      because plural variants legitimately differ (`refreshCount_one`
 *      says "Refreshed once" with no `{{count}}` in English, while
 *      `refreshCount` has `{{count}}`). The aggregate must match across
 *      locales.
 *
 * Each locale is a directory of per-namespace files (`<lng>/<namespace>.json`),
 * picked up via `import.meta.glob`, so adding a new locale (a `fr/` directory)
 * requires no edits to this file — the parity assertions automatically include
 * it. A fourth check compares the file set of every locale, so a namespace
 * file missing from one locale is named directly rather than surfacing as a
 * long list of missing keys.
 *
 * The same sentinel skip lives in `scripts/gen-strings.mjs` (so they
 * don't leak into `strings.generated.ts`) and `eslint-rules/no-unused-
 * keys.js` (so they don't get flagged as orphans). The convention is
 * duplicated by intent — each consumer documents its own contract.
 */
import { describe, it, expect } from "vitest";
import { assemble } from "../catalog";
import { LOCALE_CODES } from "../locales";

// `import.meta.glob` returns `{ "./en/health.json": {...}, ... }` at build
// time when `eager: true`. Vitest reads the same Vite config that production
// does, so this works in tests without an extra plugin.
const catalogModules = import.meta.glob<unknown>("./*/*.json", {
  eager: true,
  import: "default",
});

const filesByLocale: Record<string, string[]> = {};
const modulesByLocale: Record<string, Record<string, unknown>> = {};
for (const [path, mod] of Object.entries(catalogModules)) {
  // path looks like "./en/health.json".
  const [, code = "", file = ""] = /^\.\/([^/]+)\/([^/]+)$/.exec(path) ?? [];
  (filesByLocale[code] ??= []).push(file);
  (modulesByLocale[code] ??= {})[path] = mod;
}

const catalogs: Record<string, Record<string, unknown>> = {};
for (const [code, modules] of Object.entries(modulesByLocale)) {
  catalogs[code] = assemble(modules);
}

const PLURAL_SUFFIX = /_(zero|one|two|few|many|other)$/;
const TOKEN_RE = /\{\{(\w+)\}\}/g;

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

/**
 * Aggregate the union of `{{token}}` names per *base* (plural-stripped)
 * key. Plural variants legitimately differ inside one locale, so we
 * compare unions across locales rather than per-variant.
 */
const tokensPerBase = (catalog: Record<string, unknown>): Map<string, Set<string>> => {
  const flat = flatten(catalog);
  const byBase = new Map<string, Set<string>>();
  for (const [key, value] of Object.entries(flat)) {
    const base = stripPluralSuffix(key);
    const tokens = byBase.get(base) ?? new Set<string>();
    for (const match of value.matchAll(TOKEN_RE)) {
      tokens.add(match[1] as string);
    }
    byBase.set(base, tokens);
  }
  return byBase;
};

describe("locale catalogs", () => {
  const reference = catalogs.en;
  if (!reference) {
    throw new Error("locales/en/ must exist as the canonical reference catalog");
  }

  it("LOCALE_CODES matches the locale directories on disk", () => {
    // Catches the "added a code to LOCALE_CODES, forgot the JSON" failure mode
    // (i18next init silently ignores unknown codes today) and the inverse:
    // a JSON dropped in without registering its code.
    expect(new Set(LOCALE_CODES), "LOCALE_CODES vs locales/<code>/ must agree").toEqual(
      new Set(Object.keys(catalogs)),
    );
  });

  it("every locale has the same namespace files as en", () => {
    const referenceFiles = [...(filesByLocale.en ?? [])].sort();
    for (const [locale, files] of Object.entries(filesByLocale)) {
      if (locale === "en") continue;
      expect([...files].sort(), `${locale}/ namespace files vs en/`).toEqual(referenceFiles);
    }
  });

  it("every locale shares the same logical key shape (plural variants stripped)", () => {
    const referenceKeys = logicalKeys(reference);
    for (const [locale, catalog] of Object.entries(catalogs)) {
      if (locale === "en") continue;
      const candidate = logicalKeys(catalog);

      const missing = [...referenceKeys].filter((k) => !candidate.has(k));
      const extra = [...candidate].filter((k) => !referenceKeys.has(k));

      expect(missing, `${locale} is missing keys present in en`).toEqual([]);
      expect(extra, `${locale} has keys not in en`).toEqual([]);
    }
  });

  it("every logical key uses the same interpolation tokens across locales", () => {
    // Without this check, a translator dropping `{{count}}` from
    // `ja.notifications.summary` would render "件の通知" with no number and
    // never trip CI. The runtime fail is silent, so we surface it here.
    const referenceTokens = tokensPerBase(reference);
    for (const [locale, catalog] of Object.entries(catalogs)) {
      if (locale === "en") continue;
      const candidateTokens = tokensPerBase(catalog);

      for (const [base, refTokens] of referenceTokens) {
        const localTokens = candidateTokens.get(base) ?? new Set<string>();
        expect(
          [...localTokens].sort(),
          `${locale} key '${base}' has different interpolation tokens than en`,
        ).toEqual([...refTokens].sort());
      }
    }
  });

  it("no string is empty in any locale", () => {
    for (const [locale, catalog] of Object.entries(catalogs)) {
      const flat = flatten(catalog);
      for (const [key, value] of Object.entries(flat)) {
        expect(value.trim(), `${locale} key '${key}' is empty`).not.toBe("");
      }
    }
  });

  it("every locale's strings are actual strings (not arrays/objects/null)", () => {
    for (const [locale, catalog] of Object.entries(catalogs)) {
      const flat = flatten(catalog);
      for (const [key, value] of Object.entries(flat)) {
        expect(typeof value, `${locale} key '${key}' is not a string`).toBe(
          "string",
        );
      }
    }
  });
});
