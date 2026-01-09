import { useEffect, useState } from "react";
import { X, Keyboard } from "lucide-react";
import { cn } from "../../lib/utils";

interface Shortcut {
  keys: string[];
  description: string;
  section: string;
}

const shortcuts: Shortcut[] = [
  { keys: ["?"], description: "Show this help", section: "General" },
  { keys: ["g", "d"], description: "Go to Dashboard", section: "Navigation" },
  { keys: ["g", "c"], description: "Go to Campaigns", section: "Navigation" },
  { keys: ["g", "s"], description: "Go to Settings", section: "Navigation" },
  { keys: ["/"], description: "Focus search (when available)", section: "General" },
  { keys: ["Esc"], description: "Close dialogs / Clear focus", section: "General" },
];

interface KeyboardShortcutsProps {
  onNavigate?: (path: string) => void;
}

export function KeyboardShortcuts({ onNavigate }: KeyboardShortcutsProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [pressedKeys, setPressedKeys] = useState<string[]>([]);

  useEffect(() => {
    let timeout: NodeJS.Timeout;

    const handleKeyDown = (e: KeyboardEvent) => {
      // Don't trigger shortcuts when typing in inputs
      if (
        e.target instanceof HTMLInputElement ||
        e.target instanceof HTMLTextAreaElement ||
        e.target instanceof HTMLSelectElement
      ) {
        return;
      }

      // Handle Escape key
      if (e.key === "Escape") {
        setIsOpen(false);
        setPressedKeys([]);
        return;
      }

      // Handle ? key to open help
      if (e.key === "?" && !e.shiftKey) {
        e.preventDefault();
        setIsOpen(prev => !prev);
        return;
      }

      // Handle navigation shortcuts
      const key = e.key.toLowerCase();
      const newKeys = [...pressedKeys, key];
      setPressedKeys(newKeys);

      // Clear timeout if exists
      if (timeout) clearTimeout(timeout);

      // Set new timeout to reset keys
      timeout = setTimeout(() => setPressedKeys([]), 1000);

      // Check for navigation shortcuts
      const sequence = newKeys.join("");
      if (sequence === "gd") {
        e.preventDefault();
        onNavigate?.("/dashboard");
        setPressedKeys([]);
      } else if (sequence === "gc") {
        e.preventDefault();
        onNavigate?.("/campaigns");
        setPressedKeys([]);
      } else if (sequence === "gs") {
        e.preventDefault();
        onNavigate?.("/settings");
        setPressedKeys([]);
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => {
      window.removeEventListener("keydown", handleKeyDown);
      if (timeout) clearTimeout(timeout);
    };
  }, [pressedKeys, onNavigate]);

  if (!isOpen) {
    return (
      <button
        onClick={() => setIsOpen(true)}
        className="fixed bottom-4 right-4 bg-slate-800 hover:bg-slate-700 text-slate-300 rounded-full p-3 shadow-lg border border-slate-700 transition-all hover:scale-105 z-40"
        title="Keyboard shortcuts (press ?)"
        aria-label="Show keyboard shortcuts"
      >
        <Keyboard className="h-5 w-5" aria-hidden="true" />
      </button>
    );
  }

  const groupedShortcuts = shortcuts.reduce((acc, shortcut) => {
    if (!acc[shortcut.section]) {
      acc[shortcut.section] = [];
    }
    acc[shortcut.section].push(shortcut);
    return acc;
  }, {} as Record<string, Shortcut[]>);

  return (
    <>
      {/* Backdrop */}
      <div
        className="fixed inset-0 bg-black/50 backdrop-blur-sm z-50"
        onClick={() => setIsOpen(false)}
        aria-hidden="true"
      />

      {/* Modal */}
      <div
        className="fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 bg-slate-900 border border-slate-700 rounded-lg shadow-2xl z-50 w-full max-w-2xl max-h-[80vh] overflow-y-auto"
        role="dialog"
        aria-labelledby="keyboard-shortcuts-title"
        aria-modal="true"
      >
        <div className="sticky top-0 bg-slate-900 border-b border-slate-700 p-4 sm:p-6 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Keyboard className="h-6 w-6 text-blue-400" aria-hidden="true" />
            <h2 id="keyboard-shortcuts-title" className="text-xl font-semibold text-slate-100">
              Keyboard Shortcuts
            </h2>
          </div>
          <button
            onClick={() => setIsOpen(false)}
            className="text-slate-400 hover:text-slate-200 transition-colors p-2 hover:bg-slate-800 rounded-lg"
            aria-label="Close keyboard shortcuts"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        <div className="p-4 sm:p-6 space-y-6">
          {Object.entries(groupedShortcuts).map(([section, sectionShortcuts]) => (
            <div key={section}>
              <h3 className="text-sm font-semibold text-slate-400 uppercase tracking-wider mb-3">
                {section}
              </h3>
              <div className="space-y-2">
                {sectionShortcuts.map((shortcut, idx) => (
                  <div
                    key={idx}
                    className="flex items-center justify-between py-2 px-3 rounded-lg hover:bg-slate-800/50 transition-colors"
                  >
                    <span className="text-sm text-slate-300">{shortcut.description}</span>
                    <div className="flex items-center gap-1">
                      {shortcut.keys.map((key, keyIdx) => (
                        <kbd
                          key={keyIdx}
                          className={cn(
                            "px-2 py-1 text-xs font-mono bg-slate-800 border border-slate-700 rounded text-slate-300 shadow-sm min-w-[2rem] text-center",
                            keyIdx > 0 && "ml-1"
                          )}
                        >
                          {key}
                        </kbd>
                      ))}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          ))}

          <div className="pt-4 border-t border-slate-800">
            <p className="text-xs text-slate-500 text-center">
              Press <kbd className="px-1.5 py-0.5 bg-slate-800 border border-slate-700 rounded text-slate-400 font-mono text-xs">?</kbd> anytime to toggle this help
            </p>
          </div>
        </div>
      </div>
    </>
  );
}
