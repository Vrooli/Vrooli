const express = require('express');
const cors = require('cors');
const path = require('path');

const app = express();

// Configuration
const PORT = process.env.SAAS_LANDING_UI_PORT || 3000;
const API_BASE_URL = process.env.SAAS_LANDING_API_URL || 'http://localhost:8080/api/v1';

// Middleware
app.use(cors());
app.use(express.json());
app.use(express.static(path.join(__dirname, 'public')));

// Serve main dashboard
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

// API proxy endpoints for frontend
app.use('/api/v1', (req, res) => {
    const apiUrl = API_BASE_URL + req.path;
    
    // Forward request to API
    const options = {
        method: req.method,
        headers: {
            'Content-Type': 'application/json',
            ...req.headers
        }
    };
    
    if (req.body && Object.keys(req.body).length > 0) {
        options.body = JSON.stringify(req.body);
    }
    
    fetch(apiUrl, options)
        .then(response => response.json())
        .then(data => res.json(data))
        .catch(error => {
            console.error('API proxy error:', error);
            res.status(500).json({ error: 'API request failed' });
        });
});

// Health check
app.get('/health', (req, res) => {
    res.json({
        status: 'healthy',
        service: 'saas-landing-manager-ui',
        timestamp: new Date().toISOString()
    });
});

// Start server
app.listen(PORT, () => {
    console.log(`🚀 SaaS Landing Manager UI running on http://localhost:${PORT}`);
    console.log(`📡 API proxy to: ${API_BASE_URL}`);
});

// Graceful shutdown
process.on('SIGINT', () => {
    console.log('\n🛑 Shutting down SaaS Landing Manager UI...');
    process.exit(0);
});

process.on('SIGTERM', () => {
    console.log('\n🛑 Shutting down SaaS Landing Manager UI...');
    process.exit(0);
});