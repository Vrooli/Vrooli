const express = require('express');
const cors = require('cors');
const path = require('path');

const app = express();
const PORT = process.env.UI_PORT;

if (!PORT) {
    console.error('ERROR: UI_PORT environment variable is required');
    process.exit(1);
}

// Middleware
app.use(cors());
app.use(express.static(path.join(__dirname, 'dist')));

// Health check - compliant with UI health schema
app.get('/health', async (req, res) => {
    const apiPort = process.env.API_PORT || '16515';
    const apiUrl = `http://localhost:${apiPort}/health`;
    const timestamp = new Date().toISOString();

    let apiConnectivity = {
        connected: false,
        api_url: apiUrl,
        last_check: timestamp,
        error: null,
        latency_ms: null
    };

    try {
        const startTime = Date.now();
        const response = await fetch(apiUrl);
        const latency = Date.now() - startTime;

        if (response.ok) {
            apiConnectivity.connected = true;
            apiConnectivity.latency_ms = latency;
        } else {
            apiConnectivity.error = {
                code: 'HTTP_ERROR',
                message: `API returned status ${response.status}`,
                category: 'network',
                retryable: true
            };
        }
    } catch (error) {
        apiConnectivity.error = {
            code: 'CONNECTION_FAILED',
            message: error.message,
            category: 'network',
            retryable: true
        };
    }

    res.json({
        status: apiConnectivity.connected ? 'healthy' : 'degraded',
        service: 'symbol-search-ui',
        timestamp: timestamp,
        readiness: true,
        api_connectivity: apiConnectivity
    });
});

// Serve index.html for all routes
app.get('*', (req, res) => {
    res.sendFile(path.join(__dirname, 'dist/index.html'));
});

// Start server
app.listen(PORT, () => {
    console.log(`Symbol Search UI running on http://localhost:${PORT}`);
});
