/// <reference types="vite/client" />

declare module "*.json" {
  const value: any;
  export default value;
}

// Handle ../types.js imports
declare module "../types.js" {
  export * from "../types.d.ts";
}