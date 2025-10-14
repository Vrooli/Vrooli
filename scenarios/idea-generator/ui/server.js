const express = require('express');
const path = require('path');
const cors = require('cors');
const axios = require('axios');

if (typeof window !== 'undefined' && window.parent !== window && !window.__IDEA_GENERATOR_BRIDGE_INITIALIZED) {
    try {
        const { initIframeBridgeChild } = require('@vrooli/iframe-bridge/child');

        let parentOrigin;
        if (document.referrer) {
            parentOrigin = new URL(document.referrer).origin;
        }

        initIframeBridgeChild({ parentOrigin, appId: 'idea-generator' });
        window.__IDEA_GENERATOR_BRIDGE_INITIALIZED = true;
    } catch (error) {
        console.warn('[idea-generator] iframe bridge bootstrap skipped', error);
    }
}

const app = express();
const PORT = process.env.UI_PORT || process.env.PORT || '35100';
const API_URL = process.env.API_URL || `http://localhost:${process.env.API_PORT || '15100'}`;

// Middleware
app.use(cors());
app.use(express.json());
app.use(express.static(__dirname));

// Serve the main HTML file
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

// Health check endpoint with API connectivity check
app.get('/health', async (req, res) => {
    const now = new Date().toISOString();
    let apiConnected = false;
    let apiError = null;
    let apiLatency = null;

    try {
        const start = Date.now();
        const apiResponse = await axios.get(`${API_URL}/health`, { timeout: 5000 });
        apiLatency = Date.now() - start;
        apiConnected = apiResponse.status === 200;
    } catch (error) {
        apiError = {
            code: error.code || 'UNKNOWN_ERROR',
            message: error.message,
            category: 'network',
            retryable: true
        };
    }

    res.json({
        status: apiConnected ? 'healthy' : 'degraded',
        service: 'idea-generator-ui',
        timestamp: now,
        readiness: apiConnected,
        api_connectivity: {
            connected: apiConnected,
            api_url: API_URL,
            last_check: now,
            error: apiError,
            latency_ms: apiLatency
        }
    });
});

// Proxy API requests to the backend
app.use('/api', async (req, res) => {
    try {
        const response = await axios({
            method: req.method,
            url: `${API_URL}/api${req.path}`,
            data: req.body,
            headers: {
                'Content-Type': 'application/json',
                ...req.headers
            }
        });
        res.status(response.status).json(response.data);
    } catch (error) {
        console.error('API proxy error:', error.message);
        if (error.response) {
            res.status(error.response.status).json(error.response.data);
        } else {
            res.status(500).json({ error: 'Internal server error', message: error.message });
        }
    }
});

// Start server
app.listen(PORT, () => {
    console.log(`🚀 Idea Generator UI server running on http://localhost:${PORT}`);
    console.log(`📡 Proxying API requests to ${API_URL}`);
});

module.exports = app;
