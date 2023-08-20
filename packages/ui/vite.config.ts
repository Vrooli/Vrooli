import react from "@vitejs/plugin-react-swc";
import path from "path";
import { defineConfig } from "vite";

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
            // Set up absolute imports for each top-level folder and file in the src directory
            { find: "api", replacement: path.resolve(__dirname, "./src/api") },
            { find: "assets", replacement: path.resolve(__dirname, "./src/assets") },
            { find: "components", replacement: path.resolve(__dirname, "./src/components") },
            { find: "contexts", replacement: path.resolve(__dirname, "./src/contexts") },
            { find: "forms", replacement: path.resolve(__dirname, "./src/forms") },
            { find: "hooks", replacement: path.resolve(__dirname, "./src/hooks") },
            { find: "icons", replacement: path.resolve(__dirname, "./src/icons") },
            { find: "route", replacement: path.resolve(__dirname, "./src/route") },
            { find: "tools", replacement: path.resolve(__dirname, "./src/tools") },
            { find: "utils", replacement: path.resolve(__dirname, "./src/utils") },
            { find: "views", replacement: path.resolve(__dirname, "./src/views") },
            { find: "Routes", replacement: path.resolve(__dirname, "./src/Routes") },
            { find: "serviceWorkerRegistration", replacement: path.resolve(__dirname, "./src/serviceWorkerRegistration") },
            { find: "styles", replacement: path.resolve(__dirname, "./src/styles") },
            // Imports from the shared folder
            { find: "@local/shared", replacement: path.resolve(__dirname, "../shared/src") },
        ]
    },
    build: {
        // Enable source maps for debugging. Can be disabled in production, but it only saves a few seconds
        sourcemap: false,
    }
})
