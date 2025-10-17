const express = require('express');
const path = require('path');
const fs = require('fs');
const cors = require('cors');

const app = express();
const PORT = Number(process.env.UI_PORT || process.env.PORT || 4173);

const distPath = path.join(__dirname, 'dist');
const staticRoot = fs.existsSync(distPath) ? distPath : __dirname;

if (!fs.existsSync(distPath)) {
  console.warn('[NutritionTracker] dist/ directory not found. Falling back to serving source files.');
  console.warn('  • Run `npm run build` so the lifecycle serves the latest production bundle.');
}

app.use(cors());
app.use(express.json());

// Health check endpoint for orchestrator
app.get('/health', (_req, res) => {
  res.json({
    status: 'healthy',
    scenario: 'nutrition-tracker',
    port: PORT,
    timestamp: new Date().toISOString()
  });
});

// API endpoint to proxy requests to n8n workflows
app.post('/api/:workflow', async (req, res) => {
  const { workflow } = req.params;
  const n8nUrl = process.env.N8N_URL || 'http://localhost:5678';

  try {
    const response = await fetch(`${n8nUrl}/webhook/${workflow}`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
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

// Serve the built UI (or raw source fallback) with SPA routing support
app.use(express.static(staticRoot, { index: false }));

app.get('*', (req, res, next) => {
  if (req.path.startsWith('/api/')) {
    return next();
  }

  const indexFile = path.join(staticRoot, 'index.html');
  if (!fs.existsSync(indexFile)) {
    return res.status(500).json({ error: 'UI build missing index.html' });
  }

  res.sendFile(indexFile);
});

app.listen(PORT, () => {
  console.log(`🥗 NutriTrack server running on port ${PORT}`);
  console.log(`🌐 Visit http://localhost:${PORT} to access the app`);
});
