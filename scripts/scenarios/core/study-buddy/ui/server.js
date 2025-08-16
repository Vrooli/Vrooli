const express = require('express');
const cors = require('cors');
const path = require('path');
const http = require('http');
const socketIO = require('socket.io');
require('dotenv').config();

const app = express();
const server = http.createServer(app);
const io = socketIO(server, {
    cors: {
        origin: "*",
        methods: ["GET", "POST"]
    }
});

const PORT = process.env.UI_PORT || 3500;
const API_URL = `http://localhost:${process.env.API_PORT || 8500}`;

app.use(cors());
app.use(express.json());
app.use(express.static(__dirname));

// API proxy endpoints
app.post('/api/*', async (req, res) => {
    try {
        const axios = require('axios');
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
        const axios = require('axios');
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