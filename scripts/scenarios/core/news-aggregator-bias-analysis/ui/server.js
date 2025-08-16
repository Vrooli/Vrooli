const express = require('express');
const path = require('path');

const app = express();
const PORT = process.env.PORT || 3300;

app.use(express.static('public'));

app.get('*', (req, res) => {
  res.sendFile(path.join(__dirname, 'public', 'index.html'));
});

app.listen(PORT, () => {
  console.log(`📰 News Aggregator UI running on http://localhost:${PORT}`);
});