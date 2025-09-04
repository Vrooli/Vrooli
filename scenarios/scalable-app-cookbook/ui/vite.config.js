import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 3301,
    host: true,
  },
  build: {
    outDir: 'dist',
    sourcemap: true,
  },
})