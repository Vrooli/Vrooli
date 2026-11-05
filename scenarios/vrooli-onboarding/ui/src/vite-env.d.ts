/// <reference types="vite/client" />

declare module "*.mjs" {
  export function findHardcodedStrings(): string[];
}
