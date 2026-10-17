import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'

// Mode-aware config. A regular `vite build` ships the lean prod artifact;
// `vite build --mode profile` produces a perf-build channel for performance
// tracing. The perf build is *still* a production bundle (minified, batched,
// no StrictMode double-renders) — it differs only in:
//
//   1. `react-dom/client` aliases to `react-dom/profiling` so React's
//      profiling instrumentation survives. This makes <React.Profiler>'s
//      onRender callbacks fire, which lets `src/lib/profiler.ts` emit
//      user_timing entries that show up in Chrome DevTools' Performance panel.
//   2. `esbuild.keepNames` preserves component/function names through
//      minification so CPU samples and React-track entries display real names
//      instead of mangled ones.
//
// Triggering the perf build:
//   - Direct:  `pnpm run build:profile` always uses --mode profile.
export default defineConfig(({ mode }) => {
  const isProfile = mode === 'profile'

  return {
    // ╔══════════════════════════════════════════════════════════════╗
    // ║  INTEROP-CRITICAL: Relative base for proxy/tunnel contexts  ║
    // ║                                                              ║
    // ║  When served through app-monitor's proxy at                  ║
    // ║  /apps/<name>/proxy/, absolute asset URLs (base: '/')        ║
    // ║  resolve to the domain root, breaking all JS/CSS loading.    ║
    // ║  Relative base ('./') makes assets resolve from the          ║
    // ║  current directory, which works in all three contexts.       ║
    // ║                                                              ║
    // ║  DO NOT change to '/' or remove this setting.                ║
    // ╚══════════════════════════════════════════════════════════════╝
    base: './',
    plugins: [react()],
    test: {
      globals: true,
      environment: 'jsdom',
      setupFiles: ['./src/test-setup.ts'],
      include: ['src/**/*.test.ts', 'src/**/*.test.tsx'],
      exclude: ['**/node_modules/**', '**/dist/**', '**/coverage/**'],
      passWithNoTests: true,
      clearMocks: true,
      restoreMocks: true,
      coverage: {
        provider: 'v8',
        reporter: ['text', 'json-summary', 'json'],
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
          statements: 85,
        },
      },
    },
    resolve: isProfile
      ? {
          alias: {
            'react-dom/client': 'react-dom/profiling',
            // Internal references inside react-dom/client.js do
            // `require('react-dom')`, which would resolve back to the
            // stripped-prod bundle. Force them through the profiling entry too.
            'react-dom$': 'react-dom/profiling',
          },
        }
      : undefined,
    esbuild: isProfile
      ? {
          keepNames: true,
        }
      : undefined,
    build: {
      outDir: 'dist',
      assetsDir: 'assets',
      sourcemap: isProfile,
      rollupOptions: {
        output: {
          // Split the vendor libraries that ARE on the initial-paint graph
          // (react core + router, lucide icons) into their own chunks so they
          // load in parallel and stay out of the main entry.
          //
          // recharts (~400 KB) and react-syntax-highlighter (~500 KB+) are
          // deliberately NOT listed here: they are reached only through the
          // React.lazy boundaries (detail routes / LazyScriptHighlighter), so
          // Rollup already emits them as async chunks. Forcing them into a
          // manualChunks group would tie them statically to the entry and make
          // Vite emit a <link rel="modulepreload"> for them — eagerly fetching
          // ~1 MB on the dashboard load, which is exactly what we're avoiding.
          manualChunks: {
            'react-vendor': ['react', 'react-dom', 'react-dom/client', 'react-router-dom'],
            icons: ['lucide-react'],
          },
        },
      },
    },
  }
})
