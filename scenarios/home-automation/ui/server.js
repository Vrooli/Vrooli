const express = require('express');
const cors = require('cors');
const path = require('path');
const fs = require('fs');
const WebSocket = require('ws');
const http = require('http');
const https = require('https');

const app = express();
const server = http.createServer(app);
const wss = new WebSocket.Server({ server });

function requireEnv(name) {
    const value = process.env[name];
    if (!value) {
        console.error(`[ui] Missing required environment variable: ${name}`);
        console.error('     Start the scenario through the Vrooli lifecycle to inject configuration.');
        process.exit(1);
    }
    return value;
}

function parsePort(value, label) {
    const parsed = Number.parseInt(value, 10);
    if (Number.isNaN(parsed) || parsed <= 0) {
        console.error(`[ui] ${label} must be a positive integer (received: ${value})`);
        process.exit(1);
    }
    return parsed;
}

const PORT = parsePort(requireEnv('UI_PORT'), 'UI_PORT');

const rawApiBaseUrl = process.env.API_BASE_URL;
const rawApiPort = process.env.API_PORT;

let resolvedApiUrl;
if (rawApiBaseUrl) {
    try {
        resolvedApiUrl = new URL(rawApiBaseUrl);
    } catch (error) {
        console.error(`[ui] Invalid API_BASE_URL value: ${rawApiBaseUrl}`);
        console.error(error.message);
        process.exit(1);
    }
} else if (rawApiPort) {
    const parsedPort = parsePort(rawApiPort, 'API_PORT');
    resolvedApiUrl = new URL(`http://127.0.0.1:${parsedPort}`);
} else {
    console.error('[ui] API configuration missing. Provide API_BASE_URL or API_PORT.');
    process.exit(1);
}

const API_PROTOCOL = resolvedApiUrl.protocol;
const API_HOSTNAME = resolvedApiUrl.hostname;
const API_HOST = resolvedApiUrl.host;
const API_PORT = resolvedApiUrl.port
    ? parsePort(resolvedApiUrl.port, 'API_BASE_URL port')
    : (API_PROTOCOL === 'https:' ? 443 : 80);
const API_PATH_PREFIX = resolvedApiUrl.pathname && resolvedApiUrl.pathname !== '/'
    ? resolvedApiUrl.pathname.replace(/\/$/, '')
    : '';
const API_ORIGIN = `${API_PROTOCOL}//${API_HOST}`;
const FULL_API_BASE_URL = API_PATH_PREFIX ? `${API_ORIGIN}${API_PATH_PREFIX}` : API_ORIGIN;
const API_TRANSPORT = API_PROTOCOL === 'https:' ? https : http;

const startTime = new Date();

function buildUpstreamPath(requestPath = '/') {
    const [pathPart, query = ''] = (requestPath || '/').split('?');
    const normalizedPath = pathPart.startsWith('/') ? pathPart : `/${pathPart}`;

    let upstreamPath = normalizedPath;
    if (API_PATH_PREFIX && !normalizedPath.startsWith(API_PATH_PREFIX)) {
        upstreamPath = `${API_PATH_PREFIX}${normalizedPath}`;
    }

    return query ? `${upstreamPath}?${query}` : upstreamPath;
}

function proxyToApi(req, res, upstreamOverride) {
    const upstreamPath = buildUpstreamPath(upstreamOverride || req.url);

    const options = {
        hostname: API_HOSTNAME,
        port: API_PORT,
        path: upstreamPath,
        method: req.method,
        headers: {
            ...req.headers,
            host: API_HOST
        }
    };

    if (process.env.NODE_ENV !== 'production') {
        const originalPath = req.originalUrl || req.url;
        console.debug(`[ui] proxy ${req.method} ${originalPath} -> ${FULL_API_BASE_URL}${upstreamPath}`);
    }

    const proxyReq = API_TRANSPORT.request(options, (proxyRes) => {
        res.status(proxyRes.statusCode || 502);
        Object.entries(proxyRes.headers).forEach(([key, value]) => {
            if (typeof value !== 'undefined') {
                res.setHeader(key, value);
            }
        });
        proxyRes.pipe(res);
    });

    proxyReq.on('error', (error) => {
        console.error('[ui] API proxy error:', error.message);
        if (!res.headersSent) {
            res.status(502).json({
                error: 'API unavailable',
                message: 'Home automation API is not responding',
                target: `${FULL_API_BASE_URL}${upstreamPath}`
            });
        } else {
            res.end();
        }
    });

    if (!['GET', 'HEAD', 'OPTIONS'].includes(req.method)) {
        req.pipe(proxyReq);
    } else {
        proxyReq.end();
    }
}

// Middleware
app.use(cors());
app.use(express.static(path.join(__dirname)));

// Serve the main dashboard
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

