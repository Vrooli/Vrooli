import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react-swc'
import path from 'path'

// https://vitejs.dev/config/
export default defineConfig({
    plugins: [react()],
    server: {
        host: true,
        port: 3000,
    },
    resolve: {
        alias: [
            // Set up absolute imports for each top-level folder and file in the src directory
            { find: 'api', replacement: path.resolve(__dirname, './src/api') },
            { find: 'components', replacement: path.resolve(__dirname, './src/components') },
            { find: 'forms', replacement: path.resolve(__dirname, './src/forms') },
            { find: 'pages', replacement: path.resolve(__dirname, './src/pages') },
            { find: 'tools', replacement: path.resolve(__dirname, './src/tools') },
            { find: 'utils', replacement: path.resolve(__dirname, './src/utils') },
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
    // Make sure we can use the "crypto" module in the browser
    optimizeDeps: {
        include: ['crypto'],
    },
})
