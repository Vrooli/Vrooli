import { defineConfig, type UserConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

// https://vitejs.dev/config/
export default defineConfig(({ mode }): UserConfig => {
  const isProfile = mode === 'profile'

  return {
    // INTEROP-CRITICAL: Relative assets keep prompt-manager functional behind Vrooli tunnels/proxies.
    base: './',
    plugins: [react()],
    resolve: {
      alias: {
        '@': path.resolve(__dirname, './src'),
        ...(isProfile
          ? {
              'react-dom/client': 'react-dom/profiling',
              'react-dom$': 'react-dom/profiling',
            }
          : {}),
      },
    },
    esbuild: isProfile ? { keepNames: true } : undefined,
    server: {
      port: 3000,
      open: false,
      host: true,
    },
    build: {
      outDir: 'dist',
      sourcemap: true,
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
      // INTEROP-CRITICAL: Some browser-side dependencies probe process.env; provide an empty shim.
      'process.env': {}
    }
  }
})
