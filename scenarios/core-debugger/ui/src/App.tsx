import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { AlertTriangle, Database, LifeBuoy, RefreshCw, ServerCog, TimerReset } from 'lucide-react';
import './App.css';
import { ComponentHealthCard } from './components/ComponentHealthCard';
import { IssueCard } from './components/IssueCard';
import { StatisticCard } from './components/StatisticCard';
import { StatusBadge } from './components/StatusBadge';
import type {
  ComponentHealth,
  HealthResponse,
  HealthStatus,
  IssueRecord,
  IssuesResponse,
  ScenarioConfig
} from './types';

const DEFAULT_REFRESH_MS = 30_000;
const API_FALLBACK = '/api/v1';

async function fetchJson<T>(url: string): Promise<T> {
  const response = await fetch(url, { credentials: 'same-origin' });
  if (!response.ok) {
    throw new Error(`Request failed (${response.status}) for ${url}`);
  }
  return (await response.json()) as T;
}

function normaliseStatus(status?: string): HealthStatus {
  if (status === 'critical') return 'critical';
  if (status === 'degraded') return 'degraded';
  return 'healthy';
}

export default function App() {
  const [config, setConfig] = useState<ScenarioConfig | null>(null);
  const [health, setHealth] = useState<HealthResponse | null>(null);
  const [issues, setIssues] = useState<IssueRecord[]>([]);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [configLoaded, setConfigLoaded] = useState(false);
  const fetchingRef = useRef(false);

  const apiBase = config?.apiUrl || API_FALLBACK;
  const refreshInterval = config?.refreshIntervalMs || DEFAULT_REFRESH_MS;

  useEffect(() => {
    let cancelled = false;

    const loadConfig = async () => {
      try {
        const result = await fetchJson<ScenarioConfig>('/api/config');
        if (!cancelled) {
          setConfig({
            scenario: result.scenario || 'core-debugger',
            apiUrl: result.apiUrl || API_FALLBACK,
            apiPort: result.apiPort ?? null,
            refreshIntervalMs: result.refreshIntervalMs || DEFAULT_REFRESH_MS
          });
        }
      } catch (err) {
        console.warn('Failed to load UI config, falling back to defaults.', err);
        if (!cancelled) {
          setConfig({
            scenario: 'core-debugger',
            apiUrl: API_FALLBACK,
            apiPort: null,
            refreshIntervalMs: DEFAULT_REFRESH_MS
          });
        }
      } finally {
        if (!cancelled) {
          setConfigLoaded(true);
        }
      }
    };

    loadConfig();

    return () => {
      cancelled = true;
    };
  }, []);

  const loadData = useCallback(
    async (showSpinner = false) => {
      if (fetchingRef.current) {
        return;
      }

      fetchingRef.current = true;

      if (showSpinner) {
        setRefreshing(true);
      }
      setError(null);

      try {
        const [healthResponse, issuesResponse] = await Promise.all([
          fetchJson<HealthResponse>(`${apiBase}/health`),
          fetchJson<IssuesResponse>(`${apiBase}/issues`)
        ]);

        setHealth(healthResponse);
        setIssues(Array.isArray(issuesResponse.issues) ? issuesResponse.issues : []);
        setLastUpdated(new Date());
      } catch (err) {
        console.error('Failed to refresh scenario data', err);
        setError(err instanceof Error ? err.message : 'Unknown error refreshing data');
      } finally {
        fetchingRef.current = false;
        setLoading(false);
        setRefreshing(false);
      }
    },
    [apiBase]
  );

  useEffect(() => {
    if (!configLoaded) return;

    let cancelled = false;

    const initialise = async () => {
      if (!cancelled) {
        await loadData(true);
      }
    };

    initialise();

    const interval = window.setInterval(() => {
      if (!cancelled) {
        void loadData();
      }
    }, refreshInterval);

    return () => {
      cancelled = true;
      window.clearInterval(interval);
    };
  }, [configLoaded, loadData, refreshInterval]);

  const stats = useMemo(() => {
    const components: ComponentHealth[] = health?.components ?? [];
    const totalComponents = components.length;
    const healthyComponents = components.filter((component) => component.status === 'healthy').length;
    const averageResponse = totalComponents
      ? Math.round(
          components.reduce((sum, component) => sum + Number(component.response_time_ms || 0), 0) / totalComponents
        )
      : 0;
    const activeIssues = health?.active_issues ?? issues.length;
    const knownWorkarounds = issues.reduce((acc, issue) => acc + (issue.workarounds?.length ?? 0), 0);

    return {
      activeIssues,
      healthyComponents,
      totalComponents,
      knownWorkarounds,
      averageResponse
    };
  }, [health, issues]);

  const overallStatus = normaliseStatus(health?.status);

  const handleManualRefresh = () => {
    void loadData(true);
  };

  return (
    <div className="app-shell">
      <header className="app-header">
        <div className="header-main">
          <div className="brand">
            <div className="brand-icon" aria-hidden="true">
              <ServerCog size={24} />
            </div>
            <div>
              <h1>Core Debugger</h1>
              <p>Real-time view of Vrooli&apos;s core infrastructure health.</p>
            </div>
          </div>
          <StatusBadge status={overallStatus} />
        </div>
        <div className="header-actions">
          <button type="button" className="refresh-button" onClick={handleManualRefresh} disabled={refreshing}>
            <RefreshCw className={refreshing ? 'spin' : ''} size={16} />
            {refreshing ? 'Refreshing…' : 'Refresh'}
          </button>
          <div className="meta">
            <span>
              API:
              <strong>{config?.apiPort ? ` localhost:${config.apiPort}` : ' auto (proxy)'}</strong>
            </span>
            <span>
              Interval:
              <strong> {Math.round(refreshInterval / 1000)}s</strong>
            </span>
            <span className="last-updated">
              Last updated: {lastUpdated ? lastUpdated.toLocaleTimeString() : 'waiting…'}
            </span>
          </div>
        </div>
      </header>

      <main className="app-content">
        {error ? (
          <div className="error-banner" role="alert">
            <AlertTriangle size={18} />
            <div>
              <strong>Unable to load latest status.</strong>
              <span>{error}</span>
            </div>
          </div>
        ) : null}

        <section className="panel">
          <div className="panel-header">
            <h2>System Overview</h2>
            <p>Availability snapshot across monitored core services.</p>
          </div>
          <div className="stats-grid" role="list">
            <StatisticCard
              title="Active Issues"
              value={stats.activeIssues}
              tone={stats.activeIssues > 0 ? 'warning' : 'success'}
              icon={<AlertTriangle size={18} />}
              caption={stats.activeIssues > 0 ? 'Immediate investigation required' : 'No open incidents'}
            />
            <StatisticCard
              title="Healthy Components"
              value={`${stats.healthyComponents}/${stats.totalComponents || '–'}`}
              tone="success"
              icon={<ServerCog size={18} />}
              caption="Operational / total"
            />
            <StatisticCard
              title="Known Workarounds"
              value={stats.knownWorkarounds}
              tone="info"
              icon={<LifeBuoy size={18} />}
              caption="Reusable mitigation playbooks"
            />
            <StatisticCard
              title="Avg Response Time"
              value={`${stats.averageResponse || 0} ms`}
              tone="default"
              icon={<TimerReset size={18} />}
              caption="Rolling mean per component"
            />
          </div>
        </section>

        <section className="panel">
          <div className="panel-header">
            <h2>Component Health</h2>
            <p>Heartbeat probes across CLI, orchestrator, resource manager, and setup services.</p>
          </div>
          {loading ? (
            <div className="panel-body loading-state">
              <div className="loader" aria-hidden="true" />
              <span>Gathering health metrics…</span>
            </div>
          ) : (
            <div className="panel-body">
              {health?.components?.length ? (
                <div className="component-grid">
                  {health.components.map((component) => (
                    <ComponentHealthCard key={component.component} data={component} />
                  ))}
                </div>
              ) : (
                <div className="empty-state">
                  <Database size={20} />
                  <span>No component telemetry available.</span>
                </div>
              )}
            </div>
          )}
        </section>

        <section className="panel">
          <div className="panel-header">
            <h2>Active Issues</h2>
            <p>Current incidents with deduplicated error signatures and context.</p>
          </div>
          <div className="panel-body">
            {loading ? (
              <div className="panel-body loading-state">
                <div className="loader" aria-hidden="true" />
                <span>Collecting incident feed…</span>
              </div>
            ) : issues.length ? (
              <div className="issues-grid">
                {issues.map((issue) => (
                  <IssueCard key={issue.id} issue={issue} />
                ))}
              </div>
            ) : (
              <div className="empty-state">
                <LifeBuoy size={20} />
                <span>All clear — no active issues detected.</span>
              </div>
            )}
          </div>
        </section>
      </main>
    </div>
  );
}
