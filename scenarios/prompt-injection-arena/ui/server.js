const express = require('express');
const cors = require('cors');
const path = require('path');

const app = express();

// Require UI_PORT environment variable (no defaults)
const port = process.env.UI_PORT;
if (!port) {
    console.error('❌ UI_PORT environment variable is required');
    process.exit(1);
}

// Middleware
app.use(cors());
app.use(express.static(__dirname));

// Health check endpoint
app.get('/health', (req, res) => {
    res.json({
        status: 'healthy',
        service: 'prompt-injection-arena-ui',
        timestamp: new Date().toISOString()
    });
});

// Serve the main application
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

// Start server
app.listen(port, () => {
    console.log(`🏟️ Prompt Injection Arena UI running at http://localhost:${port}`);
    console.log(`📊 Security research platform ready`);
});