import path from "node:path";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vitest/config";

export default defineConfig(({ mode }) => {
  const isProfile = mode === "profile";

  return {
    base: './',  // Required for tunnel/proxy contexts
    resolve: {
      alias: {
        "@vrooli/proto-types/landing-page-business-suite": path.resolve(__dirname, "../../../packages/proto/gen/typescript/landing-page-business-suite/v1"),
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
        output: {
          // Keep the initial public-landing payload focused on application code.
          // Monaco is only needed by the admin variant editor, while framework
          // packages are long-lived browser-cache candidates across deployments.
          manualChunks(id) {
            if (id.includes('/node_modules/monaco-editor/') || id.includes('/node_modules/@monaco-editor/')) {
              return 'monaco-editor';
            }
            if (
              id.includes('/node_modules/react/') ||
              id.includes('/node_modules/react-dom/') ||
              id.includes('/node_modules/react-router/') ||
              id.includes('/node_modules/react-router-dom/')
            ) {
              return 'react-framework';
            }
          },
        },
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
