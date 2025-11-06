import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child';
import { buildApiUrl, resolveApiBase } from '@vrooli/api-base';

const BRIDGE_FLAG = '__fileToolsBridgeInitialized';

function bootstrapIframeBridge() {
  if (typeof window === 'undefined') {
    return;
  }

  if (window.parent === window || window[BRIDGE_FLAG]) {
    return;
  }

  let parentOrigin;
  try {
    if (document.referrer) {
      parentOrigin = new URL(document.referrer).origin;
    }
  } catch (error) {
    console.warn('[file-tools] Unable to derive parent origin for iframe bridge', error);
  }

  initIframeBridgeChild({ parentOrigin, appId: 'file-tools' });
  window[BRIDGE_FLAG] = true;
}

function parseBootstrapData() {
  const bootstrapEl = document.getElementById('file-tools-bootstrap');
  if (!bootstrapEl) {
    return {};
  }

  try {
    const payload = bootstrapEl.textContent ? JSON.parse(bootstrapEl.textContent) : {};
    return payload && typeof payload === 'object' ? payload : {};
  } catch (error) {
    console.warn('[file-tools] Failed to parse bootstrap payload', error);
    return {};
  }
}

bootstrapIframeBridge();

const bootstrap = parseBootstrapData();
const state = {
  token: bootstrap.token || 'API_TOKEN_PLACEHOLDER',
  explicitApiBase: bootstrap.explicitApiBase || undefined,
  defaultApiPort: bootstrap.defaultApiPort || '15458',
  headline: {
    optimizations: 0,
    savingsBytes: 0,
    services: 1,
  },
};

const apiOrigins = (() => {
  const apiRoot = resolveApiBase({
    explicitUrl: state.explicitApiBase,
    defaultPort: state.defaultApiPort,
    appendSuffix: false,
  }).replace(/\/+$/, '');

  const apiV1 = resolveApiBase({
    explicitUrl: state.explicitApiBase,
    defaultPort: state.defaultApiPort,
    appendSuffix: true,
  }).replace(/\/+$/, '');

  return {
    root: apiRoot,
    api: apiV1,
  };
})();

const endpoints = {
  health: buildApiUrl('/health', { baseUrl: apiOrigins.root, appendSuffix: false }),
  duplicates: buildApiUrl('/files/duplicates/detect', { baseUrl: apiOrigins.api, appendSuffix: false }),
  organize: buildApiUrl('/files/organize', { baseUrl: apiOrigins.api, appendSuffix: false }),
};

const selectors = {
  apiStatus: document.getElementById('api-status'),
  apiDetails: document.getElementById('api-details'),
  integrityList: document.getElementById('integrity-list'),
  duplicateOutput: document.getElementById('duplicate-output'),
  organizationOutput: document.getElementById('organization-output'),
  metricServices: document.getElementById('metric-services'),
  metricOptimizations: document.getElementById('metric-optimizations'),
  metricSavings: document.getElementById('metric-savings'),
  roadmapGrid: document.getElementById('roadmap-grid'),
};

function ensureElement(element, label) {
  if (!element) {
    throw new Error(`Missing element: ${label}`);
  }
  return element;
}

Object.entries(selectors).forEach(([key, element]) => ensureElement(element, key));

function formatBytes(value) {
  const bytes = Number(value);
  if (!Number.isFinite(bytes) || bytes <= 0) {
    return '0 B';
  }
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  const index = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1);
  const normalized = bytes / Math.pow(1024, index);
  return `${normalized.toFixed(normalized >= 10 || index === 0 ? 0 : 1)} ${units[index]}`;
}

async function requestJson(url, options = {}) {
  const init = {
    method: options.method || 'GET',
    headers: {
      Accept: 'application/json',
      ...(options.headers || {}),
    },
  };

  if (state.token && !options.skipAuth) {
    init.headers.Authorization = `Bearer ${state.token}`;
  }

  if (options.body) {
    init.body = JSON.stringify(options.body);
    init.headers['Content-Type'] = 'application/json';
  }

  const response = await fetch(url, init);
  const text = await response.text();
  let payload = null;

  if (text) {
    try {
      payload = JSON.parse(text);
    } catch (error) {
      console.warn('[file-tools] Non-JSON response received', { url, error });
    }
  }

  if (!response.ok) {
    const errorMessage = payload && payload.error ? payload.error : response.statusText;
    const error = new Error(errorMessage || 'Request failed');
    error.payload = payload;
    error.status = response.status;
    throw error;
  }

  return payload;
}

function renderIntegritySignals(data) {
  const entries = [];

  if (data?.database === 'connected') {
    entries.push('Database connectivity verified');
  }
  if (data?.service) {
    entries.push(`Service: ${data.service}`);
  }
  if (data?.version) {
    entries.push(`Version ${data.version}`);
  }
  entries.push('Real-time checksum monitoring ready');
  entries.push('Proxy-aware UI bridge active');

  selectors.integrityList.innerHTML = entries
    .map((entry) => `<li>${entry}</li>`)
    .join('');
}

function updateHeadlineMetrics() {
  selectors.metricServices.textContent = state.headline.services >= 3 ? 'API • CLI • UI' : `${state.headline.services}/3 ready`;
  selectors.metricOptimizations.textContent = `${state.headline.optimizations}`;
  selectors.metricSavings.textContent = formatBytes(state.headline.savingsBytes);
}

function setStatus(element, status, tone = 'neutral') {
  const classList = ['status', `status-${tone}`];
  element.className = classList.join(' ');
  element.textContent = status;
}

