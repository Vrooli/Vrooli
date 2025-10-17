const express = require('express');
const path = require('path');
const fs = require('fs');
const app = express();

// Require critical environment variables - no defaults allowed
if (!process.env.UI_PORT) {
    console.error('❌ UI_PORT environment variable is required');
    process.exit(1);
}
if (!process.env.API_PORT) {
    console.error('❌ API_PORT environment variable is required');
    process.exit(1);
}

const PORT = process.env.UI_PORT;
const API_PORT = process.env.API_PORT;
const VROOLI_ROOT = process.env.VROOLI_ROOT || path.resolve(__dirname, '..', '..', '..');
const IFRAME_BRIDGE_DIST = path.join(VROOLI_ROOT, 'packages', 'iframe-bridge', 'dist');

// Only load iframe bridge if available (skip - this is server-side, not browser)
// Note: iframe bridge initialization happens in browser via bridge-init.js


// Enable CORS for API communication - restrict to specific origins
app.use((req, res, next) => {
    const allowedOrigins = [
        'http://localhost:18446',
        `http://localhost:${API_PORT}`,
        `http://localhost:${PORT}`
    ];
    const origin = req.headers.origin;

    if (allowedOrigins.includes(origin)) {
        res.header('Access-Control-Allow-Origin', origin);
    }
    res.header('Access-Control-Allow-Headers', 'Content-Type');
    next();
});

// Serve static files
app.use(express.static(__dirname));

if (fs.existsSync(IFRAME_BRIDGE_DIST)) {
    app.use('/vendor/iframe-bridge', express.static(IFRAME_BRIDGE_DIST));
} else {
    console.warn(`Warning: iframe bridge dist not found at ${IFRAME_BRIDGE_DIST}`);
}

// Health check endpoint with API connectivity check
app.get('/health', async (req, res) => {
    const apiUrl = `http://localhost:${API_PORT}`;
    let apiConnected = false;
    let apiError = null;
    let apiLatency = null;

    // Check API connectivity
    const startTime = Date.now();
    try {
        const http = require('http');
        await new Promise((resolve, reject) => {
            const healthReq = http.get(`${apiUrl}/health`, (healthRes) => {
                if (healthRes.statusCode === 200) {
                    apiConnected = true;
                    apiLatency = Date.now() - startTime;
                    resolve();
                } else {
                    reject(new Error(`API returned status ${healthRes.statusCode}`));
                }
                healthRes.resume(); // Consume response data
            });
            healthReq.on('error', (err) => reject(err));
            healthReq.setTimeout(5000, () => {
                healthReq.destroy();
                reject(new Error('API health check timeout'));
            });
        });
    } catch (error) {
        apiConnected = false;
        apiError = {
            code: 'CONNECTION_FAILED',
            message: error.message,
            category: 'network',
            retryable: true
        };
    }

    res.json({
        status: apiConnected ? 'healthy' : 'degraded',
        service: 'workflow-scheduler-ui',
        timestamp: new Date().toISOString(),
        readiness: true,
        api_connectivity: {
            connected: apiConnected,
            api_url: apiUrl,
            last_check: new Date().toISOString(),
            error: apiError,
            latency_ms: apiLatency
        }
    });
});

// API proxy configuration
app.get('/api/config', (req, res) => {
    // Require RESOURCE_PORTS to be explicitly set
    if (!process.env.RESOURCE_PORTS) {
        return res.status(500).json({
            error: 'RESOURCE_PORTS environment variable is required'
        });
    }

    let resourcePorts;
    try {
        resourcePorts = JSON.parse(process.env.RESOURCE_PORTS);
    } catch (e) {
        return res.status(500).json({
            error: 'Invalid RESOURCE_PORTS JSON format'
        });
    }

    res.json({
        apiUrl: `http://localhost:${API_PORT}`,
        uiPort: PORT,
        resources: resourcePorts
    });
});

// Fallback routing for SPAs
app.get('*', (req, res) => {
    const dashboardFile = path.join(__dirname, 'dashboard.html');
    const indexFile = path.join(__dirname, 'index.html');
    
    if (fs.existsSync(dashboardFile)) {
        res.sendFile(dashboardFile);
    } else if (fs.existsSync(indexFile)) {
        res.sendFile(indexFile);
    } else {
        res.status(404).json({ error: 'No UI files found' });
    }
});

const server = app.listen(PORT, () => {
    console.log(`✅ UI server running on http://localhost:${PORT}`);
    console.log(`📡 API endpoint: http://localhost:${API_PORT}`);
    console.log(`🏷️  Scenario: ${'unknown'}`);
});

// Graceful shutdown
process.on('SIGTERM', () => {
    server.close(() => {
        console.log('UI server stopped');
    });
});
