import React, { useEffect, useMemo, useState } from 'react';
import {
  captureIssue,
  fetchAssistantStatus,
  fetchHistory,
  fetchIssueDetail,
  spawnAgent,
  updateIssueStatus,
  type AgentType,
  type AssistantStatus,
  type HistoryIssueSummary,
  type IssueDetail,
} from './api';

type TabId = 'capture' | 'history' | 'status';

type Feedback = {
  type: 'success' | 'error' | 'info';
  message: string;
};

type ScreenshotFeedback = {
  variant: 'idle' | 'info' | 'success' | 'error';
  message: string;
};

const agentOptions: { value: AgentType; label: string }[] = [
  { value: 'claude-code', label: 'Claude Code' },
  { value: 'agent-s2', label: 'Agent S2' },
  { value: 'agent-s3', label: 'Agent S3 (experimental)' },
  { value: 'none', label: 'No agent (capture only)' },
];

const tabOrder: { id: TabId; label: string; description: string }[] = [
  {
    id: 'capture',
    label: 'Overlay',
    description: 'Rehearse the desktop capture flow and submit new issues.',
  },
  {
    id: 'history',
    label: 'History',
    description: 'Review captured work and run follow-up actions.',
  },
  {
    id: 'status',
    label: 'Status',
    description: 'Check service health before launching deeper workflows.',
  },
];

type FormState = {
  scenarioName: string;
  url: string;
  description: string;
  tags: string;
  agentType: AgentType;
  spawnAgent: boolean;
  contextNotes: string;
};

function getDefaultScenarioName(): string {
  try {
    const params = new URLSearchParams(window.location.search);
    return params.get('scenario') || 'unknown-scenario';
  } catch (_error) {
    return 'unknown-scenario';
  }
}

function getDefaultUrl(): string {
  try {
    if (document.referrer) {
      return document.referrer;
    }
    return window.location.href;
  } catch (_error) {
    return '';
  }
}

function formatTimestamp(input: string): string {
  const date = new Date(input);
  if (Number.isNaN(date.getTime())) {
    return input;
  }
  return `${date.toLocaleDateString()} ${date.toLocaleTimeString()}`;
}

function buildTags(raw: string): string[] {
  return raw
    .split(',')
    .map((tag) => tag.trim())
    .filter(Boolean);
}

