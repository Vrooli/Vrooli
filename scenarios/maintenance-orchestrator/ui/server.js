#!/usr/bin/env node

const express = require('express');
const path = require('path');
const fs = require('fs');
const http = require('http');

const app = express();
app.set('trust proxy', true);

const allowedHosts = new Set([
    'localhost',
    '127.0.0.1',
    '[::1]',
    '::1',
    'maintenance-orchestrator.itsagitime.com'
]);

app.use((req, res, next) => {
    const hostHeader = req.headers.host;
    if (!hostHeader) {
        return next();
    }

    const hostname = hostHeader.replace(/:\d+$/, '').toLowerCase();
    if (allowedHosts.has(hostname)) {
        return next();
    }

    res.status(403).send(
        `Blocked request. This host ("${hostname}") is not allowed. ` +
        `To allow it, add the hostname to the allowedHosts set in server.js.`
    );
});

const port = Number(process.env.UI_PORT) || 3251;
const API_PORT = process.env.API_PORT;
const PROXY_ROOT = '/proxy';
const PROXY_API_PREFIX = `${PROXY_ROOT}/api`;
const PROXY_API_BASE = `${PROXY_API_PREFIX}/v1`;
const DIRECT_API_URL = API_PORT ? `http://localhost:${API_PORT}` : null;
const DIRECT_API_BASE = DIRECT_API_URL ? `${DIRECT_API_URL}/api/v1` : null;

function buildOrigin(req) {
    const forwardedProto = req.headers['x-forwarded-proto'];
    const protocol = Array.isArray(forwardedProto) ? forwardedProto[0] : forwardedProto || req.protocol || 'http';
    const forwardedHost = req.headers['x-forwarded-host'];
    const host = Array.isArray(forwardedHost) ? forwardedHost[0] : forwardedHost || req.get('host');
    return host ? `${protocol}://${host}` : `${protocol}://localhost:${port}`;
}

function proxyToApi(req, res, upstreamPath) {
    if (!API_PORT) {
        res.status(503).json({
            error: 'API_PORT not configured',
            details: 'The maintenance orchestrator API is not running in this environment.'
        });
        return;
    }

    const pathWithQuery = upstreamPath || req.originalUrl.replace(/^\/proxy/, '');
    const finalPath = pathWithQuery.startsWith('/') ? pathWithQuery : `/${pathWithQuery}`;

    const options = {
        hostname: '127.0.0.1',
        port: Number(API_PORT),
        path: finalPath,
        method: req.method,
        headers: {
            ...req.headers,
            host: `localhost:${API_PORT}`
        }
    };

    const proxyReq = http.request(options, (proxyRes) => {
        res.status(proxyRes.statusCode || 500);
        Object.entries(proxyRes.headers).forEach(([key, value]) => {
            if (value !== undefined) {
                res.setHeader(key, value);
            }
        });
        proxyRes.pipe(res);
    });

    proxyReq.on('error', (error) => {
        console.error('[maintenance-orchestrator-ui] API proxy error:', error.message);
        res.status(502).json({
            error: 'API proxy failed',
            details: error.message,
            target: `http://localhost:${API_PORT}${finalPath}`
        });
    });

    if (req.method === 'GET' || req.method === 'HEAD') {
        proxyReq.end();
    } else {
        req.pipe(proxyReq);
    }
}

app.use(PROXY_ROOT, (req, res) => {
    const upstreamPath = req.originalUrl.replace(/^\/proxy/, '');
    proxyToApi(req, res, upstreamPath);
});

// Serve static files from current directory
app.use(express.static(__dirname));

// Serve main dashboard
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

// API configuration endpoint for frontend
app.get('/config', (req, res) => {
    const origin = buildOrigin(req);
    res.json({
        service: 'maintenance-orchestrator',
        uiPort: port,
        origin,
        apiUrl: DIRECT_API_URL,
        apiBase: DIRECT_API_BASE,
        proxyApiUrl: PROXY_ROOT,
        proxyApiBase: PROXY_API_BASE,
        displayApiUrl: DIRECT_API_URL || `${origin}${PROXY_ROOT}`
    });
});

// Legacy endpoint for backward compatibility
app.get('/api/config', (req, res) => {
    res.json({
        service: 'maintenance-orchestrator',
        ui_port: port,
        api_base: DIRECT_API_BASE || PROXY_API_BASE,
        proxy_api_base: PROXY_API_BASE,
        direct_api_base: DIRECT_API_BASE
    });
});

