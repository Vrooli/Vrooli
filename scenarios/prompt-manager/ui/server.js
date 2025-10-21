import express from 'express';
import path from 'path';
import fs from 'fs';
import http from 'http';
import https from 'https';
import { fileURLToPath } from 'url';
import { dirname } from 'path';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

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

// Enable CORS without hard-coded loopback origins
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

const proxyToApi = (req, res, upstreamPath) => {
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
};

// Serve static files from dist directory
app.use(express.static(path.join(__dirname, 'dist')));

// Health check endpoint
app.get('/health', async (req, res) => {
    const timestamp = new Date().toISOString();

    // Check API connectivity
    let apiConnectivity = {
        status: 'unknown',
        last_check: timestamp
    };

    if (parsedApiBaseUrl) {
        try {
            const healthUrl = new URL('/health', parsedApiBaseUrl);
            const controller = new AbortController();
            const timeoutId = setTimeout(() => controller.abort(), 2000);

            const response = await fetch(healthUrl.href, {
                signal: controller.signal,
                headers: { 'User-Agent': 'prompt-manager-ui/health-check' }
            });
            clearTimeout(timeoutId);

            if (response.ok) {
                apiConnectivity = {
                    status: 'connected',
                    endpoint: healthUrl.href,
                    response_code: response.status,
                    last_check: timestamp
                };
            } else {
                apiConnectivity = {
                    status: 'error',
                    endpoint: healthUrl.href,
                    response_code: response.status,
                    error: {
                        code: 'API_ERROR',
                        message: `API returned ${response.status}`,
                        category: 'network',
                        retryable: true
                    },
                    last_check: timestamp
                };
            }
        } catch (error) {
            apiConnectivity = {
                status: 'disconnected',
                endpoint: parsedApiBaseUrl.href,
                error: {
                    code: error.name === 'AbortError' ? 'TIMEOUT' : 'CONNECTION_FAILED',
                    message: error.message,
                    category: 'network',
                    retryable: true
                },
                last_check: timestamp
            };
        }
    }

    res.json({
        status: 'healthy',
        service: 'prompt-manager-ui',
        timestamp,
        readiness: {
            ready: true,
            checks: {
                static_files: fs.existsSync(path.join(__dirname, 'dist')),
                api_config: parsedApiBaseUrl !== null
            }
        },
        api_connectivity: apiConnectivity,
        metadata: {
            port: PORT,
            scenario: SCENARIO_NAME,
            mode: 'production'
        }
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

app.use('/api', (req, res) => {
    proxyToApi(req, res, req.originalUrl || req.url);
});

// Catch all handler: send back React's index.html file for client-side routing
app.get('*', (req, res) => {
    const indexFile = path.join(__dirname, 'dist', 'index.html');
    if (fs.existsSync(indexFile)) {
        res.sendFile(indexFile);
    } else {
        res.status(404).json({ error: 'Build not found. Please run npm run build first.' });
    }
});

const displayHost = process.env.UI_HOST || '127.0.0.1';

const server = app.listen(PORT, () => {
    console.log(`✅ Prompt Manager UI server running on http://${displayHost}:${PORT}`);
    console.log(`📡 API target: ${parsedApiBaseUrl ? parsedApiBaseUrl.href : 'unconfigured'}`);
    console.log(`🏷️  Scenario: ${SCENARIO_NAME}`);
    console.log(`🎨 Serving modern React UI from /dist/`);
    console.log(`🚀 Production mode with Vite build`);
});

// Graceful shutdown
process.on('SIGTERM', () => {
    server.close(() => {
        console.log('UI server stopped');
    });
});

process.on('SIGINT', () => {
    server.close(() => {
        console.log('UI server stopped');
        process.exit(0);
    });
});
