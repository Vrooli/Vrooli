import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: [
      { find: '@', replacement: path.resolve(__dirname, 'src') },
      {
        find: '@vrooli/iframe-bridge/child',
        replacement: path.resolve(__dirname, '../../../packages/iframe-bridge/dist/iframeBridgeChild.js'),
      },
      {
        find: '@vrooli/iframe-bridge',
        replacement: path.resolve(__dirname, '../../../packages/iframe-bridge/dist/index.js'),
      },
      {
        find: '@vrooli/api-base',
        replacement: path.resolve(__dirname, '../../../packages/api-base/dist/index.js'),
      },
    ],
  },
  build: {
    sourcemap: true
  },
  server: {
    port: 5173,
    host: true,
    fs: {
      allow: [path.resolve(__dirname, '../../../packages')],
    },
  }
})
