const express = require('express');
const cors = require('cors');
const path = require('path');
const fs = require('fs');
const WebSocket = require('ws');
const http = require('http');

const app = express();
const server = http.createServer(app);
const wss = new WebSocket.Server({ server });

const PORT = process.env.HOME_AUTOMATION_UI_PORT || 3351;
const API_BASE_URL = process.env.API_BASE_URL || `http://localhost:${process.env.HOME_AUTOMATION_API_PORT || 3350}`;
const startTime = new Date();

// Middleware
app.use(cors());
app.use(express.json());
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
        api_url: API_BASE_URL,
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
app.use('/api', async (req, res) => {
    try {
        const apiUrl = `${API_BASE_URL}${req.path}`;
        const response = await fetch(apiUrl, {
            method: req.method,
            headers: {
                'Content-Type': 'application/json',
                'Authorization': req.headers.authorization || ''
            },
            body: req.method !== 'GET' ? JSON.stringify(req.body) : undefined
        });
        
        const data = await response.json();
        res.status(response.status).json(data);
    } catch (error) {
        console.error('API proxy error:', error);
        res.status(500).json({ 
            error: 'API unavailable',
            message: 'Home automation API is not responding'
        });
    }
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

    const apiUrl = new URL('/api/v1/health', API_BASE_URL);
    const startTime = Date.now();

    return new Promise((resolve) => {
        const options = {
            hostname: apiUrl.hostname,
            port: apiUrl.port,
            path: apiUrl.pathname,
            method: 'GET',
            timeout: 5000
        };

        const req = http.request(options, (res) => {
            const endTime = Date.now();
            health.latency_ms = endTime - startTime;
            health.checks.api_reachable = true;
            health.checks.api_status_code = res.statusCode;

            if (res.statusCode >= 200 && res.statusCode < 300) {
                health.status = 'healthy';
            } else if (res.statusCode >= 500) {
                health.status = 'degraded';
                health.error = {
                    code: 'API_SERVER_ERROR',
                    message: `Home automation API returned status ${res.statusCode}`,
                    category: 'network',
                    retryable: true
                };
            } else {
                health.status = 'degraded';
                health.error = {
                    code: 'API_UNEXPECTED_STATUS',
                    message: `Home automation API returned status ${res.statusCode}`,
                    category: 'network',
                    retryable: true
                };
            }
            resolve(health);
        });

        req.on('error', (error) => {
            health.status = 'unhealthy';
            health.checks.api_reachable = false;
            health.error = {
                code: 'API_CONNECTION_FAILED',
                message: `Failed to connect to home automation API: ${error.message}`,
                category: 'network',
                retryable: true
            };
            resolve(health);
        });

        req.on('timeout', () => {
            req.abort();
            health.status = 'unhealthy';
            health.checks.api_reachable = false;
            health.error = {
                code: 'API_TIMEOUT',
                message: 'Home automation API health check timed out after 5 seconds',
                category: 'network',
                retryable: true
            };
            resolve(health);
        });

        req.end();
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

    // Check if WebSocket server is listening
    if (wss.readyState !== WebSocket.CONNECTING && wss.readyState !== WebSocket.OPEN) {
        health.status = 'degraded';
        health.error = {
            code: 'WEBSOCKET_SERVER_NOT_READY',
            message: `WebSocket server not ready, state: ${wss.readyState}`,
            category: 'internal',
            retryable: true
        };
    } else {
        health.checks.server_ready = true;
    }

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
    console.log(`🏠 Home Automation UI Server running at http://localhost:${PORT}`);
    console.log(`🔗 API Backend: ${API_BASE_URL}`);
    console.log(`📱 Dashboard: http://localhost:${PORT}`);
    console.log(`🔌 WebSocket: ws://localhost:${PORT}`);
});