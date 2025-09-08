const express = require('express');
const path = require('path');
const WebSocket = require('ws');
const http = require('http');
require('dotenv').config();

const app = express();
const PORT = process.env.UI_PORT || process.env.PORT;
const API_PORT = process.env.API_PORT;
const server = http.createServer(app);
// Only handle WebSocket on /ws path, not all connections
const wss = new WebSocket.Server({
    server,
    path: '/ws'  // Only handle WebSocket connections on /ws endpoint
});

const API_BASE = process.env.API_BASE || `http://localhost:${API_PORT}`;

// Parse JSON bodies
app.use(express.json());

// Manual proxy function for API calls
function proxyToApi(req, res, apiPath) {
    const options = {
        hostname: 'localhost',
        port: API_PORT,
        path: apiPath || req.url,
        method: req.method,
        headers: {
            ...req.headers,
            host: `localhost:${API_PORT}`
        }
    };

    console.log(`[PROXY] ${req.method} ${req.url} -> http://localhost:${API_PORT}${options.path}`);

    const proxyReq = http.request(options, (proxyRes) => {
        res.status(proxyRes.statusCode);
        Object.keys(proxyRes.headers).forEach(key => {
            res.setHeader(key, proxyRes.headers[key]);
        });
        proxyRes.pipe(res);
    });

    proxyReq.on('error', (err) => {
        console.error('Proxy error:', err.message);
        res.status(502).json({
            error: 'API server unavailable',
            details: err.message,
            target: `http://localhost:${API_PORT}${options.path}`
        });
    });

    if (req.method !== 'GET' && req.method !== 'HEAD') {
        req.pipe(proxyReq);
    } else {
        proxyReq.end();
    }
}

// API endpoints proxy
app.use('/api', (req, res) => {
    const fullApiPath = '/api' + (req.url.startsWith('/') ? req.url : '/' + req.url);
    proxyToApi(req, res, fullApiPath);
});

// WebSocket handling
wss.on('connection', (ws) => {
    console.log('New WebSocket client connected');

    // Add error handling to prevent crashes
    ws.on('error', (error) => {
        console.error('WebSocket error:', error.message);
        // Don't crash the server on WebSocket errors
    });

    // Send initial connection message (wrapped in try-catch)
    try {
        ws.send(JSON.stringify({
            type: 'connection',
            payload: { status: 'connected' }
        }));
    } catch (error) {
        console.error('Failed to send initial message:', error.message);
        return;
    }

    // Real-time metric updates removed - UI now uses initial API fetch for accurate data
    // Mock setInterval was sending memory values up to 2048, causing >100% displays

    // Handle client messages
    ws.on('message', (message) => {
        try {
            const data = JSON.parse(message);
            console.log('Received:', data);

            // Handle different message types
            switch (data.type) {
                case 'subscribe':
                    // Subscribe to specific app updates
                    console.log(`Client subscribed to: ${data.appId}`);
                    break;
                case 'command':
                    // Execute command and send response
                    handleCommand(data.command, ws);
                    break;
            }
        } catch (error) {
            console.error('WebSocket message error:', error);
        }
    });

    ws.on('close', () => {
        console.log('WebSocket client disconnected');
        // No intervals to clear - metrics come from API calls
    });

    ws.on('error', (error) => {
        console.error('WebSocket error:', error);
    });
});

// Handle commands from WebSocket
async function handleCommand(command, ws) {
    try {
        // Process command and send response
        const result = await processCommand(command);
        ws.send(JSON.stringify({
            type: 'command_response',
            payload: result
        }));
    } catch (error) {
        ws.send(JSON.stringify({
            type: 'error',
            payload: { message: error.message }
        }));
    }
}

async function processCommand(command) {
    // This would integrate with the actual system
    // For now, return mock responses
    const [cmd, ...args] = command.split(' ');

    switch (cmd) {
        case 'status':
            return { status: 'System operational', apps: 5, resources: 8 };
        case 'list':
            if (args[0] === 'apps') {
                return { apps: ['app1', 'app2', 'app3'] };
            }
            break;
        default:
            throw new Error(`Unknown command: ${cmd}`);
    }
}

// WebSocket broadcasts removed - all real-time data comes from actual system events
// Mock broadcasts were causing fake metric data (memory > 100%) to be sent to clients

// Set up periodic broadcasts
// Note: Real-time updates should be triggered by actual events from the system,
// not by periodic intervals with mock data
// setInterval(broadcastAppUpdate, 10000);  // DISABLED - should use real events
// setInterval(broadcastLogEntry, 15000);   // DISABLED - should use real events

