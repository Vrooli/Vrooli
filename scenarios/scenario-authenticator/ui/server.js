import express from 'express';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const app = express();
const PORT = Number(process.env.UI_PORT || process.env.PORT || 4173);
const API_PORT = process.env.API_PORT;

const distDir = path.join(__dirname, 'dist');
const indexHtmlPath = path.join(distDir, 'index.html');
const dashboardPagePath = path.join(__dirname, 'dashboard.html');
const CONTACT_BOOK_URL =
  process.env.CONTACT_BOOK_URL ||
  (process.env.CONTACT_BOOK_API_PORT ? `http://localhost:${process.env.CONTACT_BOOK_API_PORT}` : undefined);

if (!fs.existsSync(indexHtmlPath)) {
  console.error(
    '[scenario-authenticator-ui] dist/index.html not found. Run `pnpm run build` before starting the UI server.'
  );
  process.exit(1);
}

app.use(express.static(distDir));

app.get('/config', (_req, res) => {
  if (!API_PORT) {
    res.status(500).json({
      error: 'AUTH_API_PORT_UNDEFINED',
      message: 'Authentication API port is not configured. Ensure the scenario is started via the lifecycle system.',
    });
    return;
  }

  res.json({
    apiUrl: `http://localhost:${API_PORT}`,
    contactBookUrl: CONTACT_BOOK_URL || null,
    version: '2.0.0',
    service: 'authentication-ui',
  });
});

app.get(['/dashboard', '/admin'], (_req, res) => {
  res.sendFile(dashboardPagePath);
});

app.get('/health', (_req, res) => {
  res.json({
    status: 'healthy',
    service: 'authentication-ui',
    version: '2.0.0',
    timestamp: new Date().toISOString(),
    assetsBuilt: true,
  });
});


app.use((req, res) => {
  if (req.method !== 'GET') {
    res.status(404).end();
    return;
  }

  res.sendFile(indexHtmlPath);
});

app.listen(PORT, () => {
  console.log(`Scenario Authenticator UI running at http://localhost:${PORT}`);
  if (!API_PORT) {
    console.warn('API_PORT not provided. /config will respond with an error until the lifecycle sets it.');
  } else {
    console.log(`Proxying API requests to http://localhost:${API_PORT}`);
  }
});
