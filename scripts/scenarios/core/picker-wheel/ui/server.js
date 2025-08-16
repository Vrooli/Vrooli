const express = require('express');
const path = require('path');
const cors = require('cors');

const app = express();
const PORT = process.env.PORT || 9850;

// Middleware
app.use(cors());
app.use(express.json());
app.use(express.static(__dirname));

// Serve index.html for root route
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

// Health check endpoint
app.get('/health', (req, res) => {
    res.json({ status: 'healthy', service: 'picker-wheel-ui' });
});

// Start server
app.listen(PORT, () => {
    console.log(`🎯 Picker Wheel UI server running on http://localhost:${PORT}`);
    console.log(`   Theme: Neon Arcade`);
    console.log(`   Features: Animated wheel, confetti, sound effects`);
});