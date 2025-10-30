// vite.config.ts
import { defineConfig } from "file:///home/matthalloran8/Vrooli/scenarios/browser-automation-studio/ui/node_modules/vite/dist/node/index.js";
import react from "file:///home/matthalloran8/Vrooli/scenarios/browser-automation-studio/ui/node_modules/@vitejs/plugin-react/dist/index.js";
import path from "path";
import http from "http";
var __vite_injected_original_dirname = "/home/matthalloran8/Vrooli/scenarios/browser-automation-studio/ui";
var UI_PORT = process.env.UI_PORT || "3000";
var API_PORT = process.env.API_PORT || "8080";
var WS_PORT = process.env.WS_PORT || "8081";
var API_HOST = process.env.API_HOST || "localhost";
var WS_HOST = process.env.WS_HOST || "localhost";
var healthEndpointPlugin = () => ({
  name: "health-endpoint",
  configureServer(server) {
    server.middlewares.use(async (req, res, next) => {
      if (req.url === "/health") {
        const buildHealthPayload = () => ({
          status: "healthy",
          service: "browser-automation-studio-ui",
          timestamp: (/* @__PURE__ */ new Date()).toISOString(),
          readiness: true,
          api_connectivity: {
            connected: false,
            api_url: API_PORT ? `http://${API_HOST}:${API_PORT}/api/v1` : null,
            last_check: (/* @__PURE__ */ new Date()).toISOString(),
            error: null,
            latency_ms: null,
            upstream: null
          }
        });
        const healthResponse = buildHealthPayload();
        if (API_PORT) {
          const startTime = Date.now();
          try {
            await new Promise((resolve, reject) => {
              const healthReq = http.request(
                {
                  hostname: API_HOST,
                  port: API_PORT,
                  path: "/health",
                  method: "GET",
                  timeout: 2e3
                },
                (healthRes) => {
                  healthResponse.api_connectivity.latency_ms = Date.now() - startTime;
                  healthResponse.api_connectivity.last_check = (/* @__PURE__ */ new Date()).toISOString();
                  healthResponse.api_connectivity.connected = healthRes.statusCode >= 200 && healthRes.statusCode < 300;
                  if (!healthResponse.api_connectivity.connected) {
                    healthResponse.status = "degraded";
                    healthResponse.api_connectivity.error = {
                      code: `HTTP_${healthRes.statusCode}`,
                      message: `API returned status ${healthRes.statusCode}`,
                      category: "network",
                      retryable: true
                    };
                  }
                  let body = "";
                  healthRes.on("data", (chunk) => {
                    body += chunk;
                  });
                  healthRes.on("end", () => {
                    try {
                      healthResponse.api_connectivity.upstream = JSON.parse(body);
                    } catch {
                      healthResponse.api_connectivity.upstream = body;
                    }
                    resolve();
                  });
                }
              );
              healthReq.on("error", (error) => {
                healthResponse.api_connectivity.latency_ms = Date.now() - startTime;
                healthResponse.api_connectivity.connected = false;
                healthResponse.api_connectivity.error = {
                  code: "CONNECTION_ERROR",
                  message: `Failed to connect to API: ${error.message}`,
                  category: "network",
                  retryable: true
                };
                healthResponse.status = "degraded";
                resolve();
              });
              healthReq.on("timeout", () => {
                healthReq.destroy();
                healthResponse.api_connectivity.latency_ms = Date.now() - startTime;
                healthResponse.api_connectivity.connected = false;
                healthResponse.api_connectivity.error = {
                  code: "TIMEOUT",
                  message: "API health check timed out",
                  category: "network",
                  retryable: true
                };
                healthResponse.status = "degraded";
                resolve();
              });
              healthReq.end();
            });
          } catch (error) {
          }
        } else {
          healthResponse.status = "degraded";
          healthResponse.api_connectivity.error = {
            code: "MISSING_CONFIG",
            message: "API_PORT environment variable not configured",
            category: "configuration",
            retryable: false
          };
        }
        res.setHeader("Content-Type", "application/json");
        res.end(JSON.stringify(healthResponse));
        return;
      }
      next();
    });
  }
});
var vite_config_default = defineConfig({
  plugins: [react(), healthEndpointPlugin()],
  resolve: {
    alias: {
      "@": path.resolve(__vite_injected_original_dirname, "./src"),
      "@components": path.resolve(__vite_injected_original_dirname, "./src/components"),
      "@hooks": path.resolve(__vite_injected_original_dirname, "./src/hooks"),
      "@stores": path.resolve(__vite_injected_original_dirname, "./src/stores"),
      "@utils": path.resolve(__vite_injected_original_dirname, "./src/utils"),
      "@types": path.resolve(__vite_injected_original_dirname, "./src/types"),
      "@api": path.resolve(__vite_injected_original_dirname, "./src/api")
    }
  },
  server: {
    port: parseInt(UI_PORT),
    host: true,
    cors: true,
    proxy: {
      "/api": {
        target: `http://${API_HOST}:${API_PORT}`,
        changeOrigin: true,
        secure: false
      },
      "/ws": {
        target: `ws://${WS_HOST}:${WS_PORT}`,
        ws: true,
        changeOrigin: true
      }
    }
  },
  define: {
    // Pass environment variables to the client with VITE_ prefix for dev mode
    "import.meta.env.VITE_API_URL": JSON.stringify(process.env.API_PORT ? `http://${API_HOST}:${process.env.API_PORT}/api/v1` : void 0),
    "import.meta.env.VITE_WS_URL": JSON.stringify(process.env.WS_PORT ? `ws://${WS_HOST}:${process.env.WS_PORT}` : void 0),
    "import.meta.env.VITE_API_PORT": JSON.stringify(process.env.API_PORT || void 0),
    "import.meta.env.VITE_UI_PORT": JSON.stringify(process.env.UI_PORT || void 0),
    "import.meta.env.VITE_WS_PORT": JSON.stringify(process.env.WS_PORT || void 0)
  },
  build: {
    outDir: "dist",
    sourcemap: true,
    minify: "esbuild",
    target: "esnext"
  },
  optimizeDeps: {
    include: ["lucide-react"]
  }
});
export {
  vite_config_default as default
};
//# sourceMappingURL=data:application/json;base64,ewogICJ2ZXJzaW9uIjogMywKICAic291cmNlcyI6IFsidml0ZS5jb25maWcudHMiXSwKICAic291cmNlc0NvbnRlbnQiOiBbImNvbnN0IF9fdml0ZV9pbmplY3RlZF9vcmlnaW5hbF9kaXJuYW1lID0gXCIvaG9tZS9tYXR0aGFsbG9yYW44L1Zyb29saS9zY2VuYXJpb3MvYnJvd3Nlci1hdXRvbWF0aW9uLXN0dWRpby91aVwiO2NvbnN0IF9fdml0ZV9pbmplY3RlZF9vcmlnaW5hbF9maWxlbmFtZSA9IFwiL2hvbWUvbWF0dGhhbGxvcmFuOC9Wcm9vbGkvc2NlbmFyaW9zL2Jyb3dzZXItYXV0b21hdGlvbi1zdHVkaW8vdWkvdml0ZS5jb25maWcudHNcIjtjb25zdCBfX3ZpdGVfaW5qZWN0ZWRfb3JpZ2luYWxfaW1wb3J0X21ldGFfdXJsID0gXCJmaWxlOi8vL2hvbWUvbWF0dGhhbGxvcmFuOC9Wcm9vbGkvc2NlbmFyaW9zL2Jyb3dzZXItYXV0b21hdGlvbi1zdHVkaW8vdWkvdml0ZS5jb25maWcudHNcIjtpbXBvcnQgeyBkZWZpbmVDb25maWcgfSBmcm9tICd2aXRlJztcbmltcG9ydCByZWFjdCBmcm9tICdAdml0ZWpzL3BsdWdpbi1yZWFjdCc7XG5pbXBvcnQgcGF0aCBmcm9tICdwYXRoJztcbmltcG9ydCBodHRwIGZyb20gJ2h0dHAnO1xuXG4vLyBHZXQgZW52aXJvbm1lbnQgdmFyaWFibGVzIHdpdGggZmFsbGJhY2tzIGZvciBidWlsZCB0aW1lXG4vLyBBVURJVE9SIE5PVEU6IFRoZXNlIGZhbGxiYWNrcyBhcmUgaW50ZW50aW9uYWwgZm9yIGRldmVsb3BtZW50IG1vZGUuXG4vLyBUaGUgbGlmZWN5Y2xlIHN5c3RlbSBwcm92aWRlcyBwcm9wZXIgZW52aXJvbm1lbnQgdmFyaWFibGVzIGluIHByb2R1Y3Rpb24uXG4vLyBWaXRlIHJlcXVpcmVzIGJ1aWxkLXRpbWUgdmFsdWVzOyB1bmRlZmluZWQgaXMgdXNlZCB0byBzaWduYWwgbWlzc2luZyBjb25maWcuXG5jb25zdCBVSV9QT1JUID0gcHJvY2Vzcy5lbnYuVUlfUE9SVCB8fCAnMzAwMCc7XG5jb25zdCBBUElfUE9SVCA9IHByb2Nlc3MuZW52LkFQSV9QT1JUIHx8ICc4MDgwJztcbmNvbnN0IFdTX1BPUlQgPSBwcm9jZXNzLmVudi5XU19QT1JUIHx8ICc4MDgxJztcblxuY29uc3QgQVBJX0hPU1QgPSBwcm9jZXNzLmVudi5BUElfSE9TVCB8fCAnbG9jYWxob3N0JztcbmNvbnN0IFdTX0hPU1QgPSBwcm9jZXNzLmVudi5XU19IT1NUIHx8ICdsb2NhbGhvc3QnO1xuXG4vLyBIZWFsdGggZW5kcG9pbnQgbWlkZGxld2FyZSBwbHVnaW4gZm9yIGRldiBtb2RlXG5jb25zdCBoZWFsdGhFbmRwb2ludFBsdWdpbiA9ICgpID0+ICh7XG4gIG5hbWU6ICdoZWFsdGgtZW5kcG9pbnQnLFxuICBjb25maWd1cmVTZXJ2ZXIoc2VydmVyKSB7XG4gICAgc2VydmVyLm1pZGRsZXdhcmVzLnVzZShhc3luYyAocmVxLCByZXMsIG5leHQpID0+IHtcbiAgICAgIGlmIChyZXEudXJsID09PSAnL2hlYWx0aCcpIHtcbiAgICAgICAgY29uc3QgYnVpbGRIZWFsdGhQYXlsb2FkID0gKCkgPT4gKHtcbiAgICAgICAgICBzdGF0dXM6ICdoZWFsdGh5JyxcbiAgICAgICAgICBzZXJ2aWNlOiAnYnJvd3Nlci1hdXRvbWF0aW9uLXN0dWRpby11aScsXG4gICAgICAgICAgdGltZXN0YW1wOiBuZXcgRGF0ZSgpLnRvSVNPU3RyaW5nKCksXG4gICAgICAgICAgcmVhZGluZXNzOiB0cnVlLFxuICAgICAgICAgIGFwaV9jb25uZWN0aXZpdHk6IHtcbiAgICAgICAgICAgIGNvbm5lY3RlZDogZmFsc2UsXG4gICAgICAgICAgICBhcGlfdXJsOiBBUElfUE9SVCA/IGBodHRwOi8vJHtBUElfSE9TVH06JHtBUElfUE9SVH0vYXBpL3YxYCA6IG51bGwsXG4gICAgICAgICAgICBsYXN0X2NoZWNrOiBuZXcgRGF0ZSgpLnRvSVNPU3RyaW5nKCksXG4gICAgICAgICAgICBlcnJvcjogbnVsbCxcbiAgICAgICAgICAgIGxhdGVuY3lfbXM6IG51bGwsXG4gICAgICAgICAgICB1cHN0cmVhbTogbnVsbFxuICAgICAgICAgIH1cbiAgICAgICAgfSk7XG5cbiAgICAgICAgY29uc3QgaGVhbHRoUmVzcG9uc2UgPSBidWlsZEhlYWx0aFBheWxvYWQoKTtcblxuICAgICAgICAvLyBBdHRlbXB0IHRvIGNoZWNrIEFQSSBoZWFsdGhcbiAgICAgICAgaWYgKEFQSV9QT1JUKSB7XG4gICAgICAgICAgY29uc3Qgc3RhcnRUaW1lID0gRGF0ZS5ub3coKTtcbiAgICAgICAgICB0cnkge1xuICAgICAgICAgICAgYXdhaXQgbmV3IFByb21pc2UoKHJlc29sdmUsIHJlamVjdCkgPT4ge1xuICAgICAgICAgICAgICBjb25zdCBoZWFsdGhSZXEgPSBodHRwLnJlcXVlc3QoXG4gICAgICAgICAgICAgICAge1xuICAgICAgICAgICAgICAgICAgaG9zdG5hbWU6IEFQSV9IT1NULFxuICAgICAgICAgICAgICAgICAgcG9ydDogQVBJX1BPUlQsXG4gICAgICAgICAgICAgICAgICBwYXRoOiAnL2hlYWx0aCcsXG4gICAgICAgICAgICAgICAgICBtZXRob2Q6ICdHRVQnLFxuICAgICAgICAgICAgICAgICAgdGltZW91dDogMjAwMCxcbiAgICAgICAgICAgICAgICB9LFxuICAgICAgICAgICAgICAgIChoZWFsdGhSZXMpID0+IHtcbiAgICAgICAgICAgICAgICAgIGhlYWx0aFJlc3BvbnNlLmFwaV9jb25uZWN0aXZpdHkubGF0ZW5jeV9tcyA9IERhdGUubm93KCkgLSBzdGFydFRpbWU7XG4gICAgICAgICAgICAgICAgICBoZWFsdGhSZXNwb25zZS5hcGlfY29ubmVjdGl2aXR5Lmxhc3RfY2hlY2sgPSBuZXcgRGF0ZSgpLnRvSVNPU3RyaW5nKCk7XG4gICAgICAgICAgICAgICAgICBoZWFsdGhSZXNwb25zZS5hcGlfY29ubmVjdGl2aXR5LmNvbm5lY3RlZCA9IGhlYWx0aFJlcy5zdGF0dXNDb2RlID49IDIwMCAmJiBoZWFsdGhSZXMuc3RhdHVzQ29kZSA8IDMwMDtcblxuICAgICAgICAgICAgICAgICAgaWYgKCFoZWFsdGhSZXNwb25zZS5hcGlfY29ubmVjdGl2aXR5LmNvbm5lY3RlZCkge1xuICAgICAgICAgICAgICAgICAgICBoZWFsdGhSZXNwb25zZS5zdGF0dXMgPSAnZGVncmFkZWQnO1xuICAgICAgICAgICAgICAgICAgICBoZWFsdGhSZXNwb25zZS5hcGlfY29ubmVjdGl2aXR5LmVycm9yID0ge1xuICAgICAgICAgICAgICAgICAgICAgIGNvZGU6IGBIVFRQXyR7aGVhbHRoUmVzLnN0YXR1c0NvZGV9YCxcbiAgICAgICAgICAgICAgICAgICAgICBtZXNzYWdlOiBgQVBJIHJldHVybmVkIHN0YXR1cyAke2hlYWx0aFJlcy5zdGF0dXNDb2RlfWAsXG4gICAgICAgICAgICAgICAgICAgICAgY2F0ZWdvcnk6ICduZXR3b3JrJyxcbiAgICAgICAgICAgICAgICAgICAgICByZXRyeWFibGU6IHRydWVcbiAgICAgICAgICAgICAgICAgICAgfTtcbiAgICAgICAgICAgICAgICAgIH1cblxuICAgICAgICAgICAgICAgICAgbGV0IGJvZHkgPSAnJztcbiAgICAgICAgICAgICAgICAgIGhlYWx0aFJlcy5vbignZGF0YScsIChjaHVuaykgPT4geyBib2R5ICs9IGNodW5rOyB9KTtcbiAgICAgICAgICAgICAgICAgIGhlYWx0aFJlcy5vbignZW5kJywgKCkgPT4ge1xuICAgICAgICAgICAgICAgICAgICB0cnkge1xuICAgICAgICAgICAgICAgICAgICAgIGhlYWx0aFJlc3BvbnNlLmFwaV9jb25uZWN0aXZpdHkudXBzdHJlYW0gPSBKU09OLnBhcnNlKGJvZHkpO1xuICAgICAgICAgICAgICAgICAgICB9IGNhdGNoIHtcbiAgICAgICAgICAgICAgICAgICAgICBoZWFsdGhSZXNwb25zZS5hcGlfY29ubmVjdGl2aXR5LnVwc3RyZWFtID0gYm9keTtcbiAgICAgICAgICAgICAgICAgICAgfVxuICAgICAgICAgICAgICAgICAgICByZXNvbHZlKCk7XG4gICAgICAgICAgICAgICAgICB9KTtcbiAgICAgICAgICAgICAgICB9XG4gICAgICAgICAgICAgICk7XG5cbiAgICAgICAgICAgICAgaGVhbHRoUmVxLm9uKCdlcnJvcicsIChlcnJvcikgPT4ge1xuICAgICAgICAgICAgICAgIGhlYWx0aFJlc3BvbnNlLmFwaV9jb25uZWN0aXZpdHkubGF0ZW5jeV9tcyA9IERhdGUubm93KCkgLSBzdGFydFRpbWU7XG4gICAgICAgICAgICAgICAgaGVhbHRoUmVzcG9uc2UuYXBpX2Nvbm5lY3Rpdml0eS5jb25uZWN0ZWQgPSBmYWxzZTtcbiAgICAgICAgICAgICAgICBoZWFsdGhSZXNwb25zZS5hcGlfY29ubmVjdGl2aXR5LmVycm9yID0ge1xuICAgICAgICAgICAgICAgICAgY29kZTogJ0NPTk5FQ1RJT05fRVJST1InLFxuICAgICAgICAgICAgICAgICAgbWVzc2FnZTogYEZhaWxlZCB0byBjb25uZWN0IHRvIEFQSTogJHtlcnJvci5tZXNzYWdlfWAsXG4gICAgICAgICAgICAgICAgICBjYXRlZ29yeTogJ25ldHdvcmsnLFxuICAgICAgICAgICAgICAgICAgcmV0cnlhYmxlOiB0cnVlXG4gICAgICAgICAgICAgICAgfTtcbiAgICAgICAgICAgICAgICBoZWFsdGhSZXNwb25zZS5zdGF0dXMgPSAnZGVncmFkZWQnO1xuICAgICAgICAgICAgICAgIHJlc29sdmUoKTtcbiAgICAgICAgICAgICAgfSk7XG5cbiAgICAgICAgICAgICAgaGVhbHRoUmVxLm9uKCd0aW1lb3V0JywgKCkgPT4ge1xuICAgICAgICAgICAgICAgIGhlYWx0aFJlcS5kZXN0cm95KCk7XG4gICAgICAgICAgICAgICAgaGVhbHRoUmVzcG9uc2UuYXBpX2Nvbm5lY3Rpdml0eS5sYXRlbmN5X21zID0gRGF0ZS5ub3coKSAtIHN0YXJ0VGltZTtcbiAgICAgICAgICAgICAgICBoZWFsdGhSZXNwb25zZS5hcGlfY29ubmVjdGl2aXR5LmNvbm5lY3RlZCA9IGZhbHNlO1xuICAgICAgICAgICAgICAgIGhlYWx0aFJlc3BvbnNlLmFwaV9jb25uZWN0aXZpdHkuZXJyb3IgPSB7XG4gICAgICAgICAgICAgICAgICBjb2RlOiAnVElNRU9VVCcsXG4gICAgICAgICAgICAgICAgICBtZXNzYWdlOiAnQVBJIGhlYWx0aCBjaGVjayB0aW1lZCBvdXQnLFxuICAgICAgICAgICAgICAgICAgY2F0ZWdvcnk6ICduZXR3b3JrJyxcbiAgICAgICAgICAgICAgICAgIHJldHJ5YWJsZTogdHJ1ZVxuICAgICAgICAgICAgICAgIH07XG4gICAgICAgICAgICAgICAgaGVhbHRoUmVzcG9uc2Uuc3RhdHVzID0gJ2RlZ3JhZGVkJztcbiAgICAgICAgICAgICAgICByZXNvbHZlKCk7XG4gICAgICAgICAgICAgIH0pO1xuXG4gICAgICAgICAgICAgIGhlYWx0aFJlcS5lbmQoKTtcbiAgICAgICAgICAgIH0pO1xuICAgICAgICAgIH0gY2F0Y2ggKGVycm9yKSB7XG4gICAgICAgICAgICAvLyBBbHJlYWR5IGhhbmRsZWQgaW4gcHJvbWlzZVxuICAgICAgICAgIH1cbiAgICAgICAgfSBlbHNlIHtcbiAgICAgICAgICBoZWFsdGhSZXNwb25zZS5zdGF0dXMgPSAnZGVncmFkZWQnO1xuICAgICAgICAgIGhlYWx0aFJlc3BvbnNlLmFwaV9jb25uZWN0aXZpdHkuZXJyb3IgPSB7XG4gICAgICAgICAgICBjb2RlOiAnTUlTU0lOR19DT05GSUcnLFxuICAgICAgICAgICAgbWVzc2FnZTogJ0FQSV9QT1JUIGVudmlyb25tZW50IHZhcmlhYmxlIG5vdCBjb25maWd1cmVkJyxcbiAgICAgICAgICAgIGNhdGVnb3J5OiAnY29uZmlndXJhdGlvbicsXG4gICAgICAgICAgICByZXRyeWFibGU6IGZhbHNlXG4gICAgICAgICAgfTtcbiAgICAgICAgfVxuXG4gICAgICAgIHJlcy5zZXRIZWFkZXIoJ0NvbnRlbnQtVHlwZScsICdhcHBsaWNhdGlvbi9qc29uJyk7XG4gICAgICAgIHJlcy5lbmQoSlNPTi5zdHJpbmdpZnkoaGVhbHRoUmVzcG9uc2UpKTtcbiAgICAgICAgcmV0dXJuO1xuICAgICAgfVxuICAgICAgbmV4dCgpO1xuICAgIH0pO1xuICB9XG59KTtcblxuZXhwb3J0IGRlZmF1bHQgZGVmaW5lQ29uZmlnKHtcbiAgcGx1Z2luczogW3JlYWN0KCksIGhlYWx0aEVuZHBvaW50UGx1Z2luKCldLFxuICByZXNvbHZlOiB7XG4gICAgYWxpYXM6IHtcbiAgICAgICdAJzogcGF0aC5yZXNvbHZlKF9fZGlybmFtZSwgJy4vc3JjJyksXG4gICAgICAnQGNvbXBvbmVudHMnOiBwYXRoLnJlc29sdmUoX19kaXJuYW1lLCAnLi9zcmMvY29tcG9uZW50cycpLFxuICAgICAgJ0Bob29rcyc6IHBhdGgucmVzb2x2ZShfX2Rpcm5hbWUsICcuL3NyYy9ob29rcycpLFxuICAgICAgJ0BzdG9yZXMnOiBwYXRoLnJlc29sdmUoX19kaXJuYW1lLCAnLi9zcmMvc3RvcmVzJyksXG4gICAgICAnQHV0aWxzJzogcGF0aC5yZXNvbHZlKF9fZGlybmFtZSwgJy4vc3JjL3V0aWxzJyksXG4gICAgICAnQHR5cGVzJzogcGF0aC5yZXNvbHZlKF9fZGlybmFtZSwgJy4vc3JjL3R5cGVzJyksXG4gICAgICAnQGFwaSc6IHBhdGgucmVzb2x2ZShfX2Rpcm5hbWUsICcuL3NyYy9hcGknKSxcbiAgICB9LFxuICB9LFxuICBzZXJ2ZXI6IHtcbiAgICBwb3J0OiBwYXJzZUludChVSV9QT1JUKSxcbiAgICBob3N0OiB0cnVlLFxuICAgIGNvcnM6IHRydWUsXG4gICAgcHJveHk6IHtcbiAgICAgICcvYXBpJzoge1xuICAgICAgICB0YXJnZXQ6IGBodHRwOi8vJHtBUElfSE9TVH06JHtBUElfUE9SVH1gLFxuICAgICAgICBjaGFuZ2VPcmlnaW46IHRydWUsXG4gICAgICAgIHNlY3VyZTogZmFsc2UsXG4gICAgICB9LFxuICAgICAgJy93cyc6IHtcbiAgICAgICAgdGFyZ2V0OiBgd3M6Ly8ke1dTX0hPU1R9OiR7V1NfUE9SVH1gLFxuICAgICAgICB3czogdHJ1ZSxcbiAgICAgICAgY2hhbmdlT3JpZ2luOiB0cnVlLFxuICAgICAgfSxcbiAgICB9LFxuICB9LFxuICBkZWZpbmU6IHtcbiAgICAvLyBQYXNzIGVudmlyb25tZW50IHZhcmlhYmxlcyB0byB0aGUgY2xpZW50IHdpdGggVklURV8gcHJlZml4IGZvciBkZXYgbW9kZVxuICAgICdpbXBvcnQubWV0YS5lbnYuVklURV9BUElfVVJMJzogSlNPTi5zdHJpbmdpZnkocHJvY2Vzcy5lbnYuQVBJX1BPUlQgPyBgaHR0cDovLyR7QVBJX0hPU1R9OiR7cHJvY2Vzcy5lbnYuQVBJX1BPUlR9L2FwaS92MWAgOiB1bmRlZmluZWQpLFxuICAgICdpbXBvcnQubWV0YS5lbnYuVklURV9XU19VUkwnOiBKU09OLnN0cmluZ2lmeShwcm9jZXNzLmVudi5XU19QT1JUID8gYHdzOi8vJHtXU19IT1NUfToke3Byb2Nlc3MuZW52LldTX1BPUlR9YCA6IHVuZGVmaW5lZCksXG4gICAgJ2ltcG9ydC5tZXRhLmVudi5WSVRFX0FQSV9QT1JUJzogSlNPTi5zdHJpbmdpZnkocHJvY2Vzcy5lbnYuQVBJX1BPUlQgfHwgdW5kZWZpbmVkKSxcbiAgICAnaW1wb3J0Lm1ldGEuZW52LlZJVEVfVUlfUE9SVCc6IEpTT04uc3RyaW5naWZ5KHByb2Nlc3MuZW52LlVJX1BPUlQgfHwgdW5kZWZpbmVkKSxcbiAgICAnaW1wb3J0Lm1ldGEuZW52LlZJVEVfV1NfUE9SVCc6IEpTT04uc3RyaW5naWZ5KHByb2Nlc3MuZW52LldTX1BPUlQgfHwgdW5kZWZpbmVkKSxcbiAgfSxcbiAgYnVpbGQ6IHtcbiAgICBvdXREaXI6ICdkaXN0JyxcbiAgICBzb3VyY2VtYXA6IHRydWUsXG4gICAgbWluaWZ5OiAnZXNidWlsZCcsXG4gICAgdGFyZ2V0OiAnZXNuZXh0JyxcbiAgfSxcbiAgb3B0aW1pemVEZXBzOiB7XG4gICAgaW5jbHVkZTogWydsdWNpZGUtcmVhY3QnXSxcbiAgfSxcbn0pO1xuIl0sCiAgIm1hcHBpbmdzIjogIjtBQUFxWCxTQUFTLG9CQUFvQjtBQUNsWixPQUFPLFdBQVc7QUFDbEIsT0FBTyxVQUFV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
