const express = require('express');
const path = require('path');
const fs = require('fs');
const app = express();
const UI_PORT = process.env.UI_PORT || process.env.PORT;

// Environment variables with fallbacks

// Middleware for CORS and JSON
app.use(express.json());
app.use(express.urlencoded({ extended: true }));
app.use((req, res, next) => {
    res.header('Access-Control-Allow-Origin', '*');
    res.header('Access-Control-Allow-Methods', 'GET, POST, PUT, DELETE, OPTIONS');
    res.header('Access-Control-Allow-Headers', 'Content-Type, Authorization');
    if (req.method === 'OPTIONS') {
        return res.sendStatus(200);
    }
    next();
});

// Serve static files from current directory
app.use(express.static(__dirname));

// Health check endpoint (CRITICAL for lifecycle management)
app.get('/health', (req, res) => {
    res.json({ 
        status: 'healthy',
        scenario: 'unknown',
        port: UI_PORT,
        apiPort: API_PORT,
        timestamp: new Date().toISOString()
    });
});

// API configuration endpoint
app.get('/api/config', (req, res) => {
    res.json({ 
        apiUrl: `http://localhost:${API_PORT}`,
        uiPort: UI_PORT,
        scenario: 'unknown',
        resources: process.env.RESOURCE_PORTS ? JSON.parse(process.env.RESOURCE_PORTS) : {}
    });
});

// Proxy API requests to backend (optional but useful)
app.all('/api/*', (req, res) => {
    const apiUrl = `http://localhost:${API_PORT}${req.url}`;
    res.redirect(307, apiUrl);
});

// SPA fallback routing - serve appropriate HTML file
app.get('*', (req, res) => {
    // Priority order for HTML files
    const htmlFiles = [
        'dashboard.html',
        'index.html', 
        'app.html',
        'main.html'
    ];
    
    for (const file of htmlFiles) {
        const filePath = path.join(__dirname, file);
        if (fs.existsSync(filePath)) {
            return res.sendFile(filePath);
        }
    }
    
    // If no HTML found, return JSON error
    res.status(404).json({ 
        error: 'No UI files found',
        searched: htmlFiles,
        scenario: 'unknown'
    });
});

// Start server
const server = app.listen(UI_PORT, '0.0.0.0', () => {
    console.log('=====================================');
    console.log(`✅ ${'unknown'} UI Server Started`);
    console.log('=====================================');
    console.log(`📍 UI:     http://localhost:${UI_PORT}`);
    console.log(`🔌 API:    http://localhost:${API_PORT}`);
    console.log(`💚 Health: http://localhost:${UI_PORT}/health`);
    console.log('=====================================');
});

// Graceful shutdown
const shutdown = () => {
    console.log(`\n🛑 Stopping ${'unknown'} UI server...`);
    server.close(() => {
        console.log('UI server stopped gracefully');
        process.exit(0);
    });
};

process.on('SIGTERM', shutdown);
process.on('SIGINT', shutdown);

// Handle errors
server.on('error', (err) => {
    if (err.code === 'EADDRINUSE') {
        console.error(`❌ Port ${UI_PORT} is already in use`);
    } else {
        console.error('Server error:', err);
    }
    process.exit(1);
});
