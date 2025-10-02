import express from 'express';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';
import { exec as execCallback } from 'child_process';
import { promisify } from 'util';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const app = express();
const PORT = Number(process.env.UI_PORT || process.env.PORT || 4173);
const API_PORT = process.env.AUTH_API_PORT || process.env.API_PORT;

const distDir = path.join(__dirname, 'dist');
const indexHtmlPath = path.join(distDir, 'index.html');
const dashboardPagePath = path.join(__dirname, 'dashboard.html');
const adminPagePath = path.join(__dirname, 'admin.html');
const userCardScriptPath = path.join(__dirname, 'user-card.js');
const CONTACT_BOOK_URL =
  process.env.CONTACT_BOOK_URL ||
  (process.env.CONTACT_BOOK_API_PORT ? `http://localhost:${process.env.CONTACT_BOOK_API_PORT}` : undefined);

const execAsync = promisify(execCallback);
const workspaceRoot = path.resolve(__dirname, '..', '..', '..');
const SCENARIO_CACHE_TTL_MS = 60_000;

let cachedScenarios;
let cachedScenariosFetchedAt = 0;
let cachedApiPort = API_PORT;

async function resolveApiPort() {
  if (cachedApiPort) {
    return cachedApiPort;
  }

  try {
    const { stdout } = await execAsync(
      'vrooli scenario port scenario-authenticator API_PORT',
      {
        cwd: workspaceRoot,
        env: process.env,
      }
    );

    const port = stdout.trim();
    if (port) {
      cachedApiPort = port;
      return cachedApiPort;
    }
  } catch (error) {
    console.warn(
      '[scenario-authenticator-ui] Unable to resolve API port via CLI lookup',
      error
    );
  }

  return null;
}

async function loadScenariosFromCli() {
  const { stdout } = await execAsync('vrooli scenario list --json', {
    cwd: workspaceRoot,
    env: process.env,
  });

  let payload = stdout.trim();
  if (!payload) {
    return [];
  }

  const braceIndex = payload.search(/[\[{]/);
  if (braceIndex > 0) {
    payload = payload.slice(braceIndex);
  }

  let parsed;
  try {
    parsed = JSON.parse(payload);
  } catch (error) {
    console.warn('[scenario-authenticator-ui] Failed to parse scenario list payload', error);
    return [];
  }

  const rawScenarios = Array.isArray(parsed)
    ? parsed
    : Array.isArray(parsed?.scenarios)
    ? parsed.scenarios
    : [];

  const normalized = rawScenarios
    .map((scenario) => {
      const name = scenario?.name || scenario?.slug || scenario?.id || '';
      if (!name) {
        return null;
      }
      const displayName =
        scenario?.display_name ||
        scenario?.displayName ||
        scenario?.title ||
        scenario?.label ||
        name;

      return {
        name,
        displayName,
      };
    })
    .filter(Boolean)
    .sort((a, b) => a.displayName.localeCompare(b.displayName));

  return normalized;
}

if (!fs.existsSync(indexHtmlPath)) {
  console.error(
    '[scenario-authenticator-ui] dist/index.html not found. Run `pnpm run build` before starting the UI server.'
  );
  process.exit(1);
}

app.use(express.static(distDir, { index: false }));

// Auth SPA routes
const authRoutes = [
  '/auth',
  '/auth/',
  '/auth/login',
  '/auth/signup',
  '/auth/register',
  '/auth/forgot-password',
];

authRoutes.forEach((route) => {
  app.get(route, (_req, res) => {
    res.sendFile(indexHtmlPath);
  });
});

app.get('/auth/*', (_req, res) => {
  res.sendFile(indexHtmlPath);
});

// Legacy shortcuts
app.get(['/login', '/signup', '/forgot-password'], (_req, res) => {
  res.redirect(301, '/auth/login');
});

app.get('/user-card.js', (_req, res) => {
  res.sendFile(userCardScriptPath);
});

app.get('/config', async (_req, res) => {
  const port = await resolveApiPort();
  if (!port) {
    res.status(500).json({
      error: 'AUTH_API_PORT_UNDEFINED',
      message:
        'Authentication API port is not configured. Ensure the scenario is started via the lifecycle system.',
    });
    return;
  }

  res.json({
    apiUrl: `http://localhost:${port}`,
    contactBookUrl: CONTACT_BOOK_URL || null,
    version: '2.0.0',
    service: 'authentication-ui',
  });
});

app.get('/scenarios', async (_req, res) => {
  const now = Date.now();
  if (cachedScenarios && now - cachedScenariosFetchedAt < SCENARIO_CACHE_TTL_MS) {
    res.json({ scenarios: cachedScenarios, cached: true });
    return;
  }

  try {
    const scenarios = await loadScenariosFromCli();
    cachedScenarios = scenarios;
    cachedScenariosFetchedAt = Date.now();
    res.json({ scenarios });
  } catch (error) {
    console.error('[scenario-authenticator-ui] Failed to list scenarios', error);
    res.status(500).json({ error: 'FAILED_TO_LIST_SCENARIOS' });
  }
});

app.get('/', (_req, res) => {
  res.sendFile(dashboardPagePath);
});

app.get('/dashboard', (_req, res) => {
  res.sendFile(dashboardPagePath);
});

app.get('/admin', (_req, res) => {
  res.sendFile(adminPagePath);
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
