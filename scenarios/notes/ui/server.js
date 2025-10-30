const express = require('express');
const path = require('path');
const cors = require('cors');

const app = express();

// Port configuration - REQUIRED, no defaults
const PORT = process.env.UI_PORT;
if (!PORT) {
    console.error('❌ UI_PORT environment variable is required');
    process.exit(1);
}

// API configuration - REQUIRED
const API_PORT = process.env.API_PORT;
if (!API_PORT) {
    console.error('❌ API_PORT environment variable is required');
    process.exit(1);
}

const API_URL = process.env.API_URL || `http://localhost:${API_PORT}`;

// Enable CORS

// Health check endpoint for orchestrator
app.get('/health', async (req, res) => {
    // Check API connectivity
    let apiConnected = false;
    let apiLatency = null;
    let apiError = null;

    try {
        const start = Date.now();
        const response = await fetch(`${API_URL}/health`);
        apiLatency = Date.now() - start;

        if (response.ok) {
            apiConnected = true;
        } else {
            apiError = {
                code: 'HTTP_ERROR',
                message: `API returned status ${response.status}`,
                category: 'network',
                retryable: true
            };
        }
    } catch (error) {
        apiError = {
            code: 'CONNECTION_FAILED',
            message: error.message,
            category: 'network',
            retryable: true
        };
    }

    // Determine overall status
    const status = apiConnected ? 'healthy' : 'degraded';
    const readiness = apiConnected;

    res.json({
        status: status,
        service: 'smartnotes-ui',
        timestamp: new Date().toISOString(),
        readiness: readiness,
        api_connectivity: {
            connected: apiConnected,
            api_url: API_URL,
            last_check: new Date().toISOString(),
            latency_ms: apiLatency,
            error: apiError
        }
    });
});

app.use(cors());

// Proxy API requests (optional, for development)
app.use('/api', (req, res) => {
    const apiPath = req.path;
    fetch(`${API_URL}/api${apiPath}`, {
        method: req.method,
        headers: req.headers,
        body: req.method !== 'GET' ? JSON.stringify(req.body) : undefined
    })
    .then(response => response.json())
    .then(data => res.json(data))
    .catch(error => res.status(500).json({ error: error.message }));
});

// Serve index.html with injected API_PORT for root and zen routes
const serveIndexWithPort = (htmlFile) => (req, res) => {
    const fs = require('fs');
    const htmlPath = path.join(__dirname, htmlFile);

    fs.readFile(htmlPath, 'utf8', (err, html) => {
        if (err) {
            res.status(500).send('Error loading page');
            return;
        }

        // Inject API_PORT into the HTML before sending
        const injectedHtml = html.replace(
            '</head>',
            `<script>window.API_PORT = '${API_PORT}';</script></head>`
        );

        res.send(injectedHtml);
    });
};

// Serve zen mode with API_PORT injected
app.get('/zen', serveIndexWithPort('zen-index.html'));

// Serve root with API_PORT injected
app.get('/', serveIndexWithPort('index.html'));

// Serve static files for everything else
app.use(express.static(__dirname));

app.listen(PORT, () => {
    console.log(`📝 SmartNotes UI running on http://localhost:${PORT}`);
    console.log(`🔗 API endpoint: ${API_URL}`);
});