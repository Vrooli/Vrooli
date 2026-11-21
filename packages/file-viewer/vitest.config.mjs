import { defineConfig } from "vitest/config";

import { fileViewer } from "./vite/index.js";

export default defineConfig({
  // The package tests its own plugin: it provides the slide runtime's virtual module.
  plugins: [fileViewer()],
  resolve: {
    // The library resolves its own React copy; hooks need exactly one.
    dedupe: ["react", "react-dom"],
  },
  test: {
    globals: true,
    environment: "jsdom",
    pool: "forks",
    poolOptions: { forks: { minForks: 1, maxForks: 2 } },
    // Real renderers in jsdom: the heaviest interaction tests take seconds, more under load.
    testTimeout: 20000,
    setupFiles: ["./src/test-setup.ts"],
    server: { deps: { inline: [/react-component-library/] } },
    include: ["src/**/*.test.{ts,tsx}", "vite/**/*.test.ts"],
  },
});
