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
app.get('/health', async (req, res) => {
    const timestamp = new Date().toISOString();
    let apiConnectivity = {
        connected: false,
        api_url: API_URL,
        last_check: timestamp,
        error: null,
        latency_ms: null
    };

    try {
        const fetch = (await import('node-fetch')).default;
        const startTime = Date.now();
        const response = await fetch(`${API_URL}/health`, {
            method: 'GET',
            timeout: 5000
        });
        const latency = Date.now() - startTime;

        if (response.ok) {
            apiConnectivity.connected = true;
            apiConnectivity.latency_ms = latency;
        } else {
            apiConnectivity.error = {
                code: `HTTP_${response.status}`,
                message: `API health check returned status ${response.status}`,
                category: 'network',
                retryable: true
            };
        }
    } catch (error) {
        apiConnectivity.error = {
            code: error.code || 'UNKNOWN_ERROR',
            message: error.message,
            category: 'network',
            retryable: true
        };
    }

    res.json({
        status: apiConnectivity.connected ? 'healthy' : 'degraded',
        service: 'research-assistant-ui',
        timestamp: timestamp,
        readiness: true,
        api_connectivity: apiConnectivity
    });
});

app.listen(PORT, () => {
    console.log(`🚀 Research Assistant UI Server running on http://localhost:${PORT}`);
    console.log(`📡 Proxying API requests to ${API_URL}`);
    console.log(`🎯 Dashboard: http://localhost:${PORT}`);
});

module.exports = app;