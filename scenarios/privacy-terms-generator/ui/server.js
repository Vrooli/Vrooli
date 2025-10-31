const http = require('http');
const fs = require('fs');
const path = require('path');

// Lifecycle protection check - must run through Vrooli lifecycle system
if (process.env.VROOLI_LIFECYCLE_MANAGED !== 'true') {
    console.error(`❌ This server must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start privacy-terms-generator

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`);
    process.exit(1);
}

const PORT = process.env.UI_PORT;
if (!PORT) {
    console.error('❌ UI_PORT environment variable is required');
    process.exit(1);
}

const API_PORT = process.env.API_PORT;
if (!API_PORT) {
    console.error('❌ API_PORT environment variable is required');
    process.exit(1);
}

const mimeTypes = {
    '.html': 'text/html',
    '.css': 'text/css',
    '.js': 'text/javascript',
    '.json': 'application/json',
    '.png': 'image/png',
    '.jpg': 'image/jpeg',
    '.gif': 'image/gif',
    '.svg': 'image/svg+xml'
};

const server = http.createServer((req, res) => {
    // Health check endpoint
    if (req.url === '/health') {
        const apiUrl = `http://localhost:${API_PORT}/health`;
        const checkStart = Date.now();

        // Check API connectivity
        http.get(apiUrl, (apiRes) => {
            const latency = Date.now() - checkStart;
            let body = '';
            apiRes.on('data', chunk => body += chunk);
            apiRes.on('end', () => {
                const healthResponse = {
                    status: 'healthy',
                    service: 'privacy-terms-generator-ui',
                    timestamp: new Date().toISOString(),
                    readiness: true,
                    api_connectivity: {
                        connected: apiRes.statusCode === 200,
                        api_url: apiUrl,
                        last_check: new Date().toISOString(),
                        error: apiRes.statusCode !== 200 ? {
                            code: 'API_UNHEALTHY',
                            message: `API returned status ${apiRes.statusCode}`,
                            category: 'resource',
                            retryable: true
                        } : null,
                        latency_ms: latency
                    }
                };
                res.writeHead(200, { 'Content-Type': 'application/json' });
                res.end(JSON.stringify(healthResponse));
            });
        }).on('error', (err) => {
            const healthResponse = {
                status: 'degraded',
                service: 'privacy-terms-generator-ui',
                timestamp: new Date().toISOString(),
                readiness: true,
                api_connectivity: {
                    connected: false,
                    api_url: apiUrl,
                    last_check: new Date().toISOString(),
                    error: {
                        code: err.code || 'CONNECTION_FAILED',
                        message: err.message,
                        category: 'network',
                        retryable: true
                    },
                    latency_ms: null
                }
            };
            res.writeHead(200, { 'Content-Type': 'application/json' });
            res.end(JSON.stringify(healthResponse));
        });
        return;
    }
    
    // Serve static files
    let filePath = req.url === '/' ? '/index.html' : req.url;
    filePath = path.join(__dirname, filePath);
    
    fs.readFile(filePath, (err, content) => {
        if (err) {
            if (err.code === 'ENOENT') {
                res.writeHead(404, { 'Content-Type': 'text/html' });
                res.end('<h1>404 - File Not Found</h1>');
            } else {
                res.writeHead(500);
                res.end(`Server Error: ${err.code}`);
            }
        } else {
            const ext = path.extname(filePath);
            const contentType = mimeTypes[ext] || 'application/octet-stream';
            res.writeHead(200, { 'Content-Type': contentType });
            res.end(content, 'utf-8');
        }
    });
});

server.listen(PORT, () => {
    console.log(`Privacy & Terms Generator UI running on http://localhost:${PORT}`);
});