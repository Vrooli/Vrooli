import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  base: './',  // Required for universal deployment (tunnel/proxy contexts)
  test: {
    environment: 'jsdom',
    setupFiles: './src/test-utils/setupTests.ts',
    coverage: {
      reporter: ['text', 'html'],
    },
    environmentOptions: {
      jsdom: {
        url: 'http://localhost:3000/',  // Base URL for jsdom test environment
      },
    },
  },
});
