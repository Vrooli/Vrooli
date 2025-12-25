import { MessageSquare, Plus, Sparkles, Zap, Shield } from "lucide-react";
import { Button } from "../ui/button";

interface EmptyStateProps {
  onNewChat: () => void;
  isCreating: boolean;
}

export function EmptyState({ onNewChat, isCreating }: EmptyStateProps) {
  return (
    <div className="flex-1 flex items-center justify-center bg-slate-950 p-8" data-testid="empty-state">
      <div className="text-center max-w-lg">
        {/* Icon */}
        <div className="relative inline-flex mb-6">
          <div className="w-20 h-20 rounded-2xl bg-gradient-to-br from-indigo-500/20 to-purple-500/20 flex items-center justify-center">
            <MessageSquare className="h-10 w-10 text-indigo-400" />
          </div>
          <div className="absolute -top-1 -right-1 w-6 h-6 rounded-full bg-indigo-500 flex items-center justify-center">
            <Sparkles className="h-3 w-3 text-white" />
          </div>
        </div>

        {/* Title & Description */}
        <h2 className="text-2xl font-bold text-white mb-3">Welcome to Agent Inbox</h2>
        <p className="text-slate-400 mb-8 max-w-md mx-auto">
          Your central hub for AI-powered conversations. Start a new chat to get help with coding,
          writing, research, or anything else you need.
        </p>

        {/* CTA Button */}
        <Button
          onClick={onNewChat}
          disabled={isCreating}
          size="lg"
          className="gap-2 px-8"
          data-testid="empty-state-new-chat"
        >
          <Plus className="h-5 w-5" />
          Start New Chat
        </Button>

        {/* Features */}
        <div className="mt-12 grid grid-cols-1 sm:grid-cols-3 gap-6 text-left">
          <FeatureCard
            icon={<Sparkles className="h-5 w-5 text-indigo-400" />}
            title="Smart Responses"
            description="Get intelligent answers powered by advanced AI models"
          />
          <FeatureCard
            icon={<Zap className="h-5 w-5 text-yellow-400" />}
            title="Real-time Streaming"
            description="See responses as they're generated for faster interactions"
          />
          <FeatureCard
            icon={<Shield className="h-5 w-5 text-green-400" />}
            title="Organized Inbox"
            description="Star, archive, and label conversations for easy access"
          />
        </div>

        {/* Quick tips */}
        <div className="mt-10 p-4 bg-white/5 rounded-xl border border-white/10">
          <h4 className="text-sm font-medium text-white mb-2">Quick Tips</h4>
          <ul className="text-xs text-slate-500 space-y-1 text-left">
            <li>• Press <kbd className="px-1 py-0.5 rounded bg-white/10 text-slate-400">Enter</kbd> to send messages quickly</li>
            <li>• Use <kbd className="px-1 py-0.5 rounded bg-white/10 text-slate-400">Shift+Enter</kbd> for multi-line messages</li>
            <li>• Star important conversations to find them easily later</li>
            <li>• Archive completed chats to keep your inbox clean</li>
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
