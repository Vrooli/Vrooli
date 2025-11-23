import { startScenarioServer } from '@vrooli/api-base/server';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const uiPort = process.env.UI_PORT;
const apiPort = process.env.API_PORT;

if (!apiPort) {
    console.error('❌ API_PORT environment variable is required for Ecosystem Manager UI server');
    process.exit(1);
}

const distDir = path.resolve(__dirname, 'dist');

startScenarioServer({
    uiPort,
    apiPort,
    distDir,
    serviceName: 'ecosystem-manager-ui',
    version: '1.0.0',
    corsOrigins: '*',
    verbose: process.env.NODE_ENV !== 'production',
    wsPathPrefix: '/ws',
    wsPathTransform: (incomingPath) => incomingPath || '/ws',
    configBuilder: (env) => ({
        apiUrl: `http://127.0.0.1:${env.API_PORT}/api`,  // No /v1 suffix
        wsUrl: `ws://127.0.0.1:${env.API_PORT}/ws`,
        apiPort: env.API_PORT,
        wsPort: env.API_PORT,
        uiPort: env.UI_PORT,
        version: '1.0.0',
        service: 'ecosystem-manager-ui',
    }),
});
