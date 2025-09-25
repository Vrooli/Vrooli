const express = require('express');
const path = require('path');
const app = express();

// Get port from environment or use default
const PORT = process.env.UI_PORT || 3000;
const API_PORT = process.env.API_PORT || 8080;

// Serve static files
app.use(express.static(__dirname));

// API proxy configuration endpoint
app.get('/config', (req, res) => {
    res.json({
        apiUrl: `http://localhost:${API_PORT}`,
        version: '1.0.0'
    });
});

// Serve index.html for all routes (SPA)
app.get('*', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

// Start server
app.listen(PORT, () => {
    console.log(`💕 Date Night Planner UI running on port ${PORT}`);
    console.log(`🔗 API expected on port ${API_PORT}`);
    console.log(`🌐 Access UI at http://localhost:${PORT}`);
});