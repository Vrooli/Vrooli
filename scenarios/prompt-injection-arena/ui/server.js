const express = require('express');
const cors = require('cors');
const path = require('path');
const http = require('http');

const app = express();

// Require lifecycle-provided environment variables (no defaults)
const uiPort = process.env.UI_PORT;
const apiPort = process.env.API_PORT;

if (!uiPort || !apiPort) {
    console.error('❌ Missing required environment variables for Prompt Injection Arena UI');
    if (!uiPort) console.error('   • UI_PORT is required');
    if (!apiPort) console.error('   • API_PORT is required');
    process.exit(1);
}

// Respect reverse proxy headers so we can derive external URLs
app.set('trust proxy', true);

// Middleware
app.use(cors());

// Runtime configuration for the frontend
app.get('/config.js', (req, res) => {
    const forwardedHost = req.get('x-forwarded-host');
    const forwardedProto = req.get('x-forwarded-proto');
    const forwardedPrefix = req.get('x-forwarded-prefix');

    const protocolHeader = forwardedProto || req.protocol || 'http';
    const protocol = String(protocolHeader).split(',')[0].trim();
    const host = forwardedHost || req.get('host');

    const originalUrl = req.originalUrl || '/config.js';
    const basePath = originalUrl.replace(/\/config\.js(?:\?.*)?$/, '/');

    const cleanedPrefix = (forwardedPrefix || basePath || '/').replace(/\/$/, '');
    const apiProxyPath = `${cleanedPrefix}/_ui/api`.replace(/\/{2,}/g, '/');

    const directApiBase = process.env.API_BASE_URL || `http://localhost:${apiPort}`;

    const config = {
        generatedAt: new Date().toISOString(),
        apiPort: Number(apiPort),
        apiBase: apiProxyPath || '/_ui/api',
        apiProxyPath: apiProxyPath || '/_ui/api',
        apiDirectBase: directApiBase,
        runtime: {
            protocol,
            host,
            basePath,
            forwardedPrefix: forwardedPrefix || null
        }
    };

    res.type('application/javascript');
    res.send(`window.__PROMPT_INJECTION_CONFIG__ = ${JSON.stringify(config)};`);
});

const apiPortNumber = Number(apiPort);

const proxyToApi = (req, res) => {
    const suffix = req.originalUrl.replace(/^\/(_ui\/)?api/, '');
    const targetPath = suffix ? (suffix.startsWith('/') ? suffix : `/${suffix}`) : '/';

    const options = {
        hostname: '127.0.0.1',
        port: apiPortNumber,
        path: targetPath,
        method: req.method,
        headers: {
            ...req.headers,
            host: `localhost:${apiPortNumber}`
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
        console.error('[prompt-injection-arena-ui] API proxy error:', error.message);
        if (!res.headersSent) {
            res.status(502).json({
                error: 'API proxy failed',
                details: error.message,
                target: `http://localhost:${apiPortNumber}${targetPath}`
            });
        } else {
            res.end();
        }
    });

    if (req.method === 'GET' || req.method === 'HEAD' || req.method === 'OPTIONS') {
        proxyReq.end();
    } else {
        req.pipe(proxyReq);
    }
};

app.use('/_ui/api', proxyToApi);
app.use('/api', proxyToApi);

app.use(express.static(__dirname));

// Health check endpoint
app.get('/health', (req, res) => {
    res.json({
        status: 'healthy',
        service: 'prompt-injection-arena-ui',
        timestamp: new Date().toISOString()
    });
});

// Serve the main application
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

// Start server
app.listen(uiPort, () => {
    console.log(`🏟️ Prompt Injection Arena UI running at http://localhost:${uiPort}`);
    console.log(`📊 API expected at http://localhost:${apiPort}`);
});
