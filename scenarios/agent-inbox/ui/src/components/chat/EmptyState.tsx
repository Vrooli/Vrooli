import { MessageSquare, Sparkles, Zap, Shield } from "lucide-react";
import { MessageInput, type MessagePayload } from "./MessageInput";
import type { Model } from "../../lib/api";

interface EmptyStateProps {
  onStartChat: (payload: MessagePayload) => void;
  isCreating: boolean;
  models: Model[];
}

export function EmptyState({ onStartChat, isCreating, models }: EmptyStateProps) {
  // Use the first model as default for capability checking
  const defaultModel = models[0] ?? null;

  return (
    <div className="flex-1 flex items-center justify-center bg-slate-950 p-4 sm:p-8" data-testid="empty-state">
      <div className="w-full max-w-2xl">
        {/* Header */}
        <div className="text-center mb-8">
          <div className="relative inline-flex mb-4">
            <div className="w-16 h-16 sm:w-20 sm:h-20 rounded-2xl bg-gradient-to-br from-indigo-500/20 to-purple-500/20 flex items-center justify-center">
              <MessageSquare className="h-8 w-8 sm:h-10 sm:w-10 text-indigo-400" />
            </div>
            <div className="absolute -top-1 -right-1 w-5 h-5 sm:w-6 sm:h-6 rounded-full bg-indigo-500 flex items-center justify-center">
              <Sparkles className="h-2.5 w-2.5 sm:h-3 sm:w-3 text-white" />
            </div>
          </div>
          <h2 className="text-xl sm:text-2xl font-bold text-white mb-2">What can I help you with?</h2>
          <p className="text-sm sm:text-base text-slate-400 max-w-md mx-auto">
            Ask me anything about coding, writing, research, or any other topic.
          </p>
        </div>

        {/* Chat Input - Primary CTA */}
        <div className="mb-8">
          <MessageInput
            onSend={onStartChat}
            isLoading={isCreating}
            placeholder="Type your message to start a conversation..."
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

        {/* Quick tips - Hidden on mobile for cleaner experience */}
        <div className="hidden sm:block mt-8 p-4 bg-white/5 rounded-xl border border-white/10">
          <h4 className="text-sm font-medium text-white mb-2">Quick Tips</h4>
          <ul className="text-xs text-slate-500 space-y-1 text-left">
            <li>Press <kbd className="px-1 py-0.5 rounded bg-white/10 text-slate-400">Ctrl+N</kbd> to create a new chat anytime</li>
            <li>Press <kbd className="px-1 py-0.5 rounded bg-white/10 text-slate-400">?</kbd> to view all keyboard shortcuts</li>
            <li>Star important conversations to find them easily later</li>
          </ul>
        </div>
      </div>
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
      <h4 className="text-sm font-medium text-white mb-1">{title}</h4>
      <p className="text-xs text-slate-500">{description}</p>
    </div>
  );
}
