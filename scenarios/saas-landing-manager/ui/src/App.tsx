import { useCallback, useEffect, useMemo, useState } from 'react';
import * as Tabs from '@radix-ui/react-tabs';
import * as Dialog from '@radix-ui/react-dialog';
import clsx from 'clsx';

type SaaSScenario = {
  id: string;
  scenario_name: string;
  display_name: string;
  description: string;
  saas_type: string;
  industry: string;
  revenue_potential: string;
  has_landing_page: boolean;
  landing_page_url: string;
  last_scan: string;
  confidence_score: number;
  metadata: Record<string, unknown>;
};

type DashboardSummary = {
  total_pages: number;
  active_ab_tests: number;
  average_conversion_rate: number;
  scenarios: SaaSScenario[];
};

type Template = {
  id: string;
  name: string;
  category: string;
  saas_type: string;
  industry: string;
  preview_url?: string;
  usage_count?: number;
  rating?: number;
};

type ScanResponse = {
  total_scenarios: number;
  saas_scenarios: number;
  newly_detected: number;
  scenarios: SaaSScenario[];
};

type GenerateResponse = {
  landing_page_id: string;
  preview_url: string;
  deployment_status: string;
  ab_test_variants: string[];
};

type OperationNotice = {
  variant: 'success' | 'error';
  message: string;
  details?: string;
};

const win = typeof window !== 'undefined' ? window : undefined;
const basePath = win?.__SAAS_LANDING_MANAGER_BASE_PATH ?? '';

function ensureLeadingSlash(path: string) {
  if (!path.startsWith('/')) {
    return `/${path}`;
  }
  return path;
}

function buildUrl(path: string) {
  return `${basePath}${ensureLeadingSlash(path)}`;
}

async function requestJson<T>(path: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers);
  if (!headers.has('accept')) {
    headers.set('accept', 'application/json');
  }

  const hasBody = init?.body !== undefined && init.body !== null && init.body !== '';
  if (hasBody && !headers.has('content-type')) {
    headers.set('content-type', 'application/json');
  }

  const response = await fetch(buildUrl(path), {
    ...init,
    headers,
  });

  const contentType = response.headers.get('content-type') ?? '';
  const payload = contentType.includes('application/json') ? await response.json() : await response.text();

  if (!response.ok) {
    const errorMessage = typeof payload === 'string' && payload ? payload : response.statusText;
    throw new Error(errorMessage || 'Request failed');
  }

  return payload as T;
}

function formatCurrency(value: string) {
  if (!value) {
    return 'N/A';
  }

  const numeric = Number.parseFloat(value.replace(/[^0-9.]/g, ''));
  if (Number.isNaN(numeric)) {
    return value;
  }

  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    maximumFractionDigits: 0,
  }).format(numeric);
}

function formatDate(timestamp: string) {
  if (!timestamp) {
    return 'Unknown';
  }

  const date = new Date(timestamp);
  if (Number.isNaN(date.getTime())) {
    return timestamp;
  }

  return date.toLocaleString();
}

function formatRelativeTime(timestamp: string) {
  if (!timestamp) {
    return 'N/A';
  }

  const target = new Date(timestamp).getTime();
  if (Number.isNaN(target)) {
    return timestamp;
  }

  const diffMs = Date.now() - target;
  const diffMinutes = Math.round(diffMs / 60000);
  if (diffMinutes < 1) {
    return 'just now';
  }
  if (diffMinutes < 60) {
    return `${diffMinutes}m ago`;
  }
  const diffHours = Math.round(diffMinutes / 60);
  if (diffHours < 24) {
    return `${diffHours}h ago`;
  }
  const diffDays = Math.round(diffHours / 24);
  if (diffDays < 7) {
    return `${diffDays}d ago`;
  }
  const diffWeeks = Math.round(diffDays / 7);
  if (diffWeeks < 5) {
    return `${diffWeeks}w ago`;
  }
  const diffMonths = Math.round(diffDays / 30);
  if (diffMonths < 12) {
    return `${diffMonths}mo ago`;
  }
  const diffYears = Math.round(diffDays / 365);
  return `${diffYears}y ago`;
}

