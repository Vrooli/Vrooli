import { X, Keyboard } from "lucide-react";

interface Shortcut {
  keys: string[];
  description: string;
}

const CANVAS_SHORTCUTS: Shortcut[] = [
  { keys: ["Arrow keys"], description: "Pan the canvas" },
  { keys: ["+", "="], description: "Zoom in" },
  { keys: ["-"], description: "Zoom out" },
  { keys: ["Scroll"], description: "Zoom in/out" },
  { keys: ["Click + drag"], description: "Move an item" },
  { keys: ["?"], description: "Toggle this help" },
];

interface Props {
  open: boolean;
  onClose: () => void;
}

export function KeyboardShortcutHelp({ open, onClose }: Props) {
  if (!open) return null;

  return (
    <div
      data-testid="keyboard-shortcut-help"
      className="absolute inset-0 z-20 flex items-center justify-center bg-black/60"
      onClick={onClose}
      role="dialog"
      aria-label="Keyboard shortcuts"
    >
      <div
        className="bg-slate-800 border border-white/10 rounded-lg p-4 max-w-xs w-full shadow-xl"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-2">
            <Keyboard className="h-4 w-4 text-slate-400" aria-hidden="true" />
            <h2 className="text-sm font-medium text-white">Keyboard Shortcuts</h2>
          </div>
          <button
            onClick={onClose}
            className="p-0.5 rounded text-slate-400 hover:text-white"
            aria-label="Close shortcuts help"
          >
            <X className="h-4 w-4" aria-hidden="true" />
          </button>
        </div>
        <dl className="space-y-1.5">
          {CANVAS_SHORTCUTS.map((shortcut) => (
            <div key={shortcut.description} className="flex items-center justify-between text-xs">
              <dt className="text-slate-400">{shortcut.description}</dt>
              <dd className="flex gap-1">
                {shortcut.keys.map((key) => (
                  <kbd
                    key={key}
                    className="px-1.5 py-0.5 rounded bg-slate-700 text-slate-300 font-mono text-[10px]"
                  >
                    {key}
                  </kbd>
                ))}
              </dd>
            </div>
          ))}
        </dl>
      </div>
    </div>
  );
}
