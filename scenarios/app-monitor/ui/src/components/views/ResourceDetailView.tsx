import { useCallback, useEffect, useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import {
  ArrowLeft,
  CheckCircle2,
  ExternalLink,
  Info,
  Loader2,
  Play,
  RefreshCw,
  Square,
  XCircle,
} from 'lucide-react';
import clsx from 'clsx';
import { resourceService } from '@/services/api';
import { useResourcesStore } from '@/state/resourcesStore';
import type { ResourceDetail, ResourceStatusSummary } from '@/types';
import './ResourceDetailView.css';

type ResourceAction = 'start' | 'stop' | 'refresh';

type ActionFeedback = {
  type: 'success' | 'error';
  message: string;
};

const coerceBoolean = (value: unknown): boolean | null => {
  if (typeof value === 'boolean') {
    return value;
  }
  if (typeof value === 'number') {
    if (Number.isNaN(value)) {
      return null;
    }
    return value !== 0;
  }
  if (typeof value === 'string') {
    const normalized = value.trim().toLowerCase();
    if (!normalized) {
      return null;
    }
    if (['true', 'yes', 'y', '1', 'online', 'running', 'enabled', 'healthy', 'installed'].includes(normalized)) {
      return true;
    }
    if (['false', 'no', 'n', '0', 'offline', 'disabled', 'stopped', 'unhealthy'].includes(normalized)) {
      return false;
    }
  }
  return null;
};

const extractRecord = (value: unknown): Record<string, unknown> | null => {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return null;
  }
  return value as Record<string, unknown>;
};

const isUrl = (value: string): boolean => /^https?:\/\//i.test(value.trim());

const formatKey = (key: string): string => {
  if (!key) {
    return '';
  }
  const spaced = key
    .replace(/[_-]+/g, ' ')
    .replace(/([a-z0-9])([A-Z])/g, '$1 $2')
    .replace(/\s+/g, ' ')
    .trim();
  return spaced
    .split(' ')
    .map(part => (part.length > 0 ? part[0].toUpperCase() + part.slice(1) : part))
    .join(' ');
};

const formatBoolean = (value: boolean | null | undefined): string => {
  if (value === true) {
    return 'Yes';
  }
  if (value === false) {
    return 'No';
  }
  return 'Unknown';
};

const renderValue = (value: unknown, depth = 0): JSX.Element => {
  if (value === null || value === undefined) {
    return <span className="kv-null">—</span>;
  }

  if (typeof value === 'boolean') {
    return <span className={value ? 'kv-boolean true' : 'kv-boolean false'}>{value ? 'True' : 'False'}</span>;
  }

  if (typeof value === 'number') {
    return <span className="kv-number">{Number.isFinite(value) ? value.toLocaleString() : String(value)}</span>;
  }

  if (typeof value === 'string') {
    const trimmed = value.trim();
    if (trimmed.length === 0) {
      return <span className="kv-null">—</span>;
    }
    if (isUrl(trimmed)) {
      // Omit noreferrer so external inspectors keep the Referer for proxy routing heuristics.
      return (
        <a className="kv-link" href={trimmed} target="_blank" rel="noopener">
          {trimmed}
          <ExternalLink className="kv-link-icon" size={14} />
        </a>
      );
    }
    return <span className="kv-string">{trimmed}</span>;
  }

  if (Array.isArray(value)) {
    if (value.length === 0) {
      return <span className="kv-null">(empty)</span>;
    }
    if (depth >= 2) {
      return <pre className="kv-json-inline">{JSON.stringify(value, null, 2)}</pre>;
    }
    return (
      <ul className={clsx('kv-list', { nested: depth > 0 })}>
        {value.map((item, index) => (
          <li key={index}>{renderValue(item, depth + 1)}</li>
        ))}
      </ul>
    );
  }

  const record = extractRecord(value);
  if (record) {
    if (depth >= 2) {
      return <pre className="kv-json-inline">{JSON.stringify(record, null, 2)}</pre>;
    }
    const entries = Object.entries(record).sort(([a], [b]) => a.localeCompare(b));
    if (entries.length === 0) {
      return <span className="kv-null">(empty)</span>;
    }
    const renderAsJson = entries.length > 12 || entries.some(([, nestedValue]) => {
      const nestedRecord = extractRecord(nestedValue);
      return nestedRecord !== null || Array.isArray(nestedValue);
    });
    if (renderAsJson) {
      return <pre className="kv-json-inline">{JSON.stringify(record, null, 2)}</pre>;
    }
    return (
      <div className={clsx('kv-nested', { nested: depth > 0 })}>
        {entries.map(([key, nestedValue]) => (
          <div className="kv-row" key={key}>
            <div className="kv-label">{formatKey(key)}</div>
            <div className="kv-value">{renderValue(nestedValue, depth + 1)}</div>
          </div>
        ))}
      </div>
    );
  }

  return <span className="kv-string">{String(value)}</span>;
};

