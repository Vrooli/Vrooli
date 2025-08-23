const express = require('express');
const path = require('path');
const WebSocket = require('ws');
const http = require('http');
const axios = require('axios');
require('dotenv').config();

const app = express();
const server = http.createServer(app);
const wss = new WebSocket.Server({ server });

const PORT = process.env.UI_PORT || 3000;
const API_PORT = process.env.SERVICE_PORT || 8090;
const API_BASE = process.env.API_BASE || `http://localhost:${API_PORT}`;

// Serve static files
app.use(express.static(__dirname));
app.use(express.json());

// Proxy API requests to Go backend
app.use('/api', async (req, res) => {
    try {
        const apiUrl = `${API_BASE}${req.url}`;
        const response = await axios({
            method: req.method,
            url: apiUrl,
            data: req.body,
            headers: {
                'Content-Type': 'application/json'
            }
        });
        res.json(response.data);
    } catch (error) {
        console.error('API proxy error:', error.message);
        res.status(error.response?.status || 500).json({
            error: error.message,
            details: error.response?.data
        });
    }
});

// WebSocket handling
wss.on('connection', (ws) => {
    console.log('New WebSocket client connected');
    
    // Send initial connection message
    ws.send(JSON.stringify({
        type: 'connection',
        payload: { status: 'connected' }
    }));
    
    // Simulate real-time updates (in production, this would connect to actual monitoring)
    const metricsInterval = setInterval(() => {
        if (ws.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify({
                type: 'metric_update',
                payload: {
                    cpu: Math.floor(Math.random() * 100),
                    memory: Math.floor(Math.random() * 2048),
                    network: Math.floor(Math.random() * 1000),
                    disk: Math.floor(Math.random() * 100)
                }
            }));
        }
    }, 5000);
    
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
        clearInterval(metricsInterval);
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

// Simulate app status updates
function broadcastAppUpdate() {
    const mockUpdate = {
        type: 'app_update',
        payload: {
            id: `app${Math.floor(Math.random() * 5) + 1}`,
            status: ['running', 'stopped', 'error'][Math.floor(Math.random() * 3)],
            cpu: Math.floor(Math.random() * 100),
            memory: Math.floor(Math.random() * 1024)
        }
    };
    
    wss.clients.forEach((client) => {
        if (client.readyState === WebSocket.OPEN) {
            client.send(JSON.stringify(mockUpdate));
        }
    });
}

// Simulate log entries
function broadcastLogEntry() {
    const levels = ['info', 'warning', 'error', 'debug'];
    const messages = [
        'Application started successfully',
        'Health check completed',
        'Resource allocation updated',
        'Connection established',
        'Cache cleared',
        'Configuration reloaded',
        'Backup completed',
        'API request processed'
    ];
    
    const mockLog = {
        type: 'log_entry',
        payload: {
            level: levels[Math.floor(Math.random() * levels.length)],
            message: messages[Math.floor(Math.random() * messages.length)],
            timestamp: new Date().toISOString()
        }
    };
    
    wss.clients.forEach((client) => {
        if (client.readyState === WebSocket.OPEN) {
            client.send(JSON.stringify(mockLog));
        }
    });
}

// Set up periodic broadcasts
setInterval(broadcastAppUpdate, 10000);
setInterval(broadcastLogEntry, 15000);

// Health check endpoint
app.get('/health', (req, res) => {
    res.json({
        status: 'healthy',
        uptime: process.uptime(),
        connections: wss.clients.size
    });
});

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
    `);
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