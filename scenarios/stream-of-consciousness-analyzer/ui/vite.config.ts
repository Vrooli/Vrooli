import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { sourceLibraryResolver } from "../../../packages/react-component-library/tooling/resolve-specifier.mjs";

export default defineConfig({
  // INTEROP-CRITICAL: base must be './' for tunnel/proxy/iframe contexts
  base: './',
  plugins: [sourceLibraryResolver({ devOnly: true }), react()],
  test: {
    globals: true,
    environment: 'jsdom',
    coverage: {
      provider: 'v8',
      reporter: ['json-summary', 'json', 'text'],
      reportOnFailure: true,
    }
  }
});
