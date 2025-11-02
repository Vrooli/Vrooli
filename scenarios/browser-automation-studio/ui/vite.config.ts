import { defineConfig } from 'vite';
import type { ViteDevServer, Plugin, Connect } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';
import http from 'http';

// Get environment variables with fallbacks for build time
// AUDITOR NOTE: These fallbacks are intentional for development mode.
// The lifecycle system provides proper environment variables in production.
// Vite requires build-time values; undefined is used to signal missing config.
const UI_PORT = process.env.UI_PORT || '3000';
const API_PORT = process.env.API_PORT || '8080';
const WS_PORT = process.env.WS_PORT || '8081';

const API_HOST = process.env.API_HOST || 'localhost';
const WS_HOST = process.env.WS_HOST || 'localhost';

const bootstrapEntry = path.resolve(__dirname, 'src/bootstrap.tsx');
const mainEntry = path.resolve(__dirname, 'index.html');
const composerEntry = path.resolve(__dirname, 'src/export/composer.html');

interface HealthResponse {
  status: 'healthy' | 'degraded';
  service: string;
  timestamp: string;
  readiness: boolean;
  api_connectivity: {
    connected: boolean;
    api_url: string | null;
    last_check: string;
    error: { code: string; message: string; category: string; retryable: boolean } | null;
    latency_ms: number | null;
    upstream: unknown;
  };
}

// Health endpoint middleware plugin for dev mode
// Extracted logic to reduce config file complexity
const healthEndpointPlugin = (): Plugin => ({
  name: 'health-endpoint',
  configureServer(server: ViteDevServer) {
    server.middlewares.use(async (req: Connect.IncomingMessage, res: http.ServerResponse, next: () => void) => {
      if (req.url !== '/health') {
        return next();
      }

      const healthResponse: HealthResponse = {
        status: 'healthy',
        service: 'browser-automation-studio-ui',
        timestamp: new Date().toISOString(),
        readiness: true,
        api_connectivity: {
          connected: false,
          api_url: API_PORT ? `http://${API_HOST}:${API_PORT}/api/v1` : null,
          last_check: new Date().toISOString(),
          error: null,
          latency_ms: null,
          upstream: null,
        },
      };

      if (API_PORT) {
        const startTime = Date.now();
        try {
          await new Promise<void>((resolve) => {
            const healthReq = http.request(
              {
                hostname: API_HOST,
                port: API_PORT,
                path: '/health',
                method: 'GET',
                timeout: 2000,
              },
              (healthRes) => {
                healthResponse.api_connectivity.latency_ms = Date.now() - startTime;
                healthResponse.api_connectivity.last_check = new Date().toISOString();
                healthResponse.api_connectivity.connected =
                  healthRes.statusCode !== undefined && healthRes.statusCode >= 200 && healthRes.statusCode < 300;

                if (!healthResponse.api_connectivity.connected) {
                  healthResponse.status = 'degraded';
                  healthResponse.api_connectivity.error = {
                    code: `HTTP_${healthRes.statusCode}`,
                    message: `API returned status ${healthRes.statusCode}`,
                    category: 'network',
                    retryable: true,
                  };
                }

                let body = '';
                healthRes.on('data', (chunk: Buffer) => {
                  body += chunk.toString();
                });
                healthRes.on('end', () => {
                  try {
                    healthResponse.api_connectivity.upstream = JSON.parse(body);
                  } catch {
                    healthResponse.api_connectivity.upstream = body;
                  }
                  resolve();
                });
              }
            );

            healthReq.on('error', (error: Error) => {
              healthResponse.api_connectivity.latency_ms = Date.now() - startTime;
              healthResponse.api_connectivity.connected = false;
              healthResponse.api_connectivity.error = {
                code: 'CONNECTION_ERROR',
                message: `Failed to connect to API: ${error.message}`,
                category: 'network',
                retryable: true,
              };
              healthResponse.status = 'degraded';
              resolve();
            });

            healthReq.on('timeout', () => {
              healthReq.destroy();
              healthResponse.api_connectivity.latency_ms = Date.now() - startTime;
              healthResponse.api_connectivity.connected = false;
              healthResponse.api_connectivity.error = {
                code: 'TIMEOUT',
                message: 'API health check timed out',
                category: 'network',
                retryable: true,
              };
              healthResponse.status = 'degraded';
              resolve();
            });

            healthReq.end();
          });
        } catch {
          // Error already handled in promise
        }
      } else {
        healthResponse.status = 'degraded';
        healthResponse.api_connectivity.error = {
          code: 'MISSING_CONFIG',
          message: 'API_PORT environment variable not configured',
          category: 'configuration',
          retryable: false,
        };
      }

      res.setHeader('Content-Type', 'application/json');
      res.end(JSON.stringify(healthResponse));
    });
  },
});

