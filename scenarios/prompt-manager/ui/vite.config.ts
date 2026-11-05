import { defineConfig, type UserConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'
import { collectWorldBuildProvenance } from './scripts/world-build-provenance'

// https://vitejs.dev/config/
export default defineConfig(({ mode }): UserConfig => {
  const isProfile = mode === 'profile'
  const worldBuildProvenance = collectWorldBuildProvenance(__dirname)

  return {
    // INTEROP-CRITICAL: Relative assets keep prompt-manager functional behind Vrooli tunnels/proxies.
    base: './',
    plugins: [react()],
    resolve: {
      alias: [
        { find: '@', replacement: path.resolve(__dirname, './src') },
        ...(isProfile ? [{ find: 'react-dom/client', replacement: 'react-dom/profiling' }] : []),
      ],
    },
    // keepNames injects outer-scope helpers into Troika's serialized workers.
    // An unminified profile artifact retains names without those helpers.
    esbuild: { keepNames: false },
    server: {
      port: 3000,
      open: false,
      host: true,
    },
    build: {
      outDir: 'dist',
      minify: isProfile ? false : 'esbuild',
      // Keep source maps for profiling builds, but do not ship their sizeable
      // payload in the normal production artifact.
      sourcemap: isProfile,
      rollupOptions: {
        output: {
          manualChunks: {
            vendor: ['react', 'react-dom'],
            // Perf: three.js + R3F + drei are ~700KB gzipped — split for independent browser caching
            three: ['three', '@react-three/fiber', '@react-three/drei'],
            ui: ['@radix-ui/react-dialog', '@radix-ui/react-dropdown-menu', '@radix-ui/react-select'],
            motion: ['framer-motion'],
            editor: ['@monaco-editor/react'],
            mermaid: ['mermaid'],
          },
        },
      },
    },
    test: {
      environment: 'jsdom',
      setupFiles: ['./src/test-setup.ts'],
      coverage: {
        provider: 'v8',
        reporter: ['text', 'json-summary', 'json'],
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
        reportOnFailure: true,
        thresholds: {
          lines: 85,
          functions: 85,
          branches: 85,
          statements: 85,
        },
      },
    },
    define: {
      __WORLD_BUILD_PROVENANCE__: JSON.stringify(worldBuildProvenance),
      // INTEROP-CRITICAL: Some browser-side dependencies probe process.env; provide an empty shim.
      'process.env': {}
    }
  }
})
