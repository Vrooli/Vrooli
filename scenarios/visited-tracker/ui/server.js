const express = require('express');
const path = require('path');
const http = require('http');
const fs = require('fs');

const DEFAULT_VERSION = '1.0.0';
const SERVICE_NAME = 'visited-tracker';
const LOOPBACK_HOST = process.env.VROOLI_LOOPBACK_HOST || '127.0.0.1';
const LOOPBACK_ORIGIN = `http://${LOOPBACK_HOST}`;
const PROXY_ROOT = '/proxy';
const PROXY_API_PREFIX = `${PROXY_ROOT}/api`;
const PROXY_API_BASE = `${PROXY_API_PREFIX}/v1`;
const PROXY_HEALTH_URL = `${PROXY_ROOT}/health`;
const DOCS_CONTENT_ROUTE = /^\/(?:.*\/)?docs-content\/?$/;

const STATIC_ROOT = (() => {
    const distPath = path.join(__dirname, 'dist');
    return fs.existsSync(distPath) ? distPath : __dirname;
})();

const buildHealthPayload = ({ apiPort, displayUrl }) => ({
    status: 'healthy',
    service: 'visited-tracker-ui',
    timestamp: new Date().toISOString(),
    readiness: true,
    api_connectivity: {
        connected: false,
        api_url: displayUrl ?? (apiPort ? `${LOOPBACK_ORIGIN}:${apiPort}` : null),
        proxy_url: PROXY_ROOT,
        last_check: new Date().toISOString(),
        error: null,
        latency_ms: null
    }
});

const buildOrigin = (req, fallbackPort) => {
    const forwardedProto = req.headers['x-forwarded-proto'];
    const forwardedHost = req.headers['x-forwarded-host'];
    const protocol = Array.isArray(forwardedProto) ? forwardedProto[0] : forwardedProto || req.protocol || 'http';
    const host = Array.isArray(forwardedHost) ? forwardedHost[0] : forwardedHost || req.get('host');

    if (host) {
        return `${protocol}://${host}`;
    }

    const resolvedPort = fallbackPort ? Number(fallbackPort) : undefined;
    return resolvedPort ? `${protocol}://${LOOPBACK_HOST}:${resolvedPort}` : null;
};

function createApp({ apiPort } = {}) {
    const resolvedApiPort = apiPort ?? process.env.API_PORT;
    const numericApiPort = resolvedApiPort ? Number(resolvedApiPort) : undefined;
    const directApiUrl = numericApiPort ? `${LOOPBACK_ORIGIN}:${numericApiPort}` : null;
    const app = express();

    app.set('trust proxy', true);

    const proxyToApi = (req, res, upstreamPath) => {
        if (!numericApiPort) {
            res.status(503).json({
                error: 'API_PORT not configured',
                details: 'The visited-tracker API is not running or port is unknown.'
            });
            return;
        }

        const pathWithQuery = upstreamPath ?? req.originalUrl.replace(PROXY_ROOT, '');
        const finalPath = pathWithQuery.startsWith('/') ? pathWithQuery : `/${pathWithQuery}`;

        const options = {
            hostname: LOOPBACK_HOST,
            port: numericApiPort,
            path: finalPath,
            method: req.method,
            headers: {
                ...req.headers,
                host: `${LOOPBACK_HOST}:${numericApiPort}`
            }
        };

        const proxyReq = http.request(options, (proxyRes) => {
            res.status(proxyRes.statusCode || 500);
            Object.entries(proxyRes.headers).forEach(([key, value]) => {
                if (typeof value !== 'undefined') {
                    res.setHeader(key, value);
                }
            });
            proxyRes.pipe(res);
        });

        proxyReq.on('error', (error) => {
            console.error('[visited-tracker-ui] API proxy error:', error.message);
            res.status(502).json({
                error: 'API proxy failed',
                details: error.message,
                target: `${directApiUrl ?? 'api'}${finalPath}`
            });
        });

        if (req.method === 'GET' || req.method === 'HEAD') {
            proxyReq.end();
        } else {
            req.pipe(proxyReq);
        }
    };

    app.use(PROXY_ROOT, (req, res) => {
        const upstreamPath = req.originalUrl.replace(PROXY_ROOT, '');
        proxyToApi(req, res, upstreamPath);
    });

    const docsMarkdownPath = path.join(__dirname, '../docs/PROMPT_USAGE.md');

    const serveDocsContent = (req, res) => {
        fs.readFile(docsMarkdownPath, 'utf8', (err, data) => {
            if (err) {
                console.error('[visited-tracker-ui] Failed to read documentation:', err);
                res.status(404).type('text/plain').send('Documentation not found');
                return;
            }
            res.type('text/markdown');
            if (req.method === 'HEAD') {
                res.setHeader('Content-Length', Buffer.byteLength(data, 'utf8'));
                res.end();
                return;
            }
            res.send(data);
        });
    };

    app.use((req, res, next) => {
        if (!DOCS_CONTENT_ROUTE.test(req.path)) {
            return next();
        }

        if (req.method !== 'GET' && req.method !== 'HEAD') {
            res.set('Allow', 'GET, HEAD');
            res.status(405).end();
            return;
        }

        serveDocsContent(req, res);
    });

    app.use(express.static(STATIC_ROOT));

    app.get('/', (_req, res) => {
        res.sendFile(path.join(STATIC_ROOT, 'index.html'));
    });

    app.get('/config', (req, res) => {
        const origin = buildOrigin(req, process.env.UI_PORT ?? process.env.PORT);
        const displayApiUrl = directApiUrl || (origin ? `${origin}${PROXY_ROOT}` : null);

        res.json({
            apiUrl: directApiUrl,
            directApiBase: directApiUrl ? `${directApiUrl}/api/v1` : null,
            proxyApiUrl: PROXY_ROOT,
            proxyApiBase: PROXY_API_BASE,
            proxyHealthUrl: PROXY_HEALTH_URL,
            displayApiUrl,
            version: DEFAULT_VERSION,
            service: SERVICE_NAME
        });
    });

    app.get('/docs', (_req, res) => {
        res.sendFile(path.join(STATIC_ROOT, 'docs.html'));
    });

    app.get('/health', async (_req, res) => {
        const healthResponse = buildHealthPayload({
            apiPort: numericApiPort,
            displayUrl: directApiUrl
        });

        if (!numericApiPort) {
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
                    hostname: LOOPBACK_HOST,
                    port: numericApiPort,
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
        console.log(`🚀 Visited Tracker Dashboard running on ${LOOPBACK_ORIGIN}:${runtimePort}`);
        if (runtimeApiPort) {
            console.log(`📊 API: ${LOOPBACK_ORIGIN}:${runtimeApiPort}`);
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
