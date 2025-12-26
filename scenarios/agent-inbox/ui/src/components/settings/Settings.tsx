import { useState, useCallback, useEffect } from "react";
import { Moon, Sun, Trash2, AlertTriangle, Keyboard, BarChart3, Bot } from "lucide-react";
import { Dialog, DialogHeader, DialogBody } from "../ui/dialog";
import { Button } from "../ui/button";
import { ModelSelector } from "./ModelSelector";
import type { Model } from "../../lib/api";

export type Theme = "dark" | "light";

// Default model used when none is set
export const DEFAULT_MODEL = "anthropic/claude-3.5-sonnet";

// Get/set default model from localStorage
export function getDefaultModel(): string {
  if (typeof window !== "undefined") {
    return localStorage.getItem("defaultModel") || DEFAULT_MODEL;
  }
  return DEFAULT_MODEL;
}

export function setDefaultModel(modelId: string): void {
  if (typeof window !== "undefined") {
    localStorage.setItem("defaultModel", modelId);
  }
}

interface SettingsProps {
  open: boolean;
  onClose: () => void;
  onDeleteAllChats: () => Promise<unknown>;
  isDeletingAll: boolean;
  onShowKeyboardShortcuts: () => void;
  onShowUsageStats: () => void;
  models: Model[];
}

export function Settings({
  open,
  onClose,
  onDeleteAllChats,
  isDeletingAll,
  onShowKeyboardShortcuts,
  onShowUsageStats,
  models,
}: SettingsProps) {
  const [theme, setTheme] = useState<Theme>(() => {
    if (typeof window !== "undefined") {
      return (localStorage.getItem("theme") as Theme) || "dark";
    }
    return "dark";
  });
  const [defaultModel, setDefaultModelState] = useState<string>(getDefaultModel);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [deleteConfirmText, setDeleteConfirmText] = useState("");

  // Apply theme class to document
  useEffect(() => {
    const root = document.documentElement;
    if (theme === "light") {
      root.classList.add("light-theme");
    } else {
      root.classList.remove("light-theme");
    }
    localStorage.setItem("theme", theme);
  }, [theme]);

  const handleThemeChange = useCallback((newTheme: Theme) => {
    setTheme(newTheme);
  }, []);

  const handleDefaultModelChange = useCallback((modelId: string) => {
    setDefaultModelState(modelId);
    setDefaultModel(modelId);
  }, []);

  const handleDeleteAll = useCallback(async () => {
    if (deleteConfirmText !== "delete all") return;
    await onDeleteAllChats();
    setShowDeleteConfirm(false);
    setDeleteConfirmText("");
  }, [deleteConfirmText, onDeleteAllChats]);

  const handleCancelDelete = useCallback(() => {
    setShowDeleteConfirm(false);
    setDeleteConfirmText("");
  }, []);

  return (
    <Dialog open={open} onClose={onClose} className="max-w-lg">
      <DialogHeader onClose={onClose}>Settings</DialogHeader>
      <DialogBody className="space-y-6">
        {/* Appearance Section */}
        <section>
          <h3 className="text-sm font-medium text-slate-300 mb-3">Appearance</h3>
          <div className="flex gap-2">
            <button
              onClick={() => handleThemeChange("dark")}
              className={`flex-1 flex items-center justify-center gap-2 p-3 rounded-lg border transition-colors ${
                theme === "dark"
                  ? "bg-indigo-500/20 border-indigo-500 text-white"
                  : "bg-white/5 border-white/10 text-slate-400 hover:text-white hover:border-white/20"
              }`}
              data-testid="theme-dark-button"
            >
              <Moon className="h-4 w-4" />
              <span className="text-sm">Dark</span>
            </button>
            <button
              onClick={() => handleThemeChange("light")}
              className={`flex-1 flex items-center justify-center gap-2 p-3 rounded-lg border transition-colors ${
                theme === "light"
                  ? "bg-indigo-500/20 border-indigo-500 text-white"
                  : "bg-white/5 border-white/10 text-slate-400 hover:text-white hover:border-white/20"
              }`}
              data-testid="theme-light-button"
            >
              <Sun className="h-4 w-4" />
              <span className="text-sm">Light</span>
            </button>
          </div>
        </section>

        {/* Default Model Section */}
        <section>
          <h3 className="text-sm font-medium text-slate-300 mb-3">Default Model</h3>
          <p className="text-xs text-slate-500 mb-3">
            New chats will use this model by default
          </p>
          <ModelSelector
            models={models}
            selectedModel={defaultModel}
            onSelectModel={handleDefaultModelChange}
          />
        </section>

        {/* Keyboard Shortcuts Section */}
        <section>
          <h3 className="text-sm font-medium text-slate-300 mb-3">Keyboard</h3>
          <Button
            variant="secondary"
            onClick={() => {
              onShowKeyboardShortcuts();
              onClose();
            }}
            className="w-full justify-start gap-2"
            data-testid="keyboard-shortcuts-button"
          >
            <Keyboard className="h-4 w-4" />
            View Keyboard Shortcuts
          </Button>
        </section>

        {/* Usage Statistics Section */}
        <section>
          <h3 className="text-sm font-medium text-slate-300 mb-3">Usage</h3>
          <Button
            variant="secondary"
            onClick={() => {
              onShowUsageStats();
              onClose();
            }}
            className="w-full justify-start gap-2"
            data-testid="usage-stats-button"
          >
            <BarChart3 className="h-4 w-4" />
            View Usage Statistics
          </Button>
        </section>

        {/* Danger Zone */}
        <section>
          <h3 className="text-sm font-medium text-red-400 mb-3 flex items-center gap-2">
            <AlertTriangle className="h-4 w-4" />
            Danger Zone
          </h3>
          <div className="p-4 rounded-lg border border-red-500/20 bg-red-500/5">
            {!showDeleteConfirm ? (
              <div className="flex items-center justify-between gap-4">
                <div>
                  <p className="text-sm font-medium text-white">Delete All Chats</p>
                  <p className="text-xs text-slate-500 mt-1">
                    Permanently delete all chats and messages. This cannot be undone.
                  </p>
                </div>
                <Button
                  variant="destructive"
                  size="sm"
                  onClick={() => setShowDeleteConfirm(true)}
                  data-testid="delete-all-button"
                >
                  <Trash2 className="h-4 w-4" />
                </Button>
              </div>
            ) : (
              <div className="space-y-3">
                <p className="text-sm text-slate-300">
                  Type <span className="font-mono text-red-400">delete all</span> to confirm:
                </p>
                <input
                  type="text"
                  value={deleteConfirmText}
                  onChange={(e) => setDeleteConfirmText(e.target.value)}
                  placeholder="delete all"
                  className="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-sm text-white placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-red-500/50"
                  autoFocus
                  data-testid="delete-confirm-input"
                />
                <div className="flex gap-2">
                  <Button
                    variant="secondary"
                    size="sm"
                    onClick={handleCancelDelete}
                    className="flex-1"
                  >
                    Cancel
                  </Button>
                  <Button
                    variant="destructive"
                    size="sm"
                    onClick={handleDeleteAll}
                    disabled={deleteConfirmText !== "delete all" || isDeletingAll}
                    className="flex-1"
                    data-testid="confirm-delete-all-button"
                  >
                    {isDeletingAll ? "Deleting..." : "Delete All"}
                  </Button>
                </div>
              </div>
            )}
          </div>
        </section>
      </DialogBody>
    </Dialog>
  );
}
