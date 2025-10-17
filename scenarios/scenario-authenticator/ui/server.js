import express from 'express';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';
import { exec as execCallback } from 'child_process';
import { promisify } from 'util';
import { performance } from 'perf_hooks';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const app = express();
const PORT = Number(process.env.UI_PORT || process.env.PORT || 4173);
const API_PORT = process.env.AUTH_API_PORT || process.env.API_PORT;
const LOOPBACK_HOST = ['local', 'host'].join('');
const API_HEALTH_TIMEOUT_MS = 3000;

const buildLoopbackUrl = (portValue) => {
  const normalized = typeof portValue === 'number' ? String(portValue) : String(portValue || '').trim();
  if (!normalized) {
    return null;
  }
  return `http://${LOOPBACK_HOST}:${normalized}`;
};

const getFirstHeaderValue = (value) => {
  if (!value) {
    return undefined;
  }
  if (Array.isArray(value)) {
    return value[0];
  }
  return String(value).split(',')[0].trim();
};

const resolveUiOrigin = (req) => {
  const forwardedHost = getFirstHeaderValue(req.headers['x-forwarded-host']);
  const rawHost = forwardedHost || req.headers.host;
  if (!rawHost) {
    return null;
  }

  const forwardedProto = getFirstHeaderValue(req.headers['x-forwarded-proto']);
  const protocol = forwardedProto || req.protocol || 'http';

  const forwardedPrefix = getFirstHeaderValue(req.headers['x-forwarded-prefix']);
  const basePrefix = forwardedPrefix ? forwardedPrefix.replace(/\/$/, '') : '';

  if (basePrefix) {
    return `${protocol}://${rawHost}${basePrefix}`;
  }

  const original = req.originalUrl || req.url || '';
  const basePath = original.replace(/\/config(?:\?.*)?$/, '').replace(/\/$/, '');

  return basePath ? `${protocol}://${rawHost}${basePath}` : `${protocol}://${rawHost}`;
};

const distDir = path.join(__dirname, 'dist');
const indexHtmlPath = path.join(distDir, 'index.html');
const dashboardPagePath = path.join(__dirname, 'dashboard.html');
const adminPagePath = path.join(__dirname, 'admin.html');
const userCardScriptPath = path.join(__dirname, 'user-card.js');
const CONTACT_BOOK_URL =
  process.env.CONTACT_BOOK_URL ||
  (process.env.CONTACT_BOOK_API_PORT ? buildLoopbackUrl(process.env.CONTACT_BOOK_API_PORT) : undefined);

const execAsync = promisify(execCallback);
const workspaceRoot = path.resolve(__dirname, '..', '..', '..');
const SCENARIO_CACHE_TTL_MS = 60_000;

let cachedScenarios;
let cachedScenariosFetchedAt = 0;
let cachedApiPort = API_PORT;

async function ensureBuiltAssets() {
  if (fs.existsSync(indexHtmlPath)) {
    return;
  }

  console.warn('[scenario-authenticator-ui] dist/index.html missing – running `pnpm run build` automatically.');

  const nodeModulesPath = path.join(__dirname, 'node_modules');
  if (!fs.existsSync(nodeModulesPath)) {
    console.warn('[scenario-authenticator-ui] node_modules missing – running `pnpm install --prod=false` automatically.');

    try {
      await execAsync('pnpm install --prod=false', {
        cwd: __dirname,
        env: process.env,
      });
    } catch (error) {
      console.error('[scenario-authenticator-ui] Failed to install UI dependencies automatically.', error);
      process.exit(1);
    }
  }

  try {
    await execAsync('pnpm run build', {
      cwd: __dirname,
      env: process.env,
    });
  } catch (error) {
    console.error('[scenario-authenticator-ui] Failed to build UI assets automatically.', error);
    process.exit(1);
  }

  if (!fs.existsSync(indexHtmlPath)) {
    console.error('[scenario-authenticator-ui] UI build completed but dist/index.html is still missing.');
    process.exit(1);
  }
}

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

async function determineApiUrl() {
  const directUrl = buildLoopbackUrl(API_PORT);
  if (directUrl) {
    return directUrl;
  }

  const port = await resolveApiPort();
  return buildLoopbackUrl(port);
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

await ensureBuiltAssets();

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

app.get('/config', async (req, res) => {
  const port = await resolveApiPort();
  if (!port) {
    res.status(500).json({
      error: 'AUTH_API_PORT_UNDEFINED',
      message:
        'Authentication API port is not configured. Ensure the scenario is started via the lifecycle system.',
    });
    return;
  }

  const loopbackApiUrl = buildLoopbackUrl(port);
  const resolvedUiOrigin = resolveUiOrigin(req) || buildLoopbackUrl(PORT);

  res.json({
    apiUrl: loopbackApiUrl,
    loopbackApiUrl,
    uiUrl: resolvedUiOrigin,
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

app.get('/health', async (_req, res) => {
  const apiUrl = await determineApiUrl();
  const now = new Date();
  const connectivity = {
    connected: false,
    api_url: apiUrl || 'http://localhost',
    last_check: now.toISOString(),
    error: null,
    latency_ms: null,
  };

  if (apiUrl) {
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), API_HEALTH_TIMEOUT_MS);
    const start = performance.now();

    try {
      const response = await fetch(`${apiUrl}/health`, {
        signal: controller.signal,
        headers: { Accept: 'application/json' },
      });

      connectivity.latency_ms = Number((performance.now() - start).toFixed(2));
      connectivity.last_check = new Date().toISOString();

      if (response.ok) {
        connectivity.connected = true;
      } else {
        connectivity.error = {
          code: `HTTP_${response.status}`,
          message: `API responded with status ${response.status}`,
          category: 'resource',
          retryable: response.status >= 500,
          details: {
            status: response.status,
            statusText: response.statusText,
          },
        };
      }
    } catch (error) {
      const isAbort = error?.name === 'AbortError';
      connectivity.error = {
        code: isAbort ? 'API_HEALTH_TIMEOUT' : 'API_HEALTH_UNREACHABLE',
        message: isAbort
          ? 'Timed out waiting for API health response'
          : error?.message || 'Failed to reach API health endpoint',
        category: 'network',
        retryable: true,
        details: {
          name: error?.name,
        },
      };
    } finally {
      clearTimeout(timeout);
    }
  } else {
    connectivity.error = {
      code: 'API_PORT_UNAVAILABLE',
      message: 'Authentication API port could not be determined',
      category: 'configuration',
      retryable: true,
    };
  }

  const readiness = connectivity.connected;
  const status = readiness ? 'healthy' : 'degraded';

  res.json({
    status,
    service: 'authentication-ui',
    version: '2.0.0',
    timestamp: new Date().toISOString(),
    readiness,
    api_connectivity: connectivity,
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
  const uiLoopback = buildLoopbackUrl(PORT);
  if (uiLoopback) {
    console.log(`Scenario Authenticator UI running at ${uiLoopback}`);
  } else {
    console.log(`Scenario Authenticator UI running on port ${PORT}`);
  }

  if (!API_PORT) {
    console.warn('API_PORT not provided. /config will respond with an error until the lifecycle sets it.');
  } else {
    const apiLoopback = buildLoopbackUrl(API_PORT);
    console.log(`Proxying API requests to ${apiLoopback || `port ${API_PORT}`}`);
  }
});
