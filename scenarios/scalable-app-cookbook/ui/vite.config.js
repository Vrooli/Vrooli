import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// The port default lives here rather than in the npm scripts: package scripts
// run through cmd.exe on Windows, which cannot expand ${UI_PORT:-3301}.
const UI_PORT = parseInt(process.env.UI_PORT) || 3301

export default defineConfig({
  base: './',
  plugins: [react()],
  server: {
    port: UI_PORT,
    host: true,
  },
  preview: {
    port: UI_PORT,
    host: true,
  },
})
