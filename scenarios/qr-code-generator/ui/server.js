const express = require('express');
const path = require('path');
const app = express();
const PORT = process.env.UI_PORT || process.env.PORT;

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

app.listen(PORT, () => {
    console.log(`✨ QR Code Generator UI running on http://localhost:${PORT}`);
    console.log(`⚡ Retro mode activated!`);
});