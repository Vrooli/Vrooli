const express = require('express');
const path = require('path');
const cors = require('cors');

const app = express();
const PORT = process.env.PORT || 3950;
const API_URL = process.env.API_URL || 'http://localhost:8950';

// Enable CORS
app.use(cors());

// Serve static files
app.use(express.static(__dirname));

// Proxy API requests (optional, for development)
app.use('/api', (req, res) => {
    const apiPath = req.path;
    fetch(`${API_URL}/api${apiPath}`, {
        method: req.method,
        headers: req.headers,
        body: req.method !== 'GET' ? JSON.stringify(req.body) : undefined
    })
    .then(response => response.json())
    .then(data => res.json(data))
    .catch(error => res.status(500).json({ error: error.message }));
});

// Serve zen mode
app.get('/zen', (req, res) => {
    res.sendFile(path.join(__dirname, 'zen-index.html'));
});

// Serve index.html for all other routes
app.get('*', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

app.listen(PORT, () => {
    console.log(`📝 SmartNotes UI running on http://localhost:${PORT}`);
    console.log(`🔗 API endpoint: ${API_URL}`);
});