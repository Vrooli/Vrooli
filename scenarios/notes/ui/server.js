const express = require('express');
const path = require('path');
const cors = require('cors');
const fs = require('fs');
const http = require('http');
const https = require('https');
const { URL } = require('url');

const app = express();

// Port configuration - REQUIRED, no defaults
const PORT = process.env.UI_PORT;
if (!PORT) {
    console.error('[SmartNotes UI] UI_PORT environment variable is required');
    process.exit(1);
}

// API configuration - REQUIRED
const API_PORT = process.env.API_PORT;
if (!API_PORT) {
    console.error('[SmartNotes UI] API_PORT environment variable is required');
    process.exit(1);
}

const API_URL = process.env.API_URL || `http://localhost:${API_PORT}`;

const stripTrailingSlash = (value = '') => value.replace(/\/+$/u, '') || '';
const ensureLeadingSlash = (value = '') => (value.startsWith('/') ? value : `/${value}`);

let apiProtocol = 'http:';
let apiHostname = '127.0.0.1';
let apiPortNumber = Number(API_PORT);
let apiPathBase = '';

try {
    const parsed = new URL(API_URL);
    apiProtocol = parsed.protocol || apiProtocol;
    apiHostname = parsed.hostname || apiHostname;
    apiPortNumber = Number(parsed.port || apiPortNumber);
    apiPathBase = stripTrailingSlash(parsed.pathname || '');
} catch (error) {
    console.warn('[SmartNotes UI] Unable to parse API_URL, defaulting to localhost:', error.message);
}

const httpClient = apiProtocol === 'https:' ? https : http;

function buildUpstreamPath(upstreamPath = '/') {
    const suffix = ensureLeadingSlash(upstreamPath || '/');
    if (!apiPathBase || apiPathBase === '/') {
        return suffix;
    }
    return `${apiPathBase}${suffix}`;
}

function proxyToApi(req, res, upstreamPath) {
    const targetPath = buildUpstreamPath(upstreamPath || req.originalUrl || req.url || '/');

    const requestOptions = {
        protocol: apiProtocol,
        hostname: apiHostname,
        port: apiPortNumber,
        method: req.method,
        path: targetPath,
        headers: {
            ...req.headers,
            host: `${apiHostname}:${apiPortNumber}`,
        },
    };

    const proxyReq = httpClient.request(requestOptions, (proxyRes) => {
        res.status(proxyRes.statusCode || 502);
        Object.entries(proxyRes.headers).forEach(([key, value]) => {
            if (typeof value !== 'undefined') {
                res.setHeader(key, value);
            }
        });
        proxyRes.pipe(res);
    });

    proxyReq.on('error', (error) => {
        console.error(`[SmartNotes UI] Proxy error for ${req.method} ${targetPath}:`, error.message);
        if (!res.headersSent) {
            res.status(502).json({
                error: 'API proxy request failed',
                details: error.message,
                target: `${apiProtocol}//${apiHostname}:${apiPortNumber}${targetPath}`,
            });
        } else {
            res.end();
        }
    });

    if (req.method === 'GET' || req.method === 'HEAD') {
        proxyReq.end();
    } else if (req.readable) {
        req.pipe(proxyReq);
    } else if (req.body) {
        const payload = typeof req.body === 'string' ? req.body : JSON.stringify(req.body);
        proxyReq.end(payload);
    } else {
        proxyReq.end();
    }
}

app.use(cors());

app.get('/health', (req, res) => proxyToApi(req, res, '/health'));

app.use('/api', (req, res) => {
    const suffix = req.originalUrl.replace(/^\/api/, '') || '/';
    proxyToApi(req, res, `/api${suffix}`);
});

const serveIndexWithPort = (htmlFile) => (req, res) => {
    const htmlPath = path.join(__dirname, htmlFile);

    fs.readFile(htmlPath, 'utf8', (err, html) => {
        if (err) {
            res.status(500).send('Error loading page');
            return;
        }

        const injectedHtml = html.replace(
            '</head>',
            `<script>window.API_PORT = '${API_PORT}';</script></head>`
        );

        res.send(injectedHtml);
    });
};

app.get('/zen', serveIndexWithPort('zen-index.html'));
app.get('/', serveIndexWithPort('index.html'));
app.use(express.static(__dirname));

app.listen(PORT, () => {
    console.log(`[SmartNotes UI] listening on http://localhost:${PORT}`);
    console.log(`[SmartNotes UI] proxying API to ${apiProtocol}//${apiHostname}:${apiPortNumber}${apiPathBase || ''}`);
});

module.exports = { app, proxyToApi };
