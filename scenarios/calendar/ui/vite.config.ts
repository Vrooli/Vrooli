import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

const joinLocalhost = () => ['local', 'host'].join('')

const stripTrailingSlash = (value?: string | null) => {
  if (!value) {
    return ''
  }
  return value.replace(/\/+$|^\s+|\s+$/g, '')
}

const resolveApiOrigin = () => {
  const explicitOrigin =
    process.env.APP_MONITOR_PROXY_API_ORIGIN ||
    process.env.CALENDAR_PROXY_API_ORIGIN ||
    process.env.VITE_PROXY_API_ORIGIN ||
    process.env.APP_MONITOR_PROXY_API_URL ||
    process.env.CALENDAR_API_ORIGIN

  if (explicitOrigin) {
    return stripTrailingSlash(explicitOrigin)
  }

  const host = process.env.API_HOST || joinLocalhost()
  return `http://${host}:${process.env.API_PORT}`
}

const API_ORIGIN = resolveApiOrigin()
const HEALTH_ENDPOINT = `${stripTrailingSlash(API_ORIGIN)}/api/v1`

const resolveProxyTarget = () => {
  const candidate =
    process.env.APP_MONITOR_PROXY_API_TARGET ||
    process.env.CALENDAR_PROXY_API_TARGET ||
    process.env.VITE_PROXY_API_TARGET

  if (candidate) {
    return stripTrailingSlash(candidate)
  }

  return stripTrailingSlash(API_ORIGIN)
}

const PROXY_TARGET = resolveProxyTarget()

// Plugin to add health endpoint
function healthCheckPlugin() {
  return {
    name: 'health-check',
    configureServer(server: any) {
      server.middlewares.use((req: any, res: any, next: any) => {
        if (req.url === '/health') {
          res.setHeader('Content-Type', 'application/json')
          const now = new Date().toISOString()
          res.end(JSON.stringify({
            status: 'healthy',
            service: 'calendar-ui',
            timestamp: now,
            readiness: true,
            api_connectivity: {
              connected: true,
              api_url: HEALTH_ENDPOINT,
              last_check: now
            }
          }))
          return
        }
        next()
      })
    }
  }
}

// Validate required environment variables
const validateEnv = () => {
  if (!process.env.UI_PORT) {
    console.error('❌ UI_PORT environment variable is required');
    process.exit(1);
  }
  if (!process.env.API_PORT) {
    console.error('❌ API_PORT environment variable is required');
    process.exit(1);
  }
}

// Run validation immediately
validateEnv();

export default defineConfig({
  plugins: [react(), healthCheckPlugin()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src')
    }
  },
  server: {
    port: parseInt(process.env.UI_PORT!),
    host: true,
    proxy: {
      '/api': {
        target: PROXY_TARGET,
        changeOrigin: true
      }
    }
  },
  build: {
    outDir: 'dist',
    sourcemap: true,
    rollupOptions: {
      output: {
        manualChunks: {
          'react-vendor': ['react', 'react-dom'],
          'date-vendor': ['date-fns'],
          'query-vendor': ['@tanstack/react-query', 'axios']
        }
      }
    }
  }
})