// Health check endpoint
app.get('/health', async (req, res) => {
    let overallStatus = 'healthy';
    let errors = [];
    let readiness = true;

    // Schema-compliant health response
    const healthResponse = {
        status: overallStatus,
        service: 'home-automation-ui',
        timestamp: new Date().toISOString(),
        readiness: true,
        dependencies: {}
    };

    // Check static file availability
    const staticFileHealth = checkStaticFiles();
    healthResponse.dependencies.static_files = staticFileHealth;
    if (staticFileHealth.status !== 'healthy') {
        overallStatus = 'degraded';
        if (staticFileHealth.status === 'unhealthy') {
            readiness = false;
        }
        if (staticFileHealth.error) {
            errors.push(staticFileHealth.error);
        }
    }

    // Check API connectivity (required by schema)
    const apiHealth = await checkAPIConnectivity();
    healthResponse.api_connectivity = {
        connected: apiHealth.status === 'healthy',
        api_url: FULL_API_BASE_URL,
        last_check: new Date().toISOString(),
        error: apiHealth.error || null,
        latency_ms: apiHealth.latency_ms || null
    };
    
    if (apiHealth.status !== 'healthy') {
        if (overallStatus !== 'unhealthy') {
            overallStatus = 'degraded';
        }
        if (apiHealth.error) {
            errors.push(apiHealth.error);
        }
    }

    // Check WebSocket server functionality
    const wsHealth = checkWebSocketServer();
    healthResponse.dependencies.websocket_server = wsHealth;
    if (wsHealth.status !== 'healthy') {
        if (overallStatus !== 'unhealthy') {
            overallStatus = 'degraded';
        }
        if (wsHealth.error) {
            errors.push(wsHealth.error);
        }
    }

    // Check Express server health
    const serverHealth = checkServerHealth();
    healthResponse.dependencies.server_health = serverHealth;
    if (serverHealth.status !== 'healthy') {
        if (overallStatus !== 'unhealthy') {
            overallStatus = 'degraded';
        }
        if (serverHealth.error) {
            errors.push(serverHealth.error);
        }
    }

    // Update final status
    healthResponse.status = overallStatus;
    healthResponse.readiness = readiness;

    // Add errors if any
    if (errors.length > 0) {
        healthResponse.errors = errors;
    }

    // Add metrics
    const uptime = Math.floor((new Date() - startTime) / 1000);
    const totalDeps = Object.keys(healthResponse.dependencies).length;
    const healthyDeps = countHealthyDependencies(healthResponse.dependencies);
    
    healthResponse.metrics = {
        total_dependencies: totalDeps,
        healthy_dependencies: healthyDeps,
        uptime_seconds: uptime,
        port: PORT,
        websocket_connections: wss.clients.size
    };

    // Return appropriate HTTP status
    const statusCode = overallStatus === 'unhealthy' ? 503 : 200;
    res.status(statusCode).json(healthResponse);
});

// Proxy API requests to backend
app.use('/api', (req, res) => {
    proxyToApi(req, res, req.originalUrl || req.url);
});

// WebSocket connection for real-time device updates
wss.on('connection', (ws) => {
    console.log('Client connected to WebSocket');
    
    // Send initial connection confirmation
    ws.send(JSON.stringify({
        type: 'connection',
        status: 'connected',
        timestamp: new Date().toISOString()
    }));
    
    // Handle client messages
    ws.on('message', (message) => {
        try {
            const data = JSON.parse(message);
            console.log('Received WebSocket message:', data);
            
            // Echo back for now - implementation will integrate with API WebSocket
            ws.send(JSON.stringify({
                type: 'echo',
                original: data,
                timestamp: new Date().toISOString()
            }));
        } catch (error) {
            console.error('WebSocket message error:', error);
        }
    });
    
    ws.on('close', () => {
        console.log('Client disconnected from WebSocket');
    });
    
    ws.on('error', (error) => {
        console.error('WebSocket error:', error);
    });
});

// Health check helper functions
function checkStaticFiles() {
    const health = {
        status: 'healthy',
        checks: {}
    };

    // Check for essential static files
    const requiredFiles = ['index.html', 'app.js', 'styles.css'];
    const missingFiles = [];

    requiredFiles.forEach(fileName => {
        const filePath = path.join(__dirname, fileName);
        try {
            if (fs.existsSync(filePath)) {
                const stats = fs.statSync(filePath);
                health.checks[fileName.replace('.', '_')] = {
                    exists: true,
                    size: stats.size,
                    modified: stats.mtime.toISOString()
                };
            } else {
                missingFiles.push(fileName);
                health.checks[fileName.replace('.', '_')] = {
                    exists: false
                };
            }
        } catch (error) {
            health.status = 'degraded';
            health.checks[fileName.replace('.', '_')] = {
                exists: false,
                error: error.message
            };
        }
    });

    if (missingFiles.length > 0) {
        health.status = 'degraded';
        health.error = {
            code: 'STATIC_FILES_MISSING',
            message: `Missing static files: ${missingFiles.join(', ')}`,
            category: 'resource',
            retryable: false
        };
    }

    return health;
}

