const express = require('express');
const path = require('path');
const fs = require('fs');
const http = require('http');

const app = express();
const PORT = process.env.UI_PORT || process.env.PORT;
const API_PORT = process.env.API_PORT;
const API_HOST = process.env.ORCHESTRATOR_HOST;

// Validate required environment variables
if (!PORT) {
    console.error('❌ UI_PORT or PORT environment variable is required');
    process.exit(1);
}

if (!API_PORT) {
    console.error('❌ API_PORT environment variable is required');
    process.exit(1);
}

if (!API_HOST) {
    console.error('❌ ORCHESTRATOR_HOST environment variable is required');
    process.exit(1);
}

// Enable CORS for API communication
app.use((req, res, next) => {
    res.header('Access-Control-Allow-Origin', '*');
    res.header('Access-Control-Allow-Methods', 'GET, POST, PUT, PATCH, DELETE, OPTIONS');
    res.header('Access-Control-Allow-Headers', 'Content-Type');
    next();
});

// Serve static files
app.use(express.static(__dirname));

// API proxy configuration endpoint
app.get('/api/config', (req, res) => {
    res.json({ 
        apiUrl: `http://${API_HOST}:${API_PORT}`,
        uiPort: PORT,
        host: API_HOST,
        version: '1.0.0',
        service: 'ai-model-orchestra-controller'
    });
});

// Comprehensive health check endpoint
app.get('/health', async (req, res) => {
    const healthResponse = {
        status: 'healthy',
        service: 'ai-model-orchestra-controller-ui',
        timestamp: new Date().toISOString(),
        readiness: true,
        api_connectivity: {
            connected: false,
            api_url: `http://${API_HOST}:${API_PORT}`,
            last_check: new Date().toISOString(),
            error: null,
            latency_ms: null
        }
    };
    
    // Test API connectivity
    const startTime = Date.now();
    
    try {
        await new Promise((resolve, reject) => {
            const options = {
                hostname: API_HOST,
                port: API_PORT,
                path: '/health',
                method: 'GET',
                timeout: 5000,
                headers: {
                    'Accept': 'application/json'
                }
            };
            
            const req = http.request(options, (res) => {
                const endTime = Date.now();
                healthResponse.api_connectivity.latency_ms = endTime - startTime;
                
                if (res.statusCode >= 200 && res.statusCode < 300) {
                    healthResponse.api_connectivity.connected = true;
                    healthResponse.api_connectivity.error = null;
                } else {
                    healthResponse.api_connectivity.connected = false;
                    healthResponse.api_connectivity.error = {
                        code: `HTTP_${res.statusCode}`,
                        message: `API returned status ${res.statusCode}: ${res.statusMessage}`,
                        category: 'network',
                        retryable: res.statusCode >= 500 && res.statusCode < 600
                    };
                    healthResponse.status = 'degraded';
                }
                resolve();
            });
            
            req.on('error', (error) => {
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
                } else if (error.code === 'ETIMEOUT' || error.code === 'ETIMEDOUT') {
                    errorCode = 'TIMEOUT';
                }
                
                healthResponse.api_connectivity.error = {
                    code: errorCode,
                    message: `Failed to connect to API: ${error.message}`,
                    category: category,
                    retryable: retryable,
                    details: {
                        error_code: error.code,
                        hostname: API_HOST,
                        port: API_PORT
                    }
                };
                healthResponse.status = 'unhealthy';
                resolve();
            });
            
            req.on('timeout', () => {
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
                req.destroy();
                resolve();
            });
            
            req.end();
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
    
    res.json(healthResponse);
});

// Serve main dashboard
app.get('/', (req, res) => {
    const dashboardFile = path.join(__dirname, 'dashboard.html');
    if (fs.existsSync(dashboardFile)) {
        res.sendFile(dashboardFile);
    } else {
        res.status(404).json({ 
            error: 'Dashboard file not found',
            path: dashboardFile 
        });
    }
});

// Serve dashboard at explicit path
app.get('/dashboard', (req, res) => {
    const dashboardFile = path.join(__dirname, 'dashboard.html');
    if (fs.existsSync(dashboardFile)) {
        res.sendFile(dashboardFile);
    } else {
        res.status(404).json({ 
            error: 'Dashboard file not found',
            path: dashboardFile 
        });
    }
});

// Catch-all for other routes
app.get('*', (req, res) => {
    res.status(404).json({ 
        error: 'Not found',
        path: req.path 
    });
});

const server = app.listen(PORT, () => {
    console.log(`🚀 AI Model Orchestra Controller UI running on port ${PORT}`);
    console.log(`📡 API endpoint configured: ${API_HOST}:${API_PORT}`);
    console.log(`🎛️  Service: ai-model-orchestra-controller v1.0.0`);
});

// Graceful shutdown
process.on('SIGTERM', () => {
    console.log('SIGTERM received, shutting down gracefully...');
    server.close(() => {
        console.log('✅ UI server stopped');
        process.exit(0);
    });
});

process.on('SIGINT', () => {
    console.log('SIGINT received, shutting down gracefully...');
    server.close(() => {
        console.log('✅ UI server stopped');
        process.exit(0);
    });
});