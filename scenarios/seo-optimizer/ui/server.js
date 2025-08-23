const express = require('express');
const cors = require('cors');
const path = require('path');

const app = express();
const PORT = process.env.UI_PORT || 4220;

// Middleware
app.use(cors());
app.use(express.static(__dirname));

// Routes
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

app.get('/health', (req, res) => {
    res.json({ status: 'healthy', service: 'seo-optimizer-ui' });
});

// Start server
app.listen(PORT, () => {
    console.log(`SEO Optimizer UI running on http://localhost:${PORT}`);
});