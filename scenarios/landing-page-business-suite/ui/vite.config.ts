import path from "node:path";
import { defineConfig, type UserConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig(({ mode }): UserConfig => {
  const isProfile = mode === "profile";

  return {
    base: './',  // Required for tunnel/proxy contexts
    resolve: {
      alias: {
        "@proto-lpbs": path.resolve(__dirname, "../../../packages/proto/gen/typescript/landing-page-business-suite/v1"),
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
        include: ['src/**/*.{ts,tsx}'],
        exclude: [
          'src/**/*.test.{ts,tsx}',
          'src/**/*.spec.{ts,tsx}',
          'src/**/*.d.ts',
          'src/main.tsx',
          'src/test-setup.ts',
          'src/test-utils/**',
          'src/consts/strings.generated.ts',
          'src/i18n/locales/**',
          'src/**/generated/**',
        ],
        thresholds: {
          lines: 85,
          functions: 85,
          branches: 85,
          statements: 85
        }
      }
    }
  };
});
