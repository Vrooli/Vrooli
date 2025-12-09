import { X, Keyboard } from "lucide-react";
import {
  type ShortcutDefinition,
  formatShortcut,
  useKeyboardShortcutsStore,
} from "@stores/keyboardShortcutsStore";
import ResponsiveDialog from "./ResponsiveDialog";

interface KeyboardShortcutsModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export function KeyboardShortcutsContent() {
  // Get shortcuts grouped by category (already filtered by current context)
  const shortcutsByCategory = useKeyboardShortcutsStore((state) =>
    state.getShortcutsByCategory()
  );

  const categories = Object.keys(shortcutsByCategory).sort();

  return (
    <>
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-3">
          <div className="p-2 bg-flow-accent/20 rounded-lg">
            <Keyboard size={20} className="text-flow-accent" />
          </div>
          <div>
            <h2 className="text-lg font-semibold text-flow-text">
              Keyboard Shortcuts
            </h2>
            <p className="text-sm text-flow-text-muted">
              Navigate faster with these shortcuts
            </p>
          </div>
        </div>
      </div>

      <div className="space-y-6 max-h-[60vh] overflow-y-auto pr-2">
        {categories.length === 0 ? (
          <p className="text-sm text-flow-text-muted text-center py-8">
            No shortcuts available for the current view.
          </p>
        ) : (
          categories.map((category) => (
            <div key={category}>
              <h3 className="text-sm font-semibold text-flow-text-muted uppercase tracking-wide mb-3">
                {category}
              </h3>
              <div className="space-y-2">
                {shortcutsByCategory[category].map(
                  (shortcut: ShortcutDefinition, index: number) => (
                    <div
                      key={`${shortcut.id}-${index}`}
                      className="flex items-center justify-between py-2 px-3 rounded-lg hover:bg-flow-node/50 transition-colors"
                    >
                      <span className="text-sm text-flow-text-secondary">
                        {shortcut.description}
                      </span>
                      <kbd className="inline-flex items-center gap-1 px-2 py-1 bg-flow-node border border-flow-border rounded text-xs font-mono text-flow-text-secondary">
                        {formatShortcut(shortcut)}
                      </kbd>
                    </div>
                  )
                )}
              </div>
            </div>
          ))
        )}
      </div>
    </>
  );
}

/**
 * Legacy modal wrapper so existing callers can still render shortcuts standalone.
 */
function KeyboardShortcutsModal({ isOpen, onClose }: KeyboardShortcutsModalProps) {
  return (
    <ResponsiveDialog
      isOpen={isOpen}
      onDismiss={onClose}
      ariaLabel="Keyboard Shortcuts"
      size="wide"
      overlayClassName="bg-black/70 backdrop-blur-sm"
      className="bg-flow-bg/95 border border-gray-700 shadow-2xl rounded-2xl"
    >
      <div className="flex justify-end">
        <button
          type="button"
          onClick={onClose}
          className="p-2 text-flow-text-muted hover:text-flow-text hover:bg-flow-node-hover rounded-lg transition-colors"
          aria-label="Close keyboard shortcuts"
        >
          <X size={20} />
        </button>
      </div>
      <KeyboardShortcutsContent />
    </ResponsiveDialog>
  );
}

export default KeyboardShortcutsModal;
