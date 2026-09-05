const http = require('http');
const fs = require('fs');
const path = require('path');

// Ensure the shared iframe bridge is initialized when this file is executed in a
// browser context (i.e., preview iframe) without breaking Node execution.
if (typeof window !== 'undefined' && window.parent !== window && !window.__FALL_FOLIAGE_BRIDGE_INITIALIZED) {
    try {
        const { initIframeBridgeChild } = require('@vrooli/iframe-bridge/child');

        let parentOrigin;
        if (document.referrer) {
            parentOrigin = new URL(document.referrer).origin;
        }

        initIframeBridgeChild({ parentOrigin, appId: 'fall-foliage-explorer' });
        window.__FALL_FOLIAGE_BRIDGE_INITIALIZED = true;
    } catch (error) {
        console.warn('[fall-foliage-explorer] iframe bridge bootstrap skipped', error);
    }
}

const PORT = requireConfiguredPort('UI_PORT');
const API_PORT = requireConfiguredPort('API_PORT');
const LOOPBACK_HOST = process.env.UI_API_HOST || '127.0.0.1';
const LOOPBACK_PROTOCOL = process.env.UI_API_PROTOCOL || 'http';

const mimeTypes = {
    '.html': 'text/html',
    '.js': 'text/javascript',
    '.css': 'text/css',
    '.json': 'application/json',
    '.png': 'image/png',
    '.jpg': 'image/jpg',
    '.gif': 'image/gif',
    '.svg': 'image/svg+xml',
    '.ico': 'image/x-icon'
};

function getStaticRoot() {
    if (fs.existsSync(DIST_DIR)) {
        return DIST_DIR;
    }
    return SRC_DIR;
}

function resolveStaticFilePath(requestUrl = '/') {
    const staticRoot = getStaticRoot();
    const urlPath = decodeURIComponent((requestUrl || '/').split('?')[0]);
    let relativePath = urlPath;
    if (!relativePath || relativePath === '/' || relativePath === './') {
        relativePath = '/index.html';
    }

    const trimmed = relativePath.replace(/^\/+/, '');
    const normalized = path
        .normalize(trimmed)
        .replace(/^(\.\.([\\/]|$))+/, '');
    const candidate = path.join(staticRoot, normalized);

    if (!candidate.startsWith(staticRoot)) {
        return path.join(staticRoot, 'index.html');
    }

    return candidate;
}

const ROOT_DIR = __dirname;
const DIST_DIR = path.join(ROOT_DIR, 'dist');
const SRC_DIR = path.join(ROOT_DIR, 'src');

function requirePort(value, name) {
    if (!value || !/^\d+$/.test(String(value))) {
        throw new Error(`${name} must be configured with a numeric port by the Vrooli lifecycle`);
    }
    return String(value);
}

function requireConfiguredPort(name) {
    return requirePort(process.env[name], name);
}

function setSecurityHeaders(res) {
    res.setHeader('X-Content-Type-Options', 'nosniff');
    res.setHeader('X-XSS-Protection', '0');
    res.setHeader('Referrer-Policy', 'no-referrer');
}

function injectProxyMetadata(content) {
    const apiBase = `${LOOPBACK_PROTOCOL}://${LOOPBACK_HOST}:${API_PORT}`;
    const metadata = `<script>window.__APP_MONITOR_PROXY_INFO__=${JSON.stringify({ apiBase })};</script>`;
    return content.replace('</head>', `${metadata}</head>`);
}

function proxyToApi(req, res) {
    const targetPath = req.url || '/api';
    const options = {
        hostname: LOOPBACK_HOST,
        port: API_PORT,
        path: targetPath,
        method: req.method,
        headers: {
            ...req.headers,
            host: `${LOOPBACK_HOST}:${API_PORT}`
        },
        timeout: 5000
    };

    const proxyReq = http.request(options, proxyRes => {
        res.writeHead(proxyRes.statusCode || 502, proxyRes.headers);
        proxyRes.pipe(res);
    });

    proxyReq.on('timeout', () => {
        proxyReq.destroy(new Error('API proxy timed out'));
    });

    proxyReq.on('error', error => {
        setSecurityHeaders(res);
        res.writeHead(502, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({
            status: 'error',
            error: 'API proxy unavailable',
            detail: error.message
        }));
    });

    req.pipe(proxyReq);
}

const server = http.createServer((req, res) => {
    setSecurityHeaders(res);

    // Health check endpoint for orchestrator
    if (req.url === '/health') {
        res.writeHead(200, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({
            status: 'healthy',
            scenario: 'fall-foliage-explorer',
            port: PORT,
            timestamp: new Date().toISOString()
        }));
        return;
    }

    if (req.url && req.url.startsWith('/api/')) {
        proxyToApi(req, res);
        return;
    }

    if (req.url === '/bridge/iframeBridgeChild.js') {
        const bridgePath = path.join(__dirname, 'node_modules', '@vrooli', 'iframe-bridge', 'dist', 'iframeBridgeChild.js');
        fs.readFile(bridgePath, (bridgeError, bridgeContent) => {
            if (bridgeError) {
                res.writeHead(500, { 'Content-Type': 'application/json' });
                res.end(JSON.stringify({ error: 'Bridge asset unavailable', detail: bridgeError.message }));
                return;
            }
            res.writeHead(200, { 'Content-Type': 'application/javascript' });
            res.end(bridgeContent, 'utf-8');
        });
        return;
    }

    const filePath = resolveStaticFilePath(req.url);
    const extname = path.extname(filePath).toLowerCase();
    const contentType = mimeTypes[extname] || 'application/octet-stream';

    fs.readFile(filePath, (error, content) => {
        if (error) {
            if (error.code === 'ENOENT') {
                res.writeHead(404, { 'Content-Type': 'text/html' });
                res.end('<h1>404 - File Not Found</h1>', 'utf-8');
            } else {
                res.writeHead(500);
                res.end(`Server Error: ${error.code}`, 'utf-8');
            }
        } else if (extname === '.html') {
            res.writeHead(200, { 'Content-Type': contentType });
            res.end(injectProxyMetadata(content.toString('utf-8')), 'utf-8');
        } else {
            res.writeHead(200, { 'Content-Type': contentType });
            res.end(content, 'utf-8');
        }
    });
});

server.listen(PORT, () => {
    console.log(`Fall Foliage Explorer UI running on port ${PORT}`);
});
