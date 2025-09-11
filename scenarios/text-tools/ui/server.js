const express = require('express');
const cors = require('cors');
const path = require('path');

const app = express();
const PORT = process.env.UI_PORT;

if (!PORT) {
    console.error('ERROR: UI_PORT environment variable is required');
    process.exit(1);
}

// Middleware
app.use(cors());
app.use(express.static(path.join(__dirname, '../ui-react/dist')));

// Health check
app.get('/health', (req, res) => {
    res.json({ status: 'healthy', timestamp: Date.now() });
});

// Serve index.html for all routes
app.get('*', (req, res) => {
    res.sendFile(path.join(__dirname, '../ui-react/dist/index.html'));
});

// Start server
app.listen(PORT, () => {
    console.log(`Text Tools UI running on http://localhost:${PORT}`);
});