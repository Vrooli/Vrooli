const express = require('express');
const path = require('path');
const cors = require('cors');
const fs = require('fs');

const app = express();
const PORT = process.env.UI_PORT || process.env.PORT || 3100;
const API_PORT = process.env.API_PORT || process.env.SERVICE_PORT || 8100;
const N8N_PORT = process.env.RESOURCE_PORTS ? JSON.parse(process.env.RESOURCE_PORTS).n8n : 5678;

// Middleware
app.use(cors());
app.use(express.json());

// Serve static files except script.js
app.use(express.static(__dirname, { 
    index: false,
    setHeaders: (res, path) => {
        if (path.endsWith('.js')) {
            res.setHeader('Content-Type', 'application/javascript');
        }
    }
}));

// Serve script.js with injected environment variables
app.get('/script.js', (req, res) => {
    fs.readFile(path.join(__dirname, 'script.js'), 'utf8', (err, data) => {
        if (err) {
            res.status(500).send('Error loading script');
            return;
        }
        
        // Replace hardcoded values with environment variables
        const modifiedScript = data
            .replace("const API_BASE = 'http://localhost:8100/api';", `const API_BASE = 'http://localhost:${API_PORT}/api';`)
            .replace("const N8N_BASE = 'http://localhost:5678/webhook';", `const N8N_BASE = 'http://localhost:${N8N_PORT}/webhook';`);
        
        res.setHeader('Content-Type', 'application/javascript');
        res.send(modifiedScript);
    });
});

// Serve index.html for root route
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

// Health check endpoint
app.get('/health', (req, res) => {
    res.json({ status: 'healthy', service: 'picker-wheel-ui' });
});

// Start server
app.listen(PORT, () => {
    console.log(`🎯 Picker Wheel UI server running on http://localhost:${PORT}`);
    console.log(`   Theme: Neon Arcade`);
    console.log(`   Features: Animated wheel, confetti, sound effects`);
});