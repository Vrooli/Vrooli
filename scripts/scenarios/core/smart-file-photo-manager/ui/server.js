const express = require('express');
const path = require('path');
const { createProxyMiddleware } = require('http-proxy-middleware');

const app = express();
const PORT = process.env.PORT || 3090;

// Serve static files
app.use(express.static(path.join(__dirname)));

// Proxy API requests to the Go backend
const API_PORT = process.env.API_PORT || 8080;
app.use('/api', createProxyMiddleware({
    target: `http://localhost:${API_PORT}`,
    changeOrigin: true,
    onError: (err, req, res) => {
        console.error('Proxy error:', err);
        res.status(500).json({ error: 'API connection error' });
    }
}));

// Serve index.html for all routes (SPA support)
app.get('*', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

app.listen(PORT, () => {
    console.log(`
╔════════════════════════════════════════════════════╗
║                                                    ║
║     SmartFile UI Server                          ║
║     AI-Powered File & Photo Manager              ║
║                                                    ║
║     🌐 UI:  http://localhost:${PORT}              ║
║     🔧 API: http://localhost:${API_PORT}          ║
║                                                    ║
║     Features:                                      ║
║     • Semantic file search                        ║
║     • AI-powered organization                     ║
║     • Duplicate detection                         ║
║     • Smart tagging                              ║
║     • Visual file management                      ║
║                                                    ║
╚════════════════════════════════════════════════════╝
    `);
});