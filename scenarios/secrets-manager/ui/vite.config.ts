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
  const env = loadEnv(mode, process.cwd());

  const uiPort = process.env.UI_PORT || env.UI_PORT || "4173";
  const rawApiBase = env.VITE_API_BASE_URL || process.env.VITE_API_BASE_URL || "http://localhost:15000/api/v1";

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
    }
  };
});
