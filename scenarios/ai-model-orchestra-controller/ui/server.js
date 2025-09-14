const express = require('express');
const path = require('path');
const fs = require('fs');
const app = express();
const PORT = process.env.UI_PORT || process.env.PORT;
const API_PORT = process.env.API_PORT;


// Enable CORS for API communication
app.use((req, res, next) => {
    res.header('Access-Control-Allow-Origin', '*');
    res.header('Access-Control-Allow-Headers', 'Content-Type');
    next();
});

// Serve static files
app.use(express.static(__dirname));

// Health check endpoint
app.get('/health', (req, res) => {
    res.json({ 
        status: 'healthy', 
        port: PORT,
        scenario: 'unknown'
    });
});

// API proxy configuration
app.get('/api/config', (req, res) => {
    const host = process.env.ORCHESTRATOR_HOST || 'localhost';
    res.json({ 
        apiUrl: `http://${host}:${API_PORT}`,
        uiPort: PORT,
        host: host,
        resources: process.env.RESOURCE_PORTS ? JSON.parse(process.env.RESOURCE_PORTS) : {}
    });
});

// Fallback routing for SPAs
app.get('*', (req, res) => {
    const dashboardFile = path.join(__dirname, 'dashboard.html');
    const indexFile = path.join(__dirname, 'index.html');
    
    if (fs.existsSync(dashboardFile)) {
        res.sendFile(dashboardFile);
    } else if (fs.existsSync(indexFile)) {
        res.sendFile(indexFile);
    } else {
        res.status(404).json({ error: 'No UI files found' });
    }
});

const server = app.listen(PORT, () => {
    const host = process.env.ORCHESTRATOR_HOST || 'localhost';
    console.log(`✅ UI server running on http://${host}:${PORT}`);
    console.log(`📡 API endpoint: http://${host}:${process.env.API_PORT || 8080}`);
    console.log(`🏷️  Scenario: ai-model-orchestra-controller`);
});

// Graceful shutdown
process.on('SIGTERM', () => {
    server.close(() => {
        console.log('UI server stopped');
    });
});
