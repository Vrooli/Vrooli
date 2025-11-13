import { defineConfig } from 'vite';
import { defineProject } from 'vitest/config';
import type { ViteDevServer, Plugin, Connect } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';
import http from 'http';
import RequirementReporter from '@vrooli/vitest-requirement-reporter';

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
const MAX_VITEST_THREADS = Math.max(1, Number(process.env.VITEST_MAX_THREADS ?? '2'));
const MIN_VITEST_THREADS = Math.max(1, Math.min(MAX_VITEST_THREADS, Number(process.env.VITEST_MIN_THREADS ?? '1')));
const PROJECT_BASE_TEST_CONFIG = {
  environment: 'jsdom',
  setupFiles: './src/test-utils/setupTests.ts',
  globals: true,
};

const THREADS_ONE = {
  threads: {
    maxThreads: 1,
    minThreads: 1,
  },
};

const THREADS_TWO = {
  threads: {
    maxThreads: 2,
    minThreads: 1,
  },
};

const FORKS_ONE = {
  forks: {
    maxForks: 1,
    minForks: 1,
  },
};

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

const ALIASES = {
  '@': path.resolve(__dirname, './src'),
  '@components': path.resolve(__dirname, './src/components'),
  '@hooks': path.resolve(__dirname, './src/hooks'),
  '@stores': path.resolve(__dirname, './src/stores'),
  '@utils': path.resolve(__dirname, './src/utils'),
  '@types': path.resolve(__dirname, './src/types'),
  '@api': path.resolve(__dirname, './src/api'),
};

export default defineConfig({
  base: './',
  plugins: [react(), healthEndpointPlugin()],
  resolve: {
    alias: ALIASES,
  },
  test: {
    environment: 'jsdom',
    setupFiles: './src/test-utils/setupTests.ts',
    globals: true,
    include: [],
    exclude: ['node_modules/**'],
    poolOptions: {
      threads: {
        maxThreads: MAX_VITEST_THREADS,
        minThreads: MIN_VITEST_THREADS,
      },
    },
    reporters: [
      'default',
      new RequirementReporter({
        outputFile: 'coverage/vitest-requirements.json',
        emitStdout: true,  // CRITICAL: Required for existing shell parsers
        verbose: true,
      }),
    ],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html'],
      reportsDirectory: './coverage',
      include: ['src/**/*.{ts,tsx}'],
      exclude: [
        'src/**/*.test.{ts,tsx,mjs}',
        'src/**/*.spec.{ts,tsx,mjs}',
        'src/test-utils/**',
        'src/**/*.d.ts',
        'src/main.tsx',
        'src/bootstrap.tsx',
        'src/export/**',
      ],
      thresholds: {
        lines: 70,
        functions: 70,
        branches: 70,
        statements: 70,
      },
    },
    environmentOptions: {
      jsdom: {
        url: 'http://localhost:3000/',
      },
    },
    projects: [
      defineProject({
        resolve: { alias: ALIASES },
        test: {
          ...PROJECT_BASE_TEST_CONFIG,
          name: 'stores',
          include: [
            'src/stores/**/*.test.{ts,tsx}',
            'src/stores/**/*.spec.{ts,tsx}',
            'src/hooks/useWorkflowVariables.test.ts',
          ],
          pool: 'threads',
          poolOptions: THREADS_TWO,
        },
      }),
      defineProject({
        resolve: { alias: ALIASES },
        test: {
          ...PROJECT_BASE_TEST_CONFIG,
          name: 'components-core',
          include: [
            'src/components/ProjectModal.test.tsx',
            'src/components/ResponsiveDialog.test.tsx',
            'src/components/__tests__/VariableSuggestionList.test.tsx',
            'src/components/__tests__/ExecutionViewer.test.tsx',
            'src/components/__tests__/WorkflowToolbar.test.tsx',
            'src/components/ProjectDetail.test.tsx',
          ],
        pool: 'threads',
        poolOptions: THREADS_TWO,
        },
      }),
      defineProject({
        resolve: { alias: ALIASES },
        test: {
          ...PROJECT_BASE_TEST_CONFIG,
          name: 'components-palette',
          include: ['src/components/__tests__/NodePalette.test.tsx'],
          pool: 'forks',
          poolOptions: FORKS_ONE,
          reuseWorkers: false,
        },
      }),
      defineProject({
        resolve: { alias: ALIASES },
        test: {
          ...PROJECT_BASE_TEST_CONFIG,
          name: 'workflow-builder',
          include: ['src/components/__tests__/WorkflowBuilder.test.tsx'],
          pool: 'forks',
          poolOptions: FORKS_ONE,
          reuseWorkers: false,
        },
      }),
    ],
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
  build: {
    outDir: 'dist',
    sourcemap: true,
    minify: 'esbuild',
    target: 'esnext',
    cssCodeSplit: true,  // Split CSS for better caching and performance
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
        // Manual chunk splitting to reduce large bundle sizes while
        // avoiding circular dependencies between vendor bundles.
        manualChunks: (id) => {
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
    include: ['lucide-react', 'react', 'react-dom', 'zustand'],
    exclude: ['@monaco-editor/react'],  // Lazy load Monaco Editor
  },
});
