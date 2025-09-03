const express = require('express');
const path = require('path');
const cors = require('cors');

const app = express();
const PORT = process.env.UI_PORT || process.env.PORT;

// Middleware
app.use(cors());
app.use(express.json());
app.use(express.static(__dirname));

// API endpoint to proxy requests to n8n workflows
app.post('/api/:workflow', async (req, res) => {
    const { workflow } = req.params;
    const n8nUrl = process.env.N8N_URL || 'http://localhost:5678';
    
    try {
        const response = await fetch(`${n8nUrl}/webhook/${workflow}`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(req.body)
        });
        
        const data = await response.json();
        res.json(data);
    } catch (error) {
        console.error(`Error calling workflow ${workflow}:`, error);
        res.status(500).json({ error: 'Failed to process request' });
    }
});

// Serve the main HTML file
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

app.listen(PORT, () => {
    console.log(`🥗 NutriTrack server running on port ${PORT}`);
    console.log(`🌐 Visit http://localhost:${PORT} to access the app`);
});