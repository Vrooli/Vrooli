const express = require('express');
const path = require('path');

const app = express();
const PORT = process.env.PORT || process.env.UI_PORT || 3003;
const API_PORT = process.env.API_PORT || process.env.SERVICE_PORT || 8080;

// Serve static files (except script.js which we handle specially)
app.use(express.static(__dirname, { 
    index: false,
    setHeaders: (res, path) => {
        if (path.endsWith('.js')) {
            res.setHeader('Cache-Control', 'no-cache');
        }
    }
}));

// Inject API port into the frontend dynamically
app.get('/script.js', (req, res) => {
    const fs = require('fs');
    let script = fs.readFileSync(path.join(__dirname, 'script.js'), 'utf8');
    
    // Replace the hardcoded port with actual runtime port
    script = script.replace(
        /this\.apiPort = ['"]8080['"]/g,
        `this.apiPort = '${API_PORT}'`
    );
    
    res.setHeader('Content-Type', 'application/javascript');
    res.setHeader('Cache-Control', 'no-cache');
    res.send(script);
});

// Health check endpoint
app.get('/health', (req, res) => {
    res.json({ status: 'healthy', service: 'system-monitor-ui' });
});

// Serve the main page
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

app.listen(PORT, () => {
    console.log(`System Monitor UI running on http://localhost:${PORT}`);
    console.log(`API connection configured for port ${API_PORT}`);
});