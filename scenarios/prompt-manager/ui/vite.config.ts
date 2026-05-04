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
    define: {
      // INTEROP-CRITICAL: Some browser-side dependencies probe process.env; provide an empty shim.
      'process.env': {}
    }
  }
})