const buildStatusClass = (summary?: ResourceStatusSummary | null): string => {
  if (!summary?.status) {
    return 'unknown';
  }
  return summary.status.toLowerCase();
};

const firstNonEmptyString = (...values: Array<unknown>): string | null => {
  for (const value of values) {
    if (typeof value === 'string') {
      const trimmed = value.trim();
      if (trimmed.length > 0) {
        return trimmed;
      }
    }
  }
  return null;
};

const keyValueEntries = (data: Record<string, unknown> | null | undefined): Array<[string, unknown]> => {
  if (!data) {
    return [];
  }
  return Object.entries(data).sort(([a], [b]) => a.localeCompare(b));
};

const shouldRenderAsJson = (data: Record<string, unknown>): boolean => {
  const entries = Object.entries(data);
  if (entries.length === 0) {
    return false;
  }
  if (entries.length > 12) {
    return true;
  }
  return entries.some(([, value]) => {
    if (Array.isArray(value)) {
      return value.length > 4 || value.some(item => extractRecord(item) !== null || Array.isArray(item));
    }
    const record = extractRecord(value);
    if (!record) {
      return false;
    }
    return Object.values(record).some(nested => extractRecord(nested) !== null || Array.isArray(nested));
  });
};

const renderJsonBlock = (data: Record<string, unknown> | null | undefined, className = 'kv-json'): JSX.Element | null => {
  if (!data) {
    return null;
  }
  try {
    return <pre className={className}>{JSON.stringify(data, null, 2)}</pre>;
  } catch {
    return <pre className={className}>{String(data)}</pre>;
  }
};