// Health check endpoint
app.get('/health', async (req, res) => {
    const healthResponse = {
        status: 'healthy',
        service: 'maintenance-orchestrator-ui',
        timestamp: new Date().toISOString(),
        readiness: true,
        api_connectivity: {
            connected: false,
            api_url: `http://localhost:${process.env.API_PORT}`,
            last_check: new Date().toISOString(),
            error: null,
            latency_ms: null
        }
    };
    
    // Test API connectivity
    const apiPort = process.env.API_PORT;
    if (apiPort) {
        const startTime = Date.now();
        
        try {
            await new Promise((resolve, reject) => {
                const options = {
                    hostname: 'localhost',
                    port: apiPort,
                    path: '/health',
                    method: 'GET',
                    timeout: 5000,
                    headers: {
                        'Accept': 'application/json'
                    }
                };
                
                const apiRequest = http.request(options, (apiRes) => {
                    const endTime = Date.now();
                    healthResponse.api_connectivity.latency_ms = endTime - startTime;
                    
                    if (apiRes.statusCode >= 200 && apiRes.statusCode < 300) {
                        healthResponse.api_connectivity.connected = true;
                        healthResponse.api_connectivity.error = null;
                    } else {
                        healthResponse.api_connectivity.connected = false;
                        healthResponse.api_connectivity.error = {
                            code: `HTTP_${apiRes.statusCode}`,
                            message: `API returned status ${apiRes.statusCode}: ${apiRes.statusMessage}`,
                            category: 'network',
                            retryable: apiRes.statusCode >= 500 && apiRes.statusCode < 600
                        };
                        healthResponse.status = 'degraded';
                    }
                    resolve();
                });
                
                apiRequest.on('error', (error) => {
                    const endTime = Date.now();
                    healthResponse.api_connectivity.latency_ms = endTime - startTime;
                    healthResponse.api_connectivity.connected = false;
                    
                    let errorCode = 'CONNECTION_FAILED';
                    let category = 'network';
                    let retryable = true;
                    
                    if (error.code === 'ECONNREFUSED') {
                        errorCode = 'CONNECTION_REFUSED';
                    } else if (error.code === 'ENOTFOUND') {
                        errorCode = 'HOST_NOT_FOUND';
                        category = 'configuration';
                    } else if (error.code === 'ETIMEOUT') {
                        errorCode = 'TIMEOUT';
                    }
                    
                    healthResponse.api_connectivity.error = {
                        code: errorCode,
                        message: `Failed to connect to API: ${error.message}`,
                        category: category,
                        retryable: retryable,
                        details: {
                            error_code: error.code
                        }
                    };
                    healthResponse.status = 'unhealthy';
                    resolve();
                });
                
                apiRequest.on('timeout', () => {
                    const endTime = Date.now();
                    healthResponse.api_connectivity.latency_ms = endTime - startTime;
                    healthResponse.api_connectivity.connected = false;
                    healthResponse.api_connectivity.error = {
                        code: 'TIMEOUT',
                        message: 'API health check timed out after 5 seconds',
                        category: 'network',
                        retryable: true
                    };
                    healthResponse.status = 'unhealthy';
                    apiRequest.destroy();
                    resolve();
                });
                
                apiRequest.end();
            });
        } catch (error) {
            const endTime = Date.now();
            healthResponse.api_connectivity.latency_ms = endTime - startTime;
            healthResponse.api_connectivity.connected = false;
            healthResponse.api_connectivity.error = {
                code: 'UNEXPECTED_ERROR',
                message: `Unexpected error: ${error.message}`,
                category: 'internal',
                retryable: true
            };
            healthResponse.status = 'unhealthy';
        }
    } else {
        // No API_PORT configured
        healthResponse.api_connectivity.connected = false;
        healthResponse.api_connectivity.error = {
            code: 'MISSING_CONFIG',
            message: 'API_PORT environment variable not configured',
            category: 'configuration',
            retryable: false
        };
        healthResponse.status = 'degraded';
    }
    
    // Check static file availability
    healthResponse.static_files = {
        available: false,
        error: null
    };
    
    try {
        const indexPath = path.join(__dirname, 'index.html');
        const dashboardPath = path.join(__dirname, 'dashboard.html');
        
        if (fs.existsSync(indexPath) && fs.existsSync(dashboardPath)) {
            healthResponse.static_files.available = true;
            healthResponse.static_files.files = {
                index_html: true,
                dashboard_html: true
            };
        } else {
            healthResponse.static_files.error = {
                code: 'STATIC_FILES_MISSING',
                message: 'Required HTML files not found',
                category: 'resource',
                retryable: false
            };
            healthResponse.static_files.files = {
                index_html: fs.existsSync(indexPath),
                dashboard_html: fs.existsSync(dashboardPath)
            };
            if (healthResponse.status === 'healthy') {
                healthResponse.status = 'degraded';
            }
        }
    } catch (error) {
        healthResponse.static_files.error = {
            code: 'STATIC_FILES_CHECK_FAILED',
            message: `Cannot check static files: ${error.message}`,
            category: 'resource',
            retryable: false
        };
    }
    
    // Add server metrics
    healthResponse.metrics = {
        uptime_seconds: process.uptime(),
        memory_usage_mb: process.memoryUsage().heapUsed / 1024 / 1024,
        port: port
    };
    
    res.json(healthResponse);
});

// 404 handler
app.use((req, res) => {
    res.status(404).json({ error: 'Not found' });
});

// Error handler
app.use((err, req, res, next) => {
    console.error('Server error:', err);
    res.status(500).json({ error: 'Internal server error' });
});

// Start server
app.listen(port, () => {
    console.log(`🎛️ Maintenance Orchestrator Dashboard running on http://localhost:${port}`);
    console.log(`📊 Mission Control Interface ready`);
});

// Graceful shutdown
process.on('SIGTERM', () => {
    console.log('🛑 Shutting down Maintenance Orchestrator Dashboard...');
    process.exit(0);
});

process.on('SIGINT', () => {
    console.log('🛑 Shutting down Maintenance Orchestrator Dashboard...');
    process.exit(0);
});
