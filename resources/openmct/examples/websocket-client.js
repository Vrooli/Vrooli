#!/usr/bin/env node
// WebSocket client example for Open MCT telemetry streaming

const WebSocket = require('ws');

// Connect to Open MCT WebSocket server
const ws = new WebSocket('ws://localhost:8100/api/telemetry/live');

ws.on('open', () => {
    console.log('Connected to Open MCT WebSocket server');
    
    // Send telemetry data every second
    setInterval(() => {
        const telemetryData = {
            stream: 'websocket_demo',
            timestamp: Date.now(),
            value: Math.random() * 100,
            data: {
                source: 'WebSocket Demo Client',
                type: 'random',
                metadata: {
                    iteration: Math.floor(Date.now() / 1000),
                    quality: 'good'
                }
            }
        };
        
        ws.send(JSON.stringify(telemetryData));
        console.log('Sent:', telemetryData.value);
    }, 1000);
});

ws.on('message', (data) => {
    console.log('Received:', data.toString());
});

ws.on('error', (error) => {
    console.error('WebSocket error:', error);
});

ws.on('close', () => {
    console.log('Disconnected from Open MCT WebSocket server');
    process.exit(0);
});

// Handle graceful shutdown
process.on('SIGINT', () => {
    console.log('\nClosing WebSocket connection...');
    ws.close();
});