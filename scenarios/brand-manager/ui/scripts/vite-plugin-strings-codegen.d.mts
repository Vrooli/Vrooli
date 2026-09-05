/**
 * Type declarations for `vite-plugin-strings-codegen.mjs`.
 *
 * Lets `vite.config.ts` import the plugin without `@typescript-eslint/no-unsafe-call`
 * complaining about an untyped default export.
 */
import type { Plugin } from "vite";

export default function stringsCodegen(): Plugin;
