import react from '@vitejs/plugin-react';
import { readFileSync } from 'node:fs';
import RequirementReporter from '@vrooli/vitest-requirement-reporter';
import http from 'http';
import path from 'path';
import type { Connect, Plugin, ViteDevServer } from 'vite';
import { defineConfig, type UserConfig } from 'vite';
import { defineProject } from 'vitest/config';

// Get environment variables with fallbacks for build time
// AUDITOR NOTE: These fallbacks are intentional for development mode.
// The lifecycle system provides proper environment variables in production.
// Vite requires build-time values; undefined is used to signal missing config.
const UI_PORT = process.env.UI_PORT || '3000';
const API_PORT = process.env.API_PORT || '8080';
const API_HOST = process.env.API_HOST || 'localhost';

const bootstrapEntry = path.resolve(__dirname, 'src/bootstrap.tsx');
const mainEntry = path.resolve(__dirname, 'index.html');
const composerEntry = path.resolve(__dirname, 'src/export/composer.html');
const UI_SERVICE_NAME = 'browser-automation-studio';
const UI_VERSION = process.env.npm_package_version || '1.0.0';
const MAX_VITEST_THREADS = Math.max(1, Number(process.env.VITEST_MAX_THREADS ?? '2'));
const MIN_VITEST_THREADS = Math.max(1, Math.min(MAX_VITEST_THREADS, Number(process.env.VITEST_MIN_THREADS ?? '1')));
const COLLECTING_RAW_COVERAGE = process.env.BAS_COLLECT_RAW_COVERAGE === '1';
const { projects: PROJECTS, coverage: COVERAGE } = JSON.parse(readFileSync(new URL('./vitest.projects.json', import.meta.url), 'utf8')) as {
  projects: Array<{ name: string; include: string[]; exclude?: string[]; pool: 'threads' | 'forks' }>;
  coverage: { include: string[]; exclude: string[]; thresholds: { lines: number; functions: number; branches: number; statements: number } };
};
const PROJECT_BASE_TEST_CONFIG = {
  environment: 'jsdom',
  setupFiles: ['./src/test-setup.ts'],
  globals: true,
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
  version?: string;
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
        service: UI_SERVICE_NAME,
        timestamp: new Date().toISOString(),
        readiness: true,
        version: UI_VERSION,
        api_connectivity: {
          connected: false,
          api_url: API_PORT ? `http://${API_HOST}:${API_PORT}/health` : null,
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
  '@vrooli/generated-proto': path.resolve(__dirname, '../../../packages/proto/gen/typescript'),
  '@vrooli/proto-types': path.resolve(__dirname, '../../../packages/proto/gen/typescript'),
  '@bufbuild/protobuf/codegenv2': path.resolve(__dirname, 'node_modules/@bufbuild/protobuf/dist/esm/codegenv2/index.js'),
  '@bufbuild/protobuf/wkt': path.resolve(__dirname, 'node_modules/@bufbuild/protobuf/dist/esm/wkt/index.js'),
  '@bufbuild/protobuf/wire': path.resolve(__dirname, 'node_modules/@bufbuild/protobuf/dist/esm/wire/index.js'),
  '@bufbuild/protobuf': path.resolve(__dirname, 'node_modules/@bufbuild/protobuf/dist/esm/index.js'),
  '@': path.resolve(__dirname, './src'),
  '@shared': path.resolve(__dirname, './src/shared'),
  '@routing': path.resolve(__dirname, './src/routing'),
  '@hooks': path.resolve(__dirname, './src/hooks'),
  '@stores': path.resolve(__dirname, './src/stores'),
  '@utils': path.resolve(__dirname, './src/utils'),
  '@constants': path.resolve(__dirname, './src/constants'),
  '@lib': path.resolve(__dirname, './src/lib'),
  '@components': path.resolve(__dirname, './src/components'),
};

export default defineConfig(({ mode }): UserConfig => {
  const isProfile = mode === 'profile';

  return {
  // ╔══════════════════════════════════════════════════════════════╗
  // ║  INTEROP-CRITICAL: Relative base for proxy/tunnel contexts  ║
  // ║                                                              ║
  // ║  When served through app-monitor's proxy at                  ║
  // ║  /apps/<name>/proxy/, absolute asset URLs (base: '/')        ║
  // ║  resolve to the domain root, breaking all JS/CSS loading.    ║
  // ║  Relative base ('./') makes assets resolve from the          ║
  // ║  current directory, which works in all three contexts.       ║
  // ║                                                              ║
  // ║  DO NOT change to '/' or remove this setting.                ║
  // ╚══════════════════════════════════════════════════════════════╝
  base: './',
  plugins: [react(), healthEndpointPlugin()],
  resolve: {
    alias: isProfile
      ? {
          ...ALIASES,
          'react-dom/client': 'react-dom/profiling',
          'react-dom$': 'react-dom/profiling',
        }
      : ALIASES,
  },
  esbuild: isProfile
    ? {
        keepNames: true,
      }
    : undefined,
  test: {
    environment: 'jsdom',
    setupFiles: ['./src/test-setup.ts'],
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
      // Note: 'default' reporter removed when using conciseMode to eliminate verbose output
      // Our custom reporter provides all necessary output in a concise format
      new RequirementReporter({
        outputFile: process.env.BAS_REQUIREMENTS_OUTPUT_FILE ?? 'coverage/vitest-requirements.json',
        emitStdout: true,  // CRITICAL: Required for existing shell parsers
        verbose: true,
        conciseMode: true,  // Enable concise output with failure artifacts
        artifactsDir: process.env.BAS_REQUIREMENTS_ARTIFACTS_DIR ?? 'coverage/unit',
        autoClear: true,
      }),
    ],
    coverage: {
      provider: 'v8',
      // Focused commands validate behavior without being made to satisfy the
      // scenario-wide coverage gate. `test:coverage` switches to the dedicated
      // coverage config, which explicitly enables collection and enforces the
      // merged 85% floor after every selected project has contributed.
      enabled: COLLECTING_RAW_COVERAGE,
      // Focused projects are merged by the coverage runner. Their raw reports
      // must contain only modules the project actually executes; the normal
      // suite remains scenario-wide and includes every eligible source file.
      all: !COLLECTING_RAW_COVERAGE,
      reporter: ['text', 'json', 'html', 'json-summary'],
      reportsDirectory: './coverage',
      include: COVERAGE.include,
      exclude: COVERAGE.exclude,
      reportOnFailure: true,
      thresholds: COVERAGE.thresholds,
    },
    environmentOptions: {
      jsdom: {
        url: 'http://localhost:3000/',
      },
    },
    projects: PROJECTS.map((project) => defineProject({
      resolve: { alias: ALIASES },
      test: {
        ...PROJECT_BASE_TEST_CONFIG,
        name: project.name,
        include: project.include,
        exclude: project.exclude,
        pool: project.pool,
        poolOptions: project.pool === 'forks' ? FORKS_ONE : THREADS_TWO,
        ...(project.pool === 'forks' ? { reuseWorkers: false } : {}),
      },
    })),
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
        // Routed BAS validation marks the browser context with this header.
        // Preserve it explicitly across Vite's development proxy so in-page
        // Connect requests use the same leased database/filesystem as the
        // initiating execution rather than silently falling back to primary.
        configure: (proxy) => {
          proxy.on('proxyReq', (proxyReq, req) => {
            const testMode = req.headers['x-vrooli-test-mode']
            if (testMode === '1') {
              proxyReq.setHeader('X-Vrooli-Test-Mode', testMode)
            }
          })
        },
      },
      // WebSocket is served by the main API server on the same port
      '/ws': {
        target: `ws://${API_HOST}:${API_PORT}`,
        ws: true,
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: 'dist',
    modulePreload: {
      // Avoid preloading large optional chunks (Monaco/ReactFlow/bootstrap entry)
      // during initial page load to improve Lighthouse mobile performance.
      resolveDependencies: (_filename, deps) =>
        deps.filter((dep) =>
          !dep.includes('bootstrap.js') &&
          !dep.includes('monaco-') &&
          !dep.includes('reactflow-'),
        ),
    },
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
          // Monaco editor (very large; keep out of initial dashboard load)
          if (id.includes('node_modules/monaco-editor') ||
            id.includes('node_modules/@monaco-editor')) {
            return 'monaco';
          }
          // UI component libraries
          if (id.includes('node_modules/lucide-react') ||
            id.includes('node_modules/react-hot-toast')) {
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
          // Leave the remaining dependencies in the route graph. A single
          // catch-all vendor chunk would make every route's optional
          // dependency part of the initial modulepreload set.
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
  };
});