export default defineConfig({
  plugins: [react(), healthEndpointPlugin()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
      '@components': path.resolve(__dirname, './src/components'),
      '@hooks': path.resolve(__dirname, './src/hooks'),
      '@stores': path.resolve(__dirname, './src/stores'),
      '@utils': path.resolve(__dirname, './src/utils'),
      '@types': path.resolve(__dirname, './src/types'),
      '@api': path.resolve(__dirname, './src/api'),
    },
  },
  server: {
    port: parseInt(UI_PORT),
    host: true,
    cors: true,
    proxy: {
      '/api': {
        target: `http://${API_HOST}:${API_PORT}`,
        changeOrigin: true,
        secure: false,
      },
      '/ws': {
        target: `ws://${WS_HOST}:${WS_PORT}`,
        ws: true,
        changeOrigin: true,
      },
    },
  },
  define: {
    // Pass environment variables to the client with VITE_ prefix for dev mode
    'import.meta.env.VITE_API_URL': JSON.stringify(process.env.API_PORT ? `http://${API_HOST}:${process.env.API_PORT}/api/v1` : undefined),
    'import.meta.env.VITE_WS_URL': JSON.stringify(process.env.WS_PORT ? `ws://${WS_HOST}:${process.env.WS_PORT}` : undefined),
    'import.meta.env.VITE_API_PORT': JSON.stringify(process.env.API_PORT || undefined),
    'import.meta.env.VITE_UI_PORT': JSON.stringify(process.env.UI_PORT || undefined),
    'import.meta.env.VITE_WS_PORT': JSON.stringify(process.env.WS_PORT || undefined),
  },
  build: {
    outDir: 'dist',
    sourcemap: true,
    minify: 'esbuild',
    target: 'esnext',
    rollupOptions: {
      input: {
        main: mainEntry,
        bootstrap: bootstrapEntry,
        composer: composerEntry,
      },
      output: {
        entryFileNames: (chunk) => {
          if (chunk.name === 'bootstrap') {
            return 'bootstrap.js';
          }
          if (chunk.name === 'composer') {
            return 'export/composer.js';
          }
          return 'assets/[name]-[hash].js';
        },
        chunkFileNames: 'assets/[name]-[hash].js',
        assetFileNames: 'assets/[name]-[hash][extname]',
        // Manual chunk splitting to reduce large bundle sizes
        // Splits 609KB bootstrap.js into smaller, cacheable chunks
        manualChunks: (id) => {
          // React core dependencies
          if (id.includes('node_modules/react') || id.includes('node_modules/react-dom')) {
            return 'react-vendor';
          }
          // ReactFlow and related packages (largest dependency)
          if (id.includes('node_modules/reactflow') ||
              id.includes('node_modules/@reactflow') ||
              id.includes('node_modules/@xyflow')) {
            return 'reactflow';
          }
          // UI component libraries
          if (id.includes('node_modules/lucide-react') ||
              id.includes('node_modules/react-hot-toast') ||
              id.includes('node_modules/react-markdown') ||
              id.includes('node_modules/react-syntax-highlighter') ||
              id.includes('node_modules/react-split')) {
            return 'ui-vendor';
          }
          // State management
          if (id.includes('node_modules/zustand')) {
            return 'state-vendor';
          }
          // Utility libraries
          if (id.includes('node_modules/clsx') ||
              id.includes('node_modules/date-fns') ||
              id.includes('node_modules/tailwind-merge')) {
            return 'utils-vendor';
          }
          // All other node_modules
          if (id.includes('node_modules')) {
            return 'vendor';
          }
        },
      },
    },
    // Increase chunk size warning limit to 800KB to reduce noise
    // after splitting (previous single chunk was 609KB)
    chunkSizeWarningLimit: 800,
  },
  optimizeDeps: {
    include: ['lucide-react'],
  },
});
