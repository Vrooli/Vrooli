const express = require('express');
const path = require('path');

const app = express();
const PORT = process.env.UI_PORT || 3000;

// Serve static files
app.use(express.static(__dirname));

// Health check endpoint
app.get('/health', (req, res) => {
    res.json({ status: 'healthy', service: 'task-planner-ui' });
});

// Catch all route - serve index.html
app.get('*', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

app.listen(PORT, () => {
    console.log(`Task Planner UI running on http://localhost:${PORT}`);
    console.log(`API expected at http://localhost:${process.env.SERVICE_PORT || 8090}`);
});