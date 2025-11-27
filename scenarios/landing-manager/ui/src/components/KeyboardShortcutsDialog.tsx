import { X, Zap } from 'lucide-react';

interface KeyboardShortcutsDialogProps {
  isOpen: boolean;
  onClose: () => void;
}

/**
 * Dialog showing available keyboard shortcuts for power users.
 * Displays hotkeys for generation, dry-run, refresh, and navigation.
 */
export function KeyboardShortcutsDialog({ isOpen, onClose }: KeyboardShortcutsDialogProps) {
  if (!isOpen) return null;

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/80 backdrop-blur-sm px-4"
      onClick={onClose}
      role="dialog"
      aria-modal="true"
      aria-labelledby="keyboard-shortcuts-title"
    >
      <div
        className="bg-slate-900 border border-white/10 rounded-2xl shadow-2xl max-w-md w-full p-6 space-y-4"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between">
          <h2 id="keyboard-shortcuts-title" className="text-xl font-bold text-slate-100">Keyboard Shortcuts</h2>
          <button
            onClick={onClose}
            className="text-slate-400 hover:text-slate-200 transition-colors p-1 rounded-lg hover:bg-white/5 focus:outline-none focus:ring-2 focus:ring-emerald-500"
            aria-label="Close dialog"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        <div className="space-y-3 text-sm">
          <div className="flex items-center justify-between py-2 border-b border-white/10">
            <span className="text-slate-300">Generate scenario</span>
            <kbd className="px-2.5 py-1.5 rounded bg-slate-800 border border-slate-700 font-mono text-xs text-slate-200">⌘/Ctrl + Enter</kbd>
          </div>
          <div className="flex items-center justify-between py-2 border-b border-white/10">
            <span className="text-slate-300">Dry-run preview</span>
            <kbd className="px-2.5 py-1.5 rounded bg-slate-800 border border-slate-700 font-mono text-xs text-slate-200">⌘/Ctrl + Shift + Enter</kbd>
          </div>
          <div className="flex items-center justify-between py-2 border-b border-white/10">
            <span className="text-slate-300">Refresh templates</span>
            <kbd className="px-2.5 py-1.5 rounded bg-slate-800 border border-slate-700 font-mono text-xs text-slate-200">⌘/Ctrl + R</kbd>
          </div>
          <div className="flex items-center justify-between py-2">
            <span className="text-slate-300">Skip to main content</span>
            <kbd className="px-2.5 py-1.5 rounded bg-slate-800 border border-slate-700 font-mono text-xs text-slate-200">Tab</kbd>
          </div>
        </div>

        <div className="pt-4 border-t border-white/10">
          <p className="text-xs text-slate-400 flex items-start gap-2">
            <Zap className="h-4 w-4 flex-shrink-0 mt-0.5 text-emerald-400" aria-hidden="true" />
            <span>All keyboard shortcuts work when the generation form is visible and not disabled.</span>
          </p>
        </div>

        <button
          onClick={onClose}
          className="w-full px-4 py-2.5 text-sm font-medium rounded-lg border border-emerald-500/40 bg-emerald-500/10 text-emerald-300 hover:bg-emerald-500/20 hover:border-emerald-400 transition-all focus:outline-none focus:ring-2 focus:ring-emerald-500"
        >
          Got it!
        </button>
      </div>
    </div>
  );
}
