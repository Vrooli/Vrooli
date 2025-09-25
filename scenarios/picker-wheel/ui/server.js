const express = require('express');
const path = require('path');
const http = require('http');

const app = express();
const PORT = process.env.UI_PORT || process.env.PORT;
const API_PORT = process.env.API_PORT;
const N8N_PORT = process.env.N8N_PORT || 5679;

// Manual proxy function for API calls
function proxyToApi(req, res, targetPort, apiPath) {
    const options = {
        hostname: 'localhost',
        port: targetPort,
        path: apiPath || req.url,
        method: req.method,
        headers: {
            ...req.headers,
            host: `localhost:${targetPort}`
        }
    };

    console.log(`[PROXY] ${req.method} ${req.url} -> http://localhost:${targetPort}${options.path}`);

    const proxyReq = http.request(options, (proxyRes) => {
        res.status(proxyRes.statusCode);
        Object.keys(proxyRes.headers).forEach(key => {
            res.setHeader(key, proxyRes.headers[key]);
        });
        proxyRes.pipe(res);
    });

    proxyReq.on('error', (err) => {
        console.error('Proxy error:', err.message);
        res.status(502).json({
            error: 'Server unavailable',
            details: err.message,
            target: `http://localhost:${targetPort}${options.path}`
        });
    });

    if (req.method !== 'GET' && req.method !== 'HEAD') {
        req.pipe(proxyReq);
    } else {
        proxyReq.end();
    }
}

// Health endpoint
app.get('/health', (req, res) => {
    res.json({ status: 'healthy', service: 'picker-wheel-ui' });
});

// API endpoints proxy
app.use('/api', (req, res) => {
    const fullApiPath = '/api' + (req.url.startsWith('/') ? req.url : '/' + req.url);
    proxyToApi(req, res, API_PORT, fullApiPath);
});

// n8n webhook proxy - critical for picker wheel functionality
app.use('/webhook', (req, res) => {
    const webhookPath = '/webhook' + (req.url.startsWith('/') ? req.url : '/' + req.url);
    proxyToApi(req, res, N8N_PORT, webhookPath);
});

// Serve static files
app.use(express.static(__dirname, {
    index: false,
    setHeaders: (res, path) => {
        if (path.endsWith('.js')) {
            res.setHeader('Cache-Control', 'no-cache');
            res.setHeader('Content-Type', 'application/javascript');
        }
    }
}));

// Serve the main page
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

app.listen(PORT, () => {
    console.log(`
╔════════════════════════════════════════════════════════════════════════════╗
║                           PICKER WHEEL UI                                 ║
╠════════════════════════════════════════════════════════════════════════════╣
║  UI Server:     http://localhost:${PORT}                                     ║
║  API Proxy:     http://localhost:${API_PORT}                                 ║
║  n8n Webhook:   http://localhost:${N8N_PORT}                                 ║
║  Status:        READY TO SPIN                                             ║
╚════════════════════════════════════════════════════════════════════════════╝
    `);
});
