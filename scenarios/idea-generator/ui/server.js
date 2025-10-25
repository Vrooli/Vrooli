const express = require('express');
const path = require('path');
const http = require('http');
const https = require('https');
const cors = require('cors');
const axios = require('axios');

if (typeof window !== 'undefined' && window.parent !== window && !window.__IDEA_GENERATOR_BRIDGE_INITIALIZED) {
    try {
        const { initIframeBridgeChild } = require('@vrooli/iframe-bridge/child');

        let parentOrigin;
        if (document.referrer) {
            parentOrigin = new URL(document.referrer).origin;
        }

        initIframeBridgeChild({ parentOrigin, appId: 'idea-generator' });
        window.__IDEA_GENERATOR_BRIDGE_INITIALIZED = true;
    } catch (error) {
        console.warn('[idea-generator] iframe bridge bootstrap skipped', error);
    }
}

const app = express();
const PORT = process.env.UI_PORT || process.env.PORT || '35100';
const API_URL = process.env.API_URL || `http://localhost:${process.env.API_PORT || '15100'}`;

let parsedApiUrl;
try {
    parsedApiUrl = new URL(API_URL);
} catch (error) {
    console.error('[idea-generator] API_URL could not be parsed:', API_URL, error);
    parsedApiUrl = null;
}

function proxyToApi(req, res, upstreamPath) {
    if (!parsedApiUrl) {
        res.status(502).json({
            error: 'API proxy misconfigured',
            message: 'API_URL could not be parsed',
            target: API_URL
        });
        return;
    }

    const targetPath = upstreamPath || req.originalUrl || req.url || '/';
    const targetUrl = new URL(targetPath, parsedApiUrl);
    const client = targetUrl.protocol === 'https:' ? https : http;

    const headers = {
        ...req.headers,
        host: targetUrl.host
    };

    delete headers['content-length'];

    const options = {
        hostname: targetUrl.hostname,
        port: targetUrl.port || (targetUrl.protocol === 'https:' ? 443 : 80),
        path: `${targetUrl.pathname}${targetUrl.search}`,
        method: req.method,
        headers
    };

    console.log(`[idea-generator][proxy] ${req.method} ${req.originalUrl} -> ${targetUrl.href}`);

    const proxyReq = client.request(options, (proxyRes) => {
        if (!res.headersSent) {
            res.status(proxyRes.statusCode || 500);
            Object.entries(proxyRes.headers).forEach(([key, value]) => {
                if (typeof value !== 'undefined') {
                    res.setHeader(key, value);
                }
            });
        }

        proxyRes.pipe(res);
    });

    proxyReq.on('error', (error) => {
        console.error('[idea-generator][proxy] error:', error.message);
        if (!res.headersSent) {
            res.status(502).json({
                error: 'API server unavailable',
                message: error.message,
                target: targetUrl.href
            });
        } else {
            res.end();
        }
    });

    if (req.method === 'GET' || req.method === 'HEAD' || req.method === 'OPTIONS') {
        proxyReq.end();
        return;
    }

    if (req.readable && typeof req.body === 'undefined') {
        req.pipe(proxyReq);
        return;
    }

    if (typeof req.body !== 'undefined') {
        const payload = Buffer.isBuffer(req.body)
            ? req.body
            : typeof req.body === 'string'
                ? Buffer.from(req.body)
                : Buffer.from(JSON.stringify(req.body));

        proxyReq.write(payload);
    }

    proxyReq.end();
}

// Middleware
app.use(cors());
app.use(express.static(__dirname));

// Serve the main HTML file
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

// Health check endpoint with API connectivity check
app.get('/health', async (req, res) => {
    const now = new Date().toISOString();
    let apiConnected = false;
    let apiError = null;
    let apiLatency = null;

    try {
        const start = Date.now();
        const apiResponse = await axios.get(`${API_URL}/health`, { timeout: 5000 });
        apiLatency = Date.now() - start;
        apiConnected = apiResponse.status === 200;
    } catch (error) {
        apiError = {
            code: error.code || 'UNKNOWN_ERROR',
            message: error.message,
            category: 'network',
            retryable: true
        };
    }

    res.json({
        status: apiConnected ? 'healthy' : 'degraded',
        service: 'idea-generator-ui',
        timestamp: now,
        readiness: apiConnected,
        api_connectivity: {
            connected: apiConnected,
            api_url: API_URL,
            last_check: now,
            error: apiError,
            latency_ms: apiLatency
        }
    });
});

// Proxy API requests to the backend
app.use('/api', (req, res) => {
    const pathForProxy = req.originalUrl || `/api${req.url}`;
    proxyToApi(req, res, pathForProxy);
});

// Start server
app.listen(PORT, () => {
    console.log(`🚀 Idea Generator UI server running on http://localhost:${PORT}`);
    console.log(`📡 Proxying API requests to ${API_URL}`);
});

app.proxyToApi = proxyToApi;

module.exports = app;
module.exports.proxyToApi = proxyToApi;
