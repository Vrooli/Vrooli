import { defineConfig, loadEnv } from 'vite';
import react from '@vitejs/plugin-react';

function normalizeBasePath(value?: string) {
  if (!value) {
    return '';
  }

  const trimmed = value.trim();
  if (!trimmed || trimmed === '/') {
    return '';
  }

  return trimmed.endsWith('/') ? trimmed.slice(0, -1) : trimmed;
}

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '');
  const proxyBase = normalizeBasePath(env.VROOLI_PROXY_BASE_PATH);
  const base = mode === 'production' ? './' : proxyBase ? `${proxyBase}/` : '/';
  const rawPort = env.UI_PORT || env.SAAS_LANDING_UI_PORT || env.PORT;
  const parsedPort = rawPort ? Number.parseInt(rawPort, 10) : undefined;

  return {
    plugins: [react()],
    base,
    server: {
      host: env.VITE_SERVER_HOST ?? true,
      port: Number.isFinite(parsedPort) ? parsedPort : undefined,
    },
    build: {
      outDir: 'dist',
      emptyOutDir: true,
    },
  };
});
