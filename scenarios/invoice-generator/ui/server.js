const express = require('express');
const cors = require('cors');
const path = require('path');
const fs = require('fs');

const app = express();

// Validate required environment variables
const PORT = process.env.UI_PORT;
const API_PORT = process.env.API_PORT;

if (!PORT || !API_PORT) {
    console.error('❌ Missing required environment variables');
    if (!PORT) console.error('   - UI_PORT is required');
    if (!API_PORT) console.error('   - API_PORT is required');
    process.exit(1);
}


// Health check endpoint for orchestrator (schema-compliant)
app.get('/health', async (req, res) => {
    const apiUrl = `http://localhost:${API_PORT}/health`;
    let apiConnected = false;
    let apiLatency = null;
    let apiError = null;

    // Test API connectivity
    const startTime = Date.now();
    try {
        const response = await fetch(apiUrl, { timeout: 2000 });
        if (response.ok) {
            apiConnected = true;
            apiLatency = Date.now() - startTime;
        } else {
            apiError = {
                code: 'API_UNHEALTHY',
                message: `API returned status ${response.status}`,
                category: 'resource',
                retryable: true
            };
        }
    } catch (error) {
        apiError = {
            code: error.code || 'CONNECTION_FAILED',
            message: error.message || 'Failed to connect to API',
            category: 'network',
            retryable: true
        };
    }

    // Overall status based on API connectivity
    const status = apiConnected ? 'healthy' : 'degraded';
    const readiness = apiConnected; // UI is ready only if API is reachable

    res.json({
        status: status,
        service: 'invoice-generator-ui',
        timestamp: new Date().toISOString(),
        readiness: readiness,
        api_connectivity: {
            connected: apiConnected,
            api_url: apiUrl,
            last_check: new Date().toISOString(),
            error: apiError,
            latency_ms: apiLatency
        }
    });
});

const DIST_DIR = path.join(__dirname, 'dist');
const SRC_DIR = path.join(__dirname, 'src');
const STATIC_ROOT = fs.existsSync(DIST_DIR) ? DIST_DIR : (fs.existsSync(SRC_DIR) ? SRC_DIR : __dirname);

app.use(cors());
app.use(express.static(STATIC_ROOT));

// Inject environment variables into the HTML
app.get('/', (req, res) => {
    res.sendFile(path.join(STATIC_ROOT, 'index.html'));
});

// Pass environment variables to the client
app.get('/config.js', (req, res) => {
    res.type('application/javascript');
    res.send(`
        window.API_PORT = ${API_PORT};
        window.UI_PORT = ${PORT};
    `);
});

app.listen(PORT, () => {
    console.log(`
    ⚡ Invoice Generator Pro UI
    =====================================
    🌐 UI running at: http://localhost:${PORT}
    🔌 API endpoint: http://localhost:${API_PORT}
    =====================================
    `);
});
