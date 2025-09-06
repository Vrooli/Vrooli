const express = require('express');
const path = require('path');

const app = express();
const PORT = process.env.UI_PORT || 34100;

// Serve static files
app.use(express.static(__dirname));

// Main route
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

// Health check
app.get('/health', (req, res) => {
    res.json({ 
        status: 'healthy', 
        service: 'scenario-surfer-ui',
        timestamp: Date.now()
    });
});

app.listen(PORT, () => {
    console.log(`🌊 Scenario Surfer UI running on http://localhost:${PORT}`);
});