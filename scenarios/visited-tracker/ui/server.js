const express = require('express');
const path = require('path');
const http = require('http');

const app = express();
const PORT = process.env.UI_PORT || process.env.PORT;
const API_PORT = process.env.API_PORT;

// Serve static files from current directory
app.use(express.static(__dirname));

// Provide config to frontend
app.get('/config', (req, res) => {
    res.json({
        apiUrl: `http://localhost:${API_PORT}`,
        version: '1.0.0',
        service: 'visited-tracker'
    });
});

// Serve main page
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

// Health check
app.get('/health', async (req, res) => {
    const healthResponse = {
        status: 'healthy',
        service: 'visited-tracker-ui',
        timestamp: new Date().toISOString(),
        readiness: true, // UI is ready to serve requests
        api_connectivity: {
            connected: false,
            api_url: `http://localhost:${API_PORT}`,
            last_check: new Date().toISOString(),
            error: null,
            latency_ms: null
        }
    };
    
    // Test API connectivity
    if (API_PORT) {
        const startTime = Date.now();
        
        try {
            await new Promise((resolve, reject) => {
                const options = {
                    hostname: 'localhost',
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
            // This shouldn't happen since we handle errors in the promise
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
    
    res.json(healthResponse);
});

app.listen(PORT, () => {
    console.log(`🚀 Visited Tracker Dashboard running on http://localhost:${PORT}`);
    console.log(`📊 API: http://localhost:${API_PORT}`);
});
