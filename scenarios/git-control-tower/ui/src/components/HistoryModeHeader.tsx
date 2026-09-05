import { GitCommit, History, X } from "lucide-react";
import { Badge } from "./ui/badge";
import { Button } from "./ui/button";
import type { CommitCheckRun } from "../lib/api";

export interface ViewingCommit {
  hash: string;
  subject: string;
  files: string[];
  author?: string;
  date?: string;
  checks?: CommitCheckRun[];
}

interface HistoryModeHeaderProps {
  commit: ViewingCommit;
  onExit: () => void;
  compact?: boolean;
}

export function HistoryModeHeader({ commit, onExit, compact }: HistoryModeHeaderProps) {
  if (compact) {
    return (
      <header
        className="flex items-center justify-between px-3 py-2 border-b border-amber-800/50 bg-amber-950/30 backdrop-blur-sm pt-safe"
        data-testid="mobile-header"
      >
        <div className="flex items-center gap-2 min-w-0 flex-1">
          <History className="h-4 w-4 text-amber-400 flex-shrink-0" />
          <Badge variant="warning" className="flex-shrink-0 text-xs">
            History
          </Badge>
          <GitCommit className="h-3.5 w-3.5 text-amber-400 flex-shrink-0" />
          <span className="font-mono text-xs text-amber-200">
            {commit.hash.substring(0, 7)}
          </span>
        </div>

        <Button
          variant="outline"
          size="sm"
          onClick={onExit}
          className="gap-1 border-amber-600/50 text-amber-200 hover:bg-amber-900/30 text-xs px-2"
        >
          <X className="h-3.5 w-3.5" />
          Exit
        </Button>
      </header>
    );
  }

  return (
    <header
      className="relative z-30 flex items-center justify-between px-4 py-3 border-b border-amber-800/50 bg-amber-950/30 backdrop-blur-sm"
      data-testid="status-header"
    >
      <div className="flex items-center gap-4 min-w-0 flex-1">
        <div className="flex items-center gap-2 flex-shrink-0">
          <History className="h-4 w-4 text-amber-400" />
          <Badge variant="warning" className="text-xs">
            Viewing History
          </Badge>
        </div>

        <div className="flex items-center gap-3 min-w-0" data-testid="history-commit-info">
          <div className="flex items-center gap-2 flex-shrink-0">
            <GitCommit className="h-4 w-4 text-amber-400" />
            <span className="font-mono text-sm text-amber-200">
              {commit.hash.substring(0, 7)}
            </span>
          </div>
          <span className="text-sm text-slate-300 truncate" title={commit.subject}>
            {commit.subject}
          </span>
        </div>
      </div>

      <div className="flex items-center gap-3 flex-shrink-0">
        {commit.author && (
          <span className="text-xs text-slate-500 hidden sm:block">
            by {commit.author}
          </span>
        )}

        <Button
          variant="outline"
          size="sm"
          onClick={onExit}
          className="gap-1.5 border-amber-600/50 text-amber-200 hover:bg-amber-900/30"
          data-testid="exit-history-mode"
        >
          <X className="h-3.5 w-3.5" />
          Back to Working Directory
        </Button>
      </div>
    </header>
  );
}
