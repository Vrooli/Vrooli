// DOC: docs/reference/configuration.md#launcher-shortcuts
// DOC: docs/internal/SEAMS.md#1-entry--presentation
import { useState, useCallback, useEffect } from "react";
import { Terminal, Zap, X } from "lucide-react";
import { Button } from "./ui/button";
import { DEFAULT_SHORTCUTS, type ShortcutEntry } from "../consts/shortcuts";
import { getEffectiveShortcuts } from "../lib/api";
import { slugify } from "../lib/slugify";

// [REQ:P0-006a] Terminal Launch Flow UI
// [REQ:P0-006b] Configurable Shortcut Entries
// [REQ:P1-002b] Shortcut Profile Management UI

/** Shared class string for launcher option cards. */
const optionCardClass =
  "flex w-full items-center gap-3 rounded-md border border-wc-default bg-wc-surface-input px-4 py-3 text-left transition hover:border-wc-accent hover:bg-wc-surface-input/80 disabled:opacity-50";

interface TerminalLauncherProps {
  open: boolean;
  onClose: () => void;
  onLaunch: (command?: string) => void;
  shortcuts?: ShortcutEntry[];
  isCreating?: boolean;
}

export default function TerminalLauncher({
  open,
  onClose,
  onLaunch,
  shortcuts: shortcutsProp,
  isCreating = false,
}: TerminalLauncherProps) {
  const [customCommand, setCustomCommand] = useState("");
  const [apiShortcuts, setApiShortcuts] = useState<ShortcutEntry[] | null>(null);

  // [REQ:P1-002b] Fetch configuration-driven shortcuts from API on open.
  // Falls back to DEFAULT_SHORTCUTS if the API call fails or prop is provided.
  useEffect(() => {
    if (!open || shortcutsProp) return;
    let cancelled = false;
    getEffectiveShortcuts()
      .then((data) => {
        if (!cancelled) setApiShortcuts(data);
      })
      .catch(() => {
        if (!cancelled) setApiShortcuts(null);
      });
    return () => { cancelled = true; };
  }, [open, shortcutsProp]);

  const shortcuts = shortcutsProp ?? apiShortcuts ?? DEFAULT_SHORTCUTS;

  // Custom command launch is separate because it validates non-empty input
  // and clears the text field after launching.
  const handleLaunchCustom = useCallback(() => {
    if (customCommand.trim()) {
      onLaunch(customCommand.trim());
      setCustomCommand("");
    }
  }, [customCommand, onLaunch]);

  if (!open) return null;

  return (
    <div
      data-testid="terminal-launcher"
      className="fixed inset-0 z-50 flex items-center justify-center bg-wc-backdrop-heavy"
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div className="mx-4 w-full max-w-md rounded-lg border border-wc-default bg-wc-surface-raised shadow-xl">
        <div className="flex items-center justify-between border-b border-wc-default px-4 py-3">
          <h2 className="text-lg font-semibold text-wc-text-primary">
            New Terminal
          </h2>
          <Button
            variant="ghost"
            size="icon"
            className="h-7 w-7"
            onClick={onClose}
          >
            <X className="h-4 w-4" />
          </Button>
        </div>

        <div className="space-y-3 p-4">
          {/* Empty shell option */}
          <button
            data-testid="launcher-empty-shell"
            onClick={() => onLaunch()}
            disabled={isCreating}
            className={optionCardClass}
          >
            <Terminal className="h-5 w-5 shrink-0 text-wc-accent" />
            <div>
              <div className="font-medium text-wc-text-primary">Empty Shell</div>
              <div className="text-sm text-wc-text-muted">
                Start a new terminal session
              </div>
            </div>
          </button>

          {/* Shortcut entries */}
          {shortcuts.length > 0 && (
            <div className="space-y-2">
              <div className="px-1 text-xs font-medium uppercase tracking-wider text-wc-text-faint">
                Shortcuts
              </div>
              {shortcuts.map((shortcut) => (
                <button
                  key={shortcut.command}
                  data-testid={`launcher-shortcut-${slugify(shortcut.label)}`}
                  onClick={() => onLaunch(shortcut.command)}
                  disabled={isCreating}
                  className={optionCardClass}
                >
                  <Zap className="h-5 w-5 shrink-0 text-yellow-400" />
                  <div className="min-w-0 flex-1">
                    <div className="font-medium text-wc-text-primary">
                      {shortcut.label}
                    </div>
                    <div className="truncate text-sm text-wc-text-muted">
                      {shortcut.description || shortcut.command}
                    </div>
                  </div>
                </button>
              ))}
            </div>
          )}

          {/* Custom command */}
          <div className="space-y-2">
            <div className="px-1 text-xs font-medium uppercase tracking-wider text-wc-text-faint">
              Custom Command
            </div>
            <div className="flex gap-2">
              <input
                data-testid="launcher-custom-input"
                type="text"
                value={customCommand}
                onChange={(e) => setCustomCommand(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") handleLaunchCustom();
                }}
                placeholder="Enter command..."
                className="flex-1 rounded-md border border-wc-default bg-wc-surface-input px-3 py-2 text-sm text-wc-text-primary placeholder:text-wc-text-faint focus:border-wc-accent focus:outline-none"
              />
              <Button
                data-testid="launcher-custom-launch"
                size="sm"
                onClick={handleLaunchCustom}
                disabled={isCreating || !customCommand.trim()}
              >
                Launch
              </Button>
            </div>
          </div>
        </div>

        {isCreating && (
          <div className="border-t border-wc-default px-4 py-2 text-center text-sm text-wc-text-muted">
            Creating session...
          </div>
        )}
      </div>
    </div>
  );
}
