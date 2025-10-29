const express = require('express');
const path = require('path');
const http = require('http');

const app = express();

// Validate required environment variables
if (!process.env.UI_PORT) {
    console.error('ERROR: UI_PORT environment variable is required');
    process.exit(1);
}

if (!process.env.API_PORT) {
    console.error('ERROR: API_PORT environment variable is required');
    process.exit(1);
}

const PORT = process.env.UI_PORT;
const API_PORT = process.env.API_PORT;
const N8N_PORT = process.env.N8N_PORT || null;

// Validate N8N_PORT if webhook functionality is required
if (!N8N_PORT) {
    console.warn('WARN: N8N_PORT not set - webhook functionality will be unavailable');
}

// Manual proxy function for API calls
function proxyToApi(req, res, targetPort, apiPath) {
    const options = {
        hostname: 'localhost',
        port: targetPort,
        path: apiPath || req.url,
        method: req.method,
        headers: {
            ...req.headers,
            host: `localhost:${targetPort}`
        }
    };

    console.log(`[PROXY] ${req.method} ${req.url} -> http://localhost:${targetPort}${options.path}`);

    const proxyReq = http.request(options, (proxyRes) => {
        res.status(proxyRes.statusCode);
        Object.keys(proxyRes.headers).forEach(key => {
            res.setHeader(key, proxyRes.headers[key]);
        });
        proxyRes.pipe(res);
    });

    proxyReq.on('error', (err) => {
        console.error('Proxy error:', err.message);
        res.status(502).json({
            error: 'Server unavailable',
            details: err.message,
            target: `http://localhost:${targetPort}${options.path}`
        });
    });

    if (req.method !== 'GET' && req.method !== 'HEAD') {
        req.pipe(proxyReq);
    } else {
        proxyReq.end();
    }
}

// Health endpoint
app.get('/health', async (req, res) => {
    let responseSent = false;

    const sendResponse = (response) => {
        if (responseSent) return;
        responseSent = true;
        res.json(response);
    };

    const healthCheck = {
        status: 'healthy',
        service: 'picker-wheel-ui',
        timestamp: new Date().toISOString(),
        readiness: true,
        api_connectivity: {
            connected: false,
            api_url: `http://localhost:${API_PORT}/health`,
            last_check: new Date().toISOString(),
            error: null,
            latency_ms: null
        }
    };

    // Check API connectivity
    const startTime = Date.now();
    const apiHealthReq = http.get(`http://localhost:${API_PORT}/health`, (apiRes) => {
        const latency = Date.now() - startTime;

        if (apiRes.statusCode === 200) {
            healthCheck.api_connectivity.connected = true;
            healthCheck.api_connectivity.latency_ms = latency;
            sendResponse(healthCheck);
        } else {
            healthCheck.status = 'degraded';
            healthCheck.api_connectivity.error = {
                code: 'UNEXPECTED_STATUS',
                message: `API returned status ${apiRes.statusCode}`,
                category: 'internal',
                retryable: true
            };
            sendResponse(healthCheck);
        }
    });

    apiHealthReq.on('error', (err) => {
        healthCheck.status = 'degraded';
        healthCheck.api_connectivity.error = {
            code: 'CONNECTION_FAILED',
            message: err.message,
            category: 'network',
            retryable: true,
            details: { errno: err.errno, syscall: err.syscall }
        };
        sendResponse(healthCheck);
    });

    // Timeout after 3 seconds
    const timeoutHandle = setTimeout(() => {
        apiHealthReq.destroy();
        healthCheck.status = 'degraded';
        healthCheck.api_connectivity.error = {
            code: 'TIMEOUT',
            message: 'API health check timed out after 3000ms',
            category: 'network',
            retryable: true
        };
        sendResponse(healthCheck);
    }, 3000);

    // Clear timeout if request completes
    apiHealthReq.on('close', () => {
        clearTimeout(timeoutHandle);
    });
});

// API endpoints proxy
app.use('/api', (req, res) => {
    const fullApiPath = '/api' + (req.url.startsWith('/') ? req.url : '/' + req.url);
    proxyToApi(req, res, API_PORT, fullApiPath);
});

// n8n webhook proxy - critical for picker wheel functionality
app.use('/webhook', (req, res) => {
    if (!N8N_PORT) {
        res.status(503).json({
            error: 'N8N webhook service unavailable',
            message: 'N8N_PORT environment variable not configured'
        });
        return;
    }
    const webhookPath = '/webhook' + (req.url.startsWith('/') ? req.url : '/' + req.url);
    proxyToApi(req, res, N8N_PORT, webhookPath);
});

// Serve static files
app.use(express.static(__dirname, {
    index: false,
    setHeaders: (res, path) => {
        if (path.endsWith('.js')) {
            res.setHeader('Cache-Control', 'no-cache');
            res.setHeader('Content-Type', 'application/javascript');
        }
    }
}));

// Serve the main page
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

app.listen(PORT, () => {
    console.log(`
╔════════════════════════════════════════════════════════════════════════════╗
║                           PICKER WHEEL UI                                 ║
╠════════════════════════════════════════════════════════════════════════════╣
║  UI Server:     http://localhost:${PORT}                                     ║
║  API Proxy:     http://localhost:${API_PORT}                                 ║
║  n8n Webhook:   http://localhost:${N8N_PORT}                                 ║
║  Status:        READY TO SPIN                                             ║
╚════════════════════════════════════════════════════════════════════════════╝
    `);
});
