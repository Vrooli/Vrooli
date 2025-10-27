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
    description: 'Live preview of the desktop assistant UI.',
  },
  {
    id: 'history',
    label: 'History',
    description: 'Review captures and manage follow-up actions.',
  },
  {
    id: 'status',
    label: 'Status',
    description: 'Check service health and usage metrics.',
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

  const [screenshotFeedback, setScreenshotFeedback] = useState<ScreenshotFeedback>({
    variant: 'idle',
    message: 'Screenshots are captured automatically when the desktop overlay submits an issue.',
  });
  const [contextSyncedAt, setContextSyncedAt] = useState<number | null>(null);

  const selectedAgentOption = useMemo(
    () => agentOptions.find((option) => option.value === formState.agentType) ?? agentOptions[0],
    [formState.agentType],
  );

  useEffect(() => {
    void refreshStatus();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (activeTab === 'history' && history.length === 0) {
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
      const status = await fetchAssistantStatus();
      setAssistantStatus(status);
    } catch (error) {
      console.error('Failed to load assistant status', error);
      setFeedback({ type: 'error', message: 'Assistant status request failed.' });
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
      message: 'Browser previews cannot trigger desktop screenshots. Use the hotkey in the Electron overlay to capture real context.',
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
        message: 'Issue submitted from the web preview. Attachments will be added when the desktop overlay is used.',
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

  return (
    <div className="assistant-shell">
      <aside className="assistant-sidebar">
        <div className="branding">
          <span className="logo">⚡️</span>
          <div>
            <h1>Vrooli Assistant</h1>
            <p>Overlay preview & control surface</p>
          </div>
        </div>
        <nav aria-label="Assistant navigation">
          {tabOrder.map((tab) => (
            <button
              key={tab.id}
              type="button"
              className={activeTab === tab.id ? 'tab-link active' : 'tab-link'}
              onClick={() => {
                setActiveTab(tab.id);
                setFeedback(null);
              }}
            >
              <strong>{tab.label}</strong>
              <span>{tab.description}</span>
            </button>
          ))}
        </nav>
        <div className="sidebar-footnote">
          <p>
            Need the full desktop overlay? Start the scenario locally and press <strong>Ctrl/Cmd + Shift + Space</strong>.
          </p>
        </div>
      </aside>

      <main className="assistant-content">
        {feedback && (
          <div className={`feedback feedback-${feedback.type}`} role="status">
            {feedback.message}
          </div>
        )}

        {activeTab === 'capture' && (
          <form className="capture-layout" onSubmit={handleSubmit}>
            <section className="overlay-card" aria-label="Assistant overlay preview">
              <header className="overlay-header">
                <h2>
                  <span className="overlay-indicator" aria-hidden="true" />
                  Vrooli Assistant Overlay
                </h2>
                <div className="overlay-header-actions">
                  <span>Capture Issue</span>
                  <span>View History</span>
                  <span>Settings</span>
                </div>
              </header>

              <div className="overlay-body">
                <div className="overlay-screenshot">
                  <div className="overlay-preview" aria-hidden="true">
                    {screenshotFeedback.variant === 'success' ? '✓' : '📸'}
                  </div>
                  <div className="overlay-screenshot-copy">
                    <p className={`overlay-status overlay-status-${screenshotFeedback.variant}`}>
                      {screenshotFeedback.message}
                    </p>
                    <p className="overlay-scenario">Scenario: {formState.scenarioName || 'unknown-scenario'}</p>
                    {formState.url && (
                      <p className="overlay-url" title={formState.url}>
                        {formState.url}
                      </p>
                    )}
                  </div>
                  <button type="button" className="ghost" onClick={markScreenshotUnavailable}>
                    Capture screenshot
                  </button>
                </div>

                <label className="overlay-field" htmlFor="description">
                  <span>Describe the issue</span>
                  <textarea
                    id="description"
                    value={formState.description}
                    onChange={(event) => updateFormField('description', event.target.value)}
                    rows={6}
                    placeholder="Summarise the problem, steps to reproduce, and expected behaviour."
                    required
                  />
                </label>

                <div className="overlay-controls">
                  <label className="overlay-select" htmlFor="agentType">
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

                  <label className={formState.agentType === 'none' ? 'overlay-toggle disabled' : 'overlay-toggle'}>
                    <input
                      type="checkbox"
                      checked={formState.spawnAgent && formState.agentType !== 'none'}
                      onChange={(event) => updateFormField('spawnAgent', event.target.checked)}
                      disabled={formState.agentType === 'none'}
                    />
                    <span>Spawn agent after submit</span>
                  </label>
                </div>
              </div>

              <footer className="overlay-footer">
                <div className="overlay-footer-copy">
                  <span>Hotkey</span>
                  <strong>Cmd/Ctrl + Enter</strong>
                </div>
                <button type="submit" className="primary" disabled={isSubmitting}>
                  {isSubmitting ? 'Submitting…' : 'Submit Issue'}
                </button>
              </footer>
            </section>

            <aside className="context-card" aria-label="Context and metadata">
              <header>
                <h3>Context & metadata</h3>
                <button type="button" className="ghost" onClick={syncContextFromPreview}>
                  Sync from preview
                </button>
              </header>

              {contextSyncedAt && (
                <p className="context-hint">
                  Synced {new Date(contextSyncedAt).toLocaleTimeString()} from App Monitor.
                </p>
              )}

              <label className="context-field" htmlFor="scenarioName">
                <span>Scenario</span>
                <input
                  id="scenarioName"
                  type="text"
                  value={formState.scenarioName}
                  onChange={(event) => updateFormField('scenarioName', event.target.value)}
                  placeholder="e.g. palette-generator"
                />
              </label>

              <label className="context-field" htmlFor="url">
                <span>URL / location</span>
                <input
                  id="url"
                  type="url"
                  value={formState.url}
                  onChange={(event) => updateFormField('url', event.target.value)}
                  placeholder="https://"
                />
              </label>

              <label className="context-field" htmlFor="tags">
                <span>Tags (comma separated)</span>
                <input
                  id="tags"
                  type="text"
                  value={formState.tags}
                  onChange={(event) => updateFormField('tags', event.target.value)}
                  placeholder="ui, regression"
                />
              </label>

              <label className="context-field" htmlFor="contextNotes">
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
                The desktop overlay auto-fills these values using active window metadata. When previewing here you can tweak
                them manually to mirror the scenario you are testing.
              </p>
            </aside>
          </form>
        )}

        {activeTab === 'history' && (
          <section className="panel">
            <header className="panel-header">
              <div>
                <h2>Recent issue history</h2>
                <p>Track captured work, spawn follow-up agents, or update status.</p>
              </div>
              <div className="panel-actions">
                <button type="button" onClick={() => void refreshHistory()} disabled={historyLoading}>
                  {historyLoading ? 'Refreshing…' : 'Refresh'}
                </button>
              </div>
            </header>

            <div className="history-table" role="table">
              <div className="history-header" role="row">
                <span role="columnheader">When</span>
                <span role="columnheader">Scenario</span>
                <span role="columnheader">Description</span>
                <span role="columnheader">Status</span>
                <span role="columnheader">Actions</span>
              </div>
              {history.length === 0 && !historyLoading && (
                <div className="history-empty" role="row">
                  No issues captured yet. Capture one to see it here.
                </div>
              )}
              {historyLoading && (
                <div className="history-empty" role="row">
                  Loading history…
                </div>
              )}
              {history.map((issue) => (
                <div
                  key={issue.id}
                  role="row"
                  className={selectedIssueId === issue.id ? 'history-row selected' : 'history-row'}
                >
                  <span role="cell">{formatTimestamp(issue.timestamp)}</span>
                  <span role="cell">{issue.scenario_name || '—'}</span>
                  <span role="cell" className="history-description">{issue.description}</span>
                  <span role="cell" className={`status status-${issue.status}`}>
                    {issue.status}
                    {issue.agent_session_id && <small> • Session {issue.agent_session_id.slice(0, 8)}</small>}
                    <button
                      type="button"
                      onClick={() => void handleSelectIssue(issue.id)}
                      disabled={issueDetailLoading && selectedIssueId === issue.id}
                    >
                      {issueDetailLoading && selectedIssueId === issue.id ? 'Loading…' : 'Inspect'}
                    </button>
                  </span>
                  <span role="cell" className="history-actions">
                    <button type="button" onClick={() => void handleSpawnAgent(issue.id, issue.description)}>
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
                  </span>
                </div>
              ))}
            </div>

            {issueDetail && selectedIssueId && (
              <div className="issue-detail">
                <header>
                  <h3>Issue detail</h3>
                  <button type="button" onClick={() => setSelectedIssueId(null)}>
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
              </div>
            )}
          </section>
        )}

        {activeTab === 'status' && (
          <section className="panel">
            <header className="panel-header">
              <div>
                <h2>Assistant service status</h2>
                <p>Ensure the API is reachable before attempting captures from the browser UI.</p>
              </div>
              <div className="panel-actions">
                <button type="button" onClick={() => void refreshStatus()} disabled={statusLoading}>
                  {statusLoading ? 'Refreshing…' : 'Refresh'}
                </button>
              </div>
            </header>

            {assistantStatus ? (
              <div className="status-grid">
                <article>
                  <h3>Current state</h3>
                  <p className={`pill pill-${assistantStatus.status}`}>{assistantStatus.status}</p>
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
              <div className="history-empty">
                {statusLoading ? 'Checking assistant status…' : 'No status available yet. Try refreshing.'}
              </div>
            )}

            <div className="status-help">
              <h3>Troubleshooting tips</h3>
              <ul>
                <li>Verify the scenario is running via <code>make start</code> or <code>vrooli scenario start vrooli-assistant</code>.</li>
                <li>Ensure the API port is reachable from the browser (<code>/health</code> should return 200).</li>
                <li>If the Electron daemon is required locally, launch it manually: <code>npm run start</code> inside <code>ui/electron</code>.</li>
              </ul>
            </div>
          </section>
        )}
      </main>
    </div>
  );
}

export default App;
