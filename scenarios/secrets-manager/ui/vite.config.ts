// INTEROP-CRITICAL: interop-sensitive configuration below — do not remove without checking host-frame embedding.
import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";

const healthMiddleware = () => {
  const handler = (req: any, res: any, next: () => void) => {
    if (req.url === "/health") {
      res.setHeader("Content-Type", "application/json");
      res.end(
        JSON.stringify({
          status: "ready",
          service: "secrets-manager-ui",
          timestamp: new Date().toISOString()
        })
      );
      return;
    }
    next();
  };
  return handler;
};

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), "UI_");
  const configuredUIPort = env.UI_PORT?.trim();
  const uiPort = mode === "test" && !configuredUIPort ? 4173 : Number(configuredUIPort);
  if (!Number.isInteger(uiPort) || uiPort < 1 || uiPort > 65535) {
    throw new Error("UI_PORT must be an integer between 1 and 65535");
  }

  return {
    plugins: [
      react(),
      {
        name: "secrets-manager-ui-health",
        configureServer(server) {
          server.middlewares.use(healthMiddleware());
        },
        configurePreviewServer(server) {
          server.middlewares.use(healthMiddleware());
        }
      }
    ],
    base: "./",
    server: {
      host: true,
      port: Number(uiPort)
    },
    preview: {
      host: true,
      port: Number(uiPort)
    },
    test: {
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
          "src/i18n/locales/**",
          "src/**/generated/**"
        ],
        reportOnFailure: true,
        thresholds: {
          branches: 85,
          functions: 85,
          lines: 85,
          statements: 85
        }
      }
    }
  };
});
