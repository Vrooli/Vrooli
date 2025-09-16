const express = require('express');
const path = require('path');
const sqlite3 = require('sqlite3').verbose();
const WebSocket = require('ws');
const http = require('http');

const app = express();
const server = http.createServer(app);
const wss = new WebSocket.Server({ server, port: 8100 });

const PORT = process.env.OPENMCT_PORT || 8099;
const DATA_DIR = process.env.OPENMCT_DATA_DIR || '/data';

// Initialize database
const db = new sqlite3.Database(path.join(DATA_DIR, 'telemetry.db'));

// Middleware
app.use(express.json());
app.use(express.static(path.join(__dirname, 'dist')));

// Health check endpoint
app.get('/health', (req, res) => {
    res.json({ status: 'healthy', timestamp: Date.now() });
});

// Telemetry API endpoints
app.get('/api/telemetry/:stream', (req, res) => {
    const { stream } = req.params;
    res.json({
        stream,
        name: stream.replace(/_/g, ' '),
        type: 'telemetry.point',
        values: ['timestamp', 'value']
    });
});

app.post('/api/telemetry/:stream/data', (req, res) => {
    const { stream } = req.params;
    const { timestamp, value, data } = req.body;
    
    db.run(
        'INSERT INTO telemetry (stream, timestamp, value, data) VALUES (?, ?, ?, ?)',
        [stream, timestamp || Date.now(), value, JSON.stringify(data || {})],
        (err) => {
            if (err) {
                res.status(500).json({ error: err.message });
            } else {
                res.json({ success: true });
                
                // Broadcast to WebSocket clients
                wss.clients.forEach(client => {
                    if (client.readyState === WebSocket.OPEN) {
                        client.send(JSON.stringify({
                            stream,
                            timestamp: timestamp || Date.now(),
                            value,
                            data
                        }));
                    }
                });
            }
        }
    );
});

app.get('/api/telemetry/history', (req, res) => {
    const { stream, start, end } = req.query;
    
    let query = 'SELECT * FROM telemetry WHERE 1=1';
    const params = [];
    
    if (stream) {
        query += ' AND stream = ?';
        params.push(stream);
    }
    if (start) {
        query += ' AND timestamp >= ?';
        params.push(parseInt(start));
    }
    if (end) {
        query += ' AND timestamp <= ?';
        params.push(parseInt(end));
    }
    
    query += ' ORDER BY timestamp DESC LIMIT 1000';
    
    db.all(query, params, (err, rows) => {
        if (err) {
            res.status(500).json({ error: err.message });
        } else {
            res.json(rows);
        }
    });
});

// WebSocket connection handling
wss.on('connection', (ws) => {
    console.log('New WebSocket connection');
    
    ws.on('message', (message) => {
        try {
            const data = JSON.parse(message);
            
            // Store in database
            db.run(
                'INSERT INTO telemetry (stream, timestamp, value, data) VALUES (?, ?, ?, ?)',
                [data.stream, data.timestamp || Date.now(), data.value, JSON.stringify(data.data || {})]
            );
            
            // Broadcast to other clients
            wss.clients.forEach(client => {
                if (client !== ws && client.readyState === WebSocket.OPEN) {
                    client.send(message);
                }
            });
        } catch (error) {
            console.error('WebSocket message error:', error);
        }
    });
    
    ws.on('close', () => {
        console.log('WebSocket connection closed');
    });
});

// Start server
server.listen(PORT, '0.0.0.0', () => {
    console.log(`Open MCT server running on port ${PORT}`);
    console.log(`WebSocket server running on port 8100`);
    
    // Start demo telemetry if enabled
    if (process.env.OPENMCT_ENABLE_DEMO === 'true') {
        require('./telemetry-server').startDemoTelemetry(wss);
    }
});
