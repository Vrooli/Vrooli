const express = require('express');
const path = require('path');

const app = express();
const PORT = process.env.UI_PORT || process.env.PORT;

app.use(express.static('public'));


// Health check endpoint for orchestrator
app.get('/health', (req, res) => {
    res.json({ 
        status: 'healthy',
        scenario: 'news-aggregator-bias-analysis',
        port: PORT,
        timestamp: new Date().toISOString()
    });
});

app.get('*', (req, res) => {
  res.sendFile(path.join(__dirname, 'public', 'index.html'));
});

app.listen(PORT, () => {
  console.log(`📰 News Aggregator UI running on http://localhost:${PORT}`);
});