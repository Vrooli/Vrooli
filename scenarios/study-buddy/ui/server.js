const express = require('express');
const cors = require('cors');
const path = require('path');
const http = require('http');
const socketIO = require('socket.io');
const axios = require('axios');
require('dotenv').config();

const app = express();
const server = http.createServer(app);
const io = socketIO(server, {
    cors: {
        origin: "*",
        methods: ["GET", "POST"]
    }
});

const PORT = process.env.UI_PORT || process.env.PORT;
const API_URL = `http://localhost:${process.env.API_PORT || 8500}`;

app.use(cors());
app.use(express.json());
app.use(express.static(__dirname));

// Health check endpoint for orchestrator
app.get('/health', async (req, res) => {
    const now = new Date();
    const timestamp = now.toISOString();
    let status = 'healthy';
    let readiness = true;

    const apiConnectivity = {
        connected: false,
        api_url: API_URL,
        last_check: timestamp,
        latency_ms: null,
        error: null
    };

    try {
        const start = Date.now();
        const response = await axios.get(`${API_URL}/health`, { timeout: 2000 });
        apiConnectivity.latency_ms = Date.now() - start;
        apiConnectivity.connected = response.status === 200;

        if (response.data?.readiness === false || !apiConnectivity.connected) {
            status = 'degraded';
            readiness = false;
        }
    } catch (error) {
        status = 'degraded';
        readiness = false;

        const message = error.response?.data?.message || error.message || 'Failed to reach API health endpoint';
        const retryable = !error.response || error.response.status >= 500;
        apiConnectivity.error = {
            code: error.code || 'API_HEALTH_UNAVAILABLE',
            message,
            category: 'network',
            retryable,
            details: {
                status: error.response?.status ?? null
            }
        };
    }

    res.json({
        status,
        service: 'study-buddy-ui',
        timestamp,
        readiness,
        api_connectivity: apiConnectivity,
        metrics: {
            active_connections: io.engine?.clientsCount ?? 0
        }
    });
});


// API proxy endpoints
app.post('/api/*', async (req, res) => {
    try {
        const apiPath = req.path.replace('/api', '');
        const response = await axios.post(`${API_URL}${apiPath}`, req.body);
        res.json(response.data);
    } catch (error) {
        console.error('API proxy error:', error);
        res.status(error.response?.status || 500).json({ error: error.message });
    }
});

app.get('/api/*', async (req, res) => {
    try {
        const apiPath = req.path.replace('/api', '');
        const response = await axios.get(`${API_URL}${apiPath}`, { params: req.query });
        res.json(response.data);
    } catch (error) {
        console.error('API proxy error:', error);
        res.status(error.response?.status || 500).json({ error: error.message });
    }
});

// Socket.IO for real-time features
io.on('connection', (socket) => {
    console.log('New client connected:', socket.id);
    
    // Join study room
    socket.on('join-room', (roomId) => {
        socket.join(roomId);
        socket.to(roomId).emit('user-joined', socket.id);
    });
    
    // Study timer sync
    socket.on('timer-update', (data) => {
        socket.to(data.roomId).emit('timer-sync', data.time);
    });
    
    // Collaborative notes
    socket.on('note-update', (data) => {
        socket.to(data.roomId).emit('note-sync', data);
    });
    
    // Achievement notifications
    socket.on('achievement-unlocked', (data) => {
        io.emit('achievement-notification', data);
    });
    
    socket.on('disconnect', () => {
        console.log('Client disconnected:', socket.id);
    });
});

// Serve the main page
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

server.listen(PORT, () => {
    console.log(`Study Buddy UI running on http://localhost:${PORT}`);
    console.log(`Connected to API at ${API_URL}`);
});
