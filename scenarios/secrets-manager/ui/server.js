const express = require('express');
const path = require('path');
const http = require('http');

const app = express();
const PORT = process.env.UI_PORT || process.env.PORT;
const API_PORT = process.env.API_PORT;

// Manual proxy function for API calls
function proxyToApi(req, res, apiPath) {
    const options = {
        hostname: 'localhost',
        port: API_PORT,
        path: apiPath || req.url,
        method: req.method,
        headers: {
            ...req.headers,
            host: `localhost:${API_PORT}`
        }
    };
    
    console.log(`[PROXY] ${req.method} ${req.url} -> http://localhost:${API_PORT}${options.path}`);
    
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
            error: 'API server unavailable', 
            details: err.message,
            target: `http://localhost:${API_PORT}${options.path}`
        });
    });
    
    if (req.method !== 'GET' && req.method !== 'HEAD') {
        req.pipe(proxyReq);
    } else {
        proxyReq.end();
    }
}

// Health endpoint proxy
app.use('/health', (req, res) => {
    proxyToApi(req, res, '/health');
});

// API endpoints proxy
app.use('/api', (req, res) => {
    const fullApiPath = '/api' + (req.url.startsWith('/') ? req.url : '/' + req.url);
    proxyToApi(req, res, fullApiPath);
});

// Also proxy root-level endpoints that the secrets-manager API expects
app.use('/scan', (req, res) => {
    proxyToApi(req, res, '/scan');
});

app.use('/validate', (req, res) => {
    proxyToApi(req, res, '/validate');
});

app.use('/rotate', (req, res) => {
    proxyToApi(req, res, '/rotate');
});

app.use('/secrets', (req, res) => {
    proxyToApi(req, res, '/secrets' + req.url);
});

// Serve static files with security headers
app.use(express.static(__dirname, { 
    index: false,
    setHeaders: (res, path) => {
        // Security headers
        res.setHeader('X-Content-Type-Options', 'nosniff');
        res.setHeader('X-Frame-Options', 'DENY');
        res.setHeader('X-XSS-Protection', '1; mode=block');
        res.setHeader('Referrer-Policy', 'strict-origin-when-cross-origin');

// Health check endpoint for orchestrator
app.get('/health', (req, res) => {
    res.json({ 
        status: 'healthy',
        scenario: 'secrets-manager',
        port: PORT,
        timestamp: new Date().toISOString()
    });
});

        
        if (path.endsWith('.js')) {
            res.setHeader('Cache-Control', 'no-cache');
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
║                    DARK CHROME SECURITY DASHBOARD                         ║
║                         Secrets Manager v1.0                              ║
╠════════════════════════════════════════════════════════════════════════════╣
║  UI Server:     http://localhost:${PORT}                                     ║
║  API Proxy:     http://localhost:${API_PORT}                                 ║
║  Status:        OPERATIONAL                                               ║
╚════════════════════════════════════════════════════════════════════════════╝
    `);
});
