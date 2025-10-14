const express = require('express');
const path = require('path');
const app = express();

// Get port from environment or use default
const PORT = process.env.UI_PORT || 3000;
const API_PORT = process.env.API_PORT || 8080;

// Serve static files
app.use(express.static(__dirname));

// Health check endpoint (schema-compliant)
app.get('/health', async (req, res) => {
    const now = new Date().toISOString();
    const apiUrl = `http://localhost:${API_PORT}/health`;

    let apiConnectivity = {
        connected: false,
        api_url: apiUrl,
        last_check: now,
        error: null,
        latency_ms: null
    };

    // Test API connectivity
    try {
        const startTime = Date.now();
        const response = await fetch(apiUrl);
        const latency = Date.now() - startTime;

        if (response.ok) {
            apiConnectivity.connected = true;
            apiConnectivity.latency_ms = latency;
        } else {
            apiConnectivity.error = {
                code: `HTTP_${response.status}`,
                message: `API returned status ${response.status}`,
                category: 'network',
                retryable: true
            };
        }
    } catch (error) {
        apiConnectivity.error = {
            code: 'CONNECTION_FAILED',
            message: error.message || 'Failed to connect to API',
            category: 'network',
            retryable: true
        };
    }

    res.status(200).json({
        status: apiConnectivity.connected ? 'healthy' : 'degraded',
        service: 'date-night-planner-ui',
        timestamp: now,
        readiness: true,
        api_connectivity: apiConnectivity
    });
});

// API proxy configuration endpoint
app.get('/config', (req, res) => {
    res.json({
        apiUrl: `http://localhost:${API_PORT}`,
        version: '1.0.0'
    });
});

// Serve index.html for all routes (SPA)
app.get('*', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

// Start server
app.listen(PORT, () => {
    console.log(`💕 Date Night Planner UI running on port ${PORT}`);
    console.log(`🔗 API expected on port ${API_PORT}`);
    console.log(`🌐 Access UI at http://localhost:${PORT}`);
});