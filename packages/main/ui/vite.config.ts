import react from '@vitejs/plugin-react-swc';
import path from 'path';
import { defineConfig } from 'vite';

export default defineConfig({
    plugins: [react()],
    assetsInclude: ['**/*.md'],
    resolve: {
        alias: [
            { find: '@local/shared', replacement: path.resolve(__dirname, '../shared/src') },
        ]
    },
    build: {
        target: 'esnext',
        outDir: 'dist',
        sourcemap: false,
    },
});