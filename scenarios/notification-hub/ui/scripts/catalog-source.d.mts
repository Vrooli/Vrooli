/** Type declarations for `catalog-source.mjs` (plain `.mjs` so `node` can run it directly). */
export const LOCALES_DIR: string;
export const SOURCE_LOCALE: string;
export function localeDir(locale: string): string;
export function catalogFiles(locale: string): string[];
export function readCatalog(locale: string): Record<string, unknown>;
