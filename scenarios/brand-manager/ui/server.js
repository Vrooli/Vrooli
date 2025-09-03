const express = require('express');
const cors = require('cors');
const path = require('path');

const app = express();
const PORT = process.env.PORT || 31006;
const API_URL = `http://localhost:${process.env.API_PORT || 22250}`;

// Middleware
app.use(cors());
app.use(express.json());
app.use(express.static(__dirname));

// Serve main page
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

// API proxy endpoints to avoid CORS issues
app.get('/api/*', async (req, res) => {
    const apiPath = req.path.replace('/api', '');
    try {
        const fetch = (await import('node-fetch')).default;
        const response = await fetch(`${API_URL}/api${apiPath}${req.url.includes('?') ? '?' + req.url.split('?')[1] : ''}`);
        const data = await response.json();
        res.status(response.status).json(data);
    } catch (error) {
        console.error('API proxy error:', error);
        res.status(500).json({ error: 'API connection failed', details: error.message });
    }
});

app.post('/api/*', async (req, res) => {
    const apiPath = req.path.replace('/api', '');
    try {
        const fetch = (await import('node-fetch')).default;
        const response = await fetch(`${API_URL}/api${apiPath}`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(req.body)
        });
        const data = await response.json();
        res.status(response.status).json(data);
    } catch (error) {
        console.error('API proxy error:', error);
        res.status(500).json({ error: 'API connection failed', details: error.message });
    }
});

// Health check
app.get('/health', (req, res) => {
    res.json({ 
        status: 'healthy', 
        service: 'brand-manager-ui',
        timestamp: new Date().toISOString(),
        api_url: API_URL
    });
});

// Start server
app.listen(PORT, () => {
    console.log(`🎨 Brand Manager UI running at http://localhost:${PORT}`);
    console.log(`📡 API Proxy: ${API_URL}`);
});

// Graceful shutdown
process.on('SIGTERM', () => {
    console.log('🛑 Brand Manager UI shutting down...');
    process.exit(0);
});

process.on('SIGINT', () => {
    console.log('🛑 Brand Manager UI shutting down...');
    process.exit(0);
});