const express = require('express');
const path = require('path');

const app = express();
const PORT = process.env.VEGAN_UI_PORT || 3000;

// Serve static files
app.use(express.static(__dirname));

// Main route
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

// Health check endpoint
app.get('/health', (req, res) => {
    res.json({ status: 'healthy', service: 'make-it-vegan-ui' });
});

app.listen(PORT, () => {
    console.log(`🌱 Make It Vegan UI running at http://localhost:${PORT}`);
    console.log(`💚 Ready to help with plant-based alternatives!`);
});