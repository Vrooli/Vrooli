const express = require('express');
const cors = require('cors');
const path = require('path');

const app = express();
const PORT = process.env.PORT || 3200;
const API_URL = process.env.API_URL || 'http://localhost:9200';

app.use(cors());
app.use(express.static(__dirname));

// Serve the main page
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

// Health check
app.get('/health', (req, res) => {
    res.json({ status: 'ok', service: 'typing-test-ui' });
});

app.listen(PORT, () => {
    console.log(`🎮 Typing Test UI running on http://localhost:${PORT}`);
    console.log(`📡 API URL configured as: ${API_URL}`);
});