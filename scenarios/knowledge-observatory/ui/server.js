const express = require('express');
const path = require('path');
const http = require('http');
const https = require('https');

const app = express();

const PORT = process.env.UI_PORT || process.env.PORT;
if (!PORT) {
    console.error('❌ FATAL: UI_PORT or PORT environment variable is required');
    process.exit(1);
}

const API_PORT = process.env.API_PORT;
const API_URL = process.env.API_URL || (API_PORT ? `http://localhost:${API_PORT}` : undefined);

if (!API_URL) {
    console.error('❌ FATAL: API_PORT or API_URL environment variable is required');
    process.exit(1);
}

function proxyToApi(req, res, apiPath) {
    let target;

    try {
        target = new URL(apiPath || req.originalUrl, API_URL);
    } catch (error) {
        console.error('Failed to resolve API target:', error.message);
        res.status(500).json({
            error: 'Failed to resolve API target',
            details: error.message
        });
        return;
    }

    const client = target.protocol === 'https:' ? https : http;
    const requestOptions = {
        protocol: target.protocol,
        hostname: target.hostname,
        port: target.port || (target.protocol === 'https:' ? 443 : 80),
        path: `${target.pathname}${target.search}`,
        method: req.method,
        headers: {
            ...req.headers,
            host: target.host,
        },
    };

    const proxyReq = client.request(requestOptions, (proxyRes) => {
        res.status(proxyRes.statusCode || 502);
        for (const [key, value] of Object.entries(proxyRes.headers)) {
            if (typeof value !== 'undefined') {
                res.setHeader(key, value);
            }
        }
        proxyRes.pipe(res);
    });

    proxyReq.on('error', (error) => {
        console.error('API proxy error:', error.message);
        res.status(502).json({
            error: 'API server unavailable',
            details: error.message,
            target: target.toString(),
        });
    });

    if (req.method === 'GET' || req.method === 'HEAD') {
        proxyReq.end();
    } else if (req.readable) {
        req.pipe(proxyReq);
    } else {
        proxyReq.end();
    }
}

app.get('/env.js', (_req, res) => {
    res.type('application/javascript');
    const script = `
        window.ENV = {
            API_URL: '${API_URL.replace(/'/g, "\\'")}',
            API_PORT: '${API_PORT || ''}'
        };
    `;
    res.send(script);
});

app.use(express.static(__dirname));

app.get('/health', (req, res) => {
    proxyToApi(req, res, '/health');
});

app.use('/api', (req, res) => {
    const fullApiPath = req.originalUrl.startsWith('/api') ? req.originalUrl : `/api${req.url}`;
    proxyToApi(req, res, fullApiPath);
});

app.get('*', (_req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

const server = app.listen(PORT, () => {
    console.log(`🔭 Knowledge Observatory UI running on port ${PORT}`);
    console.log(`📡 Proxying API requests to ${API_URL}`);
});

process.on('SIGINT', () => {
    console.log('\n🛑 Shutting down Knowledge Observatory UI...');
    server.close(() => {
        process.exit(0);
    });
});
