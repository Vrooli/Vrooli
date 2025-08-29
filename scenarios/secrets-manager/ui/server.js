#!/usr/bin/env node
// Dark Chrome Security Dashboard Server
// Serves the Secrets Manager cyberpunk UI

const http = require('http');
const fs = require('fs');
const path = require('path');
const url = require('url');

const PORT = process.env.PORT || process.env.UI_PORT || 28250;
const API_PORT = process.env.SECRETS_API_URL?.split(':').pop() || process.env.SERVICE_PORT || 28000;

// MIME types for different file extensions
const MIME_TYPES = {
    '.html': 'text/html',
    '.css': 'text/css',
    '.js': 'application/javascript',
    '.json': 'application/json',
    '.png': 'image/png',
    '.jpg': 'image/jpeg',
    '.gif': 'image/gif',
    '.svg': 'image/svg+xml',
    '.ico': 'image/x-icon',
    '.woff': 'font/woff',
    '.woff2': 'font/woff2',
    '.ttf': 'font/ttf',
    '.eot': 'application/vnd.ms-fontobject'
};

// Security headers for enhanced protection
const SECURITY_HEADERS = {
    'X-Content-Type-Options': 'nosniff',
    'X-Frame-Options': 'DENY',
    'X-XSS-Protection': '1; mode=block',
    'Referrer-Policy': 'strict-origin-when-cross-origin',
    'Content-Security-Policy': "default-src 'self' 'unsafe-inline' 'unsafe-eval' https://fonts.googleapis.com https://fonts.gstatic.com; img-src 'self' data:;"
};

class DarkChromeServer {
    constructor() {
        this.server = null;
        this.startTime = new Date();
    }

    start() {
        this.server = http.createServer((req, res) => {
            this.handleRequest(req, res);
        });

        this.server.listen(PORT, () => {
            this.logStartup();
        });

        // Graceful shutdown handling
        process.on('SIGTERM', () => this.shutdown());
        process.on('SIGINT', () => this.shutdown());
    }

    handleRequest(req, res) {
        const parsedUrl = url.parse(req.url, true);
        const pathname = parsedUrl.pathname;

        // Security: Prevent directory traversal
        if (pathname.includes('..')) {
            this.sendError(res, 400, 'Invalid request path');
            return;
        }

        // Handle special routes
        if (pathname === '/') {
            this.serveFile(res, 'index.html');
        } else if (pathname === '/health') {
            this.sendHealthCheck(res);
        } else if (pathname === '/api-config') {
            this.sendApiConfig(res);
        } else {
            // Serve static files
            const filePath = pathname.substring(1); // Remove leading slash
            this.serveFile(res, filePath);
        }
    }

    serveFile(res, filePath) {
        const fullPath = path.join(__dirname, filePath);
        
        // Security: Ensure file is within our directory
        if (!fullPath.startsWith(__dirname)) {
            this.sendError(res, 403, 'Access denied');
            return;
        }

        fs.readFile(fullPath, (err, data) => {
            if (err) {
                if (err.code === 'ENOENT') {
                    this.sendError(res, 404, 'File not found');
                } else {
                    this.sendError(res, 500, 'Server error');
                }
                return;
            }

            const ext = path.extname(filePath);
            const mimeType = MIME_TYPES[ext] || 'text/plain';

            // Apply security headers
            Object.entries(SECURITY_HEADERS).forEach(([header, value]) => {
                res.setHeader(header, value);
            });

            // Cache control for static assets
            if (ext === '.css' || ext === '.js') {
                res.setHeader('Cache-Control', 'public, max-age=3600'); // 1 hour
            } else if (ext === '.html') {
                res.setHeader('Cache-Control', 'no-cache'); // Always revalidate HTML
            }

            res.setHeader('Content-Type', mimeType);
            
            // Inject API configuration into HTML
            if (ext === '.html' && filePath === 'index.html') {
                const htmlContent = data.toString().replace(
                    '<!-- API_CONFIG_INJECTION -->',
                    `<script>window.API_CONFIG = { apiPort: '${API_PORT}' };</script>`
                );
                res.end(htmlContent);
            } else {
                res.end(data);
            }
        });
    }

    sendHealthCheck(res) {
        const uptime = Math.floor((Date.now() - this.startTime) / 1000);
        
        const health = {
            status: 'operational',
            service: 'secrets-manager-ui',
            version: '1.0.0',
            uptime: uptime,
            timestamp: new Date().toISOString(),
            endpoints: {
                dashboard: `http://localhost:${PORT}`,
                api: `http://localhost:${API_PORT}`,
                health: `http://localhost:${PORT}/health`
            },
            theme: 'dark-chrome-cyberpunk'
        };

        res.setHeader('Content-Type', 'application/json');
        Object.entries(SECURITY_HEADERS).forEach(([header, value]) => {
            res.setHeader(header, value);
        });
        
        res.end(JSON.stringify(health, null, 2));
    }

    sendApiConfig(res) {
        const config = {
            apiUrl: `http://localhost:${API_PORT}`,
            uiUrl: `http://localhost:${PORT}`,
            features: {
                secretScanning: true,
                secretValidation: true,
                secretProvisioning: true,
                vaultIntegration: true,
                darkChrome: true
            }
        };

        res.setHeader('Content-Type', 'application/json');
        res.setHeader('Access-Control-Allow-Origin', '*');
        Object.entries(SECURITY_HEADERS).forEach(([header, value]) => {
            res.setHeader(header, value);
        });
        
        res.end(JSON.stringify(config, null, 2));
    }

    sendError(res, statusCode, message) {
        res.statusCode = statusCode;
        res.setHeader('Content-Type', 'text/plain');
        
        Object.entries(SECURITY_HEADERS).forEach(([header, value]) => {
            res.setHeader(header, value);
        });
        
        res.end(`Error ${statusCode}: ${message}`);
        
        // Log error for debugging
        console.error(`🔴 [${new Date().toISOString()}] HTTP ${statusCode}: ${message}`);
    }

    logStartup() {
        console.log('🔐 SECRETS MANAGER - Dark Chrome Security Dashboard');
        console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
        console.log(`⚡ UI Server:     http://localhost:${PORT}`);
        console.log(`🔌 API Server:    http://localhost:${API_PORT}`);
        console.log(`📊 Health Check:  http://localhost:${PORT}/health`);
        console.log(`🌐 Dashboard:     http://localhost:${PORT}/`);
        console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
        console.log('🚀 Dark Chrome Terminal Ready');
        console.log(`📅 Started: ${this.startTime.toISOString()}`);
        console.log('🔒 Security headers enabled');
        console.log('⌨️  Keyboard shortcuts available in dashboard');
        console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
    }

    shutdown() {
        console.log('\n🛑 Shutting down Dark Chrome Security Dashboard...');
        
        if (this.server) {
            this.server.close(() => {
                console.log('✅ UI Server stopped gracefully');
                process.exit(0);
            });
        } else {
            process.exit(0);
        }
    }
}

// Error handling for uncaught exceptions
process.on('uncaughtException', (err) => {
    console.error('🔴 Uncaught Exception:', err);
    process.exit(1);
});

process.on('unhandledRejection', (reason, promise) => {
    console.error('🔴 Unhandled Rejection at:', promise, 'reason:', reason);
    process.exit(1);
});

// Start the server
const server = new DarkChromeServer();
server.start();