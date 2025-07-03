import react from "@vitejs/plugin-react-swc";
import path from "path";
import { defineConfig, loadEnv } from "vite";
// https://vitejs.dev/config/
export default defineConfig(function (props) {
    // Load environment variables into `process.env`, since 
    // the `meta` of `import.meta.env` doesn't exist when running
    // tests.
    //
    // NOTE: This still doesn't give us access to the environment 
    // variables during testing, but it at least allows us to avoid 
    // having to type `import?.meta?.env?.VITE_...` everywhere.
    var env = loadEnv(props.mode, process.cwd(), "");
    env.DEV = env.NODE_ENV !== "production";
    env.PROD = env.NODE_ENV === "production";
    var envInProcess = {
        "process.env": "".concat(JSON.stringify(env)),
    };
    return {
        plugins: [react()],
        assetsInclude: ["**/*.md"],
        define: envInProcess,
        server: {
            host: true,
            port: 3000,
        },
        build: {
            // Enable source maps for debugging. Can be disabled in production, but it only saves a few seconds
            sourcemap: true,
            rollupOptions: {
                input: {
                    index: path.resolve(__dirname, "index.html"), // Entry for popup/options page
                    background: path.resolve(__dirname, "src/background.ts"), // Entry for background script
                },
                output: {
                    entryFileNames: "[name].js", // Output as index.js and background.js
                    chunkFileNames: "chunks/[name]-[hash].js", // For any additional chunks
                    assetFileNames: "assets/[name]-[hash][extname]", // For assets like PNGs
                },
            },
        },
    };
});
