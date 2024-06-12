import react from "@vitejs/plugin-react-swc";
import path from "path";
import { defineConfig, loadEnv } from "vite";

type ViteEnv = Record<string, string> & {
    DEV: boolean;
    PROD: boolean;
};

// https://vitejs.dev/config/
export default defineConfig((props) => {
    // Load environment variables into `process.env`, since 
    // the `meta` of `import.meta.env` doesn't exist when running 
    // jest tests.
    // NOTE: This still doesn't give us access to the environment 
    // variables during testing, but it at least allows us to avoid 
    // having to type `import?.meta?.env?.VITE_...` everywhere.
    let env: ViteEnv = loadEnv(props.mode, process.cwd(), "") as ViteEnv;
    env.DEV = env.NODE_ENV !== "production";
    env.PROD = env.NODE_ENV === "production";
    const envInProcess = {
        "process.env": `${JSON.stringify(env)}`,
    };

    return {
        plugins: [react()],
        assetsInclude: ["**/*.md"],
        define: envInProcess,
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
            chunkSizeWarningLimit: 1000,
            // Enable source maps for debugging. Can be disabled in production, but it only saves a few seconds
            sourcemap: true,
            rollupOptions: {
                output: {
                    // Anything which doesn't need to be in the main bundle can be defined here as a separate chunk. 
                    // This should be done only if you've tried everything else to reduce the bundle size.
                    // Also, this doesn't guarantee that the chunk will be moved to its own bundle. But it's worth a try.
                    manualChunks: {
                        // Bundle for ad banners (if an ad-blocker decides to block this, 
                        // it won't affect the rest of the site).
                        // To help prevent blocking, it's named something random.
                        'banner-chicken': ['./src/components/BannerChicken/BannerChicken.tsx'],
                        // Codemirror bundles
                        'lang-angular': ['@codemirror/lang-angular'],
                        'lang-cpp': ['@codemirror/lang-cpp'],
                        'lang-css': ['@codemirror/lang-css'],
                        'lang-html': ['@codemirror/lang-html'],
                        'lang-java': ['@codemirror/lang-java'],
                        'lang-javascript': ['@codemirror/lang-javascript'],
                        'lang-json': ['@codemirror/lang-json'],
                        'lang-php': ['@codemirror/lang-php'],
                        'lang-python': ['@codemirror/lang-python'],
                        'lang-rust': ['@codemirror/lang-rust'],
                        'lang-sass': ['@codemirror/lang-sass'],
                        'lang-sql': ['@codemirror/legacy-modes/mode/sql'],
                        'lang-svelte': ['@replit/codemirror-lang-svelte'],
                        'lang-vue': ['@codemirror/lang-vue'],
                        'lang-xml': ['@codemirror/lang-xml'],
                        'codemirror-autocomplete': ['@codemirror/autocomplete'],
                        'codemirror-commands': ['@codemirror/commands'],
                        'codemirror-language': ['@codemirror/language'],
                        'codemirror-lint': ['@codemirror/lint'],
                        'codemirror-search': ['@codemirror/search'],
                        'codemirror-state': ['@codemirror/state'],
                        'codemirror-theme-one-dark': ['@codemirror/theme-one-dark'],
                        'codemirror-view': ['@codemirror/view'],
                    },
                },
            },
        }
    }
});
