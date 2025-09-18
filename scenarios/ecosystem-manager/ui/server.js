const express = require('express');
const http = require('http');
const path = require('path');

// Build health payload
const buildHealthPayload = (apiPort) => ({
    status: 'healthy',
    service: 'ecosystem-manager-ui',
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

// Create Express app
const createApp = (options = {}) => {
    const app = express();
    const apiPort = options.apiPort || process.env.API_PORT;

    // Static files
    app.use(express.static(path.join(__dirname)));

    // Main page
    app.get('/', (_req, res) => {
        res.sendFile(path.join(__dirname, 'index.html'));
    });

    // Health endpoint with API connectivity check
    app.get('/health', async (_req, res) => {
        const healthResponse = buildHealthPayload(apiPort);

        if (!apiPort) {
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
                    port: apiPort,
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
                        ETIMEDOUT: {
                            code: 'TIMEOUT',
                            category: 'network',
                            retryable: true
                        },
                        ENOTFOUND: {
                            code: 'HOST_NOT_FOUND',
                            category: 'configuration',
                            retryable: false
                        }
                    };

                    const meta = errorMap[error.code] || {
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
                            errno: error.code,
                            syscall: error.syscall
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
};

// Export for testing
module.exports = { createApp };

// Run server if executed directly
if (require.main === module) {
    const port = process.env.UI_PORT || 36110;
    const apiPort = process.env.API_PORT;
    
    const app = createApp({ apiPort });
    const server = app.listen(port, '0.0.0.0', () => {
        console.log(`Ecosystem Manager UI server running on port ${port}`);
        console.log(`API configured at port ${apiPort}`);
    });

    // Graceful shutdown
    process.on('SIGTERM', () => {
        console.log('SIGTERM signal received: closing HTTP server');
        server.close(() => {
            console.log('HTTP server closed');
            process.exit(0);
        });
    });
}