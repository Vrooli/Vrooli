const express = require('express');
const path = require('path');
const http = require('http');
const fs = require('fs');

const app = express();
app.set('trust proxy', true);

const PORT = Number(process.env.UI_PORT || process.env.PORT || 3000);
const API_PORT = Number(process.env.API_PORT || 8080);
const hasApiPort = Number.isFinite(API_PORT) && API_PORT > 0;
const API_HOST = process.env.API_HOST || '127.0.0.1';
const UI_HOST = process.env.UI_HOST || '127.0.0.1';
const PROXY_ROOT = '/proxy';
const STATIC_ROOT = (() => {
    const distPath = path.join(__dirname, 'dist');
    return fs.existsSync(distPath) ? distPath : __dirname;
})();

function buildOrigin(req) {
    const forwardedProto = req.headers['x-forwarded-proto'];
    const protocol = Array.isArray(forwardedProto) ? forwardedProto[0] : forwardedProto || req.protocol || 'http';
    const forwardedHost = req.headers['x-forwarded-host'];
    const host = Array.isArray(forwardedHost) ? forwardedHost[0] : forwardedHost || req.get('host') || `${UI_HOST}:${PORT}`;
    return `${protocol}://${host}`;
}

function proxyToApi(req, res, upstreamPath) {
    if (!hasApiPort) {
        res.status(503).json({
            error: 'API_PORT not configured',
            details: 'The Date Night Planner API is not running in this environment.'
        });
        return;
    }

    const rawPath = typeof upstreamPath === 'string' && upstreamPath.length > 0
        ? upstreamPath
        : req.originalUrl.slice(PROXY_ROOT.length) || '/';
    const targetPath = rawPath.startsWith('/') ? rawPath : `/${rawPath}`;

    const options = {
        hostname: API_HOST,
        port: API_PORT,
        path: targetPath,
        method: req.method,
        headers: {
            ...req.headers,
            host: `${API_HOST}:${API_PORT}`
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
        console.error('[date-night-planner-ui] API proxy error:', error.message);
        res.status(502).json({
            error: 'API proxy failed',
            details: error.message,
            target: `http://${API_HOST}:${API_PORT}${targetPath}`
        });
    });

    if (req.method === 'GET' || req.method === 'HEAD') {
        proxyReq.end();
    } else {
        req.pipe(proxyReq);
    }
}

app.use(PROXY_ROOT, (req, res) => {
    proxyToApi(req, res);
});

// Serve static files
app.use(express.static(STATIC_ROOT));

// Health check endpoint (schema-compliant)
app.get('/health', async (req, res) => {
    const now = new Date().toISOString();
    const directApiRoot = hasApiPort ? `http://${API_HOST}:${API_PORT}` : '';
    const apiUrl = directApiRoot ? `${directApiRoot}/health` : '';

    const apiConnectivity = {
        connected: false,
        api_url: apiUrl,
        last_check: now,
        error: null,
        latency_ms: null
    };

    if (hasApiPort && apiUrl) {
        try {
            const startTime = Date.now();
            const response = await fetch(apiUrl);
            const latency = Date.now() - startTime;

            if (response.ok) {
                apiConnectivity.connected = true;
                apiConnectivity.latency_ms = latency;
            } else {
                apiConnectivity.error = {
                    code: `HTTP_${response.status}`,
                    message: `API returned status ${response.status}`,
                    category: 'network',
                    retryable: true
                };
            }
        } catch (error) {
            apiConnectivity.error = {
                code: 'CONNECTION_FAILED',
                message: error.message || 'Failed to connect to API',
                category: 'network',
                retryable: true
            };
        }
    } else {
        apiConnectivity.error = {
            code: 'API_PORT_UNSET',
            message: 'API_PORT environment variable not configured',
            category: 'configuration',
            retryable: true
        };
    }

    res.status(200).json({
        status: apiConnectivity.connected ? 'healthy' : 'degraded',
        service: 'date-night-planner-ui',
        timestamp: now,
        readiness: true,
        api_connectivity: apiConnectivity
    });
});

// API proxy configuration endpoint
app.get('/config', (req, res) => {
    const origin = buildOrigin(req);
    const directApiUrl = hasApiPort ? `http://${API_HOST}:${API_PORT}` : '';
    res.json({
        service: 'date-night-planner-ui',
        version: '1.0.0',
        apiUrl: directApiUrl,
        proxyApiUrl: `${origin}${PROXY_ROOT}`,
        proxyApiPath: PROXY_ROOT,
        displayApiUrl: directApiUrl || `${origin}${PROXY_ROOT}`,
        proxyFallbackUrl: `${origin}${PROXY_ROOT}`
    });
});

// Serve index.html for all routes (SPA)
app.get('*', (req, res) => {
    const fallback = path.join(__dirname, 'index.html');
    const target = path.join(STATIC_ROOT, 'index.html');
    res.sendFile(fs.existsSync(target) ? target : fallback);
});

// Start server
app.listen(PORT, () => {
    console.log(`💕 Date Night Planner UI running on port ${PORT}`);
    console.log(`🔗 API expected on port ${API_PORT}`);
    console.log(`🌐 Access UI at http://${UI_HOST}:${PORT}`);
    console.log(`🛡️  Proxying API requests through ${PROXY_ROOT}`);
});

module.exports = {
    app,
    proxyToApi,
    PROXY_ROOT
};
