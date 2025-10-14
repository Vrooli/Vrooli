const express = require('express');
const cors = require('cors');
const path = require('path');

const app = express();

// Fail fast when critical environment variables are missing
const PORT = process.env.UI_PORT;
if (!PORT) {
    console.error('❌ UI_PORT environment variable is required');
    process.exit(1);
}

const API_URL = process.env.API_URL;
if (!API_URL) {
    console.error('❌ API_URL environment variable is required');
    process.exit(1);
}

// Middleware
app.use(cors());
app.use(express.json());
app.use(express.static(path.join(__dirname)));

// Serve the main HTML file
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

// Proxy API requests to the Go backend
app.use('/api', async (req, res) => {
    try {
        const fetch = (await import('node-fetch')).default;
        const url = `${API_URL}${req.originalUrl}`;
        
        const response = await fetch(url, {
            method: req.method,
            headers: req.headers,
            body: req.method !== 'GET' ? JSON.stringify(req.body) : undefined
        });
        
        const data = await response.json();
        res.json(data);
    } catch (error) {
        console.error('API Proxy Error:', error);
        res.status(500).json({ error: 'API request failed', details: error.message });
    }
});

// Health check endpoint
app.get('/health', (req, res) => {
    res.json({ 
        status: 'healthy', 
        service: 'research-assistant-ui',
        port: PORT,
        timestamp: new Date().toISOString()
    });
});

app.listen(PORT, () => {
    console.log(`🚀 Research Assistant UI Server running on http://localhost:${PORT}`);
    console.log(`📡 Proxying API requests to ${API_URL}`);
    console.log(`🎯 Dashboard: http://localhost:${PORT}`);
});

module.exports = app;