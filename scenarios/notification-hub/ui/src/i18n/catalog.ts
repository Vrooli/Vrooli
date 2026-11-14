/**
 * Message catalogs, assembled from the per-namespace files under
 * `./locales/<lng>/`.
 *
 * Each file holds one catalog subtree and its name is that subtree's dotted
 * key path: `health.json` defines `health.*`, `pages.dashboard.json` defines
 * `pages.dashboard.*`. Feature owners edit only their own files, so catalog
 * changes never collide. Runtime key paths (`t("health.title")`) are the same
 * as they would be in one big file.
 *
 * The English catalog is the reference. Its shape is derived from the generated
 * `strings` registry (itself built from the English files), so a locale can
 * only be typed as `Catalog` once the parity test in `locales/locales.test.ts`
 * has shown it has the same keys.
 *
 * `scripts/catalog-source.mjs` is the Node-side twin used by codegen and lint;
 * `catalog.test.ts` pins the two to the same output.
 */
import type { strings } from "../consts/strings.generated";

/** Leaves become `string`; the tree keeps the registry's keys. */
type Shape<T> = { -readonly [K in keyof T]: T[K] extends string ? string : Shape<T[K]> };

export type Catalog = Shape<typeof strings>;

type Modules = Record<string, unknown>;

const FILE_STEM = /([^/]+)\.json$/;

/** Merge `{ "pages.dashboard.json": {...} }` style modules into one nested object. */
export const assemble = (modules: Modules): Record<string, unknown> => {
  const catalog: Record<string, unknown> = {};
  for (const path of Object.keys(modules).sort()) {
    const stem = FILE_STEM.exec(path)?.[1];
    if (!stem) continue;
    const segments = stem.split(".");
    const leaf = segments.pop() as string;
    let node = catalog;
    for (const segment of segments) {
      node = (node[segment] ??= {}) as Record<string, unknown>;
    }
    node[leaf] = modules[path];
  }
  return catalog;
};

// Vite only accepts a literal options object in `import.meta.glob`, so it is
// repeated per locale rather than shared.
const asCatalog = (modules: Modules) => assemble(modules) as Catalog;

export const en = asCatalog(import.meta.glob("./locales/en/*.json", { eager: true, import: "default" }));
export const ja = asCatalog(import.meta.glob("./locales/ja/*.json", { eager: true, import: "default" }));
export const ar = asCatalog(import.meta.glob("./locales/ar/*.json", { eager: true, import: "default" }));
