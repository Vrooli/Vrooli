const express = require('express');
const path = require('path');
const cors = require('cors');

const app = express();
const PORT = process.env.UI_PORT || process.env.PORT;
const API_PORT = process.env.API_PORT;

// Enable CORS
app.use(cors());

// Health check endpoint for orchestrator
app.get('/health', (req, res) => {
    res.json({ 
        status: 'healthy',
        scenario: 'mind-maps',
        port: PORT,
        timestamp: new Date().toISOString()
    });
});


// Parse JSON bodies
app.use(express.json());

// Serve static files
app.use(express.static(__dirname));

// API proxy endpoints (forward to Go API)
const API_URL = process.env.API_URL || `http://localhost:${API_PORT}`;

app.use('/api', (req, res) => {
    const fetch = require('node-fetch');
    const url = `${API_URL}${req.url}`;
    
    fetch(url, {
        method: req.method,
        headers: req.headers,
        body: req.method !== 'GET' ? JSON.stringify(req.body) : undefined
    })
    .then(response => response.json())
    .then(data => res.json(data))
    .catch(error => {
        console.error('API proxy error:', error);
        res.status(500).json({ error: 'API request failed' });
    });
});

// Serve neural view variant
app.get('/neural', (req, res) => {
    res.sendFile(path.join(__dirname, 'index-neural.html'));
});

// Serve index.html for all other routes (SPA)
app.get('*', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

app.listen(PORT, () => {
    console.log(`Mind Maps UI server running on http://localhost:${PORT}`);
});
