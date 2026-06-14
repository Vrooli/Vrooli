import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), "");

  return {
    base: "./",
    plugins: [react()],
    server: {
      port: env.VITE_DEV_SERVER_PORT ? Number(env.VITE_DEV_SERVER_PORT) : 5173,
      host: true
    },
    preview: {
      port: env.VITE_PREVIEW_PORT ? Number(env.VITE_PREVIEW_PORT) : 4173,
      host: true
    },
    test: {
      globals: true,
      environment: "jsdom",
      setupFiles: ["./src/test-setup.ts"],
      coverage: {
        provider: "v8",
        reporter: ["text", "json-summary", "json"],
        include: ["src/**/*.{ts,tsx}"],
        exclude: [
          "src/**/*.test.{ts,tsx}",
          "src/**/*.spec.{ts,tsx}",
          "src/**/*.d.ts",
          "src/main.tsx",
          "src/test-setup.ts",
          "src/test-utils/**",
          "src/consts/strings.generated.ts",
          "src/i18n/locales/**"
        ]
      }
    }
  };
});
