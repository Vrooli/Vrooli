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
const PORT = process.env.PORT || 3203;
const API_BASE_URL = process.env.API_BASE_URL || 'http://localhost:3202';
const NODE_ENV = process.env.NODE_ENV || 'development';

// Create Express app
const app = express();

// Security middleware
app.use(helmet({
    contentSecurityPolicy: {
        directives: {
            defaultSrc: ["'self'"],
            styleSrc: ["'self'", "'unsafe-inline'"],
            scriptSrc: ["'self'", "'unsafe-inline'"],
            imgSrc: ["'self'", "data:", "https:"],
            connectSrc: ["'self'", API_BASE_URL],
            fontSrc: ["'self'"],
            objectSrc: ["'none'"],
            mediaSrc: ["'self'"],
            frameSrc: ["'none'"],
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
            message: 'Please ensure the API server is running on port 3202',
            api_url: API_BASE_URL
        });
    });
}

// Health check endpoint
app.get('/health', (req, res) => {
    res.json({
        status: 'healthy',
        service: 'scenario-to-desktop-ui',
        version: '1.0.0',
        timestamp: new Date().toISOString(),
        api_base_url: API_BASE_URL
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