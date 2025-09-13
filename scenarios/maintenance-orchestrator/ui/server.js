#!/usr/bin/env node

const express = require('express');
const path = require('path');
const fs = require('fs');

const app = express();

// Get port from environment variable (set by lifecycle system)
const port = process.env.UI_PORT || 3251;

// Serve static files from current directory
app.use(express.static(__dirname));

// Serve main dashboard
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

// API configuration endpoint for frontend
app.get('/config', (req, res) => {
    const apiPort = process.env.API_PORT; // Provided by lifecycle system
    res.json({
        apiUrl: `http://localhost:${apiPort}`,
        apiBase: `http://localhost:${apiPort}/api/v1`,
        uiPort: port,
        service: 'maintenance-orchestrator'
    });
});

// Legacy endpoint for backward compatibility
app.get('/api/config', (req, res) => {
    const apiPort = process.env.API_PORT;
    res.json({
        api_base: `http://localhost:${apiPort}/api/v1`,
        ui_port: port,
        service: 'maintenance-orchestrator'
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
                
                const http = require('http');
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