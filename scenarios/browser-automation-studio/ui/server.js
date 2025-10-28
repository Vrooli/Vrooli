import express from 'express';
import path from 'path';
import http from 'http';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const DEFAULT_VERSION = '1.0.0';
const SERVICE_NAME = 'browser-automation-studio';

const parsePort = (value, label) => {
    if (value === undefined || value === null || value === '') {
        return null;
    }

    const parsed = Number.parseInt(value, 10);
    if (Number.isNaN(parsed) || parsed <= 0) {
        console.warn(`${label ?? 'port'} must be a positive integer (received: ${value})`);
        return null;
    }

    return parsed;
};

let cachedApiPort = null;

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
        latency_ms: null,
        upstream: null
    }
});

function proxyToApi(req, res, upstreamPath, options = {}) {
    const { collectResponse = false, body, methodOverride, timeout = 5000, port } = options;
    const targetPort = port ?? cachedApiPort ?? parsePort(process.env.API_PORT, 'API_PORT');

    if (!targetPort) {
        if (collectResponse) {
            return Promise.reject(new Error('API_PORT not configured'));
        }

        res.status(502).json({
            error: 'API server unavailable',
            details: 'API_PORT not configured for Browser Automation Studio UI',
            target: null
        });
        return Promise.resolve();
    }

    const method = methodOverride ?? (req ? req.method : 'GET');
    const requestPath = upstreamPath ?? (req ? req.url : '/');
    const headers = req
        ? { ...req.headers, host: `localhost:${targetPort}` }
        : { host: `localhost:${targetPort}` };

    if (!headers.accept) {
        headers.accept = 'application/json';
    }

    delete headers.connection;

    return new Promise((resolve, reject) => {
        const proxyReq = http.request(
            {
                hostname: 'localhost',
                port: targetPort,
                path: requestPath,
                method,
                headers,
                timeout
            },
            (proxyRes) => {
                if (collectResponse) {
                    const chunks = [];
                    proxyRes.on('data', (chunk) => chunks.push(chunk));
                    proxyRes.on('end', () => {
                        const buffer = Buffer.concat(chunks);
                        resolve({
                            statusCode: proxyRes.statusCode ?? 0,
                            headers: proxyRes.headers,
                            body: buffer.toString('utf8')
                        });
                    });
                    return;
                }

                res.status(proxyRes.statusCode ?? 500);
                Object.entries(proxyRes.headers).forEach(([key, value]) => {
                    if (value !== undefined) {
                        res.setHeader(key, value);
                    }
                });
                proxyRes.pipe(res);
                resolve();
            }
        );

        proxyReq.on('error', (err) => {
            if (collectResponse) {
                reject(err);
                return;
            }

            console.error('API proxy error:', err.message);
            res.status(502).json({
                error: 'API server unavailable',
                details: err.message,
                target: `http://localhost:${targetPort}${requestPath}`
            });
            resolve();
        });

        proxyReq.on('timeout', () => {
            proxyReq.destroy(new Error('Request to API timed out'));
        });

        if (collectResponse) {
            if (body) {
                proxyReq.write(body);
            }
            proxyReq.end();
            return;
        }

        if (req && req.method !== 'GET' && req.method !== 'HEAD') {
            req.pipe(proxyReq);
        } else {
            proxyReq.end();
        }
    });
}

function createApp({ apiPort } = {}) {
    const resolvedApiPort = parsePort(apiPort ?? process.env.API_PORT, 'API_PORT');
    cachedApiPort = resolvedApiPort;

    const app = express();

    // Allow the UI to load inside sandboxed iframes where the origin becomes "null".
    app.use((req, res, next) => {
        res.setHeader('Access-Control-Allow-Origin', '*');
        res.setHeader('Access-Control-Allow-Methods', 'GET,POST,PUT,PATCH,DELETE,OPTIONS');
        res.setHeader('Access-Control-Allow-Headers', req.headers['access-control-request-headers'] || 'Content-Type, Authorization, X-Requested-With');
        if (req.method === 'OPTIONS') {
            res.status(204).end();
            return;
        }
        next();
    });

    app.use('/api', (req, res) => {
        const fullApiPath = req.url.startsWith('/api') ? req.url : `/api${req.url}`;
        proxyToApi(req, res, fullApiPath);
    });

    app.use(express.static(path.join(__dirname, 'dist')));

    app.get('/config', (_req, res) => {
        res.json({
            apiUrl: resolvedApiPort ? `http://localhost:${resolvedApiPort}/api/v1` : null,
            version: DEFAULT_VERSION,
            service: SERVICE_NAME
        });
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
            const apiResult = await proxyToApi(null, null, '/health', { collectResponse: true });
            healthResponse.api_connectivity.latency_ms = Date.now() - startTime;
            healthResponse.api_connectivity.last_check = new Date().toISOString();
            healthResponse.api_connectivity.connected = apiResult.statusCode >= 200 && apiResult.statusCode < 300;

            if (!healthResponse.api_connectivity.connected) {
                healthResponse.status = 'degraded';
                healthResponse.api_connectivity.error = {
                    code: `HTTP_${apiResult.statusCode ?? 'UNKNOWN'}`,
                    message: `API returned status ${apiResult.statusCode ?? 'unknown'}`,
                    category: 'network',
                    retryable: true
                };
            } else if (apiResult.body) {
                try {
                    healthResponse.api_connectivity.upstream = JSON.parse(apiResult.body);
                } catch {
                    healthResponse.api_connectivity.upstream = apiResult.body;
                }
            }
        } catch (error) {
            healthResponse.api_connectivity.latency_ms = Date.now() - startTime;
            healthResponse.api_connectivity.connected = false;
            healthResponse.api_connectivity.error = {
                code: 'CONNECTION_ERROR',
                message: `Failed to connect to API: ${error.message}`,
                category: 'network',
                retryable: true
            };
            healthResponse.status = 'degraded';
        }

        res.json(healthResponse);
    });

    app.get('/', (_req, res) => {
        res.sendFile(path.join(__dirname, 'dist', 'index.html'));
    });

    app.get(/.*/, (_req, res) => {
        res.sendFile(path.join(__dirname, 'dist', 'index.html'));
    });

    return app;
}

function startServer() {
    const port = process.env.UI_PORT || process.env.PORT || 3000;
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

if (import.meta.url === `file://${process.argv[1]}`) {
    startServer();
}

export { createApp, buildHealthPayload, proxyToApi };
