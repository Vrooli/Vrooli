import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

const sanitize = (value?: string, fallback = '') => {
  if (!value) {
    return fallback;
  }
  const trimmed = value.trim();
  return trimmed.length > 0 ? trimmed : fallback;
};

const apiProtocol = sanitize(process.env.API_PROTOCOL, 'http').replace(/:+$/, '') || 'http'
const apiHost = sanitize(process.env.API_HOST, '127.0.0.1') || '127.0.0.1'
const apiPort = sanitize(process.env.API_PORT, '8080')

const explicitProxyTarget = sanitize(process.env.API_PROXY_TARGET)
  || sanitize(process.env.API_BASE_URL)
  || sanitize(process.env.VITE_API_BASE_URL)

const fallbackProxyTarget = `${apiProtocol}://${apiHost}${apiPort ? `:${apiPort}` : ''}`
const proxyTarget = explicitProxyTarget || fallbackProxyTarget

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    port: parseInt(process.env.UI_PORT || process.env.PORT || '3000'),
    host: '0.0.0.0',
    allowedHosts: ['system-monitor.itsagitime.com'],
    proxy: {
      // Proxy API calls to the Go backend
      '/api': {
        target: proxyTarget,
        changeOrigin: true,
        secure: false,
      },
      // Proxy health checks
      '/health': {
        target: proxyTarget,
        changeOrigin: true,
        secure: false,
      }
    }
  },
  build: {
    outDir: 'dist',
    assetsDir: 'assets',
    sourcemap: true,
  }
})
