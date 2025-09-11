const http = require('http');
const fs = require('fs');
const path = require('path');
const url = require('url');

// Configuration - PORT is required from environment
const PORT = process.env.AUTH_UI_PORT || process.env.UI_PORT;
if (!PORT) {
    console.error('❌ AUTH_UI_PORT or UI_PORT environment variable is required');
    process.exit(1);
}

// MIME types
const mimeTypes = {
    '.html': 'text/html',
    '.js': 'text/javascript',
    '.css': 'text/css',
    '.json': 'application/json',
    '.png': 'image/png',
    '.jpg': 'image/jpeg',
    '.gif': 'image/gif',
    '.ico': 'image/x-icon'
};

// Get API port from environment
const API_PORT = process.env.AUTH_API_PORT || process.env.API_PORT || '15000';

// Simple HTTP server
const server = http.createServer((req, res) => {
    const parsedUrl = url.parse(req.url);
    let pathname = parsedUrl.pathname;
    
    // Enable CORS
    res.setHeader('Access-Control-Allow-Origin', '*');
    res.setHeader('Access-Control-Allow-Methods', 'GET, POST, PUT, DELETE, OPTIONS');
    res.setHeader('Access-Control-Allow-Headers', 'Content-Type, Authorization');
    
    if (req.method === 'OPTIONS') {
        res.writeHead(200);
        res.end();
        return;
    }
    
    // Config endpoint to provide API URL to client
    if (pathname === '/config') {
        res.writeHead(200, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({
            apiUrl: `http://localhost:${API_PORT}`,
            version: '1.0.0'
        }));
        return;
    }
    
    // Route handling
    if (pathname === '/' || pathname === '/login' || pathname === '/register') {
        pathname = '/index.html';
    }
    
    const filePath = path.join(__dirname, pathname);
    const ext = path.extname(filePath);
    const mimeType = mimeTypes[ext] || 'text/plain';
    
    // Check if file exists
    fs.access(filePath, fs.constants.F_OK, (err) => {
        if (err) {
            // File not found
            if (pathname !== '/favicon.ico') {
                console.log(`404 - File not found: ${filePath}`);
            }
            res.writeHead(404, { 'Content-Type': 'text/html' });
            res.end(`
                <html>
                    <body>
                        <h1>404 - Not Found</h1>
                        <p>The requested resource was not found.</p>
                        <p><a href="/">Go to Login</a></p>
                    </body>
                </html>
            `);
            return;
        }
        
        // Read and serve file
        fs.readFile(filePath, (err, data) => {
            if (err) {
                console.error(`Error reading file ${filePath}:`, err);
                res.writeHead(500, { 'Content-Type': 'text/plain' });
                res.end('Internal Server Error');
                return;
            }
            
            res.writeHead(200, { 
                'Content-Type': mimeType,
                'Cache-Control': ext === '.html' ? 'no-cache' : 'public, max-age=3600'
            });
            res.end(data);
        });
    });
});

// Health check endpoint
const healthCheck = (req, res) => {
    if (req.url === '/health') {
        res.writeHead(200, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({
            status: 'healthy',
            service: 'authentication-ui',
            version: '1.0.0',
            timestamp: new Date().toISOString()
        }));
        return true;
    }
    return false;
};

// Add health check to server
const originalListener = server.listeners('request')[0];
server.removeAllListeners('request');
server.on('request', (req, res) => {
    if (healthCheck(req, res)) {
        return;
    }
    originalListener(req, res);
});

// Start server
server.listen(PORT, () => {
    console.log(`Authentication UI server running on port ${PORT}`);
    console.log(`Access at: http://localhost:${PORT}`);
    console.log(`Health check: http://localhost:${PORT}/health`);
});

// Graceful shutdown
process.on('SIGTERM', () => {
    console.log('Received SIGTERM, shutting down gracefully');
    server.close(() => {
        console.log('Authentication UI server closed');
        process.exit(0);
    });
});

process.on('SIGINT', () => {
    console.log('Received SIGINT, shutting down gracefully');
    server.close(() => {
        console.log('Authentication UI server closed');
        process.exit(0);
    });
});

// Error handling
server.on('error', (error) => {
    if (error.code === 'EADDRINUSE') {
        console.error(`Port ${PORT} is already in use`);
        process.exit(1);
    } else {
        console.error('Server error:', error);
    }
});

process.on('uncaughtException', (error) => {
    console.error('Uncaught Exception:', error);
    process.exit(1);
});

process.on('unhandledRejection', (reason, promise) => {
    console.error('Unhandled Rejection at:', promise, 'reason:', reason);
    process.exit(1);
});