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
        scenario: 'product-manager-agent',
        port: PORT,
        timestamp: new Date().toISOString()
    });
});


// API proxy endpoints (forward to the Go API)
app.use('/api', (req, res) => {
    const apiUrl = `http://localhost:8800${req.url}`;
    fetch(apiUrl, {
        method: req.method,
        headers: req.headers,
        body: req.method !== 'GET' ? JSON.stringify(req.body) : undefined
    })
    .then(response => response.json())
    .then(data => res.json(data))
    .catch(error => res.status(500).json({ error: error.message }));
});

// Serve index.html for all other routes
app.get('*', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

app.listen(PORT, () => {
    console.log(`Product Manager UI running on http://localhost:${PORT}`);
});