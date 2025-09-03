const express = require('express');
const path = require('path');
const http = require('http');

const app = express();
const PORT = process.env.UI_PORT || process.env.PORT;

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

// Health endpoint
app.get('/health', (req, res) => {
    res.json({ status: 'healthy', service: 'seo-optimizer-ui' });
});

// API endpoints proxy
app.use('/api', (req, res) => {
    const fullApiPath = '/api' + (req.url.startsWith('/') ? req.url : '/' + req.url);
    proxyToApi(req, res, fullApiPath);
});

// SEO-specific endpoints that might be at root level
app.use('/audit', (req, res) => {
    proxyToApi(req, res, '/audit' + req.url);
});

app.use('/keywords', (req, res) => {
    proxyToApi(req, res, '/keywords' + req.url);
});

app.use('/content', (req, res) => {
    proxyToApi(req, res, '/content' + req.url);
});

app.use('/competitors', (req, res) => {
    proxyToApi(req, res, '/competitors' + req.url);
});

app.use('/rankings', (req, res) => {
    proxyToApi(req, res, '/rankings' + req.url);
});

// Serve static files
app.use(express.static(__dirname, { 
    index: false,
    setHeaders: (res, path) => {
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
║                         SEO OPTIMIZER UI                                  ║
╠════════════════════════════════════════════════════════════════════════════╣
║  UI Server:     http://localhost:${PORT}                                     ║
║  API Proxy:     http://localhost:${API_PORT}                                 ║
║  Status:        READY FOR OPTIMIZATION                                    ║
╚════════════════════════════════════════════════════════════════════════════╝
    `);
});