async function checkAPIConnectivity() {
    const health = {
        status: 'healthy',
        checks: {},
        latency_ms: null
    };

    const upstreamPath = buildUpstreamPath('/api/v1/health');
    const startTime = Date.now();

    return new Promise((resolve) => {
        const options = {
            hostname: API_HOSTNAME,
            port: API_PORT,
            path: upstreamPath,
            method: 'GET',
            headers: {
                host: API_HOST
            }
        };

        let completed = false;
        const finalize = (updater) => {
            if (completed) {
                return;
            }
            completed = true;
            if (typeof updater === 'function') {
                updater();
            }
            resolve(health);
        };

        const upstreamReq = API_TRANSPORT.request(options, (upstreamRes) => {
            finalize(() => {
                const endTime = Date.now();
                health.latency_ms = endTime - startTime;
                health.checks.api_reachable = true;
                health.checks.api_status_code = upstreamRes.statusCode;

                if (upstreamRes.statusCode >= 200 && upstreamRes.statusCode < 300) {
                    health.status = 'healthy';
                } else {
                    health.status = 'degraded';
                    const code = upstreamRes.statusCode >= 500 ? 'API_SERVER_ERROR' : 'API_UNEXPECTED_STATUS';
                    health.error = {
                        code,
                        message: `Home automation API returned status ${upstreamRes.statusCode}`,
                        category: 'network',
                        retryable: true
                    };
                }
            });
        });

        upstreamReq.on('error', (error) => {
            finalize(() => {
                health.status = 'unhealthy';
                health.checks.api_reachable = false;
                health.error = {
                    code: 'API_CONNECTION_FAILED',
                    message: `Failed to connect to home automation API: ${error.message}`,
                    category: 'network',
                    retryable: true
                };
            });
        });

        upstreamReq.setTimeout(5000, () => {
            upstreamReq.destroy(new Error('Request timed out'));
            finalize(() => {
                health.status = 'unhealthy';
                health.checks.api_reachable = false;
                health.error = {
                    code: 'API_TIMEOUT',
                    message: `Home automation API health check timed out after 5 seconds (${FULL_API_BASE_URL}${upstreamPath})`,
                    category: 'network',
                    retryable: true
                };
            });
        });

        upstreamReq.end();
    });
}

function checkWebSocketServer() {
    const health = {
        status: 'healthy',
        checks: {}
    };

    if (!wss) {
        health.status = 'unhealthy';
        health.error = {
            code: 'WEBSOCKET_SERVER_NOT_INITIALIZED',
            message: 'WebSocket server not initialized',
            category: 'internal',
            retryable: false
        };
        return health;
    }

    health.checks.server_initialized = true;
    health.checks.active_connections = wss.clients.size;

    // WebSocket.Server doesn't have readyState - it's always ready once initialized
    // Individual clients have readyState, but the server itself is either initialized or not
    health.checks.server_ready = true;

    return health;
}

function checkServerHealth() {
    const health = {
        status: 'healthy',
        checks: {}
    };

    // Check if server is properly configured
    if (!PORT) {
        health.status = 'unhealthy';
        health.error = {
            code: 'PORT_NOT_CONFIGURED',
            message: 'Server port not configured',
            category: 'configuration',
            retryable: false
        };
        return health;
    }

    health.checks.port_configured = true;
    health.checks.express_middleware = true;
    health.checks.cors_enabled = true;

    // Check memory usage (basic health indicator)
    const memUsage = process.memoryUsage();
    const memUsedMB = Math.round(memUsage.heapUsed / 1024 / 1024);
    health.checks.memory_usage_mb = memUsedMB;

    // Warning if memory usage is high (>300MB for a UI server with WebSocket)
    if (memUsedMB > 300) {
        health.status = 'degraded';
        health.error = {
            code: 'HIGH_MEMORY_USAGE',
            message: `High memory usage: ${memUsedMB}MB`,
            category: 'resource',
            retryable: true
        };
    }

    return health;
}

function countHealthyDependencies(deps) {
    let count = 0;
    Object.values(deps).forEach(dep => {
        if (dep.status === 'healthy') {
            count++;
        }
    });
    return count;
}

server.listen(PORT, () => {
    console.log(`[ui] Home Automation dashboard available at http://localhost:${PORT}`);
    console.log(`[ui] Proxying API requests to ${FULL_API_BASE_URL}`);
    console.log(`[ui] WebSocket endpoint ready at ws://localhost:${PORT}`);
});
