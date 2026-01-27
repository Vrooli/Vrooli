import { useState } from "react";
import { X, Bot, FolderOpen, Cpu, Zap } from "lucide-react";
import type { RunnerType } from "../../lib/api";
import type { AgentModeSettings } from "../../hooks/useAgentSettings";

interface AgentStartModalProps {
  /** Whether the modal is open */
  isOpen: boolean;
  /** Called when the modal should close */
  onClose: () => void;
  /** Called when the user confirms starting agent mode */
  onStart: (config: AgentStartConfig) => void;
  /** Default settings from useAgentSettings */
  defaultSettings: AgentModeSettings;
  /** Whether the start action is in progress */
  isLoading?: boolean;
}

export interface AgentStartConfig {
  runner_type: RunnerType;
  project_path: string;
  model: string;
  max_turns: number;
}

const RUNNER_OPTIONS: { value: RunnerType; label: string; description: string }[] = [
  {
    value: "claude-code",
    label: "Claude Code",
    description: "Anthropic's official CLI agent"
  },
  {
    value: "codex",
    label: "Codex",
    description: "OpenAI Codex CLI agent"
  },
  {
    value: "opencode",
    label: "OpenCode",
    description: "Open-source coding agent"
  }
];

/**
 * Modal for configuring agent mode settings before starting.
 * Shows runner selection, project path, and optional overrides.
 */
export function AgentStartModal({
  isOpen,
  onClose,
  onStart,
  defaultSettings,
  isLoading = false
}: AgentStartModalProps) {
  const [runnerType, setRunnerType] = useState<RunnerType>(defaultSettings.defaultRunner);
  const [projectPath, setProjectPath] = useState(defaultSettings.defaultProjectPath);
  const [model, setModel] = useState(defaultSettings.defaultModel);
  const [maxTurns, setMaxTurns] = useState(defaultSettings.defaultMaxTurns);

  if (!isOpen) return null;

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onStart({
      runner_type: runnerType,
      project_path: projectPath,
      model,
      max_turns: maxTurns
    });
  };

  const handleUseDefaults = () => {
    if (!defaultSettings.defaultProjectPath) {
      // Can't use defaults without a project path
      return;
    }
    onStart({
      runner_type: defaultSettings.defaultRunner,
      project_path: defaultSettings.defaultProjectPath,
      model: defaultSettings.defaultModel,
      max_turns: defaultSettings.defaultMaxTurns
    });
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/60 backdrop-blur-sm"
        onClick={onClose}
      />

      {/* Modal */}
      <div className="relative w-full max-w-md mx-4 bg-zinc-900 border border-zinc-700 rounded-xl shadow-2xl">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-zinc-700">
          <div className="flex items-center gap-3">
            <div className="p-2 rounded-lg bg-blue-500/10">
              <Bot className="h-5 w-5 text-blue-400" />
            </div>
            <div>
              <h2 className="text-lg font-semibold text-white">Start Agent Mode</h2>
              <p className="text-sm text-zinc-400">Configure the coding agent</p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="p-1 rounded-md text-zinc-400 hover:text-white hover:bg-zinc-700 transition-colors"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Body */}
        <form onSubmit={handleSubmit} className="p-6 space-y-5">
          {/* Runner Selection */}
          <div>
            <label className="flex items-center gap-2 text-sm font-medium text-zinc-300 mb-2">
              <Cpu className="h-4 w-4" />
              Runner
            </label>
            <div className="space-y-2">
              {RUNNER_OPTIONS.map(option => (
                <label
                  key={option.value}
                  className={`
                    flex items-start gap-3 p-3 rounded-lg cursor-pointer
                    border transition-colors
                    ${runnerType === option.value
                      ? "border-blue-500 bg-blue-500/10"
                      : "border-zinc-700 hover:border-zinc-600"
                    }
                  `}
                >
                  <input
                    type="radio"
                    name="runner"
                    value={option.value}
                    checked={runnerType === option.value}
                    onChange={(e) => setRunnerType(e.target.value as RunnerType)}
                    className="mt-1"
                  />
                  <div>
                    <div className="font-medium text-white">{option.label}</div>
                    <div className="text-xs text-zinc-400">{option.description}</div>
                  </div>
                </label>
              ))}
            </div>
          </div>

          {/* Project Path */}
          <div>
            <label className="flex items-center gap-2 text-sm font-medium text-zinc-300 mb-2">
              <FolderOpen className="h-4 w-4" />
              Project Path
            </label>
            <input
              type="text"
              value={projectPath}
              onChange={(e) => setProjectPath(e.target.value)}
              placeholder="/path/to/project"
              required
              className="
                w-full px-3 py-2 rounded-lg
                bg-zinc-800 border border-zinc-700
                text-white placeholder-zinc-500
                focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500
              "
            />
            <p className="mt-1 text-xs text-zinc-500">
              Directory where the agent will operate
            </p>
          </div>

          {/* Advanced Options */}
          <details className="group">
            <summary className="flex items-center gap-2 text-sm font-medium text-zinc-400 cursor-pointer hover:text-zinc-300">
              <Zap className="h-4 w-4" />
              Advanced Options
            </summary>
            <div className="mt-3 space-y-4 pl-6">
              {/* Model */}
              <div>
                <label className="text-sm text-zinc-400 mb-1 block">Model (optional)</label>
                <input
                  type="text"
                  value={model}
                  onChange={(e) => setModel(e.target.value)}
                  placeholder="Use runner default"
                  className="
                    w-full px-3 py-2 rounded-lg
                    bg-zinc-800 border border-zinc-700
                    text-white placeholder-zinc-500
                    focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500
                    text-sm
                  "
                />
              </div>

              {/* Max Turns */}
              <div>
                <label className="text-sm text-zinc-400 mb-1 block">Max Turns (0 = unlimited)</label>
                <input
                  type="number"
                  min="0"
                  value={maxTurns}
                  onChange={(e) => setMaxTurns(parseInt(e.target.value) || 0)}
                  className="
                    w-full px-3 py-2 rounded-lg
                    bg-zinc-800 border border-zinc-700
                    text-white placeholder-zinc-500
                    focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500
                    text-sm
                  "
                />
              </div>
            </div>
          </details>
        </form>

        {/* Footer */}
        <div className="flex items-center justify-between gap-3 px-6 py-4 border-t border-zinc-700">
          {defaultSettings.defaultProjectPath ? (
            <button
              type="button"
              onClick={handleUseDefaults}
              disabled={isLoading}
              className="
                px-4 py-2 rounded-lg text-sm font-medium
                text-zinc-300 hover:text-white hover:bg-zinc-700
                transition-colors disabled:opacity-50
              "
            >
              Use Defaults
            </button>
          ) : (
            <div />
          )}
          <div className="flex gap-2">
            <button
              type="button"
              onClick={onClose}
              disabled={isLoading}
              className="
                px-4 py-2 rounded-lg text-sm font-medium
                text-zinc-300 hover:text-white hover:bg-zinc-700
                transition-colors disabled:opacity-50
              "
            >
              Cancel
            </button>
            <button
              type="submit"
              onClick={handleSubmit}
              disabled={isLoading || !projectPath}
              className="
                px-4 py-2 rounded-lg text-sm font-medium
                bg-blue-600 text-white hover:bg-blue-500
                transition-colors disabled:opacity-50 disabled:cursor-not-allowed
              "
            >
              {isLoading ? "Starting..." : "Start Agent"}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

export default AgentStartModal;
