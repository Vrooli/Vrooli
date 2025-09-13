const express = require('express');
const path = require('path');
const fs = require('fs');

const app = express();
const PORT = process.env.UI_PORT || process.env.PORT;
const API_PORT = process.env.API_PORT;

// Check if we have a built React app
const buildPath = path.join(__dirname, 'build');
const publicPath = path.join(__dirname, 'public');
const hasBuild = fs.existsSync(buildPath);

if (hasBuild) {
    // Production mode: serve built React app
    console.log('📦 Serving built React app from build directory');
    app.use(express.static(buildPath));
} else {
    // Development mode: serve public directory for basic assets
    console.log('🛠️  Development mode: serving from public directory');
    app.use(express.static(publicPath));
}

// Provide config to frontend
app.get('/config', (req, res) => {
    res.json({
        apiUrl: `http://localhost:${API_PORT}`,
        version: '1.0.0',
        service: 'ai-chatbot-manager'
    });
});

// Health check
app.get('/health', async (req, res) => {
    const healthResponse = {
        status: 'healthy',
        service: 'ai-chatbot-manager-ui',
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
                
                const req = require('http').request(options, (res) => {
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
    
    // Add React build mode information
    healthResponse.build_mode = hasBuild ? 'production' : 'development';
    
    // Check static widget files accessibility
    const widgetPath = path.join(__dirname, 'public', 'widget.js');
    if (fs.existsSync(widgetPath)) {
        healthResponse.widget_files = {
            available: true,
            widget_js: true
        };
    } else {
        healthResponse.widget_files = {
            available: false,
            error: 'Widget.js file not found'
        };
        if (healthResponse.status === 'healthy') {
            healthResponse.status = 'degraded';
        }
    }
    
    res.json(healthResponse);
});

// For React Router - serve index.html for all non-API routes
app.get('*', (req, res) => {
    const indexPath = hasBuild 
        ? path.join(buildPath, 'index.html')
        : path.join(publicPath, 'index.html');
    
    if (fs.existsSync(indexPath)) {
        res.sendFile(indexPath);
    } else {
        res.status(404).send(`
            <html>
                <body>
                    <h1>AI Chatbot Manager UI</h1>
                    <p>React app not found. Please build the app first:</p>
                    <pre>npm run build</pre>
                    <p>Or run in development mode:</p>
                    <pre>npm run start:dev</pre>
                </body>
            </html>
        `);
    }
});

app.listen(PORT, () => {
    console.log(`🚀 AI Chatbot Manager Dashboard running on http://localhost:${PORT}`);
    console.log(`📊 API: http://localhost:${API_PORT}`);
    console.log(`🏗️  Mode: ${hasBuild ? 'Production (built)' : 'Development (public)'}`);
    
    if (!hasBuild) {
        console.log(`💡 To build the React app: npm run build`);
    }
});