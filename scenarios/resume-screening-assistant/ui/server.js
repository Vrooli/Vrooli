const express = require('express');
const cors = require('cors');
const path = require('path');

const app = express();
const PORT = process.env.UI_PORT || 3000;
const API_PORT = process.env.API_PORT || 8090;

// Enable CORS
app.use(cors());

// Serve static files
app.use(express.static(path.join(__dirname)));

// API proxy endpoint (forwards to the Go API)
app.use('/api', (req, res) => {
    const apiUrl = `http://localhost:${API_PORT}${req.url}`;
    
    // Simple proxy - in production you'd use http-proxy-middleware
    const http = require('http');
    const options = {
        hostname: 'localhost',
        port: API_PORT,
        path: `/api${req.url}`,
        method: req.method,
        headers: req.headers
    };
    
    const proxyReq = http.request(options, (proxyRes) => {
        res.status(proxyRes.statusCode);
        Object.keys(proxyRes.headers).forEach(key => {
            res.setHeader(key, proxyRes.headers[key]);
        });
        proxyRes.pipe(res);
    });
    
    proxyReq.on('error', (error) => {
        console.error('Proxy error:', error);
        res.status(500).json({ error: 'API connection failed' });
    });
    
    if (req.body) {
        proxyReq.write(JSON.stringify(req.body));
    }
    
    proxyReq.end();
});

// Health check
app.get('/health', (req, res) => {
    res.json({ status: 'healthy', ui: 'HyperRecruit 3000' });
});

// Catch all - serve the main HTML
app.get('*', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

app.listen(PORT, () => {
    console.log(`
╔══════════════════════════════════════════════╗
║       🚀 HyperRecruit 3000 UI Server         ║
╠══════════════════════════════════════════════╣
║  UI Server:    http://localhost:${PORT}        ║
║  API Backend:  http://localhost:${API_PORT}        ║
║  Status:       ✅ ONLINE                     ║
╚══════════════════════════════════════════════╝
    `);
});