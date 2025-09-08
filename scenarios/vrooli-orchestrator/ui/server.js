const express = require('express');
const cors = require('cors');
const axios = require('axios');
const path = require('path');

const app = express();

// Configuration
const UI_PORT = process.env.UI_PORT || 3002;
const API_BASE = process.env.VROOLI_ORCHESTRATOR_API_BASE || `http://localhost:${process.env.API_PORT || 15001}`;

console.log(`🎯 Vrooli Orchestrator Dashboard`);
console.log(`   UI Port: ${UI_PORT}`);
console.log(`   API Base: ${API_BASE}`);

// Middleware
app.use(cors());
app.use(express.json());
app.use(express.static(path.join(__dirname)));

// Health check endpoint
app.get('/health', (req, res) => {
    res.json({
        status: 'healthy',
        service: 'vrooli-orchestrator-ui',
        version: '1.0.0',
        api_base: API_BASE,
        timestamp: new Date().toISOString()
    });
});

// API proxy endpoints - forward to the orchestrator API
app.all('/api/*', async (req, res) => {
    try {
        const apiUrl = `${API_BASE}${req.path}`;
        
        const config = {
            method: req.method.toLowerCase(),
            url: apiUrl,
            headers: {
                'Content-Type': 'application/json',
                ...req.headers
            }
        };
        
        if (req.body && Object.keys(req.body).length > 0) {
            config.data = req.body;
        }
        
        console.log(`📡 Proxying ${req.method} ${req.path} -> ${apiUrl}`);
        
        const response = await axios(config);
        res.status(response.status).json(response.data);
        
    } catch (error) {
        console.error(`❌ API proxy error for ${req.path}:`, error.message);
        
        if (error.response) {
            res.status(error.response.status).json(error.response.data);
        } else {
            res.status(500).json({
                error: 'API proxy failed',
                message: error.message,
                api_base: API_BASE
            });
        }
    }
});

// Serve the main dashboard
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

// Catch-all for SPA routing
app.get('*', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

// Start server
app.listen(UI_PORT, () => {
    console.log(`🚀 Vrooli Orchestrator Dashboard running on http://localhost:${UI_PORT}`);
    console.log(`   API proxied from: ${API_BASE}`);
    console.log(`   Ready for profile management!`);
});