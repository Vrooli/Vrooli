#!/usr/bin/env node

/**
 * Enterprise Image Generation Pipeline UI Server
 * Serves the creative gallery-style interface
 */

const http = require('http');
const path = require('path');
const fs = require('fs');
const url = require('url');

if (typeof window !== 'undefined' && window.parent !== window && !window.__IMAGE_PIPELINE_BRIDGE_INITIALIZED) {
    try {
        const { initIframeBridgeChild } = require('@vrooli/iframe-bridge/child');

        let parentOrigin;
        if (document.referrer) {
            parentOrigin = new URL(document.referrer).origin;
        }

        initIframeBridgeChild({ parentOrigin, appId: 'image-generation-pipeline' });
        window.__IMAGE_PIPELINE_BRIDGE_INITIALIZED = true;
    } catch (error) {
        console.warn('[image-generation-pipeline] iframe bridge bootstrap skipped', error);
    }
}

class ImageGenerationUIServer {
    constructor() {
        this.port = process.env.UI_PORT || 31350;
        this.apiPort = process.env.API_PORT || 24000;
        const distDir = path.join(__dirname, 'dist');
        const srcDir = path.join(__dirname, 'src');
        if (fs.existsSync(distDir)) {
            this.staticDir = distDir;
        } else if (fs.existsSync(srcDir)) {
            this.staticDir = srcDir;
        } else {
            this.staticDir = __dirname;
        }
        this.mimeTypes = {
            '.html': 'text/html',
            '.css': 'text/css',
            '.js': 'application/javascript',
            '.json': 'application/json',
            '.png': 'image/png',
            '.jpg': 'image/jpeg',
            '.jpeg': 'image/jpeg',
            '.gif': 'image/gif',
            '.svg': 'image/svg+xml',
            '.ico': 'image/x-icon',
            '.woff': 'font/woff',
            '.woff2': 'font/woff2',
            '.ttf': 'font/ttf',
            '.eot': 'application/vnd.ms-fontobject'
        };
        
        this.setupServer();
    }

    setupServer() {
        this.server = http.createServer((req, res) => {
            this.handleRequest(req, res);
        });

        this.server.listen(this.port, () => {
            console.log('\n🎨 Enterprise Image Generation Pipeline UI');
            console.log('==========================================');
            console.log(`✨ UI Server running on: http://localhost:${this.port}`);
            console.log(`🔗 API Server expected at: http://localhost:${this.apiPort}`);
            console.log('📱 Features:');
            console.log('   • Creative Gallery-Style Interface');
            console.log('   • Voice Brief Processing');
            console.log('   • AI Image Generation');
            console.log('   • Brand Management');
            console.log('   • Campaign Organization');
            console.log('   • Quality Control Dashboard');
            console.log('==========================================\n');
        });

        this.server.on('error', (err) => {
            if (err.code === 'EADDRINUSE') {
                console.error(`❌ Port ${this.port} is already in use`);
                console.error('💡 Try setting a different UI_PORT environment variable');
            } else {
                console.error('❌ Server error:', err);
            }
            process.exit(1);
        });

        // Graceful shutdown
        process.on('SIGINT', () => {
            console.log('\n🛑 Shutting down Image Generation UI server...');
            this.server.close(() => {
                console.log('✅ Server closed successfully');
                process.exit(0);
            });
        });
    }

    handleRequest(req, res) {
        const parsedUrl = url.parse(req.url, true);
        let pathname = parsedUrl.pathname;

        // Log requests (optional, can be disabled in production)
        if (process.env.NODE_ENV !== 'production') {
            console.log(`${new Date().toISOString()} ${req.method} ${pathname}`);
        }

        // Enable CORS for API calls
        res.setHeader('Access-Control-Allow-Origin', '*');
        res.setHeader('Access-Control-Allow-Methods', 'GET, POST, PUT, DELETE, OPTIONS');
        res.setHeader('Access-Control-Allow-Headers', 'Content-Type, Authorization');

        if (req.method === 'OPTIONS') {
            res.writeHead(204);
            res.end();
            return;
        }

        // API proxy endpoints (forward to Go API server)
        if (pathname.startsWith('/api/')) {
            this.proxyToAPI(req, res);
            return;
        }

        // Health check endpoint
        if (pathname === '/health') {
            this.handleHealthCheck(req, res);
            return;
        }

        // Serve static files
        this.serveStaticFile(req, res, pathname);
    }

    proxyToAPI(req, res) {
        const apiUrl = `http://localhost:${this.apiPort}${req.url}`;
        
        const options = {
            hostname: 'localhost',
            port: this.apiPort,
            path: req.url,
            method: req.method,
            headers: req.headers
        };

        const proxyReq = http.request(options, (proxyRes) => {
            // Forward response headers
            Object.keys(proxyRes.headers).forEach(key => {
                res.setHeader(key, proxyRes.headers[key]);
            });
            
            res.writeHead(proxyRes.statusCode);
            proxyRes.pipe(res);
        });

        proxyReq.on('error', (err) => {
            console.error('API Proxy Error:', err);
            res.writeHead(502, {'Content-Type': 'application/json'});
            res.end(JSON.stringify({
                error: 'API Server Unavailable',
                message: `Cannot connect to API server on port ${this.apiPort}`,
                suggestion: 'Make sure the Go API server is running'
            }));
        });

        // Forward request body if present
        if (req.method === 'POST' || req.method === 'PUT') {
            req.pipe(proxyReq);
        } else {
            proxyReq.end();
        }
    }

