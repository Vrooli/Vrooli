const express = require('express');
const path = require('path');
const http = require('http');

const app = express();

// Require explicit UI_PORT configuration (no dangerous defaults)
if (!process.env.UI_PORT) {
    console.error('ERROR: UI_PORT environment variable is required');
    process.exit(1);
}
const PORT = parseInt(process.env.UI_PORT, 10);

// API_URL with explicit API_PORT or fallback to default port
const API_URL = process.env.API_URL || `http://localhost:${process.env.API_PORT || 16604}`;

// Serve static files (except index.html which we'll inject API_URL into)
app.use((req, res, next) => {
    if (req.path === '/' || req.path === '/index.html') {
        return next();
    }
    express.static(path.join(__dirname))(req, res, next);
});

// Main dashboard route with API URL injection
app.get('/', (req, res) => {
    const fs = require('fs');
    const indexPath = path.join(__dirname, 'index.html');
    let html = fs.readFileSync(indexPath, 'utf8');

    // Inject API URL as data attribute in body tag
    html = html.replace(/<body>/, `<body data-api-url="${API_URL}">`);

    res.send(html);
});

// Health check endpoint
app.get('/health', async (req, res) => {
    const startTime = Date.now();
    let apiConnected = false;
    let apiError = null;
    let apiLatency = null;

    try {
        // Check API connectivity
        const apiHealthUrl = `${API_URL}/health`;
        const apiResponse = await new Promise((resolve, reject) => {
            const request = http.get(apiHealthUrl, (response) => {
                let data = '';
                response.on('data', chunk => data += chunk);
                response.on('end', () => {
                    apiLatency = Date.now() - startTime;
                    if (response.statusCode === 200) {
                        apiConnected = true;
                        resolve({ connected: true });
                    } else {
                        reject(new Error(`API returned status ${response.statusCode}`));
                    }
                });
            });
            request.on('error', reject);
            request.setTimeout(5000, () => {
                request.destroy();
                reject(new Error('API health check timeout'));
            });
        });
    } catch (error) {
        apiError = {
            code: 'API_CONNECTION_FAILED',
            message: error.message,
            category: 'network',
            retryable: true
        };
    }

    const status = apiConnected ? 'healthy' : 'degraded';
    const readiness = true; // UI is ready even if API is down

    res.json({
        status,
        service: 'web-scraper-manager-ui',
        timestamp: new Date().toISOString(),
        readiness,
        api_connectivity: {
            connected: apiConnected,
            api_url: `${API_URL}/health`,
            last_check: new Date().toISOString(),
            error: apiError,
            latency_ms: apiLatency
        }
    });
});

// Start the server
app.listen(PORT, () => {
    console.log(`🚀 Web Scraper Manager Dashboard running on port ${PORT}`);
    console.log(`📊 Dashboard available at: http://localhost:${PORT}`);
    console.log(`🔍 Health check: http://localhost:${PORT}/health`);
});

module.exports = app;