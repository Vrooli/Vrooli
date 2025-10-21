#!/usr/bin/env node

/**
 * Scenario-to-Desktop UI Server
 * Serves the web interface for desktop application generation
 */

const express = require('express');
const cors = require('cors');
const helmet = require('helmet');
const compression = require('compression');
const morgan = require('morgan');
const path = require('path');
const fs = require('fs');

// Configuration
const PORT = process.env.UI_PORT || process.env.PORT || 35200;

// API port MUST be provided by lifecycle system for managed deployments
// SECURITY: No default value to prevent port conflicts and misconfigurations
if (process.env.VROOLI_LIFECYCLE_MANAGED === 'true' && !process.env.API_PORT) {
    console.error('FATAL: API_PORT environment variable is required when managed by Vrooli lifecycle');
    console.error('       The lifecycle system must explicitly allocate API port.');
    process.exit(1);
}

// For non-lifecycle (standalone) deployments, API_BASE_URL can be fully specified
const API_PORT = process.env.API_PORT;
const API_BASE_URL = process.env.API_BASE_URL || (API_PORT ? `http://localhost:${API_PORT}` : 'http://localhost:15200');
const NODE_ENV = process.env.NODE_ENV || 'development';

// Create Express app
const app = express();

const localFrameAncestors = [
    "'self'",
    'http://localhost:*',
    'http://127.0.0.1:*',
    'http://[::1]:*'
];

const extraFrameAncestors = (process.env.FRAME_ANCESTORS || '')
    .split(/\s+/)
    .filter(Boolean);

// Security middleware
app.use(helmet({
    frameguard: false,
    contentSecurityPolicy: {
        useDefaults: true,
        directives: {
            defaultSrc: ["'self'"],
            styleSrc: ["'self'", "'unsafe-inline'"],
            scriptSrc: ["'self'", "'unsafe-inline'"],
            imgSrc: ["'self'", "data:", "https:"],
            connectSrc: ["'self'", API_BASE_URL, 'ws:', 'wss:'],
            fontSrc: ["'self'"],
            objectSrc: ["'none'"],
            mediaSrc: ["'self'"],
            frameSrc: ["'none'"],
            frameAncestors: [...localFrameAncestors, ...extraFrameAncestors],
        },
    },
}));

// CORS configuration
app.use(cors({
    origin: process.env.CORS_ORIGIN || '*',
    methods: ['GET', 'POST', 'PUT', 'DELETE', 'OPTIONS'],
    allowedHeaders: ['Content-Type', 'Authorization', 'X-Requested-With'],
    credentials: false
}));

// Compression and logging
app.use(compression());
if (NODE_ENV === 'development') {
    app.use(morgan('dev'));
} else {
    app.use(morgan('combined'));
}

// Parse JSON bodies
app.use(express.json({ limit: '10mb' }));
app.use(express.urlencoded({ extended: true, limit: '10mb' }));

// Static files
app.use('/static', express.static(path.join(__dirname, 'static')));

// API proxy for development (optional)
if (NODE_ENV === 'development') {
    app.use('/api', (req, res) => {
        res.status(502).json({
            error: 'API proxy not implemented',
            message: 'Please ensure the API server is running on port 15200',
            api_url: API_BASE_URL
        });
    });
}

// Health check endpoint
app.get('/health', async (req, res) => {
    // Check API connectivity
    let apiConnected = false;
    let apiError = null;
    let apiLatency = null;
    const apiCheckStart = Date.now();

    try {
        const fetch = require('node-fetch');
        const apiHealthUrl = `${API_BASE_URL}/api/v1/health`;
        const response = await fetch(apiHealthUrl, { timeout: 2000 });
        apiConnected = response.ok;
        apiLatency = Date.now() - apiCheckStart;

        if (!response.ok) {
            apiError = {
                code: `HTTP_${response.status}`,
                message: `API health check returned status ${response.status}`,
                category: 'network',
                retryable: true
            };
        }
    } catch (err) {
        apiError = {
            code: err.code || 'CONNECTION_FAILED',
            message: err.message || 'Failed to connect to API',
            category: 'network',
            retryable: true
        };
    }

    res.json({
        status: apiConnected ? 'healthy' : 'degraded',
        service: 'scenario-to-desktop-ui',
        version: '1.0.0',
        timestamp: new Date().toISOString(),
        readiness: true,
        api_connectivity: {
            connected: apiConnected,
            api_url: `${API_BASE_URL}/api/v1/health`,
            last_check: new Date().toISOString(),
            error: apiError,
            latency_ms: apiLatency
        }
    });
});

