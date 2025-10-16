import { defineConfig } from 'vite';
import type { ServerOptions } from 'vite';
import react from '@vitejs/plugin-react';

function requireEnv(name: string): string {
  const value = process.env[name];
  if (!value) {
    throw new Error(
      `${name} environment variable is required. Run this scenario through the Vrooli lifecycle so ports are provisioned automatically.`,
    );
  }
  return value;
}

function optionalNumberEnv(name: string): number | undefined {
  const raw = process.env[name];
  if (!raw) {
    return undefined;
  }

  const parsed = Number.parseInt(raw, 10);
  if (Number.isNaN(parsed)) {
    throw new Error(`${name} must be a valid integer (received: ${raw})`);
  }
  return parsed;
}

const apiPort = requireEnv('API_PORT');

const hmrHost = process.env.VITE_HMR_HOST;
const hmrPort = optionalNumberEnv('VITE_HMR_PORT');
const hmrProtocol = process.env.VITE_HMR_PROTOCOL as 'ws' | 'wss' | undefined;

const hmrConfig = hmrHost || hmrPort || hmrProtocol
  ? {
      ...(hmrHost ? { host: hmrHost } : {}),
      ...(hmrPort !== undefined ? { clientPort: hmrPort } : {}),
      ...(hmrProtocol ? { protocol: hmrProtocol } : {}),
    }
  : undefined;

const server: ServerOptions = {
  hmr: hmrConfig,
  proxy: {
    '/api': {
      target: `http://localhost:${apiPort}`,
      changeOrigin: true,
      ws: true, // Enable WebSocket proxying
    },
    '/health': {
      target: `http://localhost:${apiPort}`,
      changeOrigin: true,
    },
  },
};

const uiPort = optionalNumberEnv('UI_PORT');
if (uiPort !== undefined) {
  server.port = uiPort;
}

const serverHost = process.env.VITE_SERVER_HOST;
if (serverHost) {
  server.host = serverHost;
}

const allowedHosts = new Set<string>();

const allowedHostsRaw = process.env.VITE_ALLOWED_HOSTS;
if (allowedHostsRaw) {
  allowedHostsRaw
    .split(',')
    .map((host) => host.trim())
    .filter(Boolean)
    .forEach((host) => allowedHosts.add(host));
}

const allowedOriginsRaw = process.env.ALLOWED_ORIGINS;
if (allowedOriginsRaw) {
  allowedOriginsRaw
    .split(',')
    .map((origin) => origin.trim())
    .filter(Boolean)
    .forEach((origin) => {
      try {
        const parsed = new URL(origin);
        if (parsed.hostname) {
          allowedHosts.add(parsed.hostname);
        }
      } catch {
        // ignore malformed origins
      }
    });
}

if (allowedHosts.size > 0) {
  server.allowedHosts = Array.from(allowedHosts);
} else {
  // Allow lifecycle-managed domains (Vite will validate origin headers separately)
  server.allowedHosts = true;
}

export default defineConfig({
  plugins: [react()],
  define: {
    // Expose API port to the frontend for WebSocket connections
    __API_PORT__: JSON.stringify(apiPort),
  },
  server,
  test: {
    environment: 'jsdom',
    setupFiles: './src/test-utils/setupTests.ts',
    coverage: {
      reporter: ['text', 'html'],
    },
    environmentOptions: {
      jsdom: {
        url: 'http://localhost/',
      },
    },
  },
});
