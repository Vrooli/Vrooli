const express = require('express');
const path = require('path');
const cors = require('cors');

const app = express();
const PORT = process.env.UI_PORT || process.env.PORT;

// Middleware
app.use(cors());
app.use(express.static(path.join(__dirname)));

// Routes
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

// Health check endpoint
app.get('/health', (req, res) => {
    res.json({ status: 'healthy', service: 'secure-document-processing-ui' });
});

// Start server
app.listen(PORT, () => {
    console.log(`SecureVault Pro UI running on http://localhost:${PORT}`);
    console.log('Press Ctrl+C to stop');
});