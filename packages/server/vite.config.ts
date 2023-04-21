import react from '@vitejs/plugin-react-swc'
import path from 'path'
import { defineConfig } from 'vite'

// https://vitejs.dev/config/
export default defineConfig({
    plugins: [react()],
    assetsInclude: ['**/*.md'],
    resolve: {
        alias: [
            // Set up absolute imports for each top-level folder and file in the src directory
            { find: 'api', replacement: path.resolve(__dirname, '../ui/src/api') },
            { find: 'assets', replacement: path.resolve(__dirname, '../ui/src/assets') },
            { find: 'components', replacement: path.resolve(__dirname, '../ui/src/components') },
            { find: 'forms', replacement: path.resolve(__dirname, '../ui/src/forms') },
            { find: 'tools', replacement: path.resolve(__dirname, '../ui/src/tools') },
            { find: 'utils', replacement: path.resolve(__dirname, '../ui/src/utils') },
            { find: 'views', replacement: path.resolve(__dirname, '../ui/src/views') },
            { find: 'Routes', replacement: path.resolve(__dirname, '../ui/src/Routes') },
            { find: 'serviceWorkerRegistration', replacement: path.resolve(__dirname, '../ui/src/serviceWorkerRegistration') },
            { find: 'styles', replacement: path.resolve(__dirname, '../ui/src/styles') },
            // Imports from the shared folder
            { find: '@shared/consts', replacement: path.resolve(__dirname, '../shared/consts/src') },
            { find: '@shared/icons', replacement: path.resolve(__dirname, '../shared/icons/src') },
            { find: '@shared/translations', replacement: path.resolve(__dirname, '../shared/translations/src') },
            { find: '@shared/utils', replacement: path.resolve(__dirname, '../shared/utils/src') },
            { find: '@shared/uuid', replacement: path.resolve(__dirname, '../shared/uuid/src') },
            { find: '@shared/validation', replacement: path.resolve(__dirname, '../shared/validation/src') },
        ]
    },
    build: {
        target: 'es2019',
        outDir: 'dist',
        sourcemap: false,
        ssr: 'src/renderer.tsx',
        ssrManifest: true,
        rollupOptions: {
            external: ['react', 'react-dom', 'react-router-dom'],
        },
    },
})
