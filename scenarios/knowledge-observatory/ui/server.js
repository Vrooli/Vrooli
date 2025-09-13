const express = require('express');
const path = require('path');
const http = require('http');
const fs = require('fs');

const app = express();
const PORT = process.env.UI_PORT || process.env.PORT;
const API_PORT = process.env.API_PORT;
const API_URL = process.env.API_URL || 'http://localhost:20260';

let startTime = Date.now();

app.use(express.static(__dirname));

app.get('/env.js', (req, res) => {
    res.type('application/javascript');
    res.send(`
        window.ENV = {
            API_URL: '${API_URL}',
            API_PORT: '${process.env.API_PORT || 20260}'
        };
    `);
});

// Comprehensive health check endpoint
app.get('/health', async (req, res) => {
    const health = {
        status: 'healthy',
        service: 'knowledge-observatory-ui',
        timestamp: new Date().toISOString(),
        readiness: true,
        api_connectivity: {
            connected: false,
            api_url: API_URL,
            last_check: new Date().toISOString(),
            latency_ms: null
        },
        static_files: {
            available: true,
            files: {}
        },
        dashboard_features: {
            knowledge_search: true,
            graph_visualization: true,
            quality_metrics: true,
            real_time_updates: true
        },
        metrics: {
            uptime_seconds: Math.floor((Date.now() - startTime) / 1000),
            node_version: process.version,
            memory_usage: process.memoryUsage()
        }
    };
    
    // Check API connectivity
    const apiHealthCheck = await checkAPIConnectivity();
    health.api_connectivity = { ...health.api_connectivity, ...apiHealthCheck };
    
    if (!apiHealthCheck.connected) {
        health.status = 'degraded';
        health.readiness = false;
        if (apiHealthCheck.error) {
            health.errors = health.errors || [];
            health.errors.push(apiHealthCheck.error);
        }
    }
    
    // Check static files
    const staticFiles = checkStaticFiles();
    health.static_files = staticFiles;
    
    if (!staticFiles.available) {
        health.status = 'degraded';
        if (staticFiles.error) {
            health.errors = health.errors || [];
            health.errors.push(staticFiles.error);
        }
    }
    
    const statusCode = health.status === 'healthy' ? 200 : 503;
    res.status(statusCode).json(health);
});

// Helper function to check API connectivity
async function checkAPIConnectivity() {
    const health = {
        connected: false,
        api_url: API_URL,
        last_check: new Date().toISOString(),
        latency_ms: null
    };
    
    const startTime = Date.now();
    
    return new Promise((resolve) => {
        const url = new URL('/health', API_URL);
        const options = {
            hostname: url.hostname,
            port: url.port || (url.protocol === 'https:' ? 443 : 80),
            path: url.pathname,
            method: 'GET',
            timeout: 5000
        };
        
        const req = http.request(options, (res) => {
            const endTime = Date.now();
            health.latency_ms = endTime - startTime;
            
            if (res.statusCode === 200) {
                health.connected = true;
                
                let body = '';
                res.on('data', chunk => body += chunk);
                res.on('end', () => {
                    try {
                        const apiHealth = JSON.parse(body);
                        health.api_status = apiHealth.status;
                        health.api_dependencies = apiHealth.dependencies;
                    } catch (e) {
                        // API returned non-JSON response
                        health.api_status = 'unknown';
                    }
                    resolve(health);
                });
            } else {
                health.error = {
                    code: 'API_UNHEALTHY',
                    message: `API returned status ${res.statusCode}`,
                    category: 'resource',
                    retryable: true
                };
                resolve(health);
            }
        });
        
        req.on('error', (err) => {
            health.error = {
                code: 'API_CONNECTION_FAILED',
                message: `Failed to connect to API: ${err.message}`,
                category: 'network',
                retryable: true
            };
            resolve(health);
        });
        
        req.on('timeout', () => {
            health.error = {
                code: 'API_CONNECTION_TIMEOUT',
                message: 'API health check timed out after 5 seconds',
                category: 'network',
                retryable: true
            };
            req.destroy();
            resolve(health);
        });
        
        req.end();
    });
}

// Helper function to check static files
function checkStaticFiles() {
    const health = {
        available: true,
        files: {}
    };
    
    const requiredFiles = {
        'index.html': path.join(__dirname, 'index.html'),
        'script.js': path.join(__dirname, 'script.js'),
        'package.json': path.join(__dirname, 'package.json')
    };
    
    for (const [name, filePath] of Object.entries(requiredFiles)) {
        try {
            if (fs.existsSync(filePath)) {
                const stats = fs.statSync(filePath);
                health.files[name] = {
                    exists: true,
                    size: stats.size,
                    modified: stats.mtime.toISOString()
                };
            } else {
                health.available = false;
                health.files[name] = { exists: false };
                health.error = {
                    code: 'STATIC_FILE_MISSING',
                    message: `Required file ${name} is missing`,
                    category: 'configuration',
                    retryable: false
                };
            }
        } catch (err) {
            health.available = false;
            health.files[name] = { 
                exists: false, 
                error: err.message 
            };
        }
    }
    
    return health;
}

app.get('*', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

const server = http.createServer(app);

server.listen(PORT, () => {
    console.log(`🔭 Knowledge Observatory UI running on port ${PORT}`);
    console.log(`📡 Connected to API at ${API_URL}`);
    console.log(`🌐 Access dashboard at http://localhost:${PORT}`);
});

process.on('SIGINT', () => {
    console.log('\n🛑 Shutting down Knowledge Observatory UI...');
    server.close(() => {
        process.exit(0);
    });
});