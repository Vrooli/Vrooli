#!/usr/bin/env node

const express = require('express');
const cors = require('cors');
const path = require('path');
const fs = require('fs');
require('dotenv').config();

const app = express();
const PORT = process.env.UI_PORT || process.env.PORT;
const API_URL = process.env.API_URL || (process.env.API_PORT ? `http://localhost:${process.env.API_PORT}` : 'http://localhost:8456');

app.use(cors());
app.use(express.json());

const DIST_DIR = path.join(__dirname, 'dist');
const STATIC_DIR = fs.existsSync(path.join(DIST_DIR, 'index.html')) ? DIST_DIR : __dirname;

if (STATIC_DIR === __dirname) {
    console.warn('⚠️  UI bundle not built. Serving files directly from ui/. Run `npm run build` for optimized assets.');
}

app.use(express.static(STATIC_DIR));

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
    res.sendFile(path.join(STATIC_DIR, 'index.html'));
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
