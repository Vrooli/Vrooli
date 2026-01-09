const express = require('express');
const path = require('path');
const http = require('http');
const https = require('https');

const app = express();
const PORT = process.env.UI_PORT;

if (!PORT) {
    console.error('❌ UI_PORT environment variable is required');
    process.exit(1);
}

const API_BASE_URL = process.env.API_BASE_URL || `http://localhost:${process.env.API_PORT || 23250}`;

let parsedApiBaseUrl;
try {
    parsedApiBaseUrl = new URL(API_BASE_URL);
} catch (error) {
    console.error('[Document Manager UI] Invalid API_BASE_URL provided:', API_BASE_URL, error);
    parsedApiBaseUrl = null;
}

function proxyToApi(req, res, upstreamPath) {
    if (!parsedApiBaseUrl) {
        res.status(502).json({
            error: 'API proxy misconfigured',
            message: 'Unable to determine API_BASE_URL for proxying requests',
            target: API_BASE_URL
        });
        return;
    }

    const targetPath = upstreamPath || req.originalUrl || req.url || '/api';
    const targetUrl = new URL(targetPath, parsedApiBaseUrl);
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

    const proxyReq = client.request(options, (proxyRes) => {
        res.status(proxyRes.statusCode || 500);
        Object.entries(proxyRes.headers).forEach(([key, value]) => {
            if (value !== undefined) {
                res.setHeader(key, value);
            }
        });
        proxyRes.pipe(res);
    });

    proxyReq.on('error', (error) => {
        console.error('[Document Manager UI] API proxy error:', error.message);
        if (!res.headersSent) {
            res.status(502).json({
                error: 'API request failed',
                message: error.message,
                target: targetUrl.href
            });
        } else {
            res.end();
        }
    });

    req.on('aborted', () => {
        proxyReq.destroy();
    });

    if (req.method === 'GET' || req.method === 'HEAD' || req.method === 'OPTIONS') {
        proxyReq.end();
        return;
    }

    req.pipe(proxyReq);
}

app.use('/api', (req, res) => {
    const requestedPath = req.originalUrl || req.url || '/api';
    const upstreamPath = requestedPath.startsWith('/api') ? requestedPath : `/api${requestedPath}`;
    proxyToApi(req, res, upstreamPath);
});

app.use(express.static(path.join(__dirname)));

app.get('/health', (req, res) => {
    res.json({
        status: 'healthy',
        service: 'document-manager-ui',
        timestamp: new Date().toISOString(),
        readiness: true,
        api_proxy_target: parsedApiBaseUrl ? parsedApiBaseUrl.origin : API_BASE_URL
    });
});

app.get('/ui-health', (req, res) => {
    res.json({
        status: 'healthy',
        service: 'document-manager-ui',
        version: '2.0.0',
        timestamp: new Date().toISOString(),
        api_proxy_target: parsedApiBaseUrl ? parsedApiBaseUrl.origin : API_BASE_URL
    });
});

app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

app.get('*', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

app.listen(PORT, () => {
    console.log(`Document Manager UI running on port ${PORT}`);
    console.log(`Access the application at: http://localhost:${PORT}`);
    console.log(`Proxying API requests to: ${API_BASE_URL}`);
});

module.exports = app;
