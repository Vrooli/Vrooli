const express = require('express');
const cors = require('cors');
const path = require('path');

const app = express();
const PORT = process.env.UI_PORT || 5173;

// Middleware
app.use(cors());
app.use(express.json());
app.use(express.static(__dirname));

// Routes
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

app.get('/health', (req, res) => {
    res.json({ status: 'healthy', service: 'roi-fit-analysis-ui' });
});

// Start server
app.listen(PORT, () => {
    console.log(`ROI Fit Analysis UI server running on port ${PORT}`);
    console.log(`Access the application at http://localhost:${PORT}`);
});