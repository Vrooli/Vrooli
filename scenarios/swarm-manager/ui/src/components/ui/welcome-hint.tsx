/**
 * Welcome Hint Component
 *
 * A dismissible hint for first-time users showing key features and shortcuts.
 * Uses localStorage to track if the user has dismissed it.
 *
 * Experience Architecture (Phase 29 Iteration 5):
 * - Addresses cognitive friction for new users
 * - Surfaces key capabilities (keyboard shortcuts, main actions)
 * - Dismissible and won't show again after dismissed
 */

import { useState, useEffect } from "react";
import { X, Keyboard, Lightbulb, Package, Zap } from "lucide-react";

const STORAGE_KEY = "swarm-manager-welcome-dismissed";

interface WelcomeHintProps {
  "data-testid"?: string;
}

export function WelcomeHint({ "data-testid": testId }: WelcomeHintProps) {
  const [isDismissed, setIsDismissed] = useState(true); // Start hidden to prevent flash

  useEffect(() => {
    // Check localStorage on mount
    try {
      const dismissed = localStorage.getItem(STORAGE_KEY);
      setIsDismissed(dismissed === "true");
    } catch {
      // localStorage not available, don't show hint
      setIsDismissed(true);
    }
  }, []);

  const handleDismiss = () => {
    setIsDismissed(true);
    try {
      localStorage.setItem(STORAGE_KEY, "true");
    } catch {
      // localStorage not available, just hide it
    }
  };

  if (isDismissed) {
    return null;
  }

  return (
    <div
      className="mb-6 rounded-lg border border-cyan-500/30 bg-cyan-500/5 p-4"
      data-testid={testId}
    >
      <div className="flex items-start justify-between gap-4">
        <div className="flex-1 space-y-3">
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium text-cyan-300">Welcome to Swarm Manager</span>
          </div>
          <p className="text-sm text-slate-300">
            Your command center for managing the Vrooli scenario ecosystem.
          </p>
          <div className="flex flex-wrap gap-4 text-xs">
            <div className="flex items-center gap-2 text-slate-300">
              <Lightbulb className="h-3.5 w-3.5 text-cyan-400" />
              <span><strong className="text-slate-300">Backlog</strong> - Track research, ideas, fixes, and execution</span>
            </div>
            <div className="flex items-center gap-2 text-slate-300">
              <Package className="h-3.5 w-3.5 text-cyan-400" />
              <span><strong className="text-slate-300">Scenarios</strong> - Monitor what's running</span>
            </div>
            <div className="flex items-center gap-2 text-slate-300">
              <Zap className="h-3.5 w-3.5 text-cyan-400" />
              <span><strong className="text-slate-300">Execution</strong> - Pending, running, completed, failed runs</span>
            </div>
          </div>
          <div className="flex items-center gap-2 rounded bg-slate-800/50 px-2.5 py-1.5 text-xs text-slate-300 w-fit">
            <Keyboard className="h-3.5 w-3.5" />
            <span>Keyboard shortcuts: Press <kbd className="rounded bg-slate-700 px-1.5 py-0.5 font-mono text-slate-300">1</kbd>-<kbd className="rounded bg-slate-700 px-1.5 py-0.5 font-mono text-slate-300">4</kbd> to switch tabs</span>
          </div>
        </div>
        <button
          onClick={handleDismiss}
          className="p-1 text-slate-300 hover:text-slate-200 transition-colors"
          aria-label="Dismiss welcome hint"
        >
          <X className="h-4 w-4" />
        </button>
      </div>
    </div>
  );
}
