import { useState } from "react";
import { GitCommit, Loader2, CheckCircle, AlertCircle } from "lucide-react";
import { Card, CardHeader, CardTitle, CardContent } from "./ui/card";
import { Button } from "./ui/button";

interface CommitPanelProps {
  stagedCount: number;
  onCommit: (message: string, conventional: boolean) => void;
  isCommitting: boolean;
  lastCommitHash?: string;
  commitError?: string;
}

export function CommitPanel({
  stagedCount,
  onCommit,
  isCommitting,
  lastCommitHash,
  commitError
}: CommitPanelProps) {
  const [message, setMessage] = useState("");
  const [useConventional, setUseConventional] = useState(false);

  const canCommit = stagedCount > 0 && message.trim().length > 0 && !isCommitting;

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (canCommit) {
      onCommit(message, useConventional);
      setMessage("");
    }
  };

  return (
    <Card data-testid="commit-panel">
      <CardHeader className="py-3">
        <CardTitle className="flex items-center gap-2 text-sm">
          <GitCommit className="h-4 w-4 text-slate-500" />
          Commit
        </CardTitle>
      </CardHeader>

      <CardContent className="pt-0 pb-3">
        <form onSubmit={handleSubmit} className="space-y-3">
          <div>
            <textarea
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              placeholder="Commit message..."
              className="w-full h-20 px-3 py-2 text-sm bg-slate-800/50 border border-slate-700 rounded-md resize-none focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent placeholder:text-slate-500"
              disabled={isCommitting}
              data-testid="commit-message-input"
            />
          </div>

          <div className="flex items-center justify-between">
            <label className="flex items-center gap-2 text-xs text-slate-400 cursor-pointer">
              <input
                type="checkbox"
                checked={useConventional}
                onChange={(e) => setUseConventional(e.target.checked)}
                disabled={isCommitting}
                className="w-3.5 h-3.5 rounded border-slate-600 bg-slate-800 text-blue-500 focus:ring-blue-500 focus:ring-offset-0"
                data-testid="conventional-commit-checkbox"
              />
              Conventional commit format
            </label>

            <Button
              type="submit"
              variant="default"
              size="sm"
              disabled={!canCommit}
              data-testid="commit-button"
            >
              {isCommitting ? (
                <>
                  <Loader2 className="h-3 w-3 mr-1 animate-spin" />
                  Committing...
                </>
              ) : (
                <>
                  <GitCommit className="h-3 w-3 mr-1" />
                  Commit ({stagedCount} file{stagedCount !== 1 ? "s" : ""})
                </>
              )}
            </Button>
          </div>

          {/* Success feedback */}
          {lastCommitHash && !commitError && (
            <div
              className="flex items-center gap-2 px-3 py-2 bg-emerald-950/30 border border-emerald-800/50 rounded-md text-xs text-emerald-400"
              data-testid="commit-success"
            >
              <CheckCircle className="h-3.5 w-3.5" />
              Committed as <code className="font-mono">{lastCommitHash}</code>
            </div>
          )}

          {/* Error feedback */}
          {commitError && (
            <div
              className="flex items-start gap-2 px-3 py-2 bg-red-950/30 border border-red-800/50 rounded-md text-xs text-red-400"
              data-testid="commit-error"
            >
              <AlertCircle className="h-3.5 w-3.5 mt-0.5 flex-shrink-0" />
              <span>{commitError}</span>
            </div>
          )}

          {/* Helper text when nothing staged */}
          {stagedCount === 0 && (
            <p className="text-xs text-slate-500 italic">
              Stage files to enable commit
            </p>
          )}
        </form>
      </CardContent>
    </Card>
  );
}
