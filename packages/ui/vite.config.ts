import react from '@vitejs/plugin-react-swc'
import path from 'path'
import { defineConfig } from 'vite'

// https://vitejs.dev/config/
export default defineConfig({
    plugins: [react()],
    assetsInclude: ['**/*.md'],
    server: {
        host: true,
        port: 3000,
    },
    resolve: {
        alias: [
            // Set up absolute imports for each top-level folder and file in the src directory
            { find: 'api', replacement: path.resolve(__dirname, './src/api') },
            { find: 'assets', replacement: path.resolve(__dirname, './src/assets') },
            { find: 'components', replacement: path.resolve(__dirname, './src/components') },
            { find: 'forms', replacement: path.resolve(__dirname, './src/forms') },
            { find: 'tools', replacement: path.resolve(__dirname, './src/tools') },
            { find: 'utils', replacement: path.resolve(__dirname, './src/utils') },
            { find: 'views', replacement: path.resolve(__dirname, './src/views') },
            { find: 'Routes', replacement: path.resolve(__dirname, './src/Routes') },
            { find: 'serviceWorkerRegistration', replacement: path.resolve(__dirname, './src/serviceWorkerRegistration') },
            { find: 'styles', replacement: path.resolve(__dirname, './src/styles') },
            // Imports from the shared folder
            { find: '@shared/consts', replacement: path.resolve(__dirname, '../shared/consts/src') },
            { find: '@shared/icons', replacement: path.resolve(__dirname, '../shared/icons/src') },
            { find: '@shared/route', replacement: path.resolve(__dirname, '../shared/route/src') },
            { find: '@shared/translations', replacement: path.resolve(__dirname, '../shared/translations/src') },
            { find: '@shared/utils', replacement: path.resolve(__dirname, '../shared/utils/src') },
            { find: '@shared/uuid', replacement: path.resolve(__dirname, '../shared/uuid/src') },
            { find: '@shared/validation', replacement: path.resolve(__dirname, '../shared/validation/src') },
        ]
    },
    build: {
        // Enable source maps for debugging. Can be disabled in production, but it only saves a few seconds
        sourcemap: false,
    }
})
