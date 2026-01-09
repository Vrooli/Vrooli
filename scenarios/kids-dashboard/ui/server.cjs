const express = require('express');
const path = require('path');

const app = express();
const PORT = process.env.UI_PORT || process.env.PORT || 3500;
const API_PORT = process.env.API_PORT;

// Serve static files from dist directory
app.use(express.static(path.join(__dirname, 'dist')));

// Health check endpoint
app.get('/health', (req, res) => {
    res.json({ status: 'healthy', service: 'kids-dashboard-ui' });
});

// Catch all route - serve index.html for SPA routing
app.get('*', (req, res) => {
    res.sendFile(path.join(__dirname, 'dist', 'index.html'));
});

app.listen(PORT, () => {
    console.log(`🎮 Kids Dashboard UI running on http://localhost:${PORT}`);
    if (API_PORT) {
        console.log(`📡 API available at http://localhost:${API_PORT}`);
    }
});