// Status endpoint
app.get('/status', (req, res) => {
    res.json({
        service: {
            name: 'scenario-to-desktop-ui',
            version: '1.0.0',
            description: 'Web UI for desktop application generation',
            status: 'running'
        },
        configuration: {
            port: PORT,
            api_base_url: API_BASE_URL,
            node_env: NODE_ENV
        },
        features: [
            'Desktop app generation interface',
            'Template browser',
            'Build status monitoring',
            'System statistics dashboard'
        ]
    });
});

// Serve main application
app.get('/', (req, res) => {
    const indexPath = path.join(__dirname, 'index.html');
    
    if (!fs.existsSync(indexPath)) {
        return res.status(500).json({
            error: 'UI files not found',
            message: 'index.html not found in UI directory'
        });
    }
    
    // Read and serve the HTML file
    let html = fs.readFileSync(indexPath, 'utf8');
    
    // Replace API URL placeholder if needed
    html = html.replace(/API_BASE_URL\s*=\s*['"][^'"]*['"]/, `API_BASE_URL = '${API_BASE_URL}'`);
    
    res.setHeader('Content-Type', 'text/html');
    res.send(html);
});

// Serve favicon (if it exists)
app.get('/favicon.ico', (req, res) => {
    const faviconPath = path.join(__dirname, 'favicon.ico');
    if (fs.existsSync(faviconPath)) {
        res.sendFile(faviconPath);
    } else {
        res.status(404).end();
    }
});

// Catch-all handler for SPA routing
app.get('*', (req, res) => {
    res.redirect('/');
});

// Error handling middleware
app.use((err, req, res, next) => {
    console.error('Error:', err);
    
    res.status(err.status || 500).json({
        error: 'Internal Server Error',
        message: NODE_ENV === 'development' ? err.message : 'Something went wrong',
        ...(NODE_ENV === 'development' && { stack: err.stack })
    });
});

// 404 handler
app.use((req, res) => {
    res.status(404).json({
        error: 'Not Found',
        message: `Route ${req.method} ${req.path} not found`
    });
});

// Graceful shutdown
function shutdown(signal) {
    console.log(`\nReceived ${signal}. Starting graceful shutdown...`);
    
    server.close(() => {
        console.log('Server closed. Goodbye!');
        process.exit(0);
    });
    
    // Force exit after timeout
    setTimeout(() => {
        console.log('Could not close connections in time, forcefully shutting down');
        process.exit(1);
    }, 10000);
}

// Start server
const server = app.listen(PORT, () => {
    console.log('🖥️  Scenario-to-Desktop UI Server');
    console.log('==================================');
    console.log(`✅ Server running on port ${PORT}`);
    console.log(`🌐 Web interface: http://localhost:${PORT}`);
    console.log(`📊 Health check: http://localhost:${PORT}/health`);
    console.log(`🔧 API endpoint: ${API_BASE_URL}`);
    console.log(`📱 Environment: ${NODE_ENV}`);
    console.log('');
    console.log('Ready to generate desktop applications! 🚀');
});

// Handle graceful shutdown
process.on('SIGTERM', () => shutdown('SIGTERM'));
process.on('SIGINT', () => shutdown('SIGINT'));

// Handle uncaught exceptions
process.on('uncaughtException', (err) => {
    console.error('Uncaught Exception:', err);
    process.exit(1);
});

process.on('unhandledRejection', (reason, promise) => {
    console.error('Unhandled Rejection at:', promise, 'reason:', reason);
    process.exit(1);
});

module.exports = app;
