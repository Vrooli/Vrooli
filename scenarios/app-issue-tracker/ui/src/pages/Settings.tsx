import { useCallback, useEffect, useMemo, useState } from "react";
import {
  ActivitySquare,
  Bot,
  Loader2,
  MonitorCog,
  SlidersHorizontal,
  Sparkles,
} from "lucide-react";
import {
  AgentSettings,
  DisplaySettings,
  ProcessorSettings,
  PromptTesterCase,
  promptTesterCases,
} from "../data/sampleData";

interface SettingsPageProps {
  apiBaseUrl: string;
  processor: ProcessorSettings;
  agent: AgentSettings;
  display: DisplaySettings;
  onProcessorChange: (settings: ProcessorSettings) => void;
  onAgentChange: (settings: AgentSettings) => void;
  onDisplayChange: (settings: DisplaySettings) => void;
  issuesProcessed?: number;
  issuesRemaining?: number | string;
}

type TabKey = "processor" | "agent" | "display" | "prompt";

interface PromptPreviewResponse {
  issue_id: string;
  agent_id: string;
  issue_title?: string;
  issue_status?: string;
  prompt_template: string;
  prompt_markdown: string;
  generated_at: string;
  source: string;
  error_message?: string;
}

const tabConfig: {
  key: TabKey;
  label: string;
  icon: React.ComponentType<{ size?: number }>;
}[] = [
  { key: "processor", label: "Processor", icon: ActivitySquare },
  { key: "agent", label: "Agent", icon: Bot },
  { key: "display", label: "Display", icon: MonitorCog },
  { key: "prompt", label: "Prompt Tester", icon: Sparkles },
];

