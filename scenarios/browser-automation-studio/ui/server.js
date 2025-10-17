import express from 'express';
import path from 'path';
import http from 'http';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const DEFAULT_VERSION = '1.0.0';
const SERVICE_NAME = 'browser-automation-studio';

const buildHealthPayload = (apiPort) => ({
    status: 'healthy',
    service: 'browser-automation-studio-ui',
    timestamp: new Date().toISOString(),
    readiness: true,
    api_connectivity: {
        connected: false,
        api_url: apiPort ? `http://localhost:${apiPort}/api/v1` : null,
        last_check: new Date().toISOString(),
        error: null,
        latency_ms: null
    }
});

function createApp({ apiPort } = {}) {
    const resolvedApiPort = apiPort ?? process.env.API_PORT;
    const app = express();

    app.use(express.static(path.join(__dirname, 'dist')));

    app.get('/config', (_req, res) => {
        res.json({
            apiUrl: resolvedApiPort ? `http://localhost:${resolvedApiPort}/api/v1` : null,
            version: DEFAULT_VERSION,
            service: SERVICE_NAME
        });
    });

    app.get('/', (_req, res) => {
        res.sendFile(path.join(__dirname, 'dist', 'index.html'));
    });

    app.get('/health', async (_req, res) => {
        const healthResponse = buildHealthPayload(resolvedApiPort);

        if (!resolvedApiPort) {
            healthResponse.status = 'degraded';
            healthResponse.api_connectivity.error = {
                code: 'MISSING_CONFIG',
                message: 'API_PORT environment variable not configured',
                category: 'configuration',
                retryable: false
            };
            return res.json(healthResponse);
        }

        const startTime = Date.now();

        try {
            await new Promise((resolve, reject) => {
                const requestOptions = {
                    hostname: 'localhost',
                    port: resolvedApiPort,
                    path: '/health',
                    method: 'GET',
                    timeout: 5000,
                    headers: {
                        Accept: 'application/json'
                    }
                };

                const apiRequest = http.request(requestOptions, (apiResponse) => {
                    healthResponse.api_connectivity.latency_ms = Date.now() - startTime;

                    if (apiResponse.statusCode >= 200 && apiResponse.statusCode < 300) {
                        healthResponse.api_connectivity.connected = true;
                        healthResponse.api_connectivity.error = null;
                    } else {
                        healthResponse.api_connectivity.connected = false;
                        healthResponse.api_connectivity.error = {
                            code: `HTTP_${apiResponse.statusCode}`,
                            message: `API returned status ${apiResponse.statusCode}: ${apiResponse.statusMessage}`,
                            category: 'network',
                            retryable: true
                        };
                        healthResponse.status = 'degraded';
                    }

                    resolve();
                });

                apiRequest.on('error', (err) => {
                    healthResponse.api_connectivity.latency_ms = Date.now() - startTime;
                    healthResponse.api_connectivity.connected = false;
                    healthResponse.api_connectivity.error = {
                        code: 'CONNECTION_ERROR',
                        message: `Failed to connect to API: ${err.message}`,
                        category: 'network',
                        retryable: true
                    };
                    healthResponse.status = 'degraded';
                    resolve();
                });

                apiRequest.on('timeout', () => {
                    apiRequest.destroy();
                    healthResponse.api_connectivity.latency_ms = Date.now() - startTime;
                    healthResponse.api_connectivity.connected = false;
                    healthResponse.api_connectivity.error = {
                        code: 'TIMEOUT',
                        message: 'API health check timed out after 5 seconds',
                        category: 'network',
                        retryable: true
                    };
                    healthResponse.status = 'degraded';
                    resolve();
                });

                apiRequest.end();
            });
        } catch (error) {
            healthResponse.api_connectivity.connected = false;
            healthResponse.api_connectivity.error = {
                code: 'REQUEST_ERROR',
                message: `Unexpected error during API health check: ${error.message}`,
                category: 'internal',
                retryable: true
            };
            healthResponse.status = 'degraded';
        }

        res.json(healthResponse);
    });

    // Catch-all handler for SPA routing
    app.get(/.*/, (_req, res) => {
        res.sendFile(path.join(__dirname, 'dist', 'index.html'));
    });

    return app;
}

function startServer() {
    const port = process.env.UI_PORT || 3000;
    const app = createApp();
    
    const server = app.listen(port, '0.0.0.0', () => {
        console.log(`Browser Automation Studio UI server listening on port ${port}`);
        console.log(`Health endpoint: http://localhost:${port}/health`);
        console.log(`UI available at: http://localhost:${port}`);
    });

    process.on('SIGTERM', () => {
        console.log('Received SIGTERM, shutting down gracefully');
        server.close(() => {
            console.log('UI server closed');
            process.exit(0);
        });
    });

    process.on('SIGINT', () => {
        console.log('Received SIGINT, shutting down gracefully');
        server.close(() => {
            console.log('UI server closed');
            process.exit(0);
        });
    });
}

// Only start server if this script is run directly
if (import.meta.url === `file://${process.argv[1]}`) {
    startServer();
}

export { createApp, buildHealthPayload };