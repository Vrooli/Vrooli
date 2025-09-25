const express = require('express');
const path = require('path');
const app = express();
const PORT = process.env.UI_PORT || process.env.PORT;
const API_PORT = process.env.API_PORT || '17320';

// Serve static files
app.use(express.static(__dirname));

// Root route
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

// Health check endpoint
app.get('/health', (req, res) => {
    res.json({ status: 'healthy', service: 'qr-code-generator-ui' });
});

// API config endpoint for frontend
app.get('/config', (req, res) => {
    res.json({ apiPort: API_PORT });
});

app.listen(PORT, () => {
    console.log(`✨ QR Code Generator UI running on http://localhost:${PORT}`);
    console.log(`⚡ Retro mode activated!`);
});