import { fileURLToPath } from "node:url";

export default {
  test: {
    root: fileURLToPath(new URL("./src", import.meta.url)),
    include: ["**/*.test.ts"],
    environment: "jsdom",
    coverage: {
      provider: "v8",
      reporter: ["text", "json-summary"],
      reportsDirectory: "../coverage",
      include: ["**/*.ts"],
      exclude: ["**/*.test.ts", "**/*.d.ts", "**/index.ts", "**/runtime.ts", "**/services.ts", "**/tts/types.ts"],
      thresholds: {
        lines: 65,
        functions: 65,
        branches: 65,
        statements: 65,
      },
    },
  },
};
