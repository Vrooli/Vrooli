import { useState, useEffect } from "react";
import {
  GitCommit,
  Loader2,
  CheckCircle,
  AlertCircle,
  ChevronDown,
  ChevronRight
} from "lucide-react";
import { Card, CardHeader, CardTitle, CardContent } from "./ui/card";
import { Button } from "./ui/button";

interface CommitPanelProps {
  stagedCount: number;
  onCommit: (
    message: string,
    options: { conventional: boolean; authorName?: string; authorEmail?: string }
  ) => void;
  isCommitting: boolean;
  lastCommitHash?: string;
  commitError?: string;
  defaultAuthorName?: string;
  defaultAuthorEmail?: string;
  collapsed?: boolean;
  onToggleCollapse?: () => void;
  fillHeight?: boolean;
}

export function CommitPanel({
  stagedCount,
  onCommit,
  isCommitting,
  lastCommitHash,
  commitError,
  defaultAuthorName,
  defaultAuthorEmail,
  collapsed = false,
  onToggleCollapse,
  fillHeight = false
}: CommitPanelProps) {
  const [message, setMessage] = useState("");
  const [useConventional, setUseConventional] = useState(false);
  const [advancedOpen, setAdvancedOpen] = useState(false);
  const [authorName, setAuthorName] = useState("");
  const [authorEmail, setAuthorEmail] = useState("");
  const [authorTouched, setAuthorTouched] = useState(false);

  const canCommit = stagedCount > 0 && message.trim().length > 0 && !isCommitting;
  const handleToggleCollapse = onToggleCollapse ?? (() => {});

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (canCommit) {
      onCommit(message, {
        conventional: useConventional,
        authorName: authorName.trim() || defaultAuthorName || undefined,
        authorEmail: authorEmail.trim() || defaultAuthorEmail || undefined
      });
      setMessage("");
    }
  };

  useEffect(() => {
    if (authorTouched) return;
    if (defaultAuthorName) {
      setAuthorName(defaultAuthorName);
    }
    if (defaultAuthorEmail) {
      setAuthorEmail(defaultAuthorEmail);
    }
  }, [authorTouched, defaultAuthorName, defaultAuthorEmail]);

  return (
    <Card
      className={`flex flex-col min-w-0 ${fillHeight ? "h-full" : "h-auto"}`}
      data-testid="commit-panel"
    >
      <CardHeader className="py-3 min-w-0">
        <CardTitle className="flex items-center gap-2 text-sm min-w-0">
          <button
            className="p-1 rounded hover:bg-slate-800/70 transition-colors"
            onClick={handleToggleCollapse}
            aria-label={collapsed ? "Expand commit panel" : "Collapse commit panel"}
            type="button"
          >
            {collapsed ? (
              <ChevronRight className="h-3 w-3 text-slate-400" />
            ) : (
              <ChevronDown className="h-3 w-3 text-slate-400" />
            )}
          </button>
          <GitCommit className="h-4 w-4 text-slate-500" />
          <span className="truncate">Commit</span>
        </CardTitle>
      </CardHeader>

      {!collapsed && (
        <CardContent className="flex-1 min-h-0 min-w-0 pt-0 pb-3 overflow-y-auto overflow-x-hidden">
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

          <div className="flex items-center justify-between gap-2 min-w-0">
            <button
              type="button"
              className="flex items-center gap-2 text-xs text-slate-400 hover:text-slate-200 transition-colors"
              onClick={() => setAdvancedOpen((prev) => !prev)}
            >
              {advancedOpen ? (
                <ChevronDown className="h-3 w-3" />
              ) : (
                <ChevronRight className="h-3 w-3" />
              )}
              Advanced
            </button>

            <Button
              type="submit"
              variant="default"
              size="sm"
              disabled={!canCommit}
              className="min-w-0 max-w-full"
              data-testid="commit-button"
            >
              {isCommitting ? (
                <span className="flex items-center min-w-0">
                  <Loader2 className="h-3 w-3 mr-1 animate-spin" />
                  <span className="truncate">Committing...</span>
                </span>
              ) : (
                <span className="flex items-center min-w-0">
                  <GitCommit className="h-3 w-3 mr-1" />
                  <span className="truncate">
                    Commit ({stagedCount} file{stagedCount !== 1 ? "s" : ""})
                  </span>
                </span>
              )}
            </Button>
          </div>

          {advancedOpen && (
            <div className="rounded-md border border-slate-800 bg-slate-900/40 p-3 space-y-3">
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

              <div className="space-y-2">
                <p className="text-xs text-slate-500">Commit identity (overrides local git config)</p>
                <div className="grid gap-2">
                  <input
                    value={authorName}
                    onChange={(e) => {
                      setAuthorName(e.target.value);
                      setAuthorTouched(true);
                    }}
                    placeholder="Author name"
                    className="w-full px-3 py-2 text-xs bg-slate-800/50 border border-slate-700 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent placeholder:text-slate-500"
                    disabled={isCommitting}
                  />
                  <input
                    value={authorEmail}
                    onChange={(e) => {
                      setAuthorEmail(e.target.value);
                      setAuthorTouched(true);
                    }}
                    placeholder="Author email"
                    className="w-full px-3 py-2 text-xs bg-slate-800/50 border border-slate-700 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent placeholder:text-slate-500"
                    disabled={isCommitting}
                  />
                </div>
              </div>
            </div>
          )}

          {/* Success feedback */}
          {lastCommitHash && !commitError && (
            <div
              className="flex items-center gap-2 px-3 py-2 bg-emerald-950/30 border border-emerald-800/50 rounded-md text-xs text-emerald-400"
              data-testid="commit-success"
            >
              <CheckCircle className="h-3.5 w-3.5" />
              Committed as <code className="font-mono break-all">{lastCommitHash}</code>
            </div>
          )}

          {/* Error feedback */}
          {commitError && (
            <div
              className="flex items-start gap-2 px-3 py-2 bg-red-950/30 border border-red-800/50 rounded-md text-xs text-red-400"
              data-testid="commit-error"
            >
              <AlertCircle className="h-3.5 w-3.5 mt-0.5 flex-shrink-0" />
              <span className="break-words">{commitError}</span>
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
      )}
    </Card>
  );
}
