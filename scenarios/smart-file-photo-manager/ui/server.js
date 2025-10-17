const express = require('express');
const path = require('path');
const { createProxyMiddleware } = require('http-proxy-middleware');

const app = express();
const PORT = process.env.UI_PORT || process.env.PORT;
const API_PORT = process.env.API_PORT;

// Serve static files
app.use(express.static(path.join(__dirname)));

// Health check endpoint for orchestrator
app.get('/health', (req, res) => {
    res.json({ 
        status: 'healthy',
        scenario: 'smart-file-photo-manager',
        port: PORT,
        timestamp: new Date().toISOString()
    });
});


// Proxy API requests to the Go backend
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
