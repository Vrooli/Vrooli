const express = require('express');
const path = require('path');
const fs = require('fs');
const http = require('http');
const https = require('https');

const app = express();

// Port configuration - REQUIRED, no defaults
const PORT = process.env.UI_PORT;
if (!PORT) {
    console.error('❌ UI_PORT environment variable is required');
    process.exit(1);
}

const API_PORT = process.env.API_PORT;
if (!API_PORT) {
    console.error('❌ API_PORT environment variable is required');
    process.exit(1);
}

const API_PROTOCOL = (process.env.API_PROTOCOL || 'http').replace(/:$/u, '');
const API_HOST = process.env.API_HOST || '127.0.0.1';
const API_BASE_URL = process.env.API_BASE_URL || `${API_PROTOCOL}://${API_HOST}${API_PORT ? `:${API_PORT}` : ''}`;

let parsedApiBaseUrl;
try {
    parsedApiBaseUrl = new URL(API_BASE_URL);
} catch (error) {
    console.error('⚠️  API_BASE_URL could not be parsed:', API_BASE_URL, error.message);
    parsedApiBaseUrl = null;
}

const SCENARIO_NAME = process.env.SCENARIO_NAME || 'prompt-manager';

// Enable CORS for API communication without assuming loopback origins
app.use((req, res, next) => {
    const origin = req.headers.origin;
    if (origin) {
        res.header('Access-Control-Allow-Origin', origin);
        res.header('Vary', 'Origin');
    } else {
        res.header('Access-Control-Allow-Origin', '*');
    }
    res.header('Access-Control-Allow-Headers', 'Content-Type, Authorization, X-Requested-With');
    res.header('Access-Control-Allow-Methods', 'GET,POST,PUT,PATCH,DELETE,OPTIONS');
    res.header('Access-Control-Allow-Credentials', 'true');

    if (req.method === 'OPTIONS') {
        res.sendStatus(204);
        return;
    }

    next();
});

function proxyToApi(req, res, upstreamPath) {
    if (!parsedApiBaseUrl) {
        res.status(502).json({
            error: 'api_proxy_unavailable',
            message: 'API_BASE_URL could not be parsed',
            details: API_BASE_URL
        });
        return;
    }

    const targetPath = upstreamPath || req.originalUrl || req.url || '/api';
    const targetUrl = new URL(targetPath, parsedApiBaseUrl);
    const client = targetUrl.protocol === 'https:' ? https : http;

    const headers = {
        ...req.headers,
        host: targetUrl.host
    };

    delete headers['content-length'];

    const options = {
        hostname: targetUrl.hostname,
        port: targetUrl.port || (targetUrl.protocol === 'https:' ? 443 : 80),
        path: `${targetUrl.pathname}${targetUrl.search}`,
        method: req.method,
        headers
    };

    const proxyRequest = client.request(options, (proxyResponse) => {
        res.status(proxyResponse.statusCode || 500);
        Object.entries(proxyResponse.headers).forEach(([key, value]) => {
            if (typeof value !== 'undefined') {
                res.setHeader(key, value);
            }
        });
        proxyResponse.pipe(res);
    });

    proxyRequest.on('error', (error) => {
        console.error('API proxy error:', error.message);
        if (!res.headersSent) {
            res.status(502).json({
                error: 'api_proxy_error',
                message: error.message,
                target: targetUrl.href
            });
        } else {
            res.end();
        }
    });

    if (['GET', 'HEAD', 'OPTIONS'].includes(req.method)) {
        proxyRequest.end();
        return;
    }

    if (req.readable && typeof req.body === 'undefined') {
        req.pipe(proxyRequest);
        return;
    }

    if (req.body) {
        const payload = typeof req.body === 'string' ? req.body : JSON.stringify(req.body);
        proxyRequest.write(payload);
    }

    proxyRequest.end();
}

// Serve static files with proper MIME types
app.use('/components', express.static(path.join(__dirname, 'components'), {
    setHeaders: (res, path) => {
        if (path.endsWith('.js')) {
            res.setHeader('Content-Type', 'application/javascript');
        }
    }
}));

app.use('/styles', express.static(path.join(__dirname, 'styles')));
app.use(express.static(__dirname));

// Health check endpoint
app.get('/health', (req, res) => {
    res.json({ 
        status: 'healthy', 
        port: PORT,
        scenario: SCENARIO_NAME
    });
});

// API proxy configuration
app.get('/api/config', (req, res) => {
    res.json({ 
        apiUrl: parsedApiBaseUrl ? parsedApiBaseUrl.origin : null,
        proxyPath: '/api',
        uiPort: Number(PORT),
        resources: process.env.RESOURCE_PORTS ? JSON.parse(process.env.RESOURCE_PORTS) : {}
    });
});

// Proxy API traffic through the secure tunnel helper
app.use('/api', (req, res) => {
    proxyToApi(req, res, req.originalUrl || req.url);
});

// Route for dashboard (default)
app.get('/', (req, res) => {
    const dashboardFile = path.join(__dirname, 'dashboard.html');
    if (fs.existsSync(dashboardFile)) {
        res.sendFile(dashboardFile);
    } else {
        res.status(404).json({ error: 'Dashboard not found' });
    }
});

// Route for dashboard (explicit)
app.get('/dashboard.html', (req, res) => {
    const dashboardFile = path.join(__dirname, 'dashboard.html');
    if (fs.existsSync(dashboardFile)) {
        res.sendFile(dashboardFile);
    } else {
        res.status(404).json({ error: 'Dashboard not found' });
    }
});

// Route for prompt detail page
app.get('/prompt-detail.html', (req, res) => {
    const detailFile = path.join(__dirname, 'prompt-detail.html');
    if (fs.existsSync(detailFile)) {
        res.sendFile(detailFile);
    } else {
        res.status(404).json({ error: 'Prompt detail page not found' });
    }
});

// Fallback routing for SPAs - redirect to appropriate page
app.get('*', (req, res) => {
    // Check if it's a component or style request
    if (req.path.startsWith('/components/') || req.path.startsWith('/styles/')) {
        res.status(404).json({ error: 'File not found' });
        return;
    }

    // Default fallback to dashboard
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

const displayHost = process.env.UI_HOST || '127.0.0.1';

const server = app.listen(PORT, () => {
    console.log(`✅ UI server running on http://${displayHost}:${PORT}`);
    console.log(`📡 API target: ${parsedApiBaseUrl ? parsedApiBaseUrl.href : 'unconfigured'}`);
    console.log(`🏷️  Scenario: ${SCENARIO_NAME}`);
    console.log(`📁 Serving modular UI components from /components/`);
    console.log(`🎨 Serving styles from /styles/`);
});

// Graceful shutdown
process.on('SIGTERM', () => {
    server.close(() => {
        console.log('UI server stopped');
    });
});
