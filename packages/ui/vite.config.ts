import react from "@vitejs/plugin-react-swc";
import fs from "fs";
import path from "path";
import { defineConfig, loadEnv } from "vite";
import { iconsSpritesheet } from 'vite-plugin-icons-spritesheet';

type ViteEnv = Record<string, string> & {
    DEV: boolean;
    PROD: boolean;
};

function resolveImportExtensions() {
    return {
        name: 'resolve-import-extensions', // Name of the plugin
        resolveId(source, importer) {
            // Check if the import ends with '.js'
            if (source.endsWith('.js')) {
                // Replace '.js' with '.tsx' or '.ts' to test for a TypeScript file
                const tsxPath = source.replace(/\.js$/, '.tsx');
                const tsPath = source.replace(/\.js$/, '.ts');
                // Compute the absolute path based on the importer's directory
                const tsxAbsolutePath = path.resolve(path.dirname(importer), tsxPath);
                const tsAbsolutePath = path.resolve(path.dirname(importer), tsPath);
                // If the .tsx file exists, resolve to it
                if (fs.existsSync(tsxAbsolutePath)) {
                    return tsxAbsolutePath;
                }
                if (fs.existsSync(tsAbsolutePath)) {
                    return tsAbsolutePath;
                }
            }
            // Otherwise, let Vite handle the resolution normally
            return null;
        },
    };
}

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
        plugins: [
            react(),
            resolveImportExtensions(),
            iconsSpritesheet([
                {
                    withTypes: true,
                    inputDir: "src/assets/icons/common",
                    outputDir: "public/sprites",
                    typesOutputFile: "src/icons/types/commonIcons.ts",
                    fileName: "common-sprite.svg",
                },
                {
                    withTypes: true,
                    inputDir: "src/assets/icons/routine",
                    outputDir: "public/sprites",
                    typesOutputFile: "src/icons/types/routineIcons.ts",
                    fileName: "routine-sprite.svg",
                },
                {
                    withTypes: true,
                    inputDir: "src/assets/icons/service",
                    outputDir: "public/sprites",
                    typesOutputFile: "src/icons/types/serviceIcons.ts",
                    fileName: "service-sprite.svg",
                },
                {
                    withTypes: true,
                    inputDir: "src/assets/icons/text",
                    outputDir: "public/sprites",
                    typesOutputFile: "src/icons/types/textIcons.ts",
                    fileName: "text-sprite.svg",
                }
            ]),
        ],
        assetsInclude: ["**/*.md"],
        define: envInProcess,
        server: {
            host: true,
            port: 3000,
        },
        resolve: {
            alias: [
                // Imports from the shared folder
                { find: "@vrooli/shared", replacement: path.resolve(__dirname, "../shared/src") },
            ],
            extensions: [".js", ".jsx", ".ts", ".tsx", ".json", ".d.ts"],
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
                        // Packages used in virtually every file
                        'vendor': ['react', 'react-dom', '@mui/material', 'formik', 'i18next', 'yup'],
                        // Bundle for ad banners (if an ad-blocker decides to block this, 
                        // it won't affect the rest of the site).
                        // To help prevent blocking, it's named something random.
                        'banner-chicken': ['./src/components/BannerChicken.tsx'],
                        // Separate landing page to reduce main bundle size
                        'landing': ['./src/views/main/LandingView.tsx'],
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
                        // Latex bundles
                        'latex': ['katex', '@matejmazur/react-katex'],
                    },
                },
            },
        }
    }
});
