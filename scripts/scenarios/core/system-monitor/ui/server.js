const express = require('express');
const path = require('path');

const app = express();
const PORT = process.env.PORT || 3003;
const API_PORT = process.env.API_PORT || 8083;

// Serve static files
app.use(express.static(__dirname));

// Inject API port into the frontend
app.get('/script.js', (req, res) => {
    const fs = require('fs');
    let script = fs.readFileSync(path.join(__dirname, 'script.js'), 'utf8');
    
    // Replace the API port placeholder with actual port
    script = script.replace(
        "(process?.env?.API_PORT || '8083')",
        `'${API_PORT}'`
    );
    
    res.setHeader('Content-Type', 'application/javascript');
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