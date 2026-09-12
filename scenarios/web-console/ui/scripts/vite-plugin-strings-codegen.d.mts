/**
 * Ambient type for the strings codegen Vite plugin.
 *
 * The plugin lives in `vite-plugin-strings-codegen.mjs` (plain JS so the
 * `pnpm strings:gen` / `strings:check` CLI and the build-time plugin share
 * one implementation without a TS-build prerequisite). This shim lets the
 * strict tsc check resolve the import in vite.config.ts without an
 * implicit-any error.
 */
import type { Plugin } from "vite";

export default function stringsCodegen(): Plugin;