export default function ResourceDetailView(): JSX.Element {
  const { resourceId } = useParams<{ resourceId: string }>();
  const navigate = useNavigate();
  const [detail, setDetail] = useState<ResourceDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [action, setAction] = useState<ResourceAction | null>(null);
  const [actionFeedback, setActionFeedback] = useState<ActionFeedback | null>(null);

  const startResource = useResourcesStore(state => state.startResource);
  const stopResource = useResourcesStore(state => state.stopResource);
  const refreshResourceStatus = useResourcesStore(state => state.refreshResource);
  const loadResources = useResourcesStore(state => state.loadResources);

  const fetchDetail = useCallback(async (opts?: { silent?: boolean }) => {
    if (!resourceId) {
      return;
    }

    setError(null);
    if (opts?.silent) {
      setRefreshing(true);
    } else {
      setLoading(true);
    }

    try {
      const data = await resourceService.getResourceDetails(resourceId);
      if (!data) {
        setDetail(null);
        setError('Resource details were not found.');
      } else {
        setDetail(data);
      }
    } catch (fetchError) {
      console.error('Failed to load resource detail', fetchError);
      setError('Failed to load resource details.');
      setDetail(null);
    } finally {
      if (opts?.silent) {
        setRefreshing(false);
      } else {
        setLoading(false);
      }
    }
  }, [resourceId]);

  useEffect(() => {
    void fetchDetail();
    void loadResources({ force: true });
  }, [fetchDetail, loadResources]);

  const summary = detail?.summary;
  const statusClass = buildStatusClass(summary);

  const cliStatus = useMemo(() => extractRecord(detail?.cliStatus), [detail?.cliStatus]);
  const endpoints = useMemo(() => extractRecord(cliStatus?.endpoints), [cliStatus]);
  const configuration = useMemo(() => extractRecord(cliStatus?.configuration), [cliStatus]);
  const containerInfo = useMemo(() => extractRecord(cliStatus?.container), [cliStatus]);

  const metrics = useMemo(() => {
    const rootMetrics = extractRecord(cliStatus?.metrics);
    if (rootMetrics) {
      return rootMetrics;
    }
    return extractRecord(cliStatus?.pressure) ?? null;
  }, [cliStatus]);

  const baseUrl = useMemo(() => {
    return firstNonEmptyString(cliStatus?.base_url, cliStatus?.baseUrl, endpoints?.base, configuration?.baseUrl);
  }, [cliStatus, endpoints, configuration]);

  const port = useMemo(() => {
    const portValues = [cliStatus?.port, configuration?.port, configuration?.apiPort];
    for (const candidate of portValues) {
      if (typeof candidate === 'number') {
        return candidate;
      }
      if (typeof candidate === 'string' && candidate.trim().length > 0) {
        const parsed = Number(candidate);
        if (Number.isFinite(parsed)) {
          return parsed;
        }
      }
    }
    return null;
  }, [cliStatus, configuration]);

  const healthy = useMemo(() => coerceBoolean(cliStatus?.healthy), [cliStatus]);
  const installed = useMemo(() => coerceBoolean(cliStatus?.installed), [cliStatus]);

  const handleNavigateBack = () => {
    navigate('/resources');
  };

  const handleStart = async () => {
    if (!resourceId) {
      return;
    }
    setAction('start');
    setActionFeedback(null);

    try {
      const response = await startResource(resourceId);
      if (response?.success) {
        if (response.warning) {
          setActionFeedback({ type: 'error', message: response.warning });
        } else {
          setActionFeedback({ type: 'success', message: 'Resource start requested.' });
        }
        await fetchDetail({ silent: true });
      } else if (response?.error) {
        setActionFeedback({ type: 'error', message: response.error });
      } else {
        setActionFeedback({ type: 'error', message: 'Failed to start resource.' });
      }
    } catch (actionError) {
      console.error('Failed to start resource', actionError);
      setActionFeedback({ type: 'error', message: 'Unexpected failure while starting resource.' });
    } finally {
      setAction(null);
    }
  };

  const handleStop = async () => {
    if (!resourceId) {
      return;
    }
    setAction('stop');
    setActionFeedback(null);

    try {
      const response = await stopResource(resourceId);
      if (response?.success) {
        if (response.warning) {
          setActionFeedback({ type: 'error', message: response.warning });
        } else {
          setActionFeedback({ type: 'success', message: 'Resource stop requested.' });
        }
        await fetchDetail({ silent: true });
      } else if (response?.error) {
        setActionFeedback({ type: 'error', message: response.error });
      } else {
        setActionFeedback({ type: 'error', message: 'Failed to stop resource.' });
      }
    } catch (actionError) {
      console.error('Failed to stop resource', actionError);
      setActionFeedback({ type: 'error', message: 'Unexpected failure while stopping resource.' });
    } finally {
      setAction(null);
    }
  };

  const handleRefresh = async () => {
    if (!resourceId) {
      return;
    }
    setAction('refresh');
    setActionFeedback(null);

    try {
      await refreshResourceStatus(resourceId);
      await fetchDetail({ silent: true });
      setActionFeedback({ type: 'success', message: 'Status refreshed.' });
    } catch (refreshError) {
      console.error('Failed to refresh resource', refreshError);
      setActionFeedback({ type: 'error', message: 'Failed to refresh status.' });
    } finally {
      setAction(null);
    }
  };

  const renderKeyValueSection = (title: string, data: Record<string, unknown> | null | undefined) => {
    const entries = keyValueEntries(data);
    if (entries.length === 0) {
      return null;
    }
    const renderAsJson = data ? shouldRenderAsJson(data) : false;
    return (
      <section className="resource-detail-section" key={title}>
        <header>
          <h3>{title}</h3>
        </header>
        {renderAsJson ? (
          renderJsonBlock(data)
        ) : (
          <div className="kv-grid">
            {entries.map(([key, value]) => (
              <div className="kv-row" key={key}>
                <div className="kv-label">{formatKey(key)}</div>
                <div className="kv-value">{renderValue(value)}</div>
              </div>
            ))}
          </div>
        )}
      </section>
    );
  };

  if (!resourceId) {
    return (
      <div className="resource-detail">
        <div className="resource-detail-header">
          <button type="button" className="back-button" onClick={() => navigate('/resources')}>
            <ArrowLeft size={16} />
            Back to Resources
          </button>
        </div>
        <div className="resource-detail-empty">No resource selected.</div>
      </div>
    );
  }

  if (loading) {
    return (
      <div className="resource-detail loading-state">
        <Loader2 className="loading-spinner" size={32} />
        <span>Loading resource details…</span>
      </div>
    );
  }

  if (error) {
    return (
      <div className="resource-detail">
        <div className="resource-detail-header">
          <button type="button" className="back-button" onClick={handleNavigateBack}>
            <ArrowLeft size={16} />
            Back to Resources
          </button>
        </div>
        <div className="resource-detail-error">
          <XCircle size={20} />
          <span>{error}</span>
          <button type="button" onClick={() => fetchDetail()} className="retry-button">
            Retry
          </button>
        </div>
      </div>
    );
  }

  if (!detail || !summary) {
    return (
      <div className="resource-detail">
        <div className="resource-detail-header">
          <button type="button" className="back-button" onClick={handleNavigateBack}>
            <ArrowLeft size={16} />
            Back to Resources
          </button>
        </div>
        <div className="resource-detail-empty">Resource details unavailable.</div>
      </div>
    );
  }

  const statusChips: Array<{ label: string; value: string; intent: 'positive' | 'negative' | 'neutral' }> = [
    {
      label: 'Running',
      value: formatBoolean(summary.running ?? null),
      intent: summary.running ? 'positive' : summary.running === false ? 'negative' : 'neutral',
    },
    {
      label: 'Enabled',
      value: summary.enabledKnown ? formatBoolean(summary.enabled) : 'Unknown',
      intent: summary.enabled ? 'positive' : summary.enabled === false ? 'negative' : 'neutral',
    },
    {
      label: 'Healthy',
      value: formatBoolean(healthy),
      intent: healthy === true ? 'positive' : healthy === false ? 'negative' : 'neutral',
    },
    {
      label: 'Installed',
      value: formatBoolean(installed),
      intent: installed === true ? 'positive' : installed === false ? 'negative' : 'neutral',
    },
  ];

  const containerEntries = keyValueEntries(containerInfo);
  const hasContainerSection = containerEntries.length > 0;

  return (
    <div className="resource-detail">
      <div className="resource-detail-header">
        <button type="button" className="back-button" onClick={handleNavigateBack}>
          <ArrowLeft size={16} />
          Back to Resources
        </button>
        <div className="header-main">
          <div className="resource-title-group">
            <h1>{detail.name}</h1>
            <span className={clsx('status-badge', statusClass)}>{summary.status.toUpperCase()}</span>
          </div>
          <div className="resource-subtitle">
            <span className="resource-id">ID: {detail.id}</span>
            {detail.category && <span className="divider">•</span>}
            {detail.category && <span className="resource-category">{detail.category}</span>}
          </div>
        </div>
        <div className="header-actions">
          <button
            type="button"
            className="header-action"
            onClick={handleRefresh}
            disabled={action === 'refresh'}
            aria-label="Refresh resource status"
          >
            {action === 'refresh' ? <Loader2 className="spinning" size={18} /> : <RefreshCw size={18} />}
            <span>Refresh</span>
          </button>
        </div>
      </div>

      {actionFeedback && (
        <div className={clsx('resource-detail-feedback', actionFeedback.type)}>
          {actionFeedback.type === 'success' ? <CheckCircle2 size={16} /> : <XCircle size={16} />}
          <span>{actionFeedback.message}</span>
        </div>
      )}

      <section className="resource-detail-section">
        <header>
          <h3>Overview</h3>
          {refreshing && (
            <span className="refresh-indicator">
              <Loader2 className="spinning" size={14} />
              Updating…
            </span>
          )}
        </header>
        <p className="resource-description-text">{detail.description || 'No description provided.'}</p>
        <div className="overview-grid">
          <div className="overview-block">
            <h4>Status</h4>
            <div className="kv-row">
              <div className="kv-label">State</div>
              <div className="kv-value">{summary.statusDetail || summary.status}</div>
            </div>
            <div className="kv-row">
              <div className="kv-label">Running</div>
              <div className="kv-value">{formatBoolean(summary.running ?? null)}</div>
            </div>
            <div className="kv-row">
              <div className="kv-label">Healthy</div>
              <div className="kv-value">{formatBoolean(healthy)}</div>
            </div>
          </div>
          <div className="overview-block">
            <h4>Connectivity</h4>
            <div className="kv-row">
              <div className="kv-label">Base URL</div>
              <div className="kv-value">{baseUrl ? renderValue(baseUrl) : <span className="kv-null">Not available</span>}</div>
            </div>
            <div className="kv-row">
              <div className="kv-label">Port</div>
              <div className="kv-value">{port ? port : <span className="kv-null">Not specified</span>}</div>
            </div>
            <div className="kv-row">
              <div className="kv-label">Category</div>
              <div className="kv-value">{detail.category || summary.type || '—'}</div>
            </div>
          </div>
          <div className="overview-block">
            <h4>Flags</h4>
            <div className="status-chip-row">
              {statusChips.map(chip => (
                <div key={chip.label} className={clsx('status-chip', chip.intent)}>
                  <span className="label">{chip.label}</span>
                  <span className="value">{chip.value}</span>
                </div>
              ))}
            </div>
          </div>
        </div>
        <div className="action-bar">
          <button
            type="button"
            className="action-btn start"
            onClick={handleStart}
            disabled={action === 'start' || summary.status === 'online'}
            aria-label="Start resource"
          >
            {action === 'start' ? <Loader2 className="spinning" size={18} /> : <Play size={18} />}
            <span>Start</span>
          </button>
          <button
            type="button"
            className="action-btn stop"
            onClick={handleStop}
            disabled={action === 'stop' || summary.status === 'offline' || summary.status === 'stopped'}
            aria-label="Stop resource"
          >
            {action === 'stop' ? <Loader2 className="spinning" size={18} /> : <Square size={18} />}
            <span>Stop</span>
          </button>
          <button
            type="button"
            className="action-btn neutral"
            onClick={handleRefresh}
            disabled={action === 'refresh'}
            aria-label="Refresh resource"
          >
            {action === 'refresh' ? <Loader2 className="spinning" size={18} /> : <RefreshCw size={18} />}
            <span>Refresh Status</span>
          </button>
        </div>
      </section>

      {renderKeyValueSection('Endpoints', endpoints)}
      {renderKeyValueSection('Runtime Configuration', configuration)}
      {renderKeyValueSection('Metrics', metrics)}

      {hasContainerSection && (
        <section className="resource-detail-section">
          <header>
            <h3>Container</h3>
          </header>
          <div className="kv-grid">
            {containerEntries.map(([key, value]) => (
              <div className="kv-row" key={key}>
                <div className="kv-label">{formatKey(key)}</div>
                <div className="kv-value">{renderValue(value)}</div>
              </div>
            ))}
          </div>
        </section>
      )}

      {renderKeyValueSection('Service Configuration', detail.serviceConfig)}
      {renderKeyValueSection('Runtime Metadata', detail.runtimeConfig)}
      {renderKeyValueSection('Capability Metadata', detail.capabilityMetadata)}
      {detail.schema ? (
        <section className="resource-detail-section">
          <header>
            <h3>Schema</h3>
          </header>
          {renderJsonBlock(detail.schema)}
        </section>
      ) : null}

      {detail.paths && Object.values(detail.paths).some(Boolean) && (
        <section className="resource-detail-section">
          <header>
            <h3>Key Files</h3>
          </header>
          <ul className="paths-list">
            {detail.paths.serviceConfig && (
              <li>
                <Info size={14} />
                <span>Service Config:</span>
                <code>{detail.paths.serviceConfig}</code>
              </li>
            )}
            {detail.paths.runtimeConfig && (
              <li>
                <Info size={14} />
                <span>Runtime:</span>
                <code>{detail.paths.runtimeConfig}</code>
              </li>
            )}
            {detail.paths.capabilities && (
              <li>
                <Info size={14} />
                <span>Capabilities:</span>
                <code>{detail.paths.capabilities}</code>
              </li>
            )}
            {detail.paths.schema && (
              <li>
                <Info size={14} />
                <span>Schema:</span>
                <code>{detail.paths.schema}</code>
              </li>
            )}
          </ul>
        </section>
      )}
    </div>
  );
}
