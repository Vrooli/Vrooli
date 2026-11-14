/**
 * Type declarations for `gen-strings.mjs`.
 *
 * The codegen runtime stays as plain `.mjs` so it can be invoked directly by
 * `node` (CLI mode for `pnpm strings:gen` / `strings:check`) without a TS
 * loader. This file gives the Vite plugin and `vite.config.ts` real types
 * for the imports — without that, `@typescript-eslint/no-unsafe-call` flags
 * every plugin construction.
 */

/** Absolute path to the canonical English catalog. */
export const SOURCE_PATH: string;

/** Absolute path to the generated registry file. */
export const TARGET_PATH: string;

/** Build the full file contents (header + strings + Strings type). */
export function generateContents(): string;

/** Write the generated file if it differs from current contents. Returns true if written. */
export function writeIfChanged(): boolean;

/** Check whether the generated file is out of date. */
export function isOutOfDate(): boolean;
