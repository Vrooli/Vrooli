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
        scenario: 'travel-map-filler',
        port: PORT,
        timestamp: new Date().toISOString()
    });
});


// Serve index.html for all routes (SPA support)
app.get('*', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

app.listen(PORT, () => {
    console.log(`🗺️ Travel Map Filler UI running at http://localhost:${PORT}`);
});
