import { defineConfig, loadEnv } from 'vite'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const apiPort = Number(env.API_PORT || process.env.API_PORT || 0) || 3000
  const uiPort = Number(env.UI_PORT || process.env.UI_PORT || 0) || 5173

  return {
    base: './',
    build: {
      outDir: 'dist',
      assetsDir: 'assets',
      target: 'es2018',
      emptyOutDir: true,
      sourcemap: true
    },
    publicDir: 'public',
    server: {
      host: true,
      port: uiPort,
      proxy: {
        '/api': {
          target: `http://localhost:${apiPort}`,
          changeOrigin: true
        },
        '/ws': {
          target: `ws://localhost:${apiPort}`,
          ws: true,
          changeOrigin: true
        }
      }
    }
  }
})
