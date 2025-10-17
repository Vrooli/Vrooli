const express = require('express');
const path = require('path');
const fs = require('fs');

const app = express();
const PORT = process.env.UI_PORT || 40000;

// Serve static files from the dist directory
const distPath = path.join(__dirname, 'dist');
if (fs.existsSync(distPath)) {
  app.use(express.static(distPath));

  // Health check endpoint
  app.get('/health', (req, res) => {
    res.json({ 
      status: 'healthy', 
      service: 'bedtime-story-generator-ui',
      timestamp: new Date().toISOString()
    });
  });

  // Handle all other routes by serving the index.html
  app.get('*', (req, res) => {
    res.sendFile(path.join(distPath, 'index.html'));
  });
} else {
  // Development mode - serve a simple message
  app.get('*', (req, res) => {
    res.send(`
      <h1>Bedtime Story Generator UI</h1>
      <p>Please build the UI first:</p>
      <pre>cd ui && npm install && npm run build</pre>
      <p>Or run in development mode:</p>
      <pre>cd ui && npm install && npm run dev</pre>
    `);
  });
}

app.listen(PORT, () => {
  console.log(`Bedtime Story Generator UI running on port ${PORT}`);
  console.log(`Visit: http://localhost:${PORT}`);
});