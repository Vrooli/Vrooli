#!/usr/bin/env node

const express = require('express');
const cors = require('cors');
const path = require('path');
require('dotenv').config();

const app = express();
const PORT = process.env.CHORE_TRACKER_PORT || 3456;
const API_URL = process.env.API_URL || 'http://localhost:8456';

app.use(cors());
app.use(express.json());
app.use(express.static('.'));

// Proxy API requests to the backend
app.use('/api', (req, res) => {
    const apiEndpoint = `${API_URL}${req.url}`;
    
    fetch(apiEndpoint, {
        method: req.method,
        headers: {
            'Content-Type': 'application/json',
            ...req.headers
        },
        body: req.method !== 'GET' ? JSON.stringify(req.body) : undefined
    })
    .then(response => response.json())
    .then(data => res.json(data))
    .catch(error => {
        console.error('API proxy error:', error);
        res.status(500).json({ error: 'API request failed' });
    });
});

// Serve the main page
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

// Health check endpoint
app.get('/health', (req, res) => {
    res.json({ 
        status: 'healthy', 
        service: 'chore-tracking-ui',
        timestamp: new Date().toISOString()
    });
});

app.listen(PORT, () => {
    console.log(`\n🎮 ChoreQuest UI Server Started!`);
    console.log(`\n   🏠 Local: http://localhost:${PORT}`);
    console.log(`   🔗 API: ${API_URL}`);
    console.log(`\n   Adventure in Every Task! 🌟\n`);
});