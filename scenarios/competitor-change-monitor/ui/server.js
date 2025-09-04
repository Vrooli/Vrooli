const express = require('express');
const path = require('path');

const app = express();
const PORT = process.env.UI_PORT || process.env.PORT;

// Serve static files
app.use(express.static(__dirname));

// Health check endpoint for orchestrator
app.get('/health', (req, res) => {
    res.json({ 
        status: 'healthy',
        scenario: 'competitor-change-monitor',
        port: PORT,
        timestamp: new Date().toISOString()
    });
});


// API proxy configuration
app.get('/config', (req, res) => {
    res.json({
        apiUrl: process.env.API_URL || 'http://localhost:8140'
    });
});

// Serve index.html for all routes
app.get('*', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

app.listen(PORT, () => {
    console.log(`🔍 Competitor Monitor UI running on http://localhost:${PORT}`);
});