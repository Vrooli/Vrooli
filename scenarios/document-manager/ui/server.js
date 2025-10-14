const express = require('express');
const path = require('path');

const app = express();
const PORT = process.env.UI_PORT;

if (!PORT) {
    console.error('❌ UI_PORT environment variable is required');
    process.exit(1);
}

// Serve static files
app.use(express.static(path.join(__dirname)));

// Health check endpoint for orchestrator
app.get('/health', async (req, res) => {
    const API_BASE_URL = process.env.API_BASE_URL || `http://localhost:${process.env.API_PORT || 17810}`;

    // Check API connectivity
    let apiConnected = false;
    let apiError = null;
    let apiLatency = null;
    const checkStart = Date.now();

    try {
        const response = await fetch(`${API_BASE_URL}/health`, {
            method: 'GET',
            timeout: 5000
        });
        apiConnected = response.ok;
        apiLatency = Date.now() - checkStart;
    } catch (error) {
        apiError = {
            code: 'CONNECTION_FAILED',
            message: `Failed to connect to API: ${error.message}`,
            category: 'network',
            retryable: true
        };
    }

    res.json({
        status: apiConnected ? 'healthy' : 'degraded',
        service: 'document-manager-ui',
        timestamp: new Date().toISOString(),
        readiness: true,
        api_connectivity: {
            connected: apiConnected,
            api_url: API_BASE_URL,
            last_check: new Date().toISOString(),
            error: apiError,
            latency_ms: apiLatency
        }
    });
});


// API proxy configuration - forward API calls to backend
const API_BASE_URL = process.env.API_BASE_URL || 'http://localhost:23250';

// Health check endpoint
app.get('/ui-health', (req, res) => {
    res.json({ 
        status: 'healthy', 
        service: 'document-manager-ui',
        version: '2.0.0',
        timestamp: new Date().toISOString()
    });
});

// Serve the main application
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

// Handle SPA routing - serve index.html for any route
app.get('*', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

app.listen(PORT, () => {
    console.log(`Document Manager UI running on port ${PORT}`);
    console.log(`Access the application at: http://localhost:${PORT}`);
    console.log(`API Backend: ${API_BASE_URL}`);
});

module.exports = app;