    handleHealthCheck(req, res) {
        const healthData = {
            status: 'healthy',
            service: 'image-generation-pipeline-ui',
            port: this.port,
            timestamp: new Date().toISOString(),
            apiServer: `http://localhost:${this.apiPort}`,
            features: [
                'creative-gallery-interface',
                'voice-brief-processing',
                'ai-image-generation',
                'brand-management',
                'campaign-organization',
                'quality-control'
            ]
        };

        res.writeHead(200, {'Content-Type': 'application/json'});
        res.end(JSON.stringify(healthData, null, 2));
    }

    serveStaticFile(req, res, pathname) {
        // Default to index.html for root path and SPA routing
        if (pathname === '/' || pathname === '/index.html' || !path.extname(pathname)) {
            pathname = '/index.html';
        }

        const filePath = path.join(this.staticDir, pathname);
        const ext = path.extname(filePath);
        const contentType = this.mimeTypes[ext] || 'application/octet-stream';

        // Security check - prevent directory traversal
        const resolvedPath = path.resolve(filePath);
        const resolvedStaticDir = path.resolve(this.staticDir);
        
        if (!resolvedPath.startsWith(resolvedStaticDir)) {
            this.send404(res);
            return;
        }

        fs.readFile(filePath, (err, data) => {
            if (err) {
                if (err.code === 'ENOENT') {
                    // For SPA routing, serve index.html for unknown routes
                    if (!ext && pathname !== '/index.html') {
                        this.serveStaticFile(req, res, '/index.html');
                        return;
                    }
                    this.send404(res);
                } else {
                    this.send500(res, err);
                }
                return;
            }

            // Set appropriate headers
            res.writeHead(200, {
                'Content-Type': contentType,
                'Cache-Control': ext === '.html' ? 'no-cache' : 'public, max-age=31536000' // 1 year for assets
            });
            res.end(data);
        });
    }

    send404(res) {
        const html404 = `
        <!DOCTYPE html>
        <html lang="en">
        <head>
            <meta charset="UTF-8">
            <meta name="viewport" content="width=device-width, initial-scale=1.0">
            <title>404 - Page Not Found</title>
            <style>
                body {
                    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
                    display: flex;
                    align-items: center;
                    justify-content: center;
                    min-height: 100vh;
                    margin: 0;
                    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
                    color: white;
                }
                .error-container {
                    text-align: center;
                    padding: 2rem;
                    background: rgba(255, 255, 255, 0.1);
                    border-radius: 1rem;
                    backdrop-filter: blur(10px);
                }
                h1 { font-size: 4rem; margin: 0; }
                h2 { margin: 1rem 0; }
                a {
                    color: white;
                    text-decoration: underline;
                    font-weight: 500;
                }
            </style>
        </head>
        <body>
            <div class="error-container">
                <h1>🎨</h1>
                <h2>Page Not Found</h2>
                <p>The requested page could not be found.</p>
                <p><a href="/">← Return to Image Generation Pipeline</a></p>
            </div>
        </body>
        </html>
        `;

        res.writeHead(404, {'Content-Type': 'text/html'});
        res.end(html404);
    }

    send500(res, error) {
        console.error('Server error:', error);
        
        const html500 = `
        <!DOCTYPE html>
        <html lang="en">
        <head>
            <meta charset="UTF-8">
            <meta name="viewport" content="width=device-width, initial-scale=1.0">
            <title>500 - Server Error</title>
            <style>
                body {
                    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
                    display: flex;
                    align-items: center;
                    justify-content: center;
                    min-height: 100vh;
                    margin: 0;
                    background: linear-gradient(135deg, #ff6b6b 0%, #ee5a24 100%);
                    color: white;
                }
                .error-container {
                    text-align: center;
                    padding: 2rem;
                    background: rgba(255, 255, 255, 0.1);
                    border-radius: 1rem;
                    backdrop-filter: blur(10px);
                }
                h1 { font-size: 4rem; margin: 0; }
                h2 { margin: 1rem 0; }
                a {
                    color: white;
                    text-decoration: underline;
                    font-weight: 500;
                }
            </style>
        </head>
        <body>
            <div class="error-container">
                <h1>⚠️</h1>
                <h2>Server Error</h2>
                <p>An internal server error occurred.</p>
                <p><a href="/">← Return to Image Generation Pipeline</a></p>
            </div>
        </body>
        </html>
        `;

        res.writeHead(500, {'Content-Type': 'text/html'});
        res.end(html500);
    }
}

// Start the server if this file is run directly
if (require.main === module) {
    new ImageGenerationUIServer();
}

module.exports = ImageGenerationUIServer;
