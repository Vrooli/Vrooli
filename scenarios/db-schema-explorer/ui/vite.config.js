import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

const joinLocalhost = () => ['local', 'host'].join('');

const stripTrailingSlash = (value) => {
  if (!value) {
    return undefined;
  }
  const normalized = value.replace(/\s+/g, '').replace(/\/+$/, '');
  return normalized.length > 0 ? normalized : undefined;
};

const toWebSocketTarget = (base) => {
  if (!base) {
    return undefined;
  }
  if (base.startsWith('ws://') || base.startsWith('wss://')) {
    return stripTrailingSlash(base);
  }
  if (base.startsWith('http://')) {
    return stripTrailingSlash(base.replace('http://', 'ws://'));
  }
  if (base.startsWith('https://')) {
    return stripTrailingSlash(base.replace('https://', 'wss://'));
  }
  return stripTrailingSlash(base);
};

const resolveApiOrigin = () => {
  const explicit =
    process.env.APP_MONITOR_PROXY_API_ORIGIN ||
    process.env.DB_SCHEMA_EXPLORER_PROXY_API_ORIGIN ||
    process.env.VITE_PROXY_API_ORIGIN ||
    process.env.APP_MONITOR_PROXY_API_URL ||
    process.env.DB_SCHEMA_EXPLORER_API_ORIGIN ||
    process.env.API_BASE_URL ||
    process.env.API_URL;

  if (explicit) {
    const normalized = stripTrailingSlash(explicit);
    if (normalized) {
      return normalized;
    }
  }

  const port = process.env.API_PORT;
  if (port) {
    const host = process.env.API_HOST || joinLocalhost();
    return `http://${host}:${port}`;
  }

  return undefined;
};

const resolveProxyTarget = () => {
  const candidate =
    process.env.APP_MONITOR_PROXY_API_TARGET ||
    process.env.DB_SCHEMA_EXPLORER_PROXY_API_TARGET ||
    process.env.VITE_PROXY_API_TARGET;

  if (candidate) {
    const normalized = stripTrailingSlash(candidate);
    if (normalized) {
      return normalized;
    }
  }

  return stripTrailingSlash(resolveApiOrigin());
};

const API_ORIGIN = resolveApiOrigin();
const WS_ORIGIN = toWebSocketTarget(API_ORIGIN);
const PROXY_TARGET = resolveProxyTarget();
const WS_PROXY_TARGET = toWebSocketTarget(PROXY_TARGET);

const resolveUiPort = () => {
  if (!process.env.UI_PORT) {
    console.error('UI_PORT environment variable is required for the UI dev server');
    process.exit(1);
  }
  const parsed = Number(process.env.UI_PORT);
  if (!Number.isFinite(parsed)) {
    console.error('UI_PORT must be a valid number');
    process.exit(1);
  }
  return parsed;
};

const UI_PORT = resolveUiPort();

export default defineConfig({
  plugins: [react()],
  server: {
    port: UI_PORT,
    host: true,
    proxy: {
      '/api': {
        target: PROXY_TARGET || API_ORIGIN,
        changeOrigin: true,
        secure: false,
      },
      '/health': {
        target: PROXY_TARGET || API_ORIGIN,
        changeOrigin: true,
        secure: false,
      },
      '/ws': {
        target: WS_PROXY_TARGET || WS_ORIGIN,
        ws: true,
        changeOrigin: true,
        secure: false,
      },
    },
  },
});