function extractCharacteristics(metadata: Record<string, unknown>) {
  const raw = metadata ? (metadata['characteristics'] as unknown) : undefined;
  if (Array.isArray(raw)) {
    return raw
      .filter((entry) => typeof entry === 'string')
      .map((value) => value.toString());
  }
  return [];
}

const emptyDashboard: DashboardSummary = {
  total_pages: 0,
  active_ab_tests: 0,
  average_conversion_rate: 0,
  scenarios: [],
};

const SaasTypeLabels: Record<string, string> = {
  b2b_tool: 'B2B Tool',
  b2c_app: 'B2C App',
  api_service: 'API Service',
  marketplace: 'Marketplace',
};

const ConfidenceDescriptions: Record<string, string> = {
  High: 'Strong signal and recent scan',
  Medium: 'Moderate confidence, consider review',
  Low: 'Limited data, investigate manually',
};

function confidenceBand(score: number) {
  if (score >= 1.5) {
    return 'High';
  }
  if (score >= 1.0) {
    return 'Medium';
  }
  return 'Low';
}

function scenarioLabel(scenario: SaaSScenario) {
  return scenario.display_name || scenario.scenario_name || scenario.id;
}

const TabItems: { value: string; label: string; description: string }[] = [
  { value: 'overview', label: 'Overview', description: 'Portfolio metrics and top opportunities' },
  { value: 'templates', label: 'Templates', description: 'Landing page blueprints by industry and strategy' },
  { value: 'operations', label: 'Operations', description: 'Scan, generate, and deploy landing pages' },
];

