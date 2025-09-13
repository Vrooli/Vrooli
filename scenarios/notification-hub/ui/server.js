const express = require('express');
const path = require('path');
const fs = require('fs');
const http = require('http');
const { createProxyMiddleware } = require('http-proxy-middleware');

const app = express();
const PORT = process.env.UI_PORT || process.env.PORT;
const API_PORT = process.env.API_PORT;
const startTime = new Date();

// Serve static files
app.use(express.static(__dirname));

// Health check endpoint
app.get('/health', async (req, res) => {
    let overallStatus = 'healthy';
    let errors = [];
    let readiness = true;

    // Schema-compliant health response
    const healthResponse = {
        status: overallStatus,
        service: 'notification-hub-ui',
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

    // Check API proxy connectivity (required by schema)
    const apiHealth = await checkAPIConnectivity();
    healthResponse.api_connectivity = {
        connected: apiHealth.status === 'healthy',
        api_url: apiHealth.checks.api_url || `http://localhost:${API_PORT}`,
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

    // Check Express server functionality
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
        api_port: API_PORT
    };

    // Return appropriate HTTP status
    const statusCode = overallStatus === 'unhealthy' ? 503 : 200;
    res.status(statusCode).json(healthResponse);
});

// API proxy to Go backend
app.use('/api', createProxyMiddleware({
    target: `http://localhost:${API_PORT}`,
    changeOrigin: true,
    onError: (err, req, res) => {
        console.error('API proxy error:', err);
        res.status(500).json({ error: 'API connection error' });
    }
}));

// Serve main dashboard
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

// Serve profile management page
app.get('/profiles', (req, res) => {
    res.sendFile(path.join(__dirname, 'profiles.html'));
});

// Serve analytics page
app.get('/analytics', (req, res) => {
    res.sendFile(path.join(__dirname, 'analytics.html'));
});

// Health check helper functions
function checkStaticFiles() {
    const health = {
        status: 'healthy',
        checks: {}
    };

    // Check for essential static files
    const requiredFiles = ['index.html', 'profiles.html', 'analytics.html'];
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

    if (!API_PORT) {
        health.status = 'unhealthy';
        health.error = {
            code: 'API_PORT_NOT_CONFIGURED',
            message: 'API_PORT environment variable not set',
            category: 'configuration',
            retryable: false
        };
        return health;
    }

    // Test API connectivity with health endpoint
    return new Promise((resolve) => {
        const startTime = Date.now();
        const options = {
            hostname: 'localhost',
            port: API_PORT,
            path: '/health',
            method: 'GET',
            timeout: 5000
        };

        const req = http.request(options, (res) => {
            const endTime = Date.now();
            health.latency_ms = endTime - startTime;
            health.checks.api_reachable = true;
            health.checks.api_status_code = res.statusCode;
            health.checks.api_url = `http://localhost:${API_PORT}`;

            if (res.statusCode >= 200 && res.statusCode < 300) {
                health.status = 'healthy';
            } else if (res.statusCode >= 500) {
                health.status = 'degraded';
                health.error = {
                    code: 'API_SERVER_ERROR',
                    message: `API returned status ${res.statusCode}`,
                    category: 'network',
                    retryable: true
                };
            } else {
                health.status = 'degraded';
                health.error = {
                    code: 'API_UNEXPECTED_STATUS',
                    message: `API returned status ${res.statusCode}`,
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
                message: `Failed to connect to API: ${error.message}`,
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
                message: 'API health check timed out after 5 seconds',
                category: 'network',
                retryable: true
            };
            resolve(health);
        });

        req.end();
    });
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
    health.checks.static_middleware = true;
    health.checks.proxy_middleware = true;

    // Check memory usage (basic health indicator)
    const memUsage = process.memoryUsage();
    const memUsedMB = Math.round(memUsage.heapUsed / 1024 / 1024);
    health.checks.memory_usage_mb = memUsedMB;

    // Warning if memory usage is high (>200MB for a simple UI server)
    if (memUsedMB > 200) {
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

app.listen(PORT, () => {
    console.log(`🔔 Notification Hub UI running on http://localhost:${PORT}`);
    console.log(`📊 Dashboard: http://localhost:${PORT}`);
    console.log(`⚙️ Profiles: http://localhost:${PORT}/profiles`);
    console.log(`📈 Analytics: http://localhost:${PORT}/analytics`);
    console.log(`🔗 API: http://localhost:${API_PORT}`);
});
