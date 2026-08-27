import { useCallback, useState } from "react";
import { ClipboardPaste, Copy, Image, MousePointer2, TextSelect, Trash2, Volume2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { ContextMenu } from "@vrooli/react-component-library/ContextMenu/1";
import { strings } from "../consts/strings";
import { readText } from "../lib/clipboard";

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
  /** Present only for a persistent tmux-backed pane. */
  mouseMode?: boolean;
  onToggleMouseMode?: (enabled: boolean) => void;
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
  mouseMode,
  onToggleMouseMode,
  onClose,
}: TerminalContextMenuProps) {
  const { t } = useTranslation();
  const [pasteState, setPasteState] = useState<PasteUIState>({ kind: "idle" });

  const handlePaste = useCallback(async () => {
    let text: string;
    try {
      const result = await readText();
      if (!result.ok) throw new Error(result.reason);
      text = result.text;
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
    <ContextMenu
      open
      position={position}
      title={t(strings.terminalContextMenu.selectAll)}
      closeLabel={t(strings.terminalContextMenu.selectAll)}
      testId="terminal-context-menu"
      onOpenChange={(next) => {
        if (!next) onClose();
      }}
      items={[
        ...(hasSelection
          ? [{ id: "copy", label: t(strings.terminalContextMenu.copy), icon: <Copy className="h-4 w-4 shrink-0" />, testId: "ctx-copy", onSelect: onCopy }]
          : []),
        ...(hasSelection && onSpeak
          ? [{ id: "speak", label: t(strings.terminalContextMenu.speak), icon: <Volume2 className="h-4 w-4 shrink-0" />, testId: "ctx-speak", onSelect: onSpeak }]
          : []),
        {
          id: "paste",
          label: pasteLabel,
          icon: <ClipboardPaste className="h-4 w-4 shrink-0" />,
          testId: "ctx-paste",
          disabled: pasteDisabled,
          state: pasteState.kind,
          closeOnSelect: false,
          onSelect: handlePaste,
        },
        ...(onUploadImage
          ? [{ id: "upload-image", label: t(strings.terminalContextMenu.uploadImage), icon: <Image className="h-4 w-4 shrink-0" />, testId: "ctx-upload-image", onSelect: onUploadImage }]
          : []),
        ...(onToggleMouseMode && mouseMode !== undefined
          ? [{
              id: "mouse-mode",
              label: mouseMode ? "Disable tmux mouse mode (this pane only)" : "Enable tmux mouse mode (this pane only)",
              icon: <MousePointer2 className="h-4 w-4 shrink-0" />,
              testId: "ctx-mouse-mode",
              pressed: mouseMode,
              onSelect: () => onToggleMouseMode(!mouseMode),
            }]
          : []),
        { id: "select-all", label: t(strings.terminalContextMenu.selectAll), icon: <TextSelect className="h-4 w-4 shrink-0" />, testId: "ctx-select-all", onSelect: onSelectAll },
        { id: "clear", label: t(strings.terminalContextMenu.clearTerminal), icon: <Trash2 className="h-4 w-4 shrink-0" />, testId: "ctx-clear", destructive: true, onSelect: onClear },
      ]}
    />
  );
}
