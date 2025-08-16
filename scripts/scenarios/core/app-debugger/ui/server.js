const express = require('express');
const cors = require('cors');
const path = require('path');

const app = express();
const PORT = process.env.PORT || 30151;

// Enable CORS for API communication
app.use(cors());

// Serve static files
app.use(express.static(__dirname));

// Main route
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

// Health check endpoint
app.get('/health', (req, res) => {
    res.json({ 
        status: 'healthy', 
        service: 'app-debugger-ui',
        timestamp: new Date().toISOString()
    });
});

// Start server
app.listen(PORT, () => {
    console.log(`App Debugger UI running on port ${PORT}`);
    console.log(`Access the debugger at http://localhost:${PORT}`);
});