import react from "@vitejs/plugin-react-swc";
import fs from "fs";
import path from "path";
import { defineConfig, loadEnv } from "vite";
import { iconsSpritesheet } from 'vite-plugin-icons-spritesheet';
function resolveImportExtensions() {
    return {
        name: 'resolve-import-extensions', // Name of the plugin
        resolveId: function (source, importer) {
            // Check if the import ends with '.js'
            if (source.endsWith('.js')) {
                // Replace '.js' with '.tsx' or '.ts' to test for a TypeScript file
                var tsxPath = source.replace(/\.js$/, '.tsx');
                var tsPath = source.replace(/\.js$/, '.ts');
                // Compute the absolute path based on the importer's directory
                var tsxAbsolutePath = path.resolve(path.dirname(importer), tsxPath);
                var tsAbsolutePath = path.resolve(path.dirname(importer), tsPath);
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
export default defineConfig(function (props) {
    // Load environment variables into `process.env`, since 
    // the `meta` of `import.meta.env` doesn't exist when running 
    // jest tests.
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
            // Remove console statements in production
            minify: 'terser',
            terserOptions: {
                compress: {
                    drop_console: true,
                    drop_debugger: true,
                },
            },
            rollupOptions: {
                // Exclude development files from production builds
                external: function (id) {
                    // Exclude test and story files from the build
                    if (id.includes('.test.') || id.includes('.stories.')) {
                        return true;
                    }
                    return false;
                },
                output: {
                    // Anything which doesn't need to be in the main bundle can be defined here as a separate chunk. 
                    // This should be done only if you've tried everything else to reduce the bundle size.
                    // Also, this doesn't guarantee that the chunk will be moved to its own bundle. But it's worth a try.
                    manualChunks: {
                        // Core React (required for everything)
                        'react-core': ['react', 'react-dom'],
                        // MUI core components (Box, Typography, etc)
                        'mui-core': ['@mui/material/Box', '@mui/material/Typography', '@mui/material/Stack', '@mui/material/Grid'],
                        // MUI extras (less common components)
                        'mui-extras': ['@mui/material/Accordion', '@mui/material/Autocomplete', '@mui/material/Stepper'],
                        // Form handling
                        'forms': ['formik', 'yup'],
                        // Internationalization
                        'i18n': ['i18next', 'react-i18next'],
                        // Codemirror bundles removed - now loaded dynamically
                        // Latex bundles
                        'latex': ['katex', '@matejmazur/react-katex'],
                    },
                },
            },
        }
    };
});
