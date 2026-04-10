import { useCallback, useState } from "react";
import { ClipboardPaste, Copy, Image, TextSelect, Trash2, Volume2 } from "lucide-react";
import ContextMenuBase, { contextMenuItemClass } from "./ContextMenuBase";

interface TerminalContextMenuProps {
  /** Viewport coordinates where the menu should appear. */
  position: { x: number; y: number };
  /** Whether a text selection is active (controls Copy visibility). */
  hasSelection: boolean;
  onCopy: () => void;
  /** Called with clipboard text after a successful read. */
  onPaste: (text: string) => void;
  onSelectAll: () => void;
  onClear: () => void;
  onUploadImage?: () => void;
  onSpeak?: () => void;
  onClose: () => void;
}

export default function TerminalContextMenu({
  position,
  hasSelection,
  onCopy,
  onPaste,
  onSelectAll,
  onClear,
  onUploadImage,
  onSpeak,
  onClose,
}: TerminalContextMenuProps) {
  const [pasteError, setPasteError] = useState(false);

  const handlePaste = useCallback(async () => {
    try {
      const text = await navigator.clipboard.readText();
      if (text) {
        onPaste(text);
      }
      onClose();
    } catch {
      setPasteError(true);
      setTimeout(() => setPasteError(false), 2000);
    }
  }, [onPaste, onClose]);

  return (
    <ContextMenuBase position={position} onClose={onClose} data-testid="terminal-context-menu">
      {hasSelection && (
        <button
          data-testid="ctx-copy"
          className={contextMenuItemClass}
          onClick={onCopy}
        >
          <Copy className="h-4 w-4 shrink-0" />
          Copy
        </button>
      )}
      {hasSelection && onSpeak && (
        <button
          data-testid="ctx-speak"
          className={contextMenuItemClass}
          onClick={onSpeak}
        >
          <Volume2 className="h-4 w-4 shrink-0" />
          Speak
        </button>
      )}
      <button
        data-testid="ctx-paste"
        className={contextMenuItemClass}
        onClick={handlePaste}
      >
        <ClipboardPaste className="h-4 w-4 shrink-0" />
        {pasteError ? "Use Ctrl+V to paste" : "Paste"}
      </button>
      {onUploadImage && (
        <button
          data-testid="ctx-upload-image"
          className={contextMenuItemClass}
          onClick={onUploadImage}
        >
          <Image className="h-4 w-4 shrink-0" />
          Upload Image
        </button>
      )}
      <button
        data-testid="ctx-select-all"
        className={contextMenuItemClass}
        onClick={onSelectAll}
      >
        <TextSelect className="h-4 w-4 shrink-0" />
        Select All
      </button>
      <button
        data-testid="ctx-clear"
        className={contextMenuItemClass}
        onClick={onClear}
      >
        <Trash2 className="h-4 w-4 shrink-0" />
        Clear Terminal
      </button>
    </ContextMenuBase>
  );
}
