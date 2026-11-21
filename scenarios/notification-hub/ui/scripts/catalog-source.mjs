/**
 * Node-side reader for the split message catalogs.
 *
 * Each locale is a directory, `src/i18n/locales/<lng>/`, holding one JSON file
 * per catalog subtree. The file name (minus `.json`) is the dotted key path of
 * the subtree it defines: `health.json` is `health.*`, `pages.dashboard.json`
 * is `pages.dashboard.*`. Splitting by subtree gives every feature owner its own
 * files, so concurrent edits never rewrite a shared catalog.
 *
 * The browser-side twin is `src/i18n/catalog.ts`; `catalog.test.ts` pins that
 * both assemble the same object.
 */
import { readdirSync, readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const ROOT = join(dirname(fileURLToPath(import.meta.url)), "..");

export const LOCALES_DIR = join(ROOT, "src/i18n/locales");

/** Reference locale: its catalog defines the key registry. */
export const SOURCE_LOCALE = "en";

/** Absolute path of one locale's catalog directory. */
export const localeDir = (locale) => join(LOCALES_DIR, locale);

/** File names (sorted) that make up one locale's catalog. */
export const catalogFiles = (locale) =>
  readdirSync(localeDir(locale))
    .filter((name) => name.endsWith(".json"))
    .sort();

/** Assemble one locale's catalog into a single nested object. */
export const readCatalog = (locale) => {
  const catalog = {};
  for (const file of catalogFiles(locale)) {
    const segments = file.slice(0, -".json".length).split(".");
    const leaf = segments.pop();
    let node = catalog;
    for (const segment of segments) node = node[segment] ??= {};
    node[leaf] = JSON.parse(readFileSync(join(localeDir(locale), file), "utf-8"));
  }
  return catalog;
};
