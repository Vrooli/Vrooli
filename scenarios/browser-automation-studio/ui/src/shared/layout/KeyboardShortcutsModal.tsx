import { X, Keyboard } from "lucide-react";
import {
  formatShortcut,
  type KeyboardShortcut,
} from "@hooks/useKeyboardShortcuts";
import ResponsiveDialog from "./ResponsiveDialog";

interface KeyboardShortcutsModalProps {
  isOpen: boolean;
  onClose: () => void;
  shortcuts: Omit<KeyboardShortcut, "action">[];
}

/**
 * Modal displaying all available keyboard shortcuts, grouped by category.
 * Accessible via Cmd/Ctrl + ? or the help button.
 */
function KeyboardShortcutsModal({
  isOpen,
  onClose,
  shortcuts,
}: KeyboardShortcutsModalProps) {
  // Group shortcuts by category
  const groupedShortcuts = shortcuts.reduce(
    (acc, shortcut) => {
      const category = shortcut.category || "General";
      if (!acc[category]) {
        acc[category] = [];
      }
      acc[category].push(shortcut);
      return acc;
    },
    {} as Record<string, Omit<KeyboardShortcut, "action">[]>
  );

  const categories = Object.keys(groupedShortcuts).sort();

  return (
    <ResponsiveDialog
      isOpen={isOpen}
      onDismiss={onClose}
      ariaLabel="Keyboard Shortcuts"
      size="wide"
    >
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-3">
          <div className="p-2 bg-flow-accent/20 rounded-lg">
            <Keyboard size={20} className="text-flow-accent" />
          </div>
          <div>
            <h2 className="text-lg font-semibold text-white">
              Keyboard Shortcuts
            </h2>
            <p className="text-sm text-gray-400">
              Navigate faster with these shortcuts
            </p>
          </div>
        </div>
        <button
          type="button"
          onClick={onClose}
          className="p-2 text-gray-400 hover:text-white hover:bg-gray-700 rounded-lg transition-colors"
          aria-label="Close keyboard shortcuts"
        >
          <X size={20} />
        </button>
      </div>

      <div className="space-y-6 max-h-[60vh] overflow-y-auto pr-2">
        {categories.map((category) => (
          <div key={category}>
            <h3 className="text-sm font-semibold text-gray-400 uppercase tracking-wide mb-3">
              {category}
            </h3>
            <div className="space-y-2">
              {groupedShortcuts[category].map((shortcut, index) => (
                <div
                  key={`${shortcut.key}-${index}`}
                  className="flex items-center justify-between py-2 px-3 rounded-lg hover:bg-gray-800/50 transition-colors"
                >
                  <span className="text-sm text-gray-200">
                    {shortcut.description}
                  </span>
                  <kbd className="inline-flex items-center gap-1 px-2 py-1 bg-gray-800 border border-gray-700 rounded text-xs font-mono text-gray-300">
                    {formatShortcut(shortcut.key, shortcut.modifiers)}
                  </kbd>
                </div>
              ))}
            </div>
          </div>
        ))}
      </div>

      <div className="mt-6 pt-4 border-t border-gray-800">
        <p className="text-xs text-gray-500 text-center">
          Press <kbd className="px-1.5 py-0.5 bg-gray-800 rounded text-gray-400">Esc</kbd> to close this dialog
        </p>
      </div>
    </ResponsiveDialog>
  );
}

export default KeyboardShortcutsModal;
