const express = require('express');
const cors = require('cors');
const path = require('path');

const app = express();
const PORT = process.env.UI_PORT || process.env.PORT;
const API_PORT = process.env.API_PORT;


// Health check endpoint for orchestrator
app.get('/health', (req, res) => {
    res.json({ 
        status: 'healthy',
        scenario: 'invoice-generator',
        port: PORT,
        timestamp: new Date().toISOString()
    });
});

app.use(cors());
app.use(express.static(__dirname));

// Inject environment variables into the HTML
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

// Pass environment variables to the client
app.get('/config.js', (req, res) => {
    res.type('application/javascript');
    res.send(`
        window.API_PORT = ${API_PORT};
        window.UI_PORT = ${PORT};
    `);
});

app.listen(PORT, () => {
    console.log(`
    ⚡ Invoice Generator Pro UI
    =====================================
    🌐 UI running at: http://localhost:${PORT}
    🔌 API endpoint: http://localhost:${API_PORT}
    =====================================
    `);
});
