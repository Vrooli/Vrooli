import path from 'path';
import { defineConfig } from 'vite';
import ssr from 'vite-plugin-ssr/plugin';

export default defineConfig({
    plugins: [ssr()],
    resolve: {
        alias: [
            { find: '@local/shared', replacement: path.resolve(__dirname, './shared/src') },
            { find: '@local/ui', replacement: path.resolve(__dirname, './ui/src') },
        ]
    },
    build: {
        target: 'esnext',
        outDir: 'dist',
        sourcemap: false,
        ssr: 'src/renderer.tsx',
        ssrManifest: true,
        rollupOptions: {
            external: ['react', 'react-dom', 'react-router-dom'],
        },
    },
});