const App = () => {
  const [dashboard, setDashboard] = useState<DashboardSummary>(emptyDashboard);
  const [dashboardLoading, setDashboardLoading] = useState(true);
  const [dashboardError, setDashboardError] = useState<string | null>(null);
  const [templates, setTemplates] = useState<Template[]>([]);
  const [templatesLoading, setTemplatesLoading] = useState(true);
  const [templatesError, setTemplatesError] = useState<string | null>(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [activeTab, setActiveTab] = useState<string>('overview');
  const [operationNotice, setOperationNotice] = useState<OperationNotice | null>(null);
  const [scanInFlight, setScanInFlight] = useState(false);
  const [generateInFlight, setGenerateInFlight] = useState(false);
  const [generateDialogOpen, setGenerateDialogOpen] = useState(false);
  const [selectedScenarioId, setSelectedScenarioId] = useState<string>('');
  const [selectedTemplateId, setSelectedTemplateId] = useState<string>('');
  const [customHero, setCustomHero] = useState('');
  const [enableAbTesting, setEnableAbTesting] = useState(true);

  const scenarios = dashboard.scenarios ?? [];

  const refreshDashboard = useCallback(async (signal?: AbortSignal) => {
    setDashboardLoading(true);
    setDashboardError(null);
    try {
      const summary = await requestJson<DashboardSummary>('/api/v1/analytics/dashboard', {
        method: 'GET',
        signal,
      });
      const normalizedSummary: DashboardSummary = {
        ...summary,
        scenarios: Array.isArray(summary.scenarios) ? summary.scenarios : [],
      };
      setDashboard(normalizedSummary);
      setSelectedScenarioId((current) => current || normalizedSummary.scenarios[0]?.id || '');
    } catch (error) {
      if ((error as Error).name === 'AbortError') {
        return;
      }
      setDashboardError((error as Error).message || 'Unable to load analytics dashboard');
      setDashboard(emptyDashboard);
    } finally {
      setDashboardLoading(false);
    }
  }, []);

  const refreshTemplates = useCallback(async (signal?: AbortSignal) => {
    setTemplatesLoading(true);
    setTemplatesError(null);
    try {
      const response = await requestJson<{ templates: Template[] }>('/api/v1/templates', {
        method: 'GET',
        signal,
      });
      const normalizedTemplates = Array.isArray(response.templates) ? response.templates : [];
      setTemplates(normalizedTemplates);
      setSelectedTemplateId((current) => current || normalizedTemplates[0]?.id || '');
    } catch (error) {
      if ((error as Error).name === 'AbortError') {
        return;
      }
      setTemplatesError((error as Error).message || 'Unable to load templates');
      setTemplates([]);
    } finally {
      setTemplatesLoading(false);
    }
  }, []);

  useEffect(() => {
    const controller = new AbortController();
    refreshDashboard(controller.signal);
    return () => controller.abort();
  }, [refreshDashboard]);

  useEffect(() => {
    const controller = new AbortController();
    refreshTemplates(controller.signal);
    return () => controller.abort();
  }, [refreshTemplates]);

  const filteredScenarios = useMemo(() => {
    if (!searchTerm.trim()) {
      return scenarios.slice().sort((a, b) => b.confidence_score - a.confidence_score);
    }

    const term = searchTerm.toLowerCase();
    return scenarios
      .filter((scenario) =>
        [scenario.display_name, scenario.scenario_name, scenario.industry, scenario.saas_type]
          .filter(Boolean)
          .some((value) => value.toLowerCase().includes(term)),
      )
      .sort((a, b) => b.confidence_score - a.confidence_score);
  }, [scenarios, searchTerm]);

  const landingPagesGenerated = useMemo(
    () => scenarios.filter((scenario) => scenario.has_landing_page).length,
    [scenarios],
  );

  const averageConfidenceScore = useMemo(() => {
    if (scenarios.length === 0) {
      return 0;
    }
    const total = scenarios.reduce((sum, scenario) => sum + (scenario.confidence_score || 0), 0);
    return total / scenarios.length;
  }, [scenarios]);

  const topOpportunities = useMemo(() => filteredScenarios.slice(0, 5), [filteredScenarios]);

  const industryDistribution = useMemo(() => {
    const counts = new Map<string, number>();
    for (const scenario of scenarios) {
      const key = scenario.industry || 'Unspecified';
      counts.set(key, (counts.get(key) ?? 0) + 1);
    }

    return Array.from(counts.entries())
      .map(([industry, count]) => ({ industry, count }))
      .sort((a, b) => b.count - a.count)
      .slice(0, 4);
  }, [scenarios]);

  const templateGroups = useMemo(() => {
    const groups = new Map<string, Template[]>();
    for (const template of templates) {
      const key = template.category || 'general';
      if (!groups.has(key)) {
        groups.set(key, []);
      }
      groups.get(key)?.push(template);
    }

    return Array.from(groups.entries()).map(([category, entries]) => ({
      category,
      templates: entries.sort((a, b) => (b.usage_count ?? 0) - (a.usage_count ?? 0)),
    }));
  }, [templates]);

  const handleScan = useCallback(
    async (force: boolean) => {
      setScanInFlight(true);
      setOperationNotice(null);
      try {
        const payload = await requestJson<ScanResponse>('/api/v1/scenarios/scan', {
          method: 'POST',
          body: JSON.stringify({
            force_rescan: force,
          }),
        });
        setOperationNotice({
          variant: 'success',
          message: force ? 'Full rescan completed successfully.' : 'Scan completed successfully.',
          details: `${payload.saas_scenarios} SaaS opportunities detected (${payload.newly_detected} new).`,
        });
        await refreshDashboard();
      } catch (error) {
        setOperationNotice({
          variant: 'error',
          message: 'Scan failed',
          details: (error as Error).message,
        });
      } finally {
        setScanInFlight(false);
      }
    },
    [refreshDashboard],
  );

  const handleGenerate = useCallback(
    async (event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      if (!selectedScenarioId) {
        setOperationNotice({ variant: 'error', message: 'Select a scenario before generating a landing page.' });
        return;
      }

      setGenerateInFlight(true);
      setOperationNotice(null);

      try {
        const payload: Record<string, unknown> = {
          scenario_id: selectedScenarioId,
          enable_ab_testing: enableAbTesting,
        };

        if (selectedTemplateId) {
          payload.template_id = selectedTemplateId;
        }

        if (customHero.trim()) {
          payload.custom_content = {
            hero_title: customHero.trim(),
          };
        }

        const response = await requestJson<GenerateResponse>('/api/v1/landing-pages/generate', {
          method: 'POST',
          body: JSON.stringify(payload),
        });

        setOperationNotice({
          variant: 'success',
          message: 'Landing page generated',
          details: `Landing page ${response.landing_page_id} created${
            response.preview_url ? ` • Preview: ${response.preview_url}` : ''
          }`,
        });
        setGenerateDialogOpen(false);
        setCustomHero('');
        await refreshDashboard();
      } catch (error) {
        setOperationNotice({
          variant: 'error',
          message: 'Landing page generation failed',
          details: (error as Error).message,
        });
      } finally {
        setGenerateInFlight(false);
      }
    },
    [enableAbTesting, selectedScenarioId, selectedTemplateId, customHero, refreshDashboard],
  );

  const renderOverview = () => (
    <div className="content-section">
      <header className="section-header">
        <div>
          <h2>Portfolio performance</h2>
          <p className="section-subtitle">
            {dashboardLoading
              ? 'Refreshing SaaS portfolio metrics...'
              : `${scenarios.length} tracked scenarios • ${landingPagesGenerated} live landing pages`}
          </p>
        </div>
        <div className="header-actions">
          <button
            className="button button-secondary"
            type="button"
            onClick={() => refreshDashboard()}
            disabled={dashboardLoading}
          >
            {dashboardLoading ? 'Refreshing…' : 'Refresh metrics'}
          </button>
        </div>
      </header>

      {dashboardError && <Notice variant="error" title="Analytics unavailable" description={dashboardError} />}

      <section className="metrics-grid">
        <MetricCard
          title="Tracked Scenarios"
          value={scenarios.length}
          helper={`Avg. confidence ${(averageConfidenceScore || 0).toFixed(2)}`}
          accent="blue"
        />
        <MetricCard
          title="Landing Pages"
          value={landingPagesGenerated}
          helper={`${(dashboard.average_conversion_rate * 100).toFixed(1)}% avg. conversion`}
          accent="violet"
        />
        <MetricCard
          title="Active A/B tests"
          value={dashboard.active_ab_tests}
          helper={dashboard.active_ab_tests > 0 ? 'Experiments running' : 'No experiments yet'}
          accent="green"
        />
        <MetricCard
          title="Latest scan"
          value={
            scenarios.length > 0
              ? formatRelativeTime(scenarios[0].last_scan)
              : 'No scans recorded'
          }
          helper={scenarios.length > 0 ? formatDate(scenarios[0].last_scan) : 'Schedule a scan to populate data'}
          accent="amber"
        />
      </section>

      <section className="two-column">
        <div className="panel">
          <div className="panel-header">
            <h3>Top opportunities</h3>
            <span className="panel-meta">Sorted by confidence score</span>
          </div>
          <ul className="opportunity-list">
            {topOpportunities.length === 0 && <li className="empty-state">No SaaS opportunities detected yet.</li>}
            {topOpportunities.map((scenario) => {
              const confidence = confidenceBand(scenario.confidence_score);
              const confidenceDescription = ConfidenceDescriptions[confidence];
              const characteristics = extractCharacteristics(scenario.metadata).slice(0, 3);
              return (
                <li key={scenario.id} className="opportunity-item">
                  <div className="opportunity-header">
                    <div>
                      <p className="opportunity-name">{scenarioLabel(scenario)}</p>
                      <p className="opportunity-meta">
                        {(SaasTypeLabels[scenario.saas_type] ?? scenario.saas_type) || 'SaaS'} • {scenario.industry || 'General'}
                      </p>
                    </div>
                    <div className={clsx('confidence-pill', confidence.toLowerCase())}>{confidence}</div>
                  </div>
                  <div className="opportunity-body">
                    <p className="opportunity-description">{scenario.description || 'No description provided yet.'}</p>
                    <div className="opportunity-stats">
                      <span>{formatCurrency(scenario.revenue_potential)} revenue potential</span>
                      <span>Last scan {formatRelativeTime(scenario.last_scan)}</span>
                    </div>
                    {characteristics.length > 0 && (
                      <div className="characteristic-tags">
                        {characteristics.map((item) => (
                          <span key={item} className="tag">
                            {item}
                          </span>
                        ))}
                      </div>
                    )}
                    <p className="opportunity-description muted">{confidenceDescription}</p>
                  </div>
                </li>
              );
            })}
          </ul>
        </div>
        <div className="panel">
          <div className="panel-header">
            <h3>Portfolio mix</h3>
            <span className="panel-meta">Industry distribution</span>
          </div>
          <ul className="distribution-list">
            {industryDistribution.length === 0 && <li className="empty-state">No industry data available.</li>}
            {industryDistribution.map(({ industry, count }) => (
              <li key={industry}>
                <div>
                  <p className="distribution-industry">{industry}</p>
                  <p className="distribution-meta">{count} scenario{count === 1 ? '' : 's'}</p>
                </div>
                <span className="distribution-count">{Math.round((count / Math.max(scenarios.length, 1)) * 100)}%</span>
              </li>
            ))}
          </ul>
        </div>
      </section>

      <section className="panel">
        <div className="panel-header">
          <div>
            <h3>Scenario registry</h3>
            <p className="panel-meta">Search and inspect detected SaaS scenarios.</p>
          </div>
          <input
            type="search"
            className="input"
            placeholder="Search by name, type, or industry"
            value={searchTerm}
            onChange={(event) => setSearchTerm(event.target.value)}
          />
        </div>
        <ScenarioTable scenarios={filteredScenarios} />
      </section>
    </div>
  );

  const renderTemplates = () => (
    <div className="content-section">
      <header className="section-header">
        <div>
          <h2>Template library</h2>
          <p className="section-subtitle">
            Curated landing page templates optimized for SaaS conversion flows.
          </p>
        </div>
        <div className="header-actions">
          <button
            className="button button-secondary"
            type="button"
            onClick={() => refreshTemplates()}
            disabled={templatesLoading}
          >
            {templatesLoading ? 'Refreshing…' : 'Refresh templates'}
          </button>
        </div>
      </header>

      {templatesError && <Notice variant="error" title="Template library unavailable" description={templatesError} />}

      <div className="template-groups">
        {templateGroups.length === 0 && !templatesLoading && (
          <div className="empty-state large">No templates have been configured yet.</div>
        )}
        {templateGroups.map(({ category, templates: entries }) => (
          <article key={category} className="template-group">
            <header>
              <h3>{category.replace(/_/g, ' ')}</h3>
              <span className="panel-meta">
                {entries.length} template{entries.length === 1 ? '' : 's'}
              </span>
            </header>
            <div className="template-grid">
              {entries.map((template) => (
                <TemplateCard
                  key={template.id}
                  template={template}
                  selected={template.id === selectedTemplateId}
                  onSelect={() => setSelectedTemplateId(template.id)}
                />
              ))}
            </div>
          </article>
        ))}
      </div>
    </div>
  );

  const renderOperations = () => (
    <div className="content-section">
      <header className="section-header">
        <div>
          <h2>Lifecycle operations</h2>
          <p className="section-subtitle">Run scans, generate landing pages, and monitor progress.</p>
        </div>
      </header>

      {operationNotice && (
        <Notice variant={operationNotice.variant} title={operationNotice.message} description={operationNotice.details} />
      )}

      <section className="panel">
        <div className="panel-header operations">
          <div>
            <h3>Scenario discovery</h3>
            <p className="panel-meta">Keep the SaaS registry fresh by running targeted or full scans.</p>
          </div>
          <div className="op-actions">
            <button
              type="button"
              className="button"
              onClick={() => handleScan(false)}
              disabled={scanInFlight}
            >
              {scanInFlight ? 'Scanning…' : 'Quick scan'}
            </button>
            <button
              type="button"
              className="button button-secondary"
              onClick={() => handleScan(true)}
              disabled={scanInFlight}
            >
              {scanInFlight ? 'Scanning…' : 'Force rescan'}
            </button>
          </div>
        </div>
        <ul className="scan-summary">
          <li>
            <span>Total tracked scenarios</span>
            <strong>{scenarios.length}</strong>
          </li>
          <li>
            <span>Landing pages deployed</span>
            <strong>{landingPagesGenerated}</strong>
          </li>
          <li>
            <span>Average confidence</span>
            <strong>{averageConfidenceScore.toFixed(2)}</strong>
          </li>
        </ul>
      </section>

      <section className="panel">
        <div className="panel-header operations">
          <div>
            <h3>Landing page generation</h3>
            <p className="panel-meta">Produce a conversion-ready landing page for any SaaS scenario.</p>
          </div>
          <Dialog.Root open={generateDialogOpen} onOpenChange={setGenerateDialogOpen}>
            <Dialog.Trigger asChild>
              <button type="button" className="button">
                New landing page
              </button>
            </Dialog.Trigger>
            <Dialog.Portal>
              <Dialog.Overlay className="dialog-overlay" />
              <Dialog.Content className="dialog-content">
                <Dialog.Title>Create landing page</Dialog.Title>
                <Dialog.Description>
                  Select the scenario to target and optionally override hero copy before generating.
                </Dialog.Description>
                <form className="dialog-form" onSubmit={handleGenerate}>
                  <label className="field">
                    <span>Scenario</span>
                    <select
                      required
                      value={selectedScenarioId}
                      onChange={(event) => setSelectedScenarioId(event.target.value)}
                    >
                      {scenarios.map((scenario) => (
                        <option key={scenario.id} value={scenario.id}>
                          {scenarioLabel(scenario)}
                        </option>
                      ))}
                    </select>
                  </label>

                  <label className="field">
                    <span>Template</span>
                    <select
                      value={selectedTemplateId}
                      onChange={(event) => setSelectedTemplateId(event.target.value)}
                    >
                      {templates.map((template) => (
                        <option key={template.id} value={template.id}>
                          {template.name}
                        </option>
                      ))}
                    </select>
                  </label>

                  <label className="field">
                    <span>Hero headline override</span>
                    <input
                      type="text"
                      placeholder="Optional headline to personalise the hero section"
                      value={customHero}
                      onChange={(event) => setCustomHero(event.target.value)}
                    />
                  </label>

                  <label className="field checkbox">
                    <input
                      type="checkbox"
                      checked={enableAbTesting}
                      onChange={(event) => setEnableAbTesting(event.target.checked)}
                    />
                    <span>Enable A/B testing variants</span>
                  </label>

                  <div className="dialog-actions">
                    <Dialog.Close asChild>
                      <button type="button" className="button button-secondary" disabled={generateInFlight}>
                        Cancel
                      </button>
                    </Dialog.Close>
                    <button type="submit" className="button" disabled={generateInFlight}>
                      {generateInFlight ? 'Generating…' : 'Generate landing page'}
                    </button>
                  </div>
                </form>
              </Dialog.Content>
            </Dialog.Portal>
          </Dialog.Root>
        </div>
        <p className="panel-meta">
          Generated pages are stored under each scenario's <code>landing/</code> directory and can be deployed via the CLI or Claude Code agent.
        </p>
      </section>
    </div>
  );

  return (
    <div className="app-shell">
      <header className="app-header">
        <div>
          <h1>SaaS Landing Page Manager</h1>
          <p>Monitor SaaS scenarios, deploy landing pages, and keep your marketing engine compounding.</p>
        </div>
        <div className="header-summary">
          <span>{scenarios.length} scenarios tracked</span>
          <span>{landingPagesGenerated} landing pages live</span>
        </div>
      </header>

      <Tabs.Root className="tabs-root" value={activeTab} onValueChange={setActiveTab}>
        <Tabs.List className="tabs-list" aria-label="SaaS Landing Manager views">
          {TabItems.map((tab) => (
            <Tabs.Trigger key={tab.value} className="tabs-trigger" value={tab.value}>
              <span>{tab.label}</span>
              <small>{tab.description}</small>
            </Tabs.Trigger>
          ))}
        </Tabs.List>

        <Tabs.Content value="overview" className="tabs-content">
          {renderOverview()}
        </Tabs.Content>

        <Tabs.Content value="templates" className="tabs-content">
          {renderTemplates()}
        </Tabs.Content>

        <Tabs.Content value="operations" className="tabs-content">
          {renderOperations()}
        </Tabs.Content>
      </Tabs.Root>
    </div>
  );
};

const Notice = ({
  variant,
  title,
  description,
}: {
  variant: OperationNotice['variant'];
  title: string;
  description?: string;
}) => (
  <div className={clsx('notice', variant)}>
    <div>
      <strong>{title}</strong>
      {description && <p>{description}</p>}
    </div>
  </div>
);

const MetricCard = ({
  title,
  value,
  helper,
  accent,
}: {
  title: string;
  value: number | string;
  helper?: string;
  accent: 'blue' | 'violet' | 'green' | 'amber';
}) => (
  <article className={clsx('metric-card', accent)}>
    <header>{title}</header>
    <strong>{typeof value === 'number' ? value.toLocaleString() : value}</strong>
    {helper && <p>{helper}</p>}
  </article>
);

const ScenarioTable = ({ scenarios }: { scenarios: SaaSScenario[] }) => (
  <div className="table-wrapper">
    <table className="scenario-table">
      <thead>
        <tr>
          <th scope="col">Scenario</th>
          <th scope="col">Type</th>
          <th scope="col">Industry</th>
          <th scope="col">Confidence</th>
          <th scope="col">Opportunity</th>
          <th scope="col">Landing page</th>
          <th scope="col">Last scan</th>
        </tr>
      </thead>
      <tbody>
        {scenarios.length === 0 && (
          <tr>
            <td colSpan={7} className="empty-state">
              No scenarios match your filters yet.
            </td>
          </tr>
        )}
        {scenarios.map((scenario) => {
          const confidence = confidenceBand(scenario.confidence_score);
          return (
            <tr key={scenario.id}>
              <td data-label="Scenario">
                <div className="scenario-name">{scenarioLabel(scenario)}</div>
                <div className="scenario-id">{scenario.scenario_name}</div>
              </td>
              <td data-label="Type">{(SaasTypeLabels[scenario.saas_type] ?? scenario.saas_type) || 'SaaS'}</td>
              <td data-label="Industry">{scenario.industry || 'Unspecified'}</td>
              <td data-label="Confidence">
                <span className={clsx('confidence-pill', confidence.toLowerCase())}>{confidence}</span>
              </td>
              <td data-label="Opportunity">{formatCurrency(scenario.revenue_potential)}</td>
              <td data-label="Landing page">
                {scenario.has_landing_page ? (
                  <a href={scenario.landing_page_url || '#'} target="_blank" rel="noreferrer">
                    View
                  </a>
                ) : (
                  <span className="status-pill">Pending</span>
                )}
              </td>
              <td data-label="Last scan">{formatRelativeTime(scenario.last_scan)}</td>
            </tr>
          );
        })}
      </tbody>
    </table>
  </div>
);

const TemplateCard = ({
  template,
  selected,
  onSelect,
}: {
  template: Template;
  selected: boolean;
  onSelect: () => void;
}) => (
  <button type="button" className={clsx('template-card', { selected })} onClick={onSelect}>
    <div className="template-card-header">
      <h4>{template.name}</h4>
      <span>{template.saas_type?.replace(/_/g, ' ') || 'General'}</span>
    </div>
    <p>{template.industry ? `${template.industry} • ${template.category}` : template.category}</p>
    <div className="template-card-footer">
      {template.usage_count !== undefined && <span>{template.usage_count} deployments</span>}
      {template.rating !== undefined && <span>{template.rating.toFixed(1)}★</span>}
    </div>
  </button>
);

export default App;
