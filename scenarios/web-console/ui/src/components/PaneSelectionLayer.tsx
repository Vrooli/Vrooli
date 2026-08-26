import type { ReactNode } from "react";
import TerminalContextMenu from "./TerminalContextMenu";
import type { TerminalPaneStatus } from "../hooks/terminal/useTerminalSession";

export interface PaneSelectionLayerProps {
  contextMenu: { x: number; y: number } | null;
  hasSelection: boolean;
  inputError: string | null;
  paneStatus: TerminalPaneStatus | null;
  uploadError: string | null;
  uploading: boolean;
  uploadingLabel: string;
  ttsSupported: boolean;
  onCopy: () => void;
  onPaste: (text: string) => Promise<{ status: "ok" } | { status: "failed"; reason: string }>;
  onSelectAll: () => void;
  onClear: () => void;
  onUploadImage: () => void;
  onSpeak: () => void;
  mouseMode?: boolean;
  onToggleMouseMode?: (enabled: boolean) => void;
  onClose: () => void;
  children?: ReactNode;
}

export function PaneSelectionLayer({
  contextMenu, hasSelection, inputError, paneStatus, uploadError, uploading, uploadingLabel,
  ttsSupported, onCopy, onPaste, onSelectAll, onClear, onUploadImage, onSpeak, onClose, children,
  mouseMode, onToggleMouseMode,
}: PaneSelectionLayerProps) {
  return <>
    {children}
    {uploading && <div data-testid="upload-overlay" role="status" aria-live="polite" aria-busy="true" className="pointer-events-none absolute inset-0 z-wc-chrome-raised flex items-center justify-center bg-black/50 text-sm text-white">{uploadingLabel}</div>}
    {uploadError && <div data-testid="upload-error" className="absolute top-2 left-2 z-wc-chrome-raised rounded bg-red-600/90 px-3 py-1.5 text-xs text-white shadow-lg">{uploadError}</div>}
    {inputError && <div data-testid="input-error" role="status" className="absolute bottom-2 left-2 right-2 z-wc-chrome-raised rounded bg-red-600/90 px-3 py-1.5 text-xs text-white shadow-lg">{inputError}</div>}
    {paneStatus && <div data-testid="terminal-pane-status" role="status" className={`absolute top-2 left-2 right-2 z-wc-chrome-raised rounded px-3 py-1.5 text-xs shadow-lg ${paneStatus.kind === "error" ? "bg-red-600/90 text-white" : "bg-slate-900/90 text-slate-100"}`}>{paneStatus.detail ?? paneStatus.kind}</div>}
    {contextMenu && <TerminalContextMenu position={contextMenu} hasSelection={hasSelection} onCopy={onCopy} onPaste={onPaste} onSelectAll={onSelectAll} onClear={onClear} onUploadImage={onUploadImage} onSpeak={ttsSupported ? onSpeak : undefined} mouseMode={mouseMode} onToggleMouseMode={onToggleMouseMode} onClose={onClose} />}
  </>;
}
