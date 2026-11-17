import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { sourceLibraryResolver } from "../../../packages/react-component-library/tooling/resolve-specifier.mjs";

export default defineConfig({
  base: './',  // Required for tunnel/proxy contexts
  plugins: [sourceLibraryResolver({ devOnly: true }), react()],
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/test-utils/setupTests.ts'],
    coverage: {
      provider: 'v8',
      reporter: ['json-summary', 'json', 'text'],
      reportOnFailure: true,
    }
  }
});
