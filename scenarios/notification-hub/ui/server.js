const express = require('express');
const path = require('path');
const { createProxyMiddleware } = require('http-proxy-middleware');

const app = express();
const PORT = process.env.UI_PORT || process.env.PORT;
const API_PORT = process.env.API_PORT;

// Serve static files
app.use(express.static(__dirname));

// Health check endpoint
app.get('/health', (req, res) => {
    res.json({ 
        status: 'healthy', 
        service: 'notification-hub-ui',
        scenario: 'unknown',
        port: PORT,
        timestamp: new Date().toISOString()
    });
});

// API proxy to Go backend
app.use('/api', createProxyMiddleware({
    target: `http://localhost:${API_PORT}`,
    changeOrigin: true,
    onError: (err, req, res) => {
        console.error('API proxy error:', err);
        res.status(500).json({ error: 'API connection error' });
    }
}));

// Serve main dashboard
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

// Serve profile management page
app.get('/profiles', (req, res) => {
    res.sendFile(path.join(__dirname, 'profiles.html'));
});

// Serve analytics page
app.get('/analytics', (req, res) => {
    res.sendFile(path.join(__dirname, 'analytics.html'));
});

app.listen(PORT, () => {
    console.log(`🔔 Notification Hub UI running on http://localhost:${PORT}`);
    console.log(`📊 Dashboard: http://localhost:${PORT}`);
    console.log(`⚙️ Profiles: http://localhost:${PORT}/profiles`);
    console.log(`📈 Analytics: http://localhost:${PORT}/analytics`);
    console.log(`🔗 API: http://localhost:${API_PORT}`);
});
