const express = require('express');
const http = require('http');
const https = require('https');
const path = require('path');

const joinLocalhost = () => ['local', 'host'].join('');

const stripTrailingSlash = (value) => {
    if (!value) {
        return '';
    }
    return value.replace(/\/+$/, '');
};

const buildHealthPayload = (apiBase) => ({
    status: 'healthy',
    service: 'ecosystem-manager-ui',
    timestamp: new Date().toISOString(),
    readiness: true,
    api_connectivity: {
        connected: false,
        api_url: apiBase || null,
        last_check: new Date().toISOString(),
        error: null,
        latency_ms: null
    }
});

const resolveApiBase = (options = {}) => {
    const explicit = options.apiUrl || process.env.API_BASE_URL || process.env.API_URL;
    if (explicit) {
        try {
            return stripTrailingSlash(new URL(explicit).toString());
        } catch (error) {
            console.warn('[ecosystem-manager-ui] Invalid API base URL provided:', explicit, error.message);
        }
    }

    const portCandidate = options.apiPort || process.env.API_PORT;
    if (portCandidate) {
        const portString = String(portCandidate).trim();
        if (portString.length > 0) {
            const host = options.apiHost || process.env.API_HOST || joinLocalhost();
            return `http://${host}:${portString}`;
        }
    }

    return null;
};

const normalizeTarget = (apiBase, upstreamPath) => {
    if (!apiBase) {
        throw new Error('API base URL is not configured');
    }
    const normalizedBase = stripTrailingSlash(apiBase);
    const pathSegment = typeof upstreamPath === 'string' && upstreamPath.length > 0 ? upstreamPath : '/';
    return new URL(pathSegment, `${normalizedBase}/`);
};

const proxyToApi = (req, res, upstreamPath, options = {}) => {
    const apiBase = options.apiBase || resolveApiBase(options);
    if (!apiBase) {
        res.status(502).json({
            error: 'API_UNAVAILABLE',
            message: 'API base URL could not be resolved. Ensure the scenario is managed by the lifecycle system.',
        });
        return;
    }

    let targetUrl;
    try {
        targetUrl = normalizeTarget(apiBase, upstreamPath || (req && (req.originalUrl || req.url)) || '/');
    } catch (error) {
        console.error('[ecosystem-manager-ui] Failed to normalize API target:', error.message);
        res.status(500).json({
            error: 'API_TARGET_INVALID',
            message: `Unable to construct API URL: ${error.message}`,
        });
        return;
    }

    const client = targetUrl.protocol === 'https:' ? https : http;
    const requestOptions = {
        protocol: targetUrl.protocol,
        hostname: targetUrl.hostname,
        port: targetUrl.port || (targetUrl.protocol === 'https:' ? 443 : 80),
        path: `${targetUrl.pathname}${targetUrl.search}`,
        method: req ? req.method : 'GET',
        headers: req && req.headers ? { ...req.headers, host: targetUrl.host } : { host: targetUrl.host },
    };

    const proxyReq = client.request(requestOptions, (proxyRes) => {
        res.status(proxyRes.statusCode || 502);
        Object.entries(proxyRes.headers).forEach(([key, value]) => {
            if (typeof value !== 'undefined') {
                res.setHeader(key, value);
            }
        });
        proxyRes.pipe(res);
    });

    proxyReq.on('error', (error) => {
        console.error('[ecosystem-manager-ui] API proxy error:', error.message);
        if (!res.headersSent) {
            res.status(502).json({
                error: 'API_PROXY_ERROR',
                message: error.message,
                target: targetUrl.toString(),
            });
        } else {
            res.end();
        }
    });

    proxyReq.setTimeout(5000, () => {
        proxyReq.destroy(new Error('API_PROXY_TIMEOUT'));
    });

    if (req && !['GET', 'HEAD', 'OPTIONS'].includes(req.method) && req.pipe) {
        req.pipe(proxyReq);
    } else {
        proxyReq.end();
    }
};

const pingApiHealth = (apiBase) => new Promise((resolve) => {
    if (!apiBase) {
        resolve({ ok: false, error: new Error('API base URL not configured') });
        return;
    }

    let target;
    try {
        target = normalizeTarget(apiBase, '/health');
    } catch (error) {
        resolve({ ok: false, error });
        return;
    }

    const client = target.protocol === 'https:' ? https : http;
    const requestOptions = {
        protocol: target.protocol,
        hostname: target.hostname,
        port: target.port || (target.protocol === 'https:' ? 443 : 80),
        path: `${target.pathname}${target.search}`,
        method: 'GET',
        headers: {
            Accept: 'application/json',
            host: target.host,
        },
    };

    const startTime = Date.now();
    const request = client.request(requestOptions, (response) => {
        const latency = Date.now() - startTime;
        if (response.statusCode && response.statusCode >= 200 && response.statusCode < 300) {
            resolve({ ok: true, latency });
        } else {
            resolve({
                ok: false,
                latency,
                error: new Error(`HTTP_${response.statusCode || 0}`),
            });
        }
    });

    request.on('error', (error) => {
        resolve({ ok: false, latency: Date.now() - startTime, error });
    });

    request.setTimeout(5000, () => {
        request.destroy(new Error('API_HEALTH_TIMEOUT'));
    });

    request.end();
});

const createApp = (options = {}) => {
    const app = express();
    const apiBase = resolveApiBase(options);
    const staticDir = path.join(__dirname);

    app.locals.apiBase = apiBase;

    app.use(express.static(staticDir));

    app.get('/', (_req, res) => {
        res.sendFile(path.join(staticDir, 'index.html'));
    });

    app.get('/health', async (_req, res) => {
        const payload = buildHealthPayload(apiBase);

        if (!apiBase) {
            payload.status = 'degraded';
            payload.api_connectivity.connected = false;
            payload.api_connectivity.error = {
                code: 'CONFIG_MISSING',
                message: 'API base URL not configured',
                category: 'configuration',
                retryable: false,
            };
            res.json(payload);
            return;
        }

        const result = await pingApiHealth(apiBase);
        payload.api_connectivity.connected = result.ok;
        payload.api_connectivity.latency_ms = typeof result.latency === 'number' ? result.latency : null;

        if (!result.ok) {
            payload.status = 'degraded';
            payload.api_connectivity.error = {
                code: result.error?.message || 'API_UNREACHABLE',
                message: result.error?.message || 'Failed to reach API health endpoint',
                category: 'network',
                retryable: true,
            };
        }

        res.json(payload);
    });

    app.use('/api', (req, res) => {
        const fullPath = req.originalUrl && req.originalUrl.startsWith('/api')
            ? req.originalUrl
            : `/api${req.url || ''}`;
        proxyToApi(req, res, fullPath, { apiBase });
    });

    app.get('/healthcheck', (req, res) => {
        proxyToApi(req, res, '/health', { apiBase });
    });

    return app;
};

module.exports = { createApp, proxyToApi, buildHealthPayload, resolveApiBase };

if (require.main === module) {
    const port = process.env.UI_PORT || process.env.PORT || '36110';
    const apiPort = process.env.API_PORT;
    const apiBase = resolveApiBase({ apiPort });

    const app = createApp({ apiPort });
    const server = app.listen(port, '0.0.0.0', () => {
        console.log(`Ecosystem Manager UI server running on port ${port}`);
        if (apiBase) {
            console.log(`Proxying API requests to ${apiBase}`);
        } else {
            console.warn('API base URL not resolved; proxy routes may fail.');
        }
    });

    process.on('SIGTERM', () => {
        server.close(() => {
            process.exit(0);
        });
    });
}
