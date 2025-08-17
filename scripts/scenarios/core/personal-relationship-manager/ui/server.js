const express = require('express');
const cors = require('cors');
const path = require('path');
require('dotenv').config();

const app = express();
const PORT = process.env.UI_PORT || 39002;

// Middleware
app.use(cors());
app.use(express.json());
app.use(express.static(__dirname));

// API proxy configuration
const API_URL = process.env.API_URL || 'http://localhost:39001';

// Proxy API requests to the Go backend
app.use('/api', (req, res) => {
    const fetch = require('node-fetch');
    const url = `${API_URL}${req.url}`;
    
    fetch(url, {
        method: req.method,
        headers: {
            'Content-Type': 'application/json',
            ...req.headers
        },
        body: req.method !== 'GET' ? JSON.stringify(req.body) : undefined
    })
    .then(response => response.json())
    .then(data => res.json(data))
    .catch(error => {
        console.error('API proxy error:', error);
        res.status(500).json({ error: 'API request failed' });
    });
});

// Serve the main HTML file
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

// Health check endpoint
app.get('/health', (req, res) => {
    res.json({ 
        status: 'healthy', 
        service: 'personal-relationship-manager-ui',
        timestamp: new Date().toISOString()
    });
});

// Start server
app.listen(PORT, () => {
    console.log(`
    💝 Personal Relationship Manager UI
    =====================================
    🌐 Server running at: http://localhost:${PORT}
    🎨 Warm, thoughtful interface for nurturing connections
    💕 Helping you stay close to the people who matter
    `);
});