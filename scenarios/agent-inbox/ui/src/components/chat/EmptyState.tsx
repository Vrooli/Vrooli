import { useState } from "react";
import { MessageSquare, Sparkles, Zap, Shield } from "lucide-react";
import { MessageInput, type MessagePayload } from "./MessageInput";
import { ModeSelector, type ChatMode } from "./ModeSelector";
import { AgentStartModal, type AgentStartConfig } from "./AgentStartModal";
import { AttachRunModal } from "./AttachRunModal";
import { useAgentSettings } from "../../hooks/useAgentSettings";
import { selectorsManifest } from "../../consts/selectors";
import type { Model, AgentRunSummary } from "../../lib/api";

interface EmptyStateProps {
  onStartChat: (payload: MessagePayload) => void;
  onStartAgentChat: (payload: MessagePayload, config: AgentStartConfig) => void;
  /** Called when user selects a run to attach. Creates a chat and attaches the run. */
  onAttachRun?: (run: AgentRunSummary) => void;
  onOpenAgentSettings?: () => void;
  isCreating: boolean;
  models: Model[];
}

export function EmptyState({
  onStartChat,
  onStartAgentChat,
  onAttachRun,
  onOpenAgentSettings,
  isCreating,
  models,
}: EmptyStateProps) {
  const emptyStateTestIds = {
    container: selectorsManifest.selectors["emptyState.container"]?.testId ?? "empty-state",
    title: selectorsManifest.selectors["emptyState.title"]?.testId ?? "empty-state-title",
    subtitle: selectorsManifest.selectors["emptyState.subtitle"]?.testId ?? "empty-state-subtitle",
    modeHint: selectorsManifest.selectors["emptyState.modeHint"]?.testId ?? "empty-state-mode-hint",
    mobileTips: selectorsManifest.selectors["emptyState.mobileTips"]?.testId ?? "empty-state-mobile-tips",
  };
  // Use the first model as default for capability checking
  const defaultModel = models[0] ?? null;

  // Mode selection state — persisted to localStorage
  const CHAT_MODE_KEY = "agent-inbox:chat-mode";
  const [selectedMode, setSelectedModeState] = useState<ChatMode>(() => {
    const stored = localStorage.getItem(CHAT_MODE_KEY);
    return stored === "agent" ? "agent" : "llm";
  });
  const setSelectedMode = (mode: ChatMode) => {
    setSelectedModeState(mode);
    localStorage.setItem(CHAT_MODE_KEY, mode);
  };
  const [showAgentConfig, setShowAgentConfig] = useState(false);
  const [showAttachModal, setShowAttachModal] = useState(false);
  const [isAttaching, setIsAttaching] = useState(false);
  const [pendingPayload, setPendingPayload] = useState<MessagePayload | null>(null);

  // Agent settings from localStorage
  const { settings: agentSettings } = useAgentSettings();

  // Handle send with mode-aware routing
  const handleSend = (payload: MessagePayload) => {
    if (selectedMode === "agent") {
      // If we have default project path, use defaults
      if (agentSettings.defaultProjectPath) {
        onStartAgentChat(payload, {
          runner_type: agentSettings.defaultRunner,
          project_path: agentSettings.defaultProjectPath,
          model: agentSettings.defaultModel,
          max_turns: agentSettings.defaultMaxTurns,
        });
      } else {
        // Show config modal to get project path
        setPendingPayload(payload);
        setShowAgentConfig(true);
      }
    } else {
      onStartChat(payload);
    }
  };

  // Handle agent config confirmation
  const handleAgentConfigConfirm = (config: AgentStartConfig) => {
    if (pendingPayload) {
      onStartAgentChat(pendingPayload, config);
      setPendingPayload(null);
      setShowAgentConfig(false);
    }
  };

  // Handle agent config cancel
  const handleAgentConfigCancel = () => {
    setPendingPayload(null);
    setShowAgentConfig(false);
  };

  // Handle attaching an existing run
  const handleAttachRun = async (run: AgentRunSummary) => {
    if (!onAttachRun) return;
    setIsAttaching(true);
    try {
      onAttachRun(run);
      setShowAttachModal(false);
    } finally {
      setIsAttaching(false);
    }
  };

  return (
    <div className="flex-1 flex items-center justify-center bg-slate-950 p-4 sm:p-8" data-testid={emptyStateTestIds.container}>
      <div className="w-full max-w-2xl">
        {/* Header */}
        <div className="text-center mb-8">
          <div className="relative inline-flex mb-4">
            <div className="w-16 h-16 sm:w-20 sm:h-20 rounded-2xl bg-gradient-to-br from-indigo-500/20 to-purple-500/20 flex items-center justify-center">
              <MessageSquare className="h-8 w-8 sm:h-10 sm:w-10 text-indigo-400" />
            </div>
            <div className="absolute -top-1 -right-1 w-5 h-5 sm:w-6 sm:h-6 rounded-full bg-indigo-500 flex items-center justify-center">
              <Sparkles className="h-2.5 w-2.5 sm:h-3 sm:h-3 text-white" />
            </div>
          </div>
          <h2 className="text-xl sm:text-2xl font-bold text-white mb-2" data-testid={emptyStateTestIds.title}>
            What can I help you with?
          </h2>
          <p className="text-sm sm:text-base text-slate-400 max-w-md mx-auto" data-testid={emptyStateTestIds.subtitle}>
            Ask me anything about coding, writing, research, or any other topic.
          </p>
        </div>

        {/* Mode Selector + Chat Input - Primary CTA */}
        <div className="mb-8 space-y-3">
          <div className="flex items-center justify-between">
            <ModeSelector
              mode={selectedMode}
              onModeChange={setSelectedMode}
              disabled={isCreating}
              onOpenAgentSettings={onOpenAgentSettings}
              onOpenAttachModal={onAttachRun ? () => setShowAttachModal(true) : undefined}
            />
            {selectedMode === "agent" && !agentSettings.defaultProjectPath && (
              <span className="text-xs text-amber-400" data-testid={emptyStateTestIds.modeHint}>
                Project path required
              </span>
            )}
          </div>
          <MessageInput
            onSend={handleSend}
            isLoading={isCreating}
            placeholder={selectedMode === "agent"
              ? "Describe what you want the agent to do..."
              : "Type your message to start a conversation..."}
            currentModel={defaultModel}
            autoFocus
          />
        </div>

        {/* Features - Compact on mobile */}
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 sm:gap-4 text-left">
          <FeatureCard
            icon={<Sparkles className="h-4 w-4 sm:h-5 sm:w-5 text-indigo-400" />}
            title="Smart Responses"
            description="Powered by advanced AI models"
          />
          <FeatureCard
            icon={<Zap className="h-4 w-4 sm:h-5 sm:w-5 text-yellow-400" />}
            title="Real-time Streaming"
            description="See responses as they're generated"
          />
          <FeatureCard
            icon={<Shield className="h-4 w-4 sm:h-5 sm:w-5 text-green-400" />}
            title="Organized Inbox"
            description="Star, archive, and label chats"
          />
        </div>

        {/* Quick tips */}
        <div className="mt-6 sm:mt-8 p-3 sm:p-4 bg-white/5 rounded-xl border border-white/10" data-testid={emptyStateTestIds.mobileTips}>
          <h3 className="text-sm font-medium text-white mb-2">Quick Tips</h3>
          <ul className="text-xs text-slate-400 space-y-1 text-left">
            <li>Press <kbd className="px-1 py-0.5 rounded bg-white/10 text-slate-400">Ctrl+N</kbd> to create a new chat anytime</li>
            <li>Press <kbd className="px-1 py-0.5 rounded bg-white/10 text-slate-400">?</kbd> to view all keyboard shortcuts</li>
            <li>Star important conversations to find them easily later</li>
          </ul>
        </div>
      </div>

      {/* Agent Start Modal - shown when agent mode selected without default project path */}
      <AgentStartModal
        isOpen={showAgentConfig}
        onClose={handleAgentConfigCancel}
        onStart={handleAgentConfigConfirm}
        defaultSettings={agentSettings}
        isLoading={isCreating}
      />

      {/* Attach Run Modal */}
      <AttachRunModal
        isOpen={showAttachModal}
        onClose={() => setShowAttachModal(false)}
        onAttach={handleAttachRun}
        isLoading={isAttaching}
      />
    </div>
  );
}

function FeatureCard({
  icon,
  title,
  description,
}: {
  icon: React.ReactNode;
  title: string;
  description: string;
}) {
  return (
    <div className="p-4 rounded-xl bg-white/5 border border-white/10">
      <div className="w-10 h-10 rounded-lg bg-white/10 flex items-center justify-center mb-3">
        {icon}
      </div>
      <h3 className="text-sm font-medium text-white mb-1">{title}</h3>
      <p className="text-xs text-slate-400">{description}</p>
    </div>
  );
}