async function refreshHealth() {
  setStatus(selectors.apiStatus, 'Checking…', 'neutral');
  selectors.apiDetails.textContent = 'Awaiting response';

  try {
    const payload = await requestJson(endpoints.health, { skipAuth: true });
    const data = payload?.data || {};

    setStatus(selectors.apiStatus, data.status === 'healthy' ? 'Healthy' : 'Degraded', data.status === 'healthy' ? 'ok' : 'warn');
    selectors.apiDetails.textContent = JSON.stringify(data, null, 2);
    renderIntegritySignals(data);

    state.headline.services = 3;
    updateHeadlineMetrics();
  } catch (error) {
    console.error('[file-tools] Health check failed', error);
    setStatus(selectors.apiStatus, 'Unavailable', 'err');
    selectors.apiDetails.textContent = JSON.stringify({ error: error.message, endpoint: endpoints.health }, null, 2);
    selectors.integrityList.innerHTML = '<li>Unable to reach API health endpoint.</li>';
  }
}

function summarizeDuplicates(payload) {
  if (!payload || typeof payload !== 'object') {
    return 'No duplicate insights available.';
  }

  const groups = payload.duplicate_groups || [];
  const summary = {
    filesScanned: payload.files_scanned || 0,
    duplicateGroups: groups.length,
    savingsBytes: payload.total_savings_bytes || 0,
  };

  state.headline.optimizations = Math.max(summary.duplicateGroups, state.headline.optimizations);
  state.headline.savingsBytes = Math.max(summary.savingsBytes, state.headline.savingsBytes);
  updateHeadlineMetrics();

  const topGroups = groups
    .filter((group) => Array.isArray(group?.files))
    .slice(0, 3)
    .map((group) => ({
      checksum: group.checksum,
      files: group.files.map((file) => file.path),
    }));

  return JSON.stringify(
    {
      scanId: payload.scan_id,
      filesScanned: summary.filesScanned,
      duplicateGroups: summary.duplicateGroups,
      potentialSavings: formatBytes(summary.savingsBytes),
      samples: topGroups,
    },
    null,
    2,
  );
}

async function runDuplicateScan() {
  selectors.duplicateOutput.textContent = 'Analyzing duplicates in ./temp …';

  try {
    const payload = await requestJson(endpoints.duplicates, {
      method: 'POST',
      body: {
        scan_paths: ['./temp'],
        detection_method: 'checksum',
        options: {
          similarity_threshold: 0.92,
          include_hidden: false,
          file_extensions: [],
        },
      },
    });

    selectors.duplicateOutput.textContent = summarizeDuplicates(payload?.data || payload);
  } catch (error) {
    console.error('[file-tools] Duplicate analysis failed', error);
    selectors.duplicateOutput.textContent = JSON.stringify(
      {
        error: error.message,
        recommendation: 'Verify ./temp exists and scenario has populated sample assets.',
      },
      null,
      2,
    );
  }
}

function summarizeOrganizationPlan(payload) {
  const plan = payload?.organization_plan || [];
  const conflicts = payload?.conflicts || [];

  return JSON.stringify(
    {
      operationId: payload?.operation_id,
      plannedMoves: plan.length,
      conflictingTargets: conflicts.length,
      sample: plan.slice(0, 3),
    },
    null,
    2,
  );
}

async function planSmartOrganization() {
  selectors.organizationOutput.textContent = 'Generating organization blueprint for ./temp …';

  try {
    const payload = await requestJson(endpoints.organize, {
      method: 'POST',
      body: {
        source_path: './temp',
        destination_path: './temp/organized-preview',
        organization_rules: [
          { rule_type: 'by_type', parameters: {} },
          { rule_type: 'by_date', parameters: {} },
        ],
        options: {
          dry_run: true,
          create_directories: true,
          handle_conflicts: 'rename',
        },
      },
    });

    selectors.organizationOutput.textContent = summarizeOrganizationPlan(payload?.data || payload);
  } catch (error) {
    console.error('[file-tools] Organization planning failed', error);
    selectors.organizationOutput.textContent = JSON.stringify(
      {
        error: error.message,
        recommendation: 'Ensure ./temp contains sample files. The dry-run never mutates data.',
      },
      null,
      2,
    );
  }
}

function renderRoadmap() {
  const roadmap = [
    {
      title: 'Edge Location Sync',
      status: 'In Validation',
      description: 'Bi-directional sync to regional storage nodes with delta compression.',
      eta: 'Target: Q1',
    },
    {
      title: 'Predictive Integrity',
      status: 'Design Complete',
      description: 'AI regression anticipates corruption patterns 72h before failure.',
      eta: 'Target: Q2',
    },
    {
      title: 'Zero-Trust Transfers',
      status: 'Prototype Ready',
      description: 'Policy-aware pipelines broker secure file exchange between scenarios.',
      eta: 'Target: Q3',
    },
    {
      title: 'Usage Monetization',
      status: 'Ideation',
      description: 'Billing adapters to expose compression, integrity, and archiving as premium add-ons.',
      eta: 'Discovery',
    },
  ];

  selectors.roadmapGrid.innerHTML = roadmap
    .map(
      (item) => `
        <article class="roadmap-card">
          <p class="eyebrow">${item.status}</p>
          <h3>${item.title}</h3>
          <p>${item.description}</p>
          <p class="subtitle">${item.eta}</p>
        </article>
      `,
    )
    .join('');
}

function bindActions() {
  const handlers = [
    ['refresh-health', refreshHealth],
    ['analyze-duplicates', runDuplicateScan],
    ['plan-organization', planSmartOrganization],
  ];

  handlers.forEach(([action, handler]) => {
    const button = document.querySelector(`[data-action="${action}"]`);
    if (!button) {
      console.warn(`[file-tools] Missing action button: ${action}`);
      return;
    }
    button.addEventListener('click', () => handler());
  });
}

function init() {
  renderRoadmap();
  updateHeadlineMetrics();
  bindActions();
  refreshHealth();
}

init();
