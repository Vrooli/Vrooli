const express = require('express');
const cors = require('cors');
const path = require('path');
const { createProxyMiddleware } = require('http-proxy-middleware');

const app = express();

// Configuration
const UI_PORT = process.env.UI_PORT || 38000;
const API_PORT = process.env.API_PORT || 18000;
const API_URL = `http://localhost:${API_PORT}`;
const NODE_ENV = process.env.NODE_ENV || 'development';

// Middleware
app.use(cors({
    origin: [`http://localhost:${UI_PORT}`, `http://localhost:3000`],
    credentials: true
}));

app.use(express.json({ limit: '50mb' }));
app.use(express.urlencoded({ extended: true, limit: '50mb' }));

// API Proxy - Forward all /api requests to the Go API server
app.use('/api', createProxyMiddleware({
    target: API_URL,
    changeOrigin: true,
    onError: (err, req, res) => {
        console.error('Proxy Error:', err.message);
        res.status(500).json({
            success: false,
            error: 'API server unavailable',
            details: err.message
        });
    },
    onProxyReq: (proxyReq, req, res) => {
        console.log(`Proxying ${req.method} ${req.url} to ${API_URL}${req.url}`);
    }
}));

// WebSocket proxy for real-time updates
app.use('/ws', createProxyMiddleware({
    target: API_URL.replace('http:', 'ws:'),
    ws: true,
    changeOrigin: true,
    onError: (err, req, res) => {
        console.error('WebSocket Proxy Error:', err.message);
    }
}));

// Health check endpoint
app.get('/health', (req, res) => {
    res.json({
        status: 'healthy',
        service: 'social-media-scheduler-ui',
        timestamp: new Date().toISOString(),
        version: '1.0.0',
        environment: NODE_ENV,
        ports: {
            ui: UI_PORT,
            api: API_PORT
        }
    });
});

// Static file serving
if (NODE_ENV === 'production') {
    // Serve React build files in production
    app.use(express.static(path.join(__dirname, 'build')));
    
    // Serve React app for all non-API routes
    app.get('*', (req, res) => {
        res.sendFile(path.join(__dirname, 'build', 'index.html'));
    });
} else {
    // Development mode - serve the development React app
    app.use(express.static(path.join(__dirname, 'public')));
    
    // Serve the main HTML file for development
    app.get('*', (req, res) => {
        res.sendFile(path.join(__dirname, 'public', 'index.html'));
    });
}

// Error handling middleware
app.use((err, req, res, next) => {
    console.error('Express Error:', err.stack);
    res.status(500).json({
        success: false,
        error: 'Internal server error',
        details: NODE_ENV === 'development' ? err.message : 'Something went wrong'
    });
});

// 404 handler for API routes
app.use('/api/*', (req, res) => {
    res.status(404).json({
        success: false,
        error: 'API endpoint not found',
        path: req.path
    });
});

// Start server
const server = app.listen(UI_PORT, () => {
    console.log(`🚀 Social Media Scheduler UI Server running on port ${UI_PORT}`);
    console.log(`📅 Web Interface: http://localhost:${UI_PORT}`);
    console.log(`🔗 API Proxy: ${API_URL}`);
    console.log(`🌍 Environment: ${NODE_ENV}`);
    
    if (NODE_ENV === 'development') {
        console.log('\n💡 Development Tips:');
        console.log('  - API calls are automatically proxied to the Go server');
        console.log('  - WebSocket connections are proxied for real-time updates');
        console.log('  - Make sure the Go API server is running on port', API_PORT);
    }
});

// Graceful shutdown
process.on('SIGTERM', () => {
    console.log('📴 Received SIGTERM, shutting down gracefully...');
    server.close(() => {
        console.log('✅ UI Server shut down');
        process.exit(0);
    });
});

process.on('SIGINT', () => {
    console.log('📴 Received SIGINT, shutting down gracefully...');
    server.close(() => {
        console.log('✅ UI Server shut down');
        process.exit(0);
    });
});

// Handle proxy errors
process.on('uncaughtException', (err) => {
    console.error('Uncaught Exception:', err);
    if (err.code === 'ECONNREFUSED' && err.port == API_PORT) {
        console.error(`⚠️  Cannot connect to API server on port ${API_PORT}`);
        console.error('   Make sure the Go API server is running:');
        console.error(`   cd ../api && go run . --mode=server`);
    }
});

module.exports = app;