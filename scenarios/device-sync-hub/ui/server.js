#!/usr/bin/env node

// Device Sync Hub UI Server
// Serves static files with dynamic configuration injection

const http = require('http');
const fs = require('fs');
const path = require('path');
const url = require('url');

// Configuration from environment variables
const UI_PORT = process.env.UI_PORT || 3301;
const API_PORT = process.env.API_PORT || 3300;
const AUTH_PORT = process.env.AUTH_PORT || 3250;

// Build URLs dynamically based on current host
function getConfigForRequest(req) {
    const protocol = req.headers['x-forwarded-proto'] || 'http';
    const host = req.headers.host;
    const hostname = host.split(':')[0];
    
    return {
        apiUrl: `${protocol}://${hostname}:${API_PORT}`,
        authUrl: `${protocol}://${hostname}:${AUTH_PORT}`,
        apiPort: API_PORT,
        authPort: AUTH_PORT
    };
}

// Mime types for static files
const mimeTypes = {
    '.html': 'text/html',
    '.js': 'application/javascript',
    '.css': 'text/css',
    '.json': 'application/json',
    '.png': 'image/png',
    '.jpg': 'image/jpeg',
    '.gif': 'image/gif',
    '.ico': 'image/x-icon',
    '.svg': 'image/svg+xml'
};

// Inject configuration into HTML files
function injectConfig(htmlContent, config) {
    return htmlContent
        .replace('<meta name="api-url" content="">', `<meta name="api-url" content="${config.apiUrl}">`)
        .replace('<meta name="auth-url" content="">', `<meta name="auth-url" content="${config.authUrl}">`)
        .replace('<meta name="api-port" content="">', `<meta name="api-port" content="${config.apiPort}">`)
        .replace('<meta name="auth-port" content="">', `<meta name="auth-port" content="${config.authPort}">`);
}

// Create HTTP server
const server = http.createServer((req, res) => {
    const parsedUrl = url.parse(req.url);
    let pathname = parsedUrl.pathname;
    
    // Default to index.html for root path
    if (pathname === '/') {
        pathname = '/index.html';
    }
    
    // Security: prevent directory traversal
    if (pathname.includes('..')) {
        res.writeHead(403, {'Content-Type': 'text/plain'});
        res.end('Forbidden');
        return;
    }
    
    const filePath = path.join(__dirname, pathname);
    const ext = path.extname(filePath);
    
    // Check if file exists
    fs.access(filePath, fs.constants.F_OK, (err) => {
        if (err) {
            // File not found
            res.writeHead(404, {'Content-Type': 'text/plain'});
            res.end('Not Found');
            return;
        }
        
        // Read and serve file
        fs.readFile(filePath, (err, data) => {
            if (err) {
                res.writeHead(500, {'Content-Type': 'text/plain'});
                res.end('Internal Server Error');
                return;
            }
            
            const mimeType = mimeTypes[ext] || 'application/octet-stream';
            res.writeHead(200, {'Content-Type': mimeType});
            
            // Inject configuration for HTML files
            if (ext === '.html') {
                const config = getConfigForRequest(req);
                const injectedContent = injectConfig(data.toString(), config);
                res.end(injectedContent);
            } else {
                res.end(data);
            }
        });
    });
});

// Start server
server.listen(UI_PORT, () => {
    console.log(`🌐 Device Sync Hub UI server running on port ${UI_PORT}`);
    console.log(`📡 API URL: http://localhost:${API_PORT}`);
    console.log(`🔐 Auth URL: http://localhost:${AUTH_PORT}`);
    console.log(`🎯 UI URL: http://localhost:${UI_PORT}`);
});

// Graceful shutdown
process.on('SIGTERM', () => {
    console.log('📴 UI server shutting down...');
    server.close(() => {
        console.log('✅ UI server stopped');
        process.exit(0);
    });
});

process.on('SIGINT', () => {
    console.log('📴 UI server shutting down...');
    server.close(() => {
        console.log('✅ UI server stopped');
        process.exit(0);
    });
});