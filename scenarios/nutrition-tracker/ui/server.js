import express from 'express';
import cors from 'cors';
import { existsSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const app = express();
const PORT = Number(process.env.UI_PORT || process.env.PORT || 4173);

const distPath = path.join(__dirname, 'dist');
const staticRoot = existsSync(distPath) ? distPath : __dirname;

if (!existsSync(distPath)) {
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

// Optional API proxy to backend service if provided
app.all('/api/*', async (req, res, next) => {
  const apiUrl = process.env.API_URL;
  if (!apiUrl) return next();

  try {
    const target = `${apiUrl}${req.originalUrl.replace(/^\\/api/, '')}`;
    const response = await fetch(target, {
      method: req.method,
      headers: { 'Content-Type': 'application/json' },
      body: ['GET', 'HEAD'].includes(req.method.toUpperCase()) ? undefined : JSON.stringify(req.body)
    });
    const data = await response.json();
    res.status(response.status).json(data);
  } catch (error) {
    console.error('API proxy error:', error);
    res.status(502).json({ error: 'Failed to reach API' });
  }
});

// Serve the built UI (or raw source fallback) with SPA routing support
app.use(express.static(staticRoot, { index: false }));

app.get('*', (req, res, next) => {
  if (req.path.startsWith('/api/')) {
    return next();
  }

  const indexFile = path.join(staticRoot, 'index.html');
  if (!existsSync(indexFile)) {
    return res.status(500).json({ error: 'UI build missing index.html' });
  }

  res.sendFile(indexFile);
});

app.listen(PORT, () => {
  console.log(`🥗 NutriTrack server running on port ${PORT}`);
  console.log(`🌐 Visit http://localhost:${PORT} to access the app`);
});
