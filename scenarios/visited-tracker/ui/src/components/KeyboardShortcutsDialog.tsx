import { Keyboard } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle
} from "./ui/dialog";

interface KeyboardShortcutsDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

interface Shortcut {
  keys: string[];
  description: string;
  context?: string;
}

const shortcuts: Shortcut[] = [
  { keys: ["N"], description: "Create new campaign", context: "Campaign list" },
  { keys: ["/"], description: "Focus search input", context: "Campaign list" },
  { keys: ["R"], description: "Refresh data", context: "Any page" },
  { keys: ["Esc"], description: "Go back to list", context: "Campaign detail" },
  { keys: ["?"], description: "Show this help", context: "Campaign list" },
  { keys: ["Ctrl", "Enter"], description: "Submit form", context: "Create dialog" },
];

export function KeyboardShortcutsDialog({
  open,
  onOpenChange
}: KeyboardShortcutsDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Keyboard className="h-5 w-5 text-blue-400" aria-hidden="true" />
            Keyboard Shortcuts
          </DialogTitle>
          <DialogDescription>
            Navigate faster with keyboard shortcuts. Works when not typing in input fields.
          </DialogDescription>
        </DialogHeader>

        <div className="mt-4 space-y-3">
          {shortcuts.map((shortcut, i) => (
            <div
              key={i}
              className="flex items-center justify-between gap-4 rounded-lg border border-white/10 bg-white/[0.02] p-3 hover:bg-white/[0.05] transition-colors"
            >
              <div className="flex-1">
                <div className="text-sm font-medium text-slate-200">
                  {shortcut.description}
                </div>
                {shortcut.context && (
                  <div className="text-xs text-slate-500 mt-0.5">
                    {shortcut.context}
                  </div>
                )}
              </div>
              <div className="flex items-center gap-1">
                {shortcut.keys.map((key, j) => (
                  <span key={j} className="flex items-center gap-1">
                    <kbd className="inline-flex h-7 min-w-7 items-center justify-center rounded border border-slate-700 bg-slate-800 px-2 text-xs font-medium text-slate-300">
                      {key}
                    </kbd>
                    {j < shortcut.keys.length - 1 && (
                      <span className="text-xs text-slate-500">+</span>
                    )}
                  </span>
                ))}
              </div>
            </div>
          ))}
        </div>

        <div className="mt-6 rounded-lg border border-blue-500/20 bg-blue-500/5 p-4">
          <div className="flex items-start gap-2 text-xs text-slate-300">
            <Keyboard className="h-4 w-4 text-blue-400 shrink-0 mt-0.5" aria-hidden="true" />
            <div className="space-y-1">
              <div className="font-medium text-slate-200">Pro tip for agents</div>
              <div className="text-slate-400">
                CLI commands are usually faster than UI interactions. Use <code className="px-1.5 py-0.5 rounded bg-slate-800 text-blue-300">visited-tracker least-visited</code> to get files programmatically.
              </div>
            </div>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
