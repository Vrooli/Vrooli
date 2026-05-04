import path from "node:path";
import { defineConfig, type UserConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig(({ mode }): UserConfig => {
  const isProfile = mode === "profile";

  return {
    base: './',  // Required for tunnel/proxy contexts
    resolve: {
      alias: {
        "@proto-lprv": path.resolve(__dirname, "../../../packages/proto/gen/typescript/landing-page-react-vite/v1"),
        ...(isProfile
          ? {
              "react-dom/client": "react-dom/profiling",
              "react-dom$": "react-dom/profiling",
            }
          : {}),
      },
      // Ensure modules imported from proto files resolve from UI's node_modules
      dedupe: ["@bufbuild/protobuf"],
    },
    esbuild: isProfile
      ? {
          keepNames: true,
        }
      : undefined,
    server: {
      fs: {
        allow: [path.resolve(__dirname, "../../../packages")],
      },
    },
    plugins: [react()],
    optimizeDeps: {
      include: ["@bufbuild/protobuf"],
    },
    build: {
      commonjsOptions: {
        include: [/node_modules/],
      },
      rollupOptions: {
        // Force resolution of @bufbuild/protobuf from UI's node_modules
        external: [],
      },
    },
    test: {
      globals: true,
      environment: 'jsdom',
      setupFiles: ['./src/test-setup.ts'],
      coverage: {
        provider: 'v8',
        reporter: ['json-summary', 'json', 'text'],
        reportOnFailure: true,
        thresholds: {
          lines: 0,
          functions: 0,
          branches: 0,
          statements: 0
        }
      }
    }
  };
});
