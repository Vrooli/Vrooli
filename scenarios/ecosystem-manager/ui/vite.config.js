import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { defineConfig } from 'vite';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const IFRAME_BRIDGE_CHILD_ENTRY = path.resolve(
    __dirname,
    'node_modules',
    '@vrooli',
    'iframe-bridge',
    'dist',
    'iframeBridgeChild.js'
);

const uiPort = Number.parseInt(process.env.UI_PORT || '', 10);

export default defineConfig({
    base: './',
    resolve: {
        alias: {
            '@vrooli/iframe-bridge/child': IFRAME_BRIDGE_CHILD_ENTRY,
        },
    },
    server: {
        host: true,
        port: uiPort,
    },
});
