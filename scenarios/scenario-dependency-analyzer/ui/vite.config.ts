import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig(({ mode }) => {
	const env = loadEnv(mode, process.cwd(), "");
	const isProfile = mode === "profile";

	return {
		base: "./",
		plugins: [react()],
		// Profile builds retain React commit instrumentation and component names
		// for performance-health traces. Normal production bundles remain lean.
		resolve: isProfile
			? {
				alias: {
					"react-dom/client": "react-dom/profiling",
					"react-dom$": "react-dom/profiling"
				}
			}
			: undefined,
		// Vite 8 otherwise selects OXC and silently ignores esbuild.keepNames.
		// The profiling build is diagnostic-only, so use esbuild there to retain
		// the component names emitted into React performance traces.
		oxc: isProfile ? false : undefined,
		esbuild: isProfile ? { keepNames: true } : undefined,
		build: isProfile
			? {
				// Vite 8 uses Rolldown for production output; this is the setting
				// that preserves function and class names in the emitted bundle.
				rolldownOptions: { output: { keepNames: true } }
			}
			: undefined,
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
        reportOnFailure: true,
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
      }
    }
  };
});