// Health check endpoint (before static files)
app.get('/health', (req, res) => {
    res.json({
        status: 'healthy',
        uptime: process.uptime(),
        connections: wss.clients.size
    });
});

// In production, serve built React files
if (process.env.NODE_ENV === 'production') {
    const staticPath = path.join(__dirname, 'dist');
    app.use(express.static(staticPath));

    // Catch all routes for client-side routing in production
    app.get('*', (req, res) => {
        // Skip API routes
        if (req.path.startsWith('/api')) {
            return res.status(404).json({ error: 'Not found' });
        }
        res.sendFile(path.join(staticPath, 'index.html'));
    });
} else {
    // In development, show helpful message
    app.get('/', (req, res) => {
        const vitePort = process.env.VITE_PORT;
        res.send(`
            <html>
                <head>
                    <title>App Monitor - Express Server</title>
                    <style>
                        body {
                            background: #0a0a0a;
                            color: #00ff41;
                            font-family: 'Courier New', monospace;
                            padding: 2rem;
                            text-align: center;
                        }
                        h1 { color: #39ff14; }
                        a {
                            color: #00ffff;
                            font-size: 1.2rem;
                        }
                        .info {
                            margin: 2rem;
                            padding: 1rem;
                            border: 1px solid #00ff41;
                        }
                    </style>
                </head>
                <body>
                    <h1>🚀 App Monitor Express Server</h1>
                    <div class="info">
                        <p>This is the Express backend server (port ${PORT})</p>
                        <p>It handles API proxying and WebSocket connections only.</p>
                        <br>
                        <p><strong>To access the React UI, go to:</strong></p>
                        <h2><a href="http://localhost:${vitePort}">http://localhost:${vitePort}</a></h2>
                        <br>
                        <p>If Vite is not running, start it with: <code>npm run dev</code></p>
                    </div>
                    <div class="info">
                        <p>Available endpoints on this server:</p>
                        <ul style="text-align: left; display: inline-block;">
                            <li>/api/* - Proxied to Go API server</li>
                            <li>/ws - WebSocket endpoint</li>
                            <li>/health - Health check</li>
                        </ul>
                    </div>
                </body>
            </html>
        `);
    });

    // Catch all other routes in development with helpful message
    app.get('*', (req, res) => {
        if (req.path.startsWith('/api')) {
            return res.status(404).json({ error: 'API endpoint not found' });
        }
        const vitePort = process.env.VITE_PORT;
        res.status(404).send(`
            <html>
                <head>
                    <title>404 - Wrong Server</title>
                    <style>
                        body {
                            background: #0a0a0a;
                            color: #ff0040;
                            font-family: 'Courier New', monospace;
                            padding: 2rem;
                            text-align: center;
                        }
                        a { color: #00ffff; }
                    </style>
                </head>
                <body>
                    <h1>❌ 404 - Wrong Server</h1>
                    <p>You're accessing the Express backend server.</p>
                    <p>The React UI is running on the Vite dev server:</p>
                    <h2><a href="http://localhost:${vitePort}${req.path}">http://localhost:${vitePort}${req.path}</a></h2>
                </body>
            </html>
        `);
    });
}

// Start server
server.listen(PORT, () => {
    console.log(`
╔════════════════════════════════════════════╗
║     VROOLI APP MONITOR - MATRIX UI        ║
║                                            ║
║  UI Server running on port ${PORT}           ║
║  WebSocket server active                   ║
║  API proxy to port ${API_PORT}                ║
║                                            ║
║  Access dashboard at:                      ║
║  http://localhost:${PORT}                     ║
╔════════════════════════════════════════════╗
    UPDATED_CODE
    
🔍 DIAGNOSTIC INFO (boop):
Working Directory: ${process.cwd()}
Script Path: ${__filename}
Process ID: ${process.pid}
Node Version: ${process.version}
Arguments: ${process.argv.join(' ')}
Environment: NODE_ENV=${process.env.NODE_ENV || 'undefined'}`);
});

// Graceful shutdown
process.on('SIGTERM', () => {
    console.log('SIGTERM received, closing server...');
    wss.clients.forEach((client) => {
        client.close();
    });
    server.close(() => {
        console.log('Server closed');
        process.exit(0);
    });
});

process.on('SIGINT', () => {
    console.log('\nSIGINT received, closing server...');
    wss.clients.forEach((client) => {
        client.close();
    });
    server.close(() => {
        console.log('Server closed');
        process.exit(0);
    });
});
