import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// Follow @vrooli/api-base guidance: always build production bundles with a
// relative base path so proxy/tunnel deployments resolve assets correctly.
export default defineConfig({
  base: './',
  plugins: [react()],
  build: {
    outDir: 'dist',
    assetsDir: 'assets',
    sourcemap: true,
  },
})
