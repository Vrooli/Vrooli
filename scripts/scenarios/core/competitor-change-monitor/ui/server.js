const express = require('express');
const path = require('path');

const app = express();
const PORT = process.env.PORT || 5240;

// Serve static files
app.use(express.static(__dirname));

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