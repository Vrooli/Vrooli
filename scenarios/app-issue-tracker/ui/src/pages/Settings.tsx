import { FormEvent, useMemo, useState } from 'react';
import {
  ActivitySquare,
  Bot,
  MonitorCog,
  Network,
  SlidersHorizontal,
  Sparkles,
} from 'lucide-react';
import {
  ActiveAgentOption,
  AgentSettings,
  DisplaySettings,
  ProcessorSettings,
  RateLimits,
} from '../data/sampleData';

interface SettingsPageProps {
  processor: ProcessorSettings;
  agent: AgentSettings;
  display: DisplaySettings;
  rateLimits: RateLimits;
  activeAgents: ActiveAgentOption[];
  onProcessorChange: (settings: ProcessorSettings) => void;
  onAgentChange: (settings: AgentSettings) => void;
  onDisplayChange: (settings: DisplaySettings) => void;
  onRateLimitChange: (settings: RateLimits) => void;
  onAgentsUpdate: (agents: ActiveAgentOption[]) => void;
}

type TabKey = 'processor' | 'agent' | 'display' | 'prompt' | 'rate';

const tabConfig: { key: TabKey; label: string; icon: React.ComponentType<{ size?: number }> }[] = [
  { key: 'processor', label: 'Processor', icon: ActivitySquare },
  { key: 'agent', label: 'Agent', icon: Bot },
  { key: 'display', label: 'Display', icon: MonitorCog },
  { key: 'prompt', label: 'Prompt Tester', icon: Sparkles },
  { key: 'rate', label: 'Rate Limits', icon: Network },
];

