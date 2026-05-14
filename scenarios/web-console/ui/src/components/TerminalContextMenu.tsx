import { useCallback, useState } from "react";
import { ClipboardPaste, Copy, Image, TextSelect, Trash2, Volume2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import ContextMenuBase, { contextMenuItemClass } from "./ContextMenuBase";
import { strings } from "../consts/strings";

/**
 * onPaste returns a thenable/promise that resolves when the pasted
 * bytes have been acknowledged by the backend (or rejected with a
 * reason). Null/undefined means the paste was rejected before the
 * wire (empty, disposed) — the menu treats these as silent success
 * so users don't see "Paste failed" on empty clipboard.
 */
export type PasteResult =
  | { status: "ok" }
  | { status: "failed"; reason: string };

interface TerminalContextMenuProps {
  /** Viewport coordinates where the menu should appear. */
  position: { x: number; y: number };
  /** Whether a text selection is active (controls Copy visibility). */
  hasSelection: boolean;
  onCopy: () => void;
  /**
   * Called with clipboard text. Must return a promise that resolves
   * when the paste is settled (ack received or timeout). The menu
   * stays open with a "Pasting…" indicator until the promise
   * resolves, then flashes "Pasted" or the failure reason.
   */
  onPaste: (text: string) => Promise<PasteResult>;
  onSelectAll: () => void;
  onClear: () => void;
  onUploadImage?: () => void;
  onSpeak?: () => void;
  onClose: () => void;
}

type PasteUIState =
  | { kind: "idle" }
  | { kind: "pending" }
  | { kind: "succeeded" }
  | { kind: "failed"; reason: string }
  | { kind: "clipboard_unavailable" };

/**
 * PASTE_SUCCESS_FLASH_MS is how long the "Pasted" confirmation stays
 * visible before the menu auto-closes. Short enough to feel snappy.
 */
const PASTE_SUCCESS_FLASH_MS = 600;
/**
 * PASTE_FAILURE_HOLD_MS is how long the failure reason stays visible
 * before the menu auto-closes. Longer than the success flash so the
 * user can actually read the reason.
 */
const PASTE_FAILURE_HOLD_MS = 3000;

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
  const { t } = useTranslation();
  const [pasteState, setPasteState] = useState<PasteUIState>({ kind: "idle" });

  const handlePaste = useCallback(async () => {
    let text: string;
    try {
      text = await navigator.clipboard.readText();
    } catch {
      setPasteState({ kind: "clipboard_unavailable" });
      setTimeout(() => setPasteState({ kind: "idle" }), 2000);
      return;
    }
    if (!text) {
      onClose();
      return;
    }
    setPasteState({ kind: "pending" });
    try {
      const res = await onPaste(text);
      if (res.status === "ok") {
        setPasteState({ kind: "succeeded" });
        setTimeout(onClose, PASTE_SUCCESS_FLASH_MS);
      } else {
        setPasteState({ kind: "failed", reason: res.reason });
        setTimeout(onClose, PASTE_FAILURE_HOLD_MS);
      }
    } catch (err) {
      const reason = err instanceof Error ? err.message : String(err);
      setPasteState({ kind: "failed", reason });
      setTimeout(onClose, PASTE_FAILURE_HOLD_MS);
    }
  }, [onPaste, onClose]);

  const pasteLabel = (() => {
    switch (pasteState.kind) {
      case "pending":
        return t(strings.terminalContextMenu.pasting);
      case "succeeded":
        return t(strings.terminalContextMenu.pasted);
      case "failed":
        return t(strings.terminalContextMenu.pasteFailed, { reason: pasteState.reason });
      case "clipboard_unavailable":
        return t(strings.terminalContextMenu.useCtrlVHint);
      case "idle":
      default:
        return t(strings.terminalContextMenu.paste);
    }
  })();
  const pasteDisabled = pasteState.kind === "pending";

  return (
    <ContextMenuBase position={position} onClose={onClose} data-testid="terminal-context-menu">
      {hasSelection && (
        <button
          data-testid="ctx-copy"
          className={contextMenuItemClass}
          onClick={onCopy}
        >
          <Copy className="h-4 w-4 shrink-0" />
          {t(strings.terminalContextMenu.copy)}
        </button>
      )}
      {hasSelection && onSpeak && (
        <button
          data-testid="ctx-speak"
          className={contextMenuItemClass}
          onClick={onSpeak}
        >
          <Volume2 className="h-4 w-4 shrink-0" />
          {t(strings.terminalContextMenu.speak)}
        </button>
      )}
      <button
        data-testid="ctx-paste"
        data-paste-state={pasteState.kind}
        className={contextMenuItemClass}
        onClick={handlePaste}
        disabled={pasteDisabled}
      >
        <ClipboardPaste className="h-4 w-4 shrink-0" />
        {pasteLabel}
      </button>
      {onUploadImage && (
        <button
          data-testid="ctx-upload-image"
          className={contextMenuItemClass}
          onClick={onUploadImage}
        >
          <Image className="h-4 w-4 shrink-0" />
          {t(strings.terminalContextMenu.uploadImage)}
        </button>
      )}
      <button
        data-testid="ctx-select-all"
        className={contextMenuItemClass}
        onClick={onSelectAll}
      >
        <TextSelect className="h-4 w-4 shrink-0" />
        {t(strings.terminalContextMenu.selectAll)}
      </button>
      <button
        data-testid="ctx-clear"
        className={contextMenuItemClass}
        onClick={onClear}
      >
        <Trash2 className="h-4 w-4 shrink-0" />
        {t(strings.terminalContextMenu.clearTerminal)}
      </button>
    </ContextMenuBase>
  );
}
