import { useCallback, useEffect, useRef, useState, type RefObject } from "react";
import type { Terminal } from "@xterm/xterm";
import { readText } from "../../lib/clipboard";
import { useTerminalTouch } from "../useTerminalTouch";
import { useMobileBackspaceRepeat } from "../useMobileBackspaceRepeat";
import type { GateResult, InputIntent } from "../../components/terminal/inputGate";
import type { InputSettlementCallback, InputSettledListener } from "./useStdinStream";

type ClipboardPasteResult = { status: "ok" } | { status: "failed"; reason: string };

export function usePaneSelection(options: {
  terminal: Terminal | null;
  containerRef: RefObject<HTMLDivElement | null>;
  sendControl: (data: string) => boolean;
  scrollBy: (lines: number, source: "touch" | "programmatic") => void;
  fontSize: number;
  isFollower: boolean;
  onFontSizeCommit: (size: number) => void;
  onFontSizePreview?: (size: number | null) => void;
  submitInput: (data: string, intent: Exclude<InputIntent, "control">) => GateResult;
  awaitOffset: (offset: number, cb: InputSettlementCallback) => () => void;
  subscribeInputSettled: (cb: InputSettledListener) => () => void;
  rejectedMessage: string;
}) {
  const { terminal, containerRef, sendControl, scrollBy, fontSize, isFollower, onFontSizeCommit, onFontSizePreview, submitInput, awaitOffset, subscribeInputSettled, rejectedMessage } = options;
  const [contextMenu, setContextMenu] = useState<{ x: number; y: number } | null>(null);
  const [inputError, setInputError] = useState<string | null>(null);
  const selfReportedOffsetsRef = useRef<Set<number>>(new Set());
  const { hasSelection, copySelection, clearSelection } = useTerminalTouch({
    terminal,
    containerRef,
    sendControl,
    scrollBy,
    fontSize,
    isFollower,
    onFontSizeCommit,
    onFontSizePreview,
    onContextMenu: useCallback((x: number, y: number) => { setContextMenu({ x, y }); }, []),
  });
  useMobileBackspaceRepeat(terminal);

  const closeContextMenu = useCallback(() => {
    clearSelection();
    setContextMenu(null);
  }, [clearSelection]);
  const handleCopy = useCallback(() => {
    void copySelection();
    closeContextMenu();
  }, [closeContextMenu, copySelection]);
  const handlePaste = useCallback((text: string): Promise<ClipboardPasteResult> => {
    const result = submitInput(text, "bulk_text");
    terminal?.focus();
    if (result.status === "rejected") return Promise.resolve({ status: "failed", reason: result.reason });
    if (result.status === "queued") return Promise.resolve({ status: "ok" });
    const { offset } = result;
    selfReportedOffsetsRef.current.add(offset);
    return new Promise((resolve) => {
      awaitOffset(offset, (ok, reason) => { resolve(ok ? { status: "ok" } : { status: "failed", reason: reason ?? "server rejected" }); });
    });
  }, [awaitOffset, submitInput, terminal]);
  const pasteFromClipboard = useCallback(async () => {
    try {
      const result = await readText();
      return result.ok ? (await handlePaste(result.text)).status === "ok" : false;
    } catch {
      return false;
    }
  }, [handlePaste]);
  useEffect(() => subscribeInputSettled((offset, ok) => {
    const wasSelfReported = selfReportedOffsetsRef.current.delete(offset);
    if (ok || wasSelfReported) return;
    setInputError(rejectedMessage);
  }), [rejectedMessage, subscribeInputSettled]);
  useEffect(() => {
    if (!inputError) return;
    const timer = window.setTimeout(() => { setInputError(null); }, 6000);
    return () => { window.clearTimeout(timer); };
  }, [inputError]);
  const selectAll = useCallback(() => {
    terminal?.selectAll();
    setContextMenu(null);
  }, [terminal]);
  const clear = useCallback(() => {
    terminal?.clear();
    setContextMenu(null);
  }, [terminal]);
  return {
    contextMenu,
    hasSelection,
    inputError,
    copySelection,
    pasteFromClipboard,
    handleCopy,
    handlePaste,
    closeContextMenu,
    clearSelection,
    selectAll,
    clear,
  };
}