export function SettingsPage({
  apiBaseUrl,
  processor,
  agent,
  display,
  onProcessorChange,
  onAgentChange,
  onDisplayChange,
  issuesProcessed = 0,
  issuesRemaining = 'unlimited',
}: SettingsPageProps) {
  const [activeTab, setActiveTab] = useState<TabKey>("processor");
  const [selectedTestCaseId, setSelectedTestCaseId] = useState<string>(
    promptTesterCases[0]?.id ?? "",
  );
  const [promptTemplate, setPromptTemplate] = useState<string>("");
  const [promptPreview, setPromptPreview] = useState<string>("");
  const [previewError, setPreviewError] = useState<string | null>(null);
  const [isPreviewLoading, setIsPreviewLoading] = useState(false);
  const [previewMeta, setPreviewMeta] = useState<{
    generatedAt: string;
    source: string;
    issueTitle?: string;
    issueStatus?: string;
  } | null>(null);

  const allowedToolsString = useMemo(
    () => agent.allowedTools.join(","),
    [agent.allowedTools],
  );

  const selectedTestCase = useMemo<PromptTesterCase | undefined>(
    () =>
      promptTesterCases.find((testCase) => testCase.id === selectedTestCaseId),
    [selectedTestCaseId],
  );

  const previewSourceLabel = useMemo(() => {
    if (!previewMeta?.source) {
      return null;
    }
    switch (previewMeta.source) {
      case "issue_directory":
        return "Source: existing issue bundle";
      case "payload":
      default:
        return "Source: mock issue payload";
    }
  }, [previewMeta?.source]);

  const formattedGeneratedAt = useMemo(() => {
    if (!previewMeta?.generatedAt) {
      return null;
    }
    const generatedDate = new Date(previewMeta.generatedAt);
    if (Number.isNaN(generatedDate.getTime())) {
      return previewMeta.generatedAt;
    }
    return generatedDate.toLocaleString();
  }, [previewMeta?.generatedAt]);

  const fetchPromptPreview = useCallback(
    async (options?: { signal?: AbortSignal }) => {
      if (!selectedTestCase) {
        setPromptTemplate("");
        setPromptPreview("");
        setPreviewMeta(null);
        return;
      }

      if (!apiBaseUrl) {
        setPreviewError("API base URL is not configured.");
        setIsPreviewLoading(false);
        return;
      }

      if (!options?.signal?.aborted) {
        setIsPreviewLoading(true);
        setPreviewError(null);
        setPromptTemplate("");
        setPromptPreview("");
        setPreviewMeta(null);
      }

      try {
        const response = await fetch(`${apiBaseUrl}/investigate/preview`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            agent_id: agent.agentId,
            issue_id: selectedTestCase.issue.id,
            issue: selectedTestCase.issue,
          }),
          signal: options?.signal,
        });

        if (!response.ok) {
          const message = await response.text();
          throw new Error(
            message || `Request failed with status ${response.status}`,
          );
        }

        const payload = (await response.json()) as PromptPreviewResponse;

        if (options?.signal?.aborted) {
          return;
        }

        setPromptTemplate(payload.prompt_template);
        setPromptPreview(payload.prompt_markdown);
        setPreviewMeta({
          generatedAt: payload.generated_at,
          source: payload.source,
          issueTitle: payload.issue_title,
          issueStatus: payload.issue_status,
        });
      } catch (error) {
        const isAbort =
          (options?.signal && options.signal.aborted) ||
          (error instanceof DOMException && error.name === "AbortError");
        if (isAbort) {
          return;
        }
        setPreviewError(
          (error as Error).message || "Failed to load prompt preview.",
        );
        setPromptTemplate("");
        setPromptPreview("");
        setPreviewMeta(null);
      } finally {
        if (!options?.signal?.aborted) {
          setIsPreviewLoading(false);
        }
      }
    },
    [apiBaseUrl, agent.agentId, selectedTestCase],
  );

  useEffect(() => {
    const controller = new AbortController();
    void fetchPromptPreview({ signal: controller.signal });
    return () => {
      controller.abort();
    };
  }, [fetchPromptPreview]);

  const handleProcessorChange = (
    field: keyof ProcessorSettings,
    value: boolean | number,
  ) => {
    onProcessorChange({
      ...processor,
      [field]: value,
    });
  };

  const handleAgentChange = (
    field: keyof AgentSettings,
    value: AgentSettings[keyof AgentSettings],
  ) => {
    if (field === "allowedTools" && typeof value === "string") {
      const tools = value
        .split(",")
        .map((tool) => tool.trim())
        .filter(Boolean);
      onAgentChange({ ...agent, allowedTools: tools });
    } else {
      onAgentChange({
        ...agent,
        [field]: value,
      });
    }
  };

  const handleDisplayChange = (
    field: keyof DisplaySettings,
    value: DisplaySettings[keyof DisplaySettings],
  ) => {
    onDisplayChange({
      ...display,
      [field]: value,
    });
  };

  return (
    <div className="settings-page">
      <header className="page-header">
        <div>
          <h2>Scenario Settings</h2>
          <p>
            Configure automation capacity, the unified agent, and UI
            preferences.
          </p>
        </div>
      </header>
      <div className="settings-layout">
        <nav className="settings-tabs">
          {tabConfig.map((tab) => {
            const Icon = tab.icon;
            const active = tab.key === activeTab;
            return (
              <button
                key={tab.key}
                className={`settings-tab ${active ? "active" : ""}`}
                onClick={() => setActiveTab(tab.key)}
              >
                <Icon size={16} />
                <span>{tab.label}</span>
              </button>
            );
          })}
        </nav>
        <section className="settings-panel">
          {activeTab === "processor" && (
            <div className="settings-group">
              <h3>Queue Processor</h3>
              <p>
                Control how many issues the unified agent tackles
                simultaneously.
              </p>
              <div className="form-grid">
                <label className="slider-field">
                  <div className="slider-label">
                    <span>Concurrent slots</span>
                    <span className="slider-value">
                      {processor.concurrentSlots}
                    </span>
                  </div>
                  <input
                    type="range"
                    min={1}
                    max={8}
                    step={1}
                    value={processor.concurrentSlots}
                    onChange={(event) =>
                      handleProcessorChange(
                        "concurrentSlots",
                        Number(event.target.value),
                      )
                    }
                  />
                  <div className="slider-range">
                    <span>1</span>
                    <span>8</span>
                  </div>
                </label>
                <label>
                  <span>Refresh interval (seconds)</span>
                  <input
                    type="number"
                    min={15}
                    max={600}
                    value={processor.refreshInterval}
                    onChange={(event) =>
                      handleProcessorChange(
                        "refreshInterval",
                        Number(event.target.value),
                      )
                    }
                  />
                </label>
                <label>
                  <span>Maximum issues to process</span>
                  <input
                    type="number"
                    min={0}
                    max={1000}
                    value={processor.maxIssues}
                    onChange={(event) =>
                      handleProcessorChange(
                        "maxIssues",
                        Number(event.target.value),
                      )
                    }
                  />
                  <small style={{ display: "block", marginTop: "4px", color: "var(--text-secondary)" }}>
                    Maximum number of issues to process (0 = unlimited).{" "}
                    {processor.maxIssues > 0 && (
                      <span style={{ fontWeight: 500 }}>
                        {issuesProcessed} processed, {issuesRemaining} remaining
                      </span>
                    )}
                  </small>
                </label>
                <label className="checkbox-field">
                  <input
                    type="checkbox"
                    checked={processor.active}
                    onChange={(event) =>
                      handleProcessorChange("active", event.target.checked)
                    }
                  />
                  <span>Processor active</span>
                </label>
              </div>
            </div>
          )}

          {activeTab === "agent" && (
            <div className="settings-group">
              <h3>Unified Agent</h3>
              <p>
                Configure the single automation agent responsible for
                investigation and remediation.
              </p>
              <div className="form-grid">
                <label>
                  <span>AI Backend Provider</span>
                  <select
                    value={agent.backend?.provider ?? "codex"}
                    onChange={(event) =>
                      handleAgentChange("backend", {
                        provider: event.target.value as "codex" | "claude-code",
                        autoFallback: agent.backend?.autoFallback ?? true,
                      })
                    }
                  >
                    <option value="codex">Codex (Primary)</option>
                    <option value="claude-code">Claude Code</option>
                  </select>
                </label>
                <label className="checkbox-field">
                  <input
                    type="checkbox"
                    checked={agent.backend?.autoFallback ?? true}
                    onChange={(event) =>
                      handleAgentChange("backend", {
                        provider: agent.backend?.provider ?? "codex",
                        autoFallback: event.target.checked,
                      })
                    }
                  />
                  <span>Auto-fallback to alternative backend</span>
                </label>
                <label>
                  <span>Agent identifier</span>
                  <input
                    type="text"
                    value={agent.agentId}
                    onChange={(event) =>
                      handleAgentChange("agentId", event.target.value)
                    }
                  />
                </label>
                <label className="slider-field">
                  <div className="slider-label">
                    <span>Maximum turns</span>
                    <span className="slider-value">
                      {agent.maximumTurns}
                    </span>
                  </div>
                  <input
                    type="range"
                    min={5}
                    max={80}
                    step={1}
                    value={agent.maximumTurns}
                    onChange={(event) =>
                      handleAgentChange(
                        "maximumTurns",
                        Number(event.target.value),
                      )
                    }
                  />
                  <div className="slider-range">
                    <span>5</span>
                    <span>80</span>
                  </div>
                </label>
                <label>
                  <span>Allowed tools</span>
                  <input
                    type="text"
                    value={allowedToolsString}
                    onChange={(event) =>
                      handleAgentChange("allowedTools", event.target.value)
                    }
                    placeholder="Comma separated"
                  />
                </label>
                <label className="slider-field">
                  <div className="slider-label">
                    <span>Task timeout (minutes)</span>
                    <span className="slider-value">
                      {agent.taskTimeout}
                    </span>
                  </div>
                  <input
                    type="range"
                    min={5}
                    max={240}
                    step={5}
                    value={agent.taskTimeout}
                    onChange={(event) =>
                      handleAgentChange(
                        "taskTimeout",
                        Number(event.target.value),
                      )
                    }
                  />
                  <div className="slider-range">
                    <span>5</span>
                    <span>240</span>
                  </div>
                </label>
                <label className="checkbox-field">
                  <input
                    type="checkbox"
                    checked={agent.skipPermissionChecks}
                    onChange={(event) =>
                      handleAgentChange(
                        "skipPermissionChecks",
                        event.target.checked,
                      )
                    }
                  />
                  <span>Skip permission checks</span>
                </label>
              </div>
            </div>
          )}

          {activeTab === "display" && (
            <div className="settings-group">
              <h3>Display</h3>
              <p>Pick between light and dark presentations.</p>
              <div className="form-grid">
                <label className="toggle-field">
                  <span>Theme</span>
                  <button
                    type="button"
                    role="switch"
                    aria-checked={display.theme === "dark"}
                    className={`toggle-switch ${display.theme === "dark" ? "active" : ""}`}
                    onClick={() =>
                      handleDisplayChange(
                        "theme",
                        display.theme === "dark" ? "light" : "dark",
                      )
                    }
                  >
                    <span className="toggle-track">
                      <span className="toggle-thumb" />
                    </span>
                    <span className="toggle-text">
                      {display.theme === "dark" ? "Dark" : "Light"}
                    </span>
                  </button>
                </label>
              </div>
            </div>
          )}

          {activeTab === "prompt" && (
            <div className="settings-group">
              <h3>Prompt Tester</h3>
              <p>Inspect the generated agent prompt without launching a run.</p>
              <div className="prompt-tester">
                <label>
                  <span>Mock issue scenario</span>
                  <select
                    value={selectedTestCaseId}
                    onChange={(event) =>
                      setSelectedTestCaseId(event.target.value)
                    }
                  >
                    {promptTesterCases.map((testCase) => (
                      <option key={testCase.id} value={testCase.id}>
                        {testCase.label}
                      </option>
                    ))}
                  </select>
                </label>
                {selectedTestCase && (
                  <div className="prompt-case-details">
                    <p className="prompt-case-summary">
                      {selectedTestCase.summary}
                    </p>
                    <div className="prompt-case-meta">
                      <span>
                        <strong>ID:</strong> {selectedTestCase.issue.id}
                      </span>
                      {selectedTestCase.issue.app_id && (
                        <span>
                          <strong>App:</strong> {selectedTestCase.issue.app_id}
                        </span>
                      )}
                      {selectedTestCase.issue.priority && (
                        <span>
                          <strong>Priority:</strong>{" "}
                          {selectedTestCase.issue.priority}
                        </span>
                      )}
                      {selectedTestCase.issue.status && (
                        <span>
                          <strong>Status:</strong>{" "}
                          {selectedTestCase.issue.status}
                        </span>
                      )}
                    </div>
                    {selectedTestCase.issue.error_context?.error_message && (
                      <div className="prompt-case-error">
                        <strong>Reported error:</strong>{" "}
                        {selectedTestCase.issue.error_context.error_message}
                      </div>
                    )}
                  </div>
                )}
                <button
                  type="button"
                  className="primary-action"
                  onClick={() => fetchPromptPreview()}
                  disabled={isPreviewLoading || !selectedTestCase}
                >
                  {isPreviewLoading ? (
                    <Loader2 className="attachment-spinner" size={16} />
                  ) : (
                    <SlidersHorizontal size={16} />
                  )}
                  <span>
                    {isPreviewLoading ? "Loading preview" : "Refresh preview"}
                  </span>
                </button>
              </div>
              {previewError && (
                <div className="prompt-error">{previewError}</div>
              )}
              {promptTemplate && (
                <div className="prompt-template">
                  <h4>Prompt Template</h4>
                  <pre>{promptTemplate}</pre>
                </div>
              )}
              {promptPreview && (
                <div className="prompt-result">
                  <h4>Agent Prompt Preview</h4>
                  {previewMeta && (
                    <div className="prompt-meta">
                      {previewMeta.issueTitle && (
                        <span>Issue: {previewMeta.issueTitle}</span>
                      )}
                      {previewSourceLabel && <span>{previewSourceLabel}</span>}
                      {formattedGeneratedAt && (
                        <span>Generated {formattedGeneratedAt}</span>
                      )}
                    </div>
                  )}
                  <pre>{promptPreview}</pre>
                </div>
              )}
            </div>
          )}

        </section>
      </div>
    </div>
  );
}
