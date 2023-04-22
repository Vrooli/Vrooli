import react from "@vitejs/plugin-react-swc"
import path from "path"
import { defineConfig } from "vite"

// https://vitejs.dev/config/
export default defineConfig({
    plugins: [react()],
    assetsInclude: ["**/*.md"],
    server: {
        host: true,
        port: 3000,
    },
    resolve: {
        alias: [
            // Imports from the shared folder
            { find: "@shared/consts", replacement: path.resolve(__dirname, "../shared/consts/src") },
            { find: "@shared/icons", replacement: path.resolve(__dirname, "../shared/icons/src") },
            { find: "@shared/translations", replacement: path.resolve(__dirname, "../shared/translations/src") },
            { find: "@shared/utils", replacement: path.resolve(__dirname, "../shared/utils/src") },
            { find: "@shared/uuid", replacement: path.resolve(__dirname, "../shared/uuid/src") },
            { find: "@shared/validation", replacement: path.resolve(__dirname, "../shared/validation/src") },
        ]
    },
    build: {
        // Enable source maps for debugging. Can be disabled in production, but it only saves a few seconds
        sourcemap: false,
    }
})
