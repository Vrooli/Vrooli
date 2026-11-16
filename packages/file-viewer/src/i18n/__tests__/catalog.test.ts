import { readdirSync, readFileSync, statSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";

import type { LocaleMessages } from "../catalog.js";
import { ar } from "../locales/ar.js";
import { en } from "../locales/en.js";
import { ja } from "../locales/ja.js";

const PLURAL_SUFFIX = /_(zero|one|two|few|many|other)$/;

/** Dotted paths of every string leaf, with plural-variant suffixes folded into their base key. */
function keysOf(node: LocaleMessages, prefix = ""): string[] {
  return Object.entries(node).flatMap(([key, value]) => {
    const path = prefix ? `${prefix}.${key}` : key;
    return typeof value === "string" ? [path.replace(PLURAL_SUFFIX, "")] : keysOf(value, path);
  });
}

function sourceFiles(directory: string): string[] {
  return readdirSync(directory).flatMap((name) => {
    const path = join(directory, name);
    if (statSync(path).isDirectory()) return name === "locales" || name === "__tests__" ? [] : sourceFiles(path);
    return /\.tsx?$/.test(name) && !/\.test\./.test(name) && !path.includes(`${join("src", "i18n")}`) ? [path] : [];
  });
}

describe("catalogs", () => {
  const english = new Set(keysOf(en));

  it.each([["ar", ar], ["ja", ja]] as const)("%s has exactly the keys English has", (_name, catalog) => {
    const keys = new Set(keysOf(catalog));
    expect([...english].filter((key) => !keys.has(key))).toEqual([]);
    expect([...keys].filter((key) => !english.has(key))).toEqual([]);
  });

  it("holds no copy that nothing renders", () => {
    // Every leaf name must appear as an identifier somewhere in the shipped source.
    const text = sourceFiles("src").map((file) => readFileSync(file, "utf8")).join("\n");
    const unused = [...english].filter((key) => !new RegExp(`\\b${key.split(".").at(-1) ?? key}\\b`).test(text));
    expect(unused).toEqual([]);
  });

  it("uses the same interpolation tokens in every locale", () => {
    // Plural variants legitimately differ inside one locale (`key_one` may say "one match"
    // with no {{count}}), so tokens are unioned per base key before comparing locales.
    const tokensByBase = (catalog: LocaleMessages): Map<string, string> => {
      const union = new Map<string, Set<string>>();
      const walk = (node: LocaleMessages, prefix: string) => {
        for (const [key, value] of Object.entries(node)) {
          const path = prefix ? `${prefix}.${key}` : key;
          if (typeof value !== "string") { walk(value, path); continue; }
          const base = path.replace(PLURAL_SUFFIX, "");
          const tokens = union.get(base) ?? new Set<string>();
          for (const match of value.matchAll(/\{\{\s*(\w+)\s*\}\}/g)) tokens.add(match[1] ?? "");
          union.set(base, tokens);
        }
      };
      walk(catalog, "");
      return new Map([...union].map(([base, tokens]) => [base, [...tokens].sort().join(",")]));
    };
    const reference = tokensByBase(en);
    for (const catalog of [ar, ja]) expect(tokensByBase(catalog)).toEqual(reference);
  });

  it("has no empty copy in any locale", () => {
    const empty = (node: LocaleMessages, prefix = ""): string[] => Object.entries(node).flatMap(([key, value]) => {
      const path = prefix ? `${prefix}.${key}` : key;
      return typeof value === "string" ? (value.trim() ? [] : [path]) : empty(value, path);
    });
    expect([...empty(en), ...empty(ar), ...empty(ja)]).toEqual([]);
  });
});
