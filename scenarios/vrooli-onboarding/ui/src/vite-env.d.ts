/// <reference types="vite/client" />

declare module "*.mjs" {
  import type { Plugin } from "vite";

  export function findHardcodedStrings(): string[];
  export function sourceLibraryResolver(options?: {
    libraryRoot?: string;
    draftOverrides?: Record<string, string>;
    devOnly?: boolean;
    usageContractsInTests?: boolean;
  }): Plugin;
}
