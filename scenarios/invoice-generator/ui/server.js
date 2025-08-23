const express = require('express');
const cors = require('cors');
const path = require('path');

const app = express();
const PORT = process.env.UI_PORT || 3100;
const SERVICE_PORT = process.env.SERVICE_PORT || 8100;

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
        window.SERVICE_PORT = ${SERVICE_PORT};
        window.UI_PORT = ${PORT};
    `);
});

app.listen(PORT, () => {
    console.log(`
    ⚡ Invoice Generator Pro UI
    =====================================
    🌐 UI running at: http://localhost:${PORT}
    🔌 API endpoint: http://localhost:${SERVICE_PORT}
    =====================================
    `);
});