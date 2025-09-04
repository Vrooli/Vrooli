const express = require('express');
const path = require('path');
const { createProxyMiddleware } = require('http-proxy-middleware');

const app = express();
const PORT = process.env.UI_PORT || process.env.PORT;
const API_URL = process.env.API_URL || 'http://localhost:8780';

// Serve static files
app.use(express.static(__dirname));

// Proxy API requests
app.use('/api', createProxyMiddleware({
    target: API_URL,
    changeOrigin: true,
    pathRewrite: {
        '^/api': ''
    }
}));

// Serve index.html for all routes
app.get('*', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

app.listen(PORT, () => {
    console.log(`🎨 Palette Gen UI running on http://localhost:${PORT}`);
});