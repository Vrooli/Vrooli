const express = require('express');
const path = require('path');
const app = express();

// Environment variables must be explicitly set - fail fast if missing
const PORT = process.env.UI_PORT;
if (!PORT) {
    console.error('❌ Error: UI_PORT environment variable is required');
    console.error('💡 This service must be run through the Vrooli lifecycle system.');
    console.error('   Use: vrooli scenario start qr-code-generator');
    process.exit(1);
}

const API_PORT = process.env.API_PORT;
if (!API_PORT) {
    console.error('❌ Error: API_PORT environment variable is required');
    process.exit(1);
}

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