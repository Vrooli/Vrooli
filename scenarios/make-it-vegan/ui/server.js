const express = require('express');
const path = require('path');
const fs = require('fs');

const app = express();

// Validate required environment variables - fail fast if missing
const PORT = process.env.UI_PORT;
if (!PORT) {
    console.error('ERROR: UI_PORT environment variable is required');
    process.exit(1);
}

const API_PORT = process.env.API_PORT;
if (!API_PORT) {
    console.error('ERROR: API_PORT environment variable is required');
    process.exit(1);
}

// Inject API URL configuration into the HTML
app.get('/', (req, res) => {
    const indexPath = path.join(__dirname, 'index.html');
    fs.readFile(indexPath, 'utf8', (err, html) => {
        if (err) {
            return res.status(500).send('Error loading page');
        }

        // Inject API configuration before app.js loads
        const configScript = `
            <script>
                window.API_URL = 'http://localhost:${API_PORT}/api';
            </script>
        `;

        // Insert config right before the app.js script tag
        html = html.replace('<script src="app.js"></script>',
            configScript + '<script src="app.js"></script>');

        res.send(html);
    });
});

// Serve static files for other resources
app.use(express.static(__dirname));

// Health check endpoint
app.get('/health', async (req, res) => {
    const timestamp = new Date().toISOString();
    const apiUrl = `http://localhost:${API_PORT}/health`;

    let apiConnectivity = {
        connected: false,
        api_url: apiUrl,
        last_check: timestamp,
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
                message: `API health check returned ${response.status}`,
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

    const healthResponse = {
        status: apiConnectivity.connected ? 'healthy' : 'degraded',
        service: 'make-it-vegan-ui',
        timestamp: timestamp,
        readiness: true, // UI is always ready to serve, even if API is down
        api_connectivity: apiConnectivity
    };

    res.json(healthResponse);
});

app.listen(PORT, () => {
    console.log(`🌱 Make It Vegan UI running at http://localhost:${PORT}`);
    console.log(`💚 Ready to help with plant-based alternatives!`);
});
