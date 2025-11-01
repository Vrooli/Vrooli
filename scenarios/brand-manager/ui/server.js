const express = require('express');
const cors = require('cors');
const path = require('path');
const http = require('http');
const https = require('https');

const app = express();

function requireEnv(name) {
    const value = process.env[name];
    if (!value) {
        throw new Error(`${name} environment variable is required. Run the scenario through the lifecycle system so configuration is injected automatically.`);
    }
    return value;
}

function parsePort(value, label) {
    const parsed = Number.parseInt(value, 10);
    if (Number.isNaN(parsed) || parsed <= 0) {
        throw new Error(`${label} must be a positive integer (received: ${value})`);
    }
    return parsed;
}

function resolveUiPort() {
    const raw = process.env.UI_PORT || process.env.PORT;
    if (!raw) {
        throw new Error('UI_PORT must be provided for the UI server.');
    }
    return parsePort(raw, 'UI_PORT');
}

const PORT = resolveUiPort();
const API_PORT = parsePort(requireEnv('API_PORT'), 'API_PORT');
const API_PROTOCOL = (process.env.API_PROTOCOL || process.env.BRAND_MANAGER_API_PROTOCOL || 'http').toLowerCase() === 'https' ? 'https' : 'http';
const API_HOST = process.env.API_HOST || process.env.BRAND_MANAGER_API_HOST || '127.0.0.1';
const API_TARGET = `${API_PROTOCOL}://${API_HOST}:${API_PORT}`;

function normalizeUpstreamPath(input) {
    if (!input) {
        return '/';
    }
    return input.startsWith('/') ? input : `/${input}`;
}

function buildProxyHeaders(req) {
    const headers = { ...req.headers };
    headers.host = `${API_HOST}:${API_PORT}`;
    headers.origin = API_TARGET;
    headers['x-forwarded-host'] = req.headers.host || headers.host;
    headers['x-forwarded-proto'] = req.headers['x-forwarded-proto'] || (req.protocol || (req.socket?.encrypted ? 'https' : 'http'));

    const remoteAddress = req.socket?.remoteAddress;
    if (remoteAddress) {
        headers['x-forwarded-for'] = headers['x-forwarded-for'] ? `${headers['x-forwarded-for']}, ${remoteAddress}` : remoteAddress;
    }

    return headers;
}

function logProxyEvent(phase, req, extras = '') {
    const method = req.method || 'GET';
    const url = req.originalUrl || req.url || '/';
    const host = req.headers?.host || 'n/a';
    const origin = req.headers?.origin || 'n/a';
    console.log(`[Proxy] ${phase} ${method} ${url} (host=${host}, origin=${origin}) ${extras}`.trim());
}

function proxyToApi(req, res, upstreamPath) {
    const basePath = upstreamPath ?? `${req.baseUrl || ''}${req.url || ''}`;
    const targetPath = normalizeUpstreamPath(basePath);
    const requestOptions = {
        protocol: `${API_PROTOCOL}:`,
        hostname: API_HOST,
        port: API_PORT,
        path: targetPath,
        method: req.method || 'GET',
        headers: buildProxyHeaders(req),
    };

    logProxyEvent('forward', req, `-> ${API_TARGET}${targetPath}`);

    const client = API_PROTOCOL === 'https' ? https : http;
    const proxyReq = client.request(requestOptions, (proxyRes) => {
        if (!res.headersSent) {
            res.writeHead(proxyRes.statusCode || 502, proxyRes.headers);
        }
        proxyRes.on('error', (err) => {
            console.error('API proxy response error:', err.message);
            if (!res.headersSent) {
                res.status(502).json({ error: 'API response stream failed', details: err.message });
            } else {
                res.end();
            }
        });
        proxyRes.pipe(res, { end: true });
    });

    proxyReq.on('error', (err) => {
        console.error('API proxy error:', err.message);
        if (!res.headersSent) {
            res.status(502).json({ error: 'API server unavailable', details: err.message });
        } else {
            res.end();
        }
    });

    req.on('error', (err) => {
        console.error('API proxy request stream error:', err.message);
        proxyReq.destroy(err);
    });

    req.on('aborted', () => {
        proxyReq.destroy();
    });

    req.pipe(proxyReq, { end: true });
}

app.use(cors());
app.use(express.static(__dirname));

app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

app.get('/health', (req, res) => {
    proxyToApi(req, res, '/health');
});

app.use('/api', (req, res) => {
    proxyToApi(req, res);
});

const server = app.listen(PORT, () => {
    const displayHost = process.env.UI_HOST || '127.0.0.1';
    console.log(`🎨 Brand Manager UI running at http://${displayHost}:${PORT}`);
    console.log(`📡 API Proxy forwarding to ${API_TARGET}`);
});

process.on('SIGTERM', () => {
    console.log('🛑 Brand Manager UI shutting down...');
    server.close(() => process.exit(0));
});

process.on('SIGINT', () => {
    console.log('🛑 Brand Manager UI shutting down...');
    server.close(() => process.exit(0));
});

module.exports = { proxyToApi };