export function SettingsPage({
  processor,
  agent,
  display,
  rateLimits,
  activeAgents,
  onProcessorChange,
  onAgentChange,
  onDisplayChange,
  onRateLimitChange,
  onAgentsUpdate,
}: SettingsPageProps) {
  const [activeTab, setActiveTab] = useState<TabKey>('processor');
  const [promptInput, setPromptInput] = useState('Summarize the root cause of ISSUE-142.');
  const [promptContext, setPromptContext] = useState('Detected API latency spike above SLA during chaos experiments.');
  const [promptResult, setPromptResult] = useState<string | null>(null);

  const allowedToolsString = useMemo(() => agent.allowedTools.join(', '), [agent.allowedTools]);

  const handleProcessorChange = (field: keyof ProcessorSettings, value: boolean | number) => {
    onProcessorChange({
      ...processor,
      [field]: value,
    });
  };

  const handleAgentChange = (field: keyof AgentSettings, value: AgentSettings[keyof AgentSettings]) => {
    if (field === 'allowedTools' && typeof value === 'string') {
      const tools = value
        .split(',')
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

  const handleDisplayChange = (field: keyof DisplaySettings, value: DisplaySettings[keyof DisplaySettings]) => {
    onDisplayChange({
      ...display,
      [field]: value,
    });
  };

  const handleRateLimitChange = (field: keyof RateLimits, value: number) => {
    onRateLimitChange({
      ...rateLimits,
      [field]: value,
    });
  };

  const runPromptTest = (event: FormEvent) => {
    event.preventDefault();
    setPromptResult(
      `resource-codex responded after simulated analysis:\n- Identified primary cause from instrumentation logs\n- Proposed fix: tune rate limiter window\n- Confidence: ${(Math.random() * 0.3 + 0.6).toFixed(2)}`,
    );
  };

  const handleRemoveAgent = (agentId: string) => {
    onAgentsUpdate(activeAgents.filter((option) => option.id !== agentId));
  };

  return (
    <div className="settings-page">
      <header className="page-header">
        <div>
          <h2>Scenario Settings</h2>
          <p>Configure automation capacity, resource-codex agents, and UI preferences.</p>
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
                className={`settings-tab ${active ? 'active' : ''}`}
                onClick={() => setActiveTab(tab.key)}
              >
                <Icon size={16} />
                <span>{tab.label}</span>
              </button>
            );
          })}
        </nav>
        <section className="settings-panel">
          {activeTab === 'processor' && (
            <div className="settings-group">
              <h3>Queue Processor</h3>
              <p>Control how many issues resource-codex tackles simultaneously.</p>
              <div className="form-grid">
                <label>
                  <span>Concurrent slots</span>
                  <input
                    type="number"
                    min={1}
                    max={12}
                    value={processor.concurrentSlots}
                    onChange={(event) => handleProcessorChange('concurrentSlots', Number(event.target.value))}
                  />
                </label>
                <label>
                  <span>Refresh interval (seconds)</span>
                  <input
                    type="number"
                    min={15}
                    max={600}
                    value={processor.refreshInterval}
                    onChange={(event) => handleProcessorChange('refreshInterval', Number(event.target.value))}
                  />
                </label>
                <label className="checkbox-field">
                  <input
                    type="checkbox"
                    checked={processor.active}
                    onChange={(event) => handleProcessorChange('active', event.target.checked)}
                  />
                  <span>Processor active</span>
                </label>
              </div>
            </div>
          )}

          {activeTab === 'agent' && (
            <div className="settings-group">
              <h3>resource-codex Agent</h3>
              <p>Fine-tune how the automation agent interacts with issue investigations.</p>
              <div className="form-grid">
                <label>
                  <span>Agent identifier</span>
                  <input
                    type="text"
                    value={agent.agentId}
                    onChange={(event) => handleAgentChange('agentId', event.target.value)}
                  />
                </label>
                <label>
                  <span>Maximum turns</span>
                  <input
                    type="number"
                    min={5}
                    max={120}
                    value={agent.maximumTurns}
                    onChange={(event) => handleAgentChange('maximumTurns', Number(event.target.value))}
                  />
                </label>
                <label>
                  <span>Allowed tools</span>
                  <input
                    type="text"
                    value={allowedToolsString}
                    onChange={(event) => handleAgentChange('allowedTools', event.target.value)}
                    placeholder="Comma separated"
                  />
                </label>
                <label>
                  <span>Task timeout (minutes)</span>
                  <input
                    type="number"
                    min={5}
                    max={180}
                    value={agent.taskTimeout}
                    onChange={(event) => handleAgentChange('taskTimeout', Number(event.target.value))}
                  />
                </label>
                <label className="checkbox-field">
                  <input
                    type="checkbox"
                    checked={agent.skipPermissionChecks}
                    onChange={(event) => handleAgentChange('skipPermissionChecks', event.target.checked)}
                  />
                  <span>Skip permission checks</span>
                </label>
              </div>

              <div className="active-agent-list">
                <h4>Active agent slots</h4>
                <ul>
                  {activeAgents.map((option) => (
                    <li key={option.id}>
                      <span>{option.label}</span>
                      <button type="button" onClick={() => handleRemoveAgent(option.id)}>
                        Remove
                      </button>
                    </li>
                  ))}
                </ul>
              </div>
            </div>
          )}

          {activeTab === 'display' && (
            <div className="settings-group">
              <h3>Display</h3>
              <p>Adjust the UI surface for dashboards and kanban boards.</p>
              <div className="form-grid">
                <label>
                  <span>Theme</span>
                  <select value={display.theme} onChange={(event) => handleDisplayChange('theme', event.target.value as DisplaySettings['theme'])}>
                    <option value="light">Light</option>
                    <option value="dark">Dark</option>
                  </select>
                </label>
                <label className="checkbox-field">
                  <input
                    type="checkbox"
                    checked={display.condensedMode}
                    onChange={(event) => handleDisplayChange('condensedMode', event.target.checked)}
                  />
                  <span>Condensed layout</span>
                </label>
                <label className="checkbox-field">
                  <input
                    type="checkbox"
                    checked={display.showTooltips}
                    onChange={(event) => handleDisplayChange('showTooltips', event.target.checked)}
                  />
                  <span>Show contextual tooltips</span>
                </label>
                <label className="checkbox-field">
                  <input
                    type="checkbox"
                    checked={display.sidebarMini}
                    onChange={(event) => handleDisplayChange('sidebarMini', event.target.checked)}
                  />
                  <span>Mini sidebar variant</span>
                </label>
              </div>
            </div>
          )}

          {activeTab === 'prompt' && (
            <div className="settings-group">
              <h3>Prompt Tester</h3>
              <p>Quickly iterate on agent prompts before shipping them to production.</p>
              <form className="prompt-tester" onSubmit={runPromptTest}>
                <label>
                  <span>Prompt</span>
                  <textarea value={promptInput} onChange={(event) => setPromptInput(event.target.value)} rows={4} />
                </label>
                <label>
                  <span>Context</span>
                  <textarea value={promptContext} onChange={(event) => setPromptContext(event.target.value)} rows={4} />
                </label>
                <button type="submit" className="primary-action">
                  <SlidersHorizontal size={16} />
                  <span>Run test</span>
                </button>
              </form>
              {promptResult && (
                <div className="prompt-result">
                  <h4>Simulated Output</h4>
                  <pre>{promptResult}</pre>
                </div>
              )}
            </div>
          )}

          {activeTab === 'rate' && (
            <div className="settings-group">
              <h3>Rate Limits</h3>
              <p>Protect downstream resources with conservative request caps.</p>
              <div className="form-grid">
                <label>
                  <span>Tokens per minute</span>
                  <input
                    type="number"
                    min={1000}
                    value={rateLimits.tokensPerMinute}
                    onChange={(event) => handleRateLimitChange('tokensPerMinute', Number(event.target.value))}
                  />
                </label>
                <label>
                  <span>Requests per hour</span>
                  <input
                    type="number"
                    min={10}
                    value={rateLimits.requestsPerHour}
                    onChange={(event) => handleRateLimitChange('requestsPerHour', Number(event.target.value))}
                  />
                </label>
                <label>
                  <span>Burst capacity</span>
                  <input
                    type="number"
                    min={1}
                    value={rateLimits.burstRequests}
                    onChange={(event) => handleRateLimitChange('burstRequests', Number(event.target.value))}
                  />
                </label>
              </div>
            </div>
          )}
        </section>
      </div>
    </div>
  );
}
