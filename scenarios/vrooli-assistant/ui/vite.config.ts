import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import { resolve } from 'node:path';
import { sourceLibraryResolver } from "../../../packages/react-component-library/tooling/resolve-specifier.mjs";

export default defineConfig(() => ({
  plugins: [sourceLibraryResolver({ devOnly: true }), react()],
  build: {
    outDir: resolve(__dirname, '../api/webui'),
    emptyOutDir: true,
    sourcemap: false
  },
  server: {
    host: '0.0.0.0',
    port: 5173
  }
}));
