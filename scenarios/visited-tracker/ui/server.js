const express = require('express');
const path = require('path');
const http = require('http');

const DEFAULT_VERSION = '1.0.0';
const SERVICE_NAME = 'visited-tracker';

const buildHealthPayload = (apiPort) => ({
    status: 'healthy',
    service: 'visited-tracker-ui',
    timestamp: new Date().toISOString(),
    readiness: true,
    api_connectivity: {
        connected: false,
        api_url: apiPort ? `http://localhost:${apiPort}` : null,
        last_check: new Date().toISOString(),
        error: null,
        latency_ms: null
    }
});

function createApp({ apiPort } = {}) {
    const resolvedApiPort = apiPort ?? process.env.API_PORT;
    const app = express();

    app.use(express.static(__dirname));

    app.get('/config', (_req, res) => {
        res.json({
            apiUrl: resolvedApiPort ? `http://localhost:${resolvedApiPort}` : null,
            version: DEFAULT_VERSION,
            service: SERVICE_NAME
        });
    });

    app.get('/', (_req, res) => {
        res.sendFile(path.join(__dirname, 'index.html'));
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
            await new Promise((resolve) => {
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
                            retryable: apiResponse.statusCode >= 500 && apiResponse.statusCode < 600
                        };
                        healthResponse.status = 'degraded';
                    }
                    resolve();
                });

                apiRequest.on('error', (error) => {
                    healthResponse.api_connectivity.latency_ms = Date.now() - startTime;
                    healthResponse.api_connectivity.connected = false;

                    const errorMap = {
                        ECONNREFUSED: {
                            code: 'CONNECTION_REFUSED',
                            category: 'network',
                            retryable: true
                        },
                        ENOTFOUND: {
                            code: 'HOST_NOT_FOUND',
                            category: 'configuration',
                            retryable: false
                        },
                        ETIMEOUT: {
                            code: 'TIMEOUT',
                            category: 'network',
                            retryable: true
                        }
                    };

                    const meta = error.code && errorMap[error.code] ? errorMap[error.code] : {
                        code: 'CONNECTION_FAILED',
                        category: 'network',
                        retryable: true
                    };

                    healthResponse.api_connectivity.error = {
                        code: meta.code,
                        message: `Failed to connect to API: ${error.message}`,
                        category: meta.category,
                        retryable: meta.retryable,
                        details: {
                            error_code: error.code
                        }
                    };
                    healthResponse.status = meta.retryable ? 'unhealthy' : 'degraded';
                    resolve();
                });

                apiRequest.on('timeout', () => {
                    healthResponse.api_connectivity.latency_ms = Date.now() - startTime;
                    healthResponse.api_connectivity.connected = false;
                    healthResponse.api_connectivity.error = {
                        code: 'TIMEOUT',
                        message: 'API health check timed out after 5 seconds',
                        category: 'network',
                        retryable: true
                    };
                    healthResponse.status = 'unhealthy';
                    apiRequest.destroy();
                    resolve();
                });

                apiRequest.end();
            });
        } catch (error) {
            healthResponse.api_connectivity.latency_ms = Date.now() - startTime;
            healthResponse.api_connectivity.connected = false;
            healthResponse.api_connectivity.error = {
                code: 'UNEXPECTED_ERROR',
                message: `Unexpected error: ${error.message}`,
                category: 'internal',
                retryable: true
            };
            healthResponse.status = 'unhealthy';
        }

        res.json(healthResponse);
    });

    return app;
}

function startServer({ uiPort, apiPort } = {}) {
    const app = createApp({ apiPort });
    const resolvedPort = uiPort ?? process.env.UI_PORT ?? process.env.PORT ?? 0;
    const server = app.listen(resolvedPort, () => {
        const address = server.address();
        const runtimePort = typeof address === 'object' && address ? address.port : resolvedPort;
        const runtimeApiPort = apiPort ?? process.env.API_PORT;
        console.log(`🚀 Visited Tracker Dashboard running on http://localhost:${runtimePort}`);
        if (runtimeApiPort) {
            console.log(`📊 API: http://localhost:${runtimeApiPort}`);
        }
    });
    return { app, server };
}

if (require.main === module) {
    startServer();
}

module.exports = {
    createApp,
    startServer
};
