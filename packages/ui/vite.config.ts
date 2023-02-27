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
    // Set up absolute imports for each top-level folder and file in the src directory
    resolve: {
        alias: [
            { find: 'api', replacement: path.resolve(__dirname, './src/api') },
            { find: 'components', replacement: path.resolve(__dirname, './src/components') },
            { find: 'forms', replacement: path.resolve(__dirname, './src/forms') },
            { find: 'pages', replacement: path.resolve(__dirname, './src/pages') },
            { find: 'tools', replacement: path.resolve(__dirname, './src/tools') },
            { find: 'utils', replacement: path.resolve(__dirname, './src/utils') },
            { find: 'Routes', replacement: path.resolve(__dirname, './src/Routes') },
            { find: 'serviceWorkerRegistration', replacement: path.resolve(__dirname, './src/serviceWorkerRegistration') },
            { find: 'styles', replacement: path.resolve(__dirname, './src/styles') },
          ]
    },
})