function App(): JSX.Element {
  const [activeTab, setActiveTab] = useState<TabId>('capture');
  const [formState, setFormState] = useState<FormState>(() => ({
    scenarioName: getDefaultScenarioName(),
    url: getDefaultUrl(),
    description: '',
    tags: '',
    agentType: 'claude-code',
    spawnAgent: true,
    contextNotes: '',
  }));
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [feedback, setFeedback] = useState<Feedback | null>(null);

  const [history, setHistory] = useState<HistoryIssueSummary[]>([]);
  const [historyLoading, setHistoryLoading] = useState(false);
  const [selectedIssueId, setSelectedIssueId] = useState<string | null>(null);
  const [issueDetail, setIssueDetail] = useState<IssueDetail | null>(null);
  const [issueDetailLoading, setIssueDetailLoading] = useState(false);

  const [assistantStatus, setAssistantStatus] = useState<AssistantStatus | null>(null);
  const [statusLoading, setStatusLoading] = useState(false);
  const [statusError, setStatusError] = useState<string | null>(null);

  const [screenshotFeedback, setScreenshotFeedback] = useState<ScreenshotFeedback>({
    variant: 'info',
    message: 'Browser previews cannot capture the desktop overlay. Launch the Electron overlay for real screenshots.',
  });
  const [contextSyncedAt, setContextSyncedAt] = useState<number | null>(null);

  const selectedAgentOption = useMemo(
    () => agentOptions.find((option) => option.value === formState.agentType) ?? agentOptions[0],
    [formState.agentType],
  );

  const activeTabMeta = useMemo(
    () => tabOrder.find((tab) => tab.id === activeTab) ?? tabOrder[0],
    [activeTab],
  );

  useEffect(() => {
    void refreshStatus();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (activeTab === 'history' && history.length === 0 && !historyLoading) {
      void refreshHistory();
    }
    if (activeTab === 'status') {
      void refreshStatus();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeTab]);

  const updateFormField = <Key extends keyof FormState>(field: Key, value: FormState[Key]) => {
    setFormState((prev) => ({ ...prev, [field]: value }));
  };

  async function refreshHistory(): Promise<void> {
    try {
      setHistoryLoading(true);
      const items = await fetchHistory();
      setHistory(items);
    } catch (error) {
      console.error('Failed to load history', error);
      setFeedback({ type: 'error', message: 'Unable to load issue history. Check API connectivity.' });
    } finally {
      setHistoryLoading(false);
    }
  }

  async function refreshStatus(): Promise<void> {
    try {
      setStatusLoading(true);
      setStatusError(null);
      const status = await fetchAssistantStatus();
      setAssistantStatus(status);
    } catch (error) {
      console.error('Failed to load assistant status', error);
      setAssistantStatus(null);
      setStatusError('Assistant service is unavailable. Ensure the API exposes /api/v1/assistant/status.');
    } finally {
      setStatusLoading(false);
    }
  }

  function syncContextFromPreview(): void {
    const scenarioName = getDefaultScenarioName();
    const url = getDefaultUrl();
    setFormState((prev) => ({
      ...prev,
      scenarioName,
      url,
    }));
    setContextSyncedAt(Date.now());
  }

  function markScreenshotUnavailable(): void {
    setScreenshotFeedback({
      variant: 'info',
      message:
        'App Monitor runs in a sandbox, so screenshots must come from the desktop overlay. Start it with ENABLE_ELECTRON_OVERLAY=1.',
    });
  }

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();

    if (!formState.description.trim()) {
      setFeedback({ type: 'error', message: 'Description is required before capturing an issue.' });
      return;
    }

    setIsSubmitting(true);
    setFeedback(null);

    try {
      const response = await captureIssue({
        description: formState.description.trim(),
        scenarioName: formState.scenarioName.trim(),
        url: formState.url.trim(),
        screenshotPath: '',
        tags: buildTags(formState.tags),
        contextData: formState.contextNotes
          ? { notes: formState.contextNotes.trim() }
          : {},
        agentType: formState.agentType,
        spawnAgent: formState.spawnAgent && formState.agentType !== 'none',
      });

      if (formState.spawnAgent && formState.agentType !== 'none') {
        await spawnAgent(response.issue_id, formState.agentType, formState.description.trim());
      }

      setFeedback({
        type: 'success',
        message: `Issue captured successfully${
          formState.spawnAgent && formState.agentType !== 'none' ? ' and agent spawned' : ''
        }. Issue ID: ${response.issue_id}.`,
      });

      setFormState((prev) => ({
        ...prev,
        description: '',
        tags: '',
        contextNotes: '',
      }));

      setScreenshotFeedback({
        variant: 'success',
        message: 'Submitted from the browser preview. Attachments will appear when the desktop overlay is used.',
      });

      if (activeTab === 'history') {
        await refreshHistory();
      }
    } catch (error) {
      console.error('Issue capture failed', error);
      setFeedback({ type: 'error', message: error instanceof Error ? error.message : 'Issue capture failed.' });
    } finally {
      setIsSubmitting(false);
    }
  }

  async function handleSelectIssue(issueId: string): Promise<void> {
    try {
      setIssueDetailLoading(true);
      setSelectedIssueId(issueId);
      const detail = await fetchIssueDetail(issueId);
      setIssueDetail(detail);
    } catch (error) {
      console.error('Failed to load issue detail', error);
      setFeedback({ type: 'error', message: 'Unable to load issue detail.' });
    } finally {
      setIssueDetailLoading(false);
    }
  }

  async function handleSpawnAgent(issueId: string, description: string): Promise<void> {
    if (selectedAgentOption.value === 'none') {
      setFeedback({ type: 'info', message: 'Select an agent type other than "No agent" to spawn an agent.' });
      return;
    }

    try {
      await spawnAgent(issueId, selectedAgentOption.value, description);
      setFeedback({ type: 'success', message: `Agent ${selectedAgentOption.label} spawned for issue ${issueId}.` });
      await refreshHistory();
    } catch (error) {
      console.error('Failed to spawn agent', error);
      setFeedback({ type: 'error', message: 'Agent spawn failed.' });
    }
  }

  async function handleStatusUpdate(issueId: string, status: string): Promise<void> {
    try {
      await updateIssueStatus(issueId, status);
      setFeedback({ type: 'success', message: `Issue ${issueId} updated to ${status}.` });
      await refreshHistory();
      if (selectedIssueId === issueId) {
        await handleSelectIssue(issueId);
      }
    } catch (error) {
      console.error('Failed to update issue status', error);
      setFeedback({ type: 'error', message: 'Unable to update issue status.' });
    }
  }

  const statusSummary = (
    <div className="status-summary" role="status" aria-live="polite">
      <span className="status-summary-label">Assistant</span>
      {statusLoading ? (
        <span className="status-pill status-pill-loading">Checking…</span>
      ) : assistantStatus ? (
        <span
          className={`status-pill status-pill-${assistantStatus.status
            .toLowerCase()
            .replace(/[^a-z0-9]+/g, '-')}`}
        >
          {assistantStatus.status}
        </span>
      ) : (
        <button type="button" className="pill-button" onClick={() => void refreshStatus()}>
          {statusError ? 'Retry' : 'Check status'}
        </button>
      )}
      {statusError && !statusLoading && <span className="status-summary-hint">{statusError}</span>}
    </div>
  );

  return (
    <div className="assistant-app">
      <header className="app-header">
        <div className="app-brand">
          <span className="app-logo" aria-hidden="true">
            ⚡️
          </span>
          <div>
            <h1>Vrooli Assistant</h1>
            <p>{activeTabMeta.description}</p>
          </div>
        </div>

        <nav className="app-nav" aria-label="Assistant sections">
          {tabOrder.map((tab) => (
            <button
              key={tab.id}
              type="button"
              className={activeTab === tab.id ? 'nav-button active' : 'nav-button'}
              onClick={() => {
                setActiveTab(tab.id);
                setFeedback(null);
              }}
            >
              {tab.label}
            </button>
          ))}
        </nav>

        <div className="app-header-actions">
          <button type="button" className="ghost" onClick={syncContextFromPreview}>
            Sync preview context
          </button>
          {statusSummary}
        </div>
      </header>

      <main className="app-main">
        <div className="content">
          {feedback && (
            <div className={`feedback feedback-${feedback.type}`} role="status">
              {feedback.message}
            </div>
          )}

          {activeTab === 'capture' && (
            <>
              <section className="info-callout" aria-label="Preview guidance">
                <div>
                  <h2>Capture issues without leaving App Monitor</h2>
                  <p>Mirror the desktop overlay flow, submit context-rich issues, and hand them off to agents in one place.</p>
                </div>
                <ul className="info-points">
                  <li>
                    <strong>Live API.</strong> Actions hit the running assistant backend, so data stays in sync.
                  </li>
                  <li>
                    <strong>Desktop extras later.</strong> Screenshots and hotkeys remain in the Electron overlay.
                  </li>
                  <li>
                    <strong>One-click context.</strong> Sync pulls the current preview URL and scenario straight into the form.
                  </li>
                </ul>
              </section>

              <form className="capture-grid" onSubmit={handleSubmit}>
                <article className="card capture-card" aria-label="Assistant overlay replica">
                  <header className="card-header">
                    <div>
                      <h3>Overlay replica</h3>
                      <p>Practise the capture flow before switching to the desktop overlay.</p>
                    </div>
                  </header>

                  <div className="capture-preview">
                    <div className={`capture-preview-artwork capture-preview-${screenshotFeedback.variant}`} aria-hidden="true">
                      {screenshotFeedback.variant === 'success' ? '✓' : '📸'}
                    </div>
                    <div className="capture-preview-copy">
                      <p className={`capture-preview-message capture-preview-${screenshotFeedback.variant}`}>
                        {screenshotFeedback.message}
                      </p>
                      <dl>
                        <div>
                          <dt>Scenario</dt>
                          <dd>{formState.scenarioName || 'unknown-scenario'}</dd>
                        </div>
                        {formState.url && (
                          <div>
                            <dt>Preview URL</dt>
                            <dd title={formState.url}>{formState.url}</dd>
                          </div>
                        )}
                      </dl>
                    </div>
                    <button type="button" className="ghost" onClick={markScreenshotUnavailable}>
                      Capture screenshot
                    </button>
                  </div>

                  <label className="field" htmlFor="description">
                    <span>Description</span>
                    <textarea
                      id="description"
                      value={formState.description}
                      onChange={(event) => updateFormField('description', event.target.value)}
                      rows={6}
                      placeholder="Summarise the problem, steps to reproduce, and expected behaviour."
                      required
                    />
                  </label>

                  <div className="field-row">
                    <label className="field" htmlFor="agentType">
                      <span>Agent hand-off</span>
                      <select
                        id="agentType"
                        value={formState.agentType}
                        onChange={(event) => {
                          const value = event.target.value as AgentType;
                          updateFormField('agentType', value);
                          if (value === 'none') {
                            updateFormField('spawnAgent', false);
                          }
                        }}
                      >
                        {agentOptions.map((option) => (
                          <option key={option.value} value={option.value}>
                            {option.label}
                          </option>
                        ))}
                      </select>
                    </label>

                    <label className={formState.agentType === 'none' ? 'toggle disabled' : 'toggle'}>
                      <input
                        type="checkbox"
                        checked={formState.spawnAgent && formState.agentType !== 'none'}
                        onChange={(event) => updateFormField('spawnAgent', event.target.checked)}
                        disabled={formState.agentType === 'none'}
                      />
                      <span>Spawn agent after submit</span>
                    </label>
                  </div>

                  <footer className="card-footer">
                    <div>
                      <span>Shortcut</span>
                      <strong>Cmd/Ctrl + Enter</strong>
                    </div>
                    <button type="submit" className="primary" disabled={isSubmitting}>
                      {isSubmitting ? 'Submitting…' : 'Submit issue'}
                    </button>
                  </footer>
                </article>

                <aside className="card context-card" aria-label="Context and metadata">
                  <header className="card-header">
                    <div>
                      <h3>Context & metadata</h3>
                      <p>Make sure the issue is tied to the right scenario and preview.</p>
                    </div>
                    <button type="button" className="ghost" onClick={syncContextFromPreview}>
                      Sync from preview
                    </button>
                  </header>

                  {contextSyncedAt && (
                    <p className="context-hint">Synced {new Date(contextSyncedAt).toLocaleTimeString()} from App Monitor.</p>
                  )}

                  <label className="field" htmlFor="scenarioName">
                    <span>Scenario</span>
                    <input
                      id="scenarioName"
                      type="text"
                      value={formState.scenarioName}
                      onChange={(event) => updateFormField('scenarioName', event.target.value)}
                      placeholder="e.g. palette-generator"
                    />
                  </label>

                  <label className="field" htmlFor="url">
                    <span>URL / location</span>
                    <input
                      id="url"
                      type="url"
                      value={formState.url}
                      onChange={(event) => updateFormField('url', event.target.value)}
                      placeholder="https://"
                    />
                  </label>

                  <label className="field" htmlFor="tags">
                    <span>Tags (comma separated)</span>
                    <input
                      id="tags"
                      type="text"
                      value={formState.tags}
                      onChange={(event) => updateFormField('tags', event.target.value)}
                      placeholder="ui, regression"
                    />
                  </label>

                  <label className="field" htmlFor="contextNotes">
                    <span>Additional context</span>
                    <textarea
                      id="contextNotes"
                      value={formState.contextNotes}
                      onChange={(event) => updateFormField('contextNotes', event.target.value)}
                      rows={4}
                      placeholder="Logs, reproduction hints, or extra observations."
                    />
                  </label>

                  <p className="context-hint">
                    The desktop overlay auto-fills these fields using the active window. Edit them here to match what you
                    see in the preview.
                  </p>
                </aside>
              </form>
            </>
          )}

          {activeTab === 'history' && (
            <section className="card history-card">
              <header className="card-header">
                <div>
                  <h2>Recent issue history</h2>
                  <p>Track captured work, spawn follow-up agents, or update status.</p>
                </div>
                <button type="button" onClick={() => void refreshHistory()} disabled={historyLoading} className="ghost">
                  {historyLoading ? 'Refreshing…' : 'Refresh'}
                </button>
              </header>

              <div className="history-layout">
                <div className="history-table-wrapper">
                  <table className="history-table">
                    <thead>
                      <tr>
                        <th scope="col">Captured</th>
                        <th scope="col">Scenario</th>
                        <th scope="col">Description</th>
                        <th scope="col">Status & actions</th>
                      </tr>
                    </thead>
                    <tbody>
                      {historyLoading && (
                        <tr>
                          <td colSpan={4} className="empty-state">
                            Loading history…
                          </td>
                        </tr>
                      )}
                      {!historyLoading && history.length === 0 && (
                        <tr>
                          <td colSpan={4} className="empty-state">
                            No issues captured yet. Capture one to see it here.
                          </td>
                        </tr>
                      )}
                      {history.map((issue) => {
                        const normalizedStatus = issue.status.toLowerCase().replace(/[^a-z0-9]+/g, '-');
                        const isSelected = selectedIssueId === issue.id;
                        return (
                          <tr key={issue.id} className={isSelected ? 'selected' : undefined}>
                            <td data-title="Captured">
                              <div className="history-when">
                                <time dateTime={issue.timestamp}>{formatTimestamp(issue.timestamp)}</time>
                                <button
                                  type="button"
                                  className="link-button"
                                  onClick={() => void handleSelectIssue(issue.id)}
                                  disabled={issueDetailLoading && selectedIssueId === issue.id}
                                >
                                  {issueDetailLoading && selectedIssueId === issue.id ? 'Loading…' : 'Inspect'}
                                </button>
                              </div>
                            </td>
                            <td data-title="Scenario">{issue.scenario_name || '—'}</td>
                            <td data-title="Description" className="history-description">
                              {issue.description}
                            </td>
                            <td data-title="Status & actions">
                              <div className="history-status">
                                <span className={`status-chip status-chip-${normalizedStatus}`}>{issue.status}</span>
                                {issue.agent_session_id && (
                                  <span className="history-session">Session {issue.agent_session_id.slice(0, 8)}</span>
                                )}
                                <div className="history-actions">
                                  <button
                                    type="button"
                                    className="ghost"
                                    onClick={() => void handleSpawnAgent(issue.id, issue.description)}
                                  >
                                    Spawn agent
                                  </button>
                                  <select
                                    value={issue.status}
                                    onChange={(event) => void handleStatusUpdate(issue.id, event.target.value)}
                                  >
                                    <option value="new">new</option>
                                    <option value="triaged">triaged</option>
                                    <option value="in_progress">in_progress</option>
                                    <option value="completed">completed</option>
                                    <option value="blocked">blocked</option>
                                  </select>
                                </div>
                              </div>
                            </td>
                          </tr>
                        );
                      })}
                    </tbody>
                  </table>
                </div>

                {issueDetail && selectedIssueId && (
                  <aside className="issue-detail-card" aria-live="polite">
                    <header>
                      <h3>Issue detail</h3>
                      <button type="button" className="ghost" onClick={() => setSelectedIssueId(null)}>
                        Close
                      </button>
                    </header>
                    <dl>
                      <div>
                        <dt>Issue ID</dt>
                        <dd>{issueDetail.id}</dd>
                      </div>
                      <div>
                        <dt>Captured</dt>
                        <dd>{formatTimestamp(issueDetail.timestamp)}</dd>
                      </div>
                      <div>
                        <dt>Scenario</dt>
                        <dd>{issueDetail.scenario_name || '—'}</dd>
                      </div>
                      <div>
                        <dt>URL</dt>
                        <dd>{issueDetail.url || '—'}</dd>
                      </div>
                      <div>
                        <dt>Description</dt>
                        <dd>{issueDetail.description}</dd>
                      </div>
                      {issueDetail.tags && issueDetail.tags.length > 0 && (
                        <div>
                          <dt>Tags</dt>
                          <dd>
                            <ul className="tag-list">
                              {issueDetail.tags.map((tag) => (
                                <li key={tag}>{tag}</li>
                              ))}
                            </ul>
                          </dd>
                        </div>
                      )}
                      {issueDetail.context_data && (
                        <div>
                          <dt>Context</dt>
                          <dd>
                            <pre>{JSON.stringify(issueDetail.context_data, null, 2)}</pre>
                          </dd>
                        </div>
                      )}
                      {issueDetail.resolution_notes && (
                        <div>
                          <dt>Resolution</dt>
                          <dd>{issueDetail.resolution_notes}</dd>
                        </div>
                      )}
                    </dl>
                  </aside>
                )}
              </div>
            </section>
          )}

          {activeTab === 'status' && (
            <section className="card status-card">
              <header className="card-header">
                <div>
                  <h2>Assistant service status</h2>
                  <p>Ensure the API is reachable before attempting captures from the browser UI.</p>
                </div>
                <button type="button" onClick={() => void refreshStatus()} disabled={statusLoading} className="ghost">
                  {statusLoading ? 'Refreshing…' : 'Refresh'}
                </button>
              </header>

              {assistantStatus ? (
                <div className="status-grid">
                  <article>
                    <h3>Current state</h3>
                    <p className={`pill pill-${assistantStatus.status.toLowerCase().replace(/[^a-z0-9]+/g, '-')}`}>
                      {assistantStatus.status}
                    </p>
                    <small>Reported by /api/v1/assistant/status</small>
                  </article>
                  <article>
                    <h3>Issues captured</h3>
                    <p className="metric">{assistantStatus.issues_captured}</p>
                    <small>Total since service boot</small>
                  </article>
                  <article>
                    <h3>Agents spawned</h3>
                    <p className="metric">{assistantStatus.agents_spawned}</p>
                    <small>Includes manual and automatic triggers</small>
                  </article>
                  <article>
                    <h3>Uptime</h3>
                    <p className="metric">{assistantStatus.uptime}</p>
                    <small>Derived from service heartbeat</small>
                  </article>
                </div>
              ) : (
                <div className="empty-state">
                  {statusLoading
                    ? 'Checking assistant status…'
                    : statusError || 'No status available yet. Try refreshing.'}
                </div>
              )}

              <div className="status-help">
                <h3>Troubleshooting tips</h3>
                <ul>
                  <li>Start the scenario via <code>make start</code> or <code>vrooli scenario start vrooli-assistant</code>.</li>
                  <li>Verify the API responds to <code>/health</code> with HTTP 200.</li>
                  <li>If you need the desktop daemon, run <code>npm run start</code> inside <code>ui/electron</code>.</li>
                </ul>
              </div>
            </section>
          )}
        </div>
      </main>
    </div>
  );
}

export default App;
