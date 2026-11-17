import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'
import { sourceLibraryResolver } from "../../../packages/react-component-library/tooling/resolve-specifier.mjs";

export default defineConfig({
  base: './',
  plugins: [sourceLibraryResolver({ devOnly: true }), react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  build: {
    outDir: 'dist',
    sourcemap: true,
  },
})