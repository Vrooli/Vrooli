import path from 'path';
import { defineConfig } from 'vite';
import ssr from 'vite-plugin-ssr/plugin';

export default defineConfig({
    plugins: [ssr()],
    resolve: {
        alias: [
            // Imports from the shared folder
            { find: ':local/consts', replacement: path.resolve(__dirname, './consts/src') },
            { find: ':local/icons', replacement: path.resolve(__dirname, './icons/src') },
            { find: ':local/translations', replacement: path.resolve(__dirname, './translations/src') },
            { find: ':local/ui', replacement: path.resolve(__dirname, './ui/src') },
            { find: ':local/utils', replacement: path.resolve(__dirname, './utils/src') },
            { find: ':local/uuid', replacement: path.resolve(__dirname, './uuid/src') },
            { find: ':local/validation', replacement: path.resolve(__dirname, './validation/src') },
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
