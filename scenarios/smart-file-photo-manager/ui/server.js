const express = require('express');
const path = require('path');
const http = require('http');
const https = require('https');
const { URL } = require('url');

const app = express();
app.set('trust proxy', true);

const PORT = Number(process.env.UI_PORT || process.env.PORT || 3000);
const RAW_API_URL = process.env.API_URL;
const API_PORT = process.env.API_PORT ? Number(process.env.API_PORT) : undefined;
const API_HOST = process.env.API_HOST || '127.0.0.1';
const API_PROTOCOL = (process.env.API_PROTOCOL || 'http').replace(/:+$/, '');
const PROXY_TIMEOUT_MS = Number(process.env.PROXY_TIMEOUT_MS || 20000);

function normalizeUpstreamPath(value) {
    if (!value) {
        return '/';
    }

    if (typeof value !== 'string') {
        return '/';
    }

    if (value.startsWith('http://') || value.startsWith('https://')) {
        const parsed = new URL(value);
        return `${parsed.pathname}${parsed.search}` || '/';
    }

    return value.startsWith('/') ? value : `/${value}`;
}

function buildTargetUrl(upstreamPath) {
    const normalizedPath = normalizeUpstreamPath(upstreamPath);

    if (RAW_API_URL) {
        try {
            const base = new URL(RAW_API_URL);
            const target = new URL(normalizedPath, base);
            return target.toString();
        } catch (error) {
            console.error('[SmartFile UI] Failed to resolve API_URL target:', error.message);
        }
    }

    if (!API_PORT || Number.isNaN(API_PORT)) {
        throw new Error('API_PORT is not configured and API_URL is not set');
    }

    const protocol = `${API_PROTOCOL || 'http'}`.replace(/:+$/, '');
    const base = `${protocol}://${API_HOST}:${API_PORT}`;
    return `${base}${normalizedPath}`;
}

function proxyToApi(req, res, upstreamPath) {
    let targetUrl;
    try {
        targetUrl = buildTargetUrl(upstreamPath || req.originalUrl || req.url);
    } catch (error) {
        console.error('[SmartFile UI] API proxy misconfiguration:', error.message);
        res.status(503).json({
            error: 'API proxy unavailable',
            details: error.message
        });
        return;
    }

    let parsed;
    try {
        parsed = new URL(targetUrl);
    } catch (error) {
        console.error('[SmartFile UI] Invalid proxy target:', error.message, targetUrl);
        res.status(502).json({
            error: 'Invalid API proxy target',
            target: targetUrl,
            details: error.message
        });
        return;
    }

    const client = parsed.protocol === 'https:' ? https : http;
    const requestOptions = {
        protocol: parsed.protocol,
        hostname: parsed.hostname,
        port: parsed.port || (parsed.protocol === 'https:' ? 443 : 80),
        path: `${parsed.pathname}${parsed.search}`,
        method: req.method,
        headers: {
            ...req.headers,
            host: parsed.host
        },
        timeout: PROXY_TIMEOUT_MS
    };

    const proxyReq = client.request(requestOptions, (proxyRes) => {
        if (!res.headersSent) {
            res.status(proxyRes.statusCode || 502);
            Object.entries(proxyRes.headers || {}).forEach(([key, value]) => {
                if (typeof value !== 'undefined') {
                    res.setHeader(key, value);
                }
            });
        }

        proxyRes.on('error', (error) => {
            console.error('[SmartFile UI] Proxy response stream error:', error.message);
            if (!res.headersSent) {
                res.status(502).json({
                    error: 'API proxy stream failed',
                    target: targetUrl,
                    details: error.message
                });
            } else {
                req.destroy(error);
            }
        });

        proxyRes.pipe(res);
    });

    proxyReq.on('timeout', () => {
        proxyReq.destroy(new Error('Proxy request timed out'));
    });

    proxyReq.on('error', (error) => {
        console.error('[SmartFile UI] Proxy request error:', error.message);
        if (!res.headersSent) {
            res.status(502).json({
                error: 'API proxy request failed',
                target: targetUrl,
                details: error.message
            });
        } else {
            res.end();
        }
    });

    if (req.method === 'GET' || req.method === 'HEAD') {
        proxyReq.end();
    } else {
        req.pipe(proxyReq, { end: true });
    }
}

// Serve static files
app.use(express.static(path.join(__dirname), {
    setHeaders: (res, filePath) => {
        if (filePath.endsWith('.js')) {
            res.setHeader('Cache-Control', 'no-cache');
        }
    }
}));

// Health check endpoint for orchestrator
app.get('/health', (req, res) => {
    res.json({
        status: 'healthy',
        scenario: 'smart-file-photo-manager',
        port: PORT,
        timestamp: new Date().toISOString()
    });
});

// Proxy API requests to the backend
app.use('/api', (req, res) => {
    proxyToApi(req, res, req.originalUrl || req.url);
});

// Serve index.html for all routes (SPA support)
app.get('*', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

app.listen(PORT, () => {
    const displayPort = PORT || 'unknown';
    console.log(`
╔════════════════════════════════════════════════════╗
║                                                    ║
║     SmartFile UI Server                             ║
║     AI-Powered File & Photo Manager                 ║
║                                                    ║
║     🌐 UI:  http://localhost:${displayPort}             ║
║     🔧 API: ${RAW_API_URL || `${API_PROTOCOL}://${API_HOST}:${API_PORT || 'n/a'}`}          ║
║                                                    ║
║     Features:                                      ║
║     • Semantic file search                         ║
║     • AI-powered organization                      ║
║     • Duplicate detection                          ║
║     • Smart tagging                                ║
║     • Visual file management                       ║
║                                                    ║
╚════════════════════════════════════════════════════╝
    `);
});

module.exports = {
    app,
    proxyToApi
};
