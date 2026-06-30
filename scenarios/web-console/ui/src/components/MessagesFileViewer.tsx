import { useEffect, useMemo, useState } from "react";
import { AlertTriangle, Check, Copy, Loader2, RotateCw, X } from "lucide-react";
import { useTranslation } from "react-i18next";

import { strings } from "../consts/strings";
import { rendererForKind } from "./file-preview/renderers";
import type { PreviewState } from "./file-preview/types";

interface MessagesFileViewerProps {
  state: PreviewState;
  onClose: () => void;
  onReopen: () => void;
  onRendererError: (message: string) => void;
}

export default function MessagesFileViewer({ state, onClose, onReopen, onRendererError }: MessagesFileViewerProps) {
  const { t } = useTranslation();
  const { open, status, model, text, error, requestedPath } = state;

  const displayPath = model?.resolvedPath ?? requestedPath ?? "";
  const targetLine = model?.line ?? null;
  const [copied, setCopied] = useState(false);
  const copyPath = () => {
    if (!displayPath) return;
    void navigator.clipboard.writeText(displayPath);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const basename = useMemo(() => {
    const fullPath = model?.resolvedPath ?? requestedPath ?? "";
    if (!fullPath) return t(strings.messagesFileViewer.filePreviewFallback);
    const parts = fullPath.split(/[\\/]/);
    return parts[parts.length - 1] || fullPath;
  }, [model?.resolvedPath, requestedPath, t]);

  useEffect(() => {
    if (!open) return;
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") onClose();
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [onClose, open]);

  if (!open) return null;

  const isLoading = status === "resolving" || status === "loadingText";
  const Renderer = model ? rendererForKind(model.kind) : null;

  return (
    <div className="fixed inset-0 z-[80]">
      <div className="absolute inset-0 bg-wc-backdrop" onClick={onClose} />
      <div
        data-testid="messages-file-viewer-panel"
        className="wc-stable-theme absolute inset-x-0 bottom-0 top-[max(1rem,var(--wc-safe-top,0px))] flex flex-col overflow-hidden rounded-t-[20px] border-t border-wc-default bg-wc-surface-raised shadow-2xl md:inset-x-8 md:bottom-8 md:top-8 md:rounded-2xl md:border"
      >
        <div className="shrink-0 border-b border-wc-default px-4 py-3">
          <div className="flex items-center gap-3">
            <h2 className="min-w-0 flex-1 truncate text-sm font-semibold text-wc-text-primary">{basename}</h2>
            <button
              type="button"
              onClick={onClose}
              className="shrink-0 rounded-full p-1.5 text-wc-text-muted transition hover:bg-wc-surface-input hover:text-wc-text-primary"
              aria-label={t(strings.messagesFileViewer.closeAriaLabel)}
            >
              <X className="h-4 w-4" />
            </button>
          </div>
          <div className="mt-1 flex items-center gap-1.5">
            <p className="min-w-0 flex-1 truncate text-xs text-wc-text-muted">
              {displayPath || t(strings.messagesFileViewer.loadingFile)}
            </p>
            {displayPath && (
              <button
                type="button"
                onClick={copyPath}
                className="shrink-0 rounded p-1 text-wc-text-muted transition hover:bg-wc-surface-input hover:text-wc-text-primary"
                aria-label={copied ? t(strings.messagesFileViewer.copied) : t(strings.messagesFileViewer.copyPath)}
                title={copied ? t(strings.messagesFileViewer.copied) : t(strings.messagesFileViewer.copyPath)}
              >
                {copied ? <Check className="h-3.5 w-3.5 text-green-400" /> : <Copy className="h-3.5 w-3.5" />}
              </button>
            )}
          </div>
          {(model?.resolutionBasis || model?.kind || targetLine) && (
            <div className="mt-2 flex flex-wrap gap-2 text-[11px] text-wc-text-faint">
              {model?.resolutionBasis && (
                <span className="rounded-full border border-wc-default px-2 py-0.5">{model.resolutionBasis}</span>
              )}
              {model?.kind && <span className="rounded-full border border-wc-default px-2 py-0.5">{model.kind}</span>}
              {targetLine && (
                <span className="rounded-full border border-wc-default px-2 py-0.5">
                  {t(strings.messagesFileViewer.linePrefix, { line: targetLine })}
                </span>
              )}
            </div>
          )}
        </div>

        <div className="min-h-0 flex-1 overflow-hidden">
          {isLoading && (
            <div className="flex h-full items-center justify-center gap-2 text-wc-text-muted">
              <Loader2 className="h-4 w-4 animate-spin" />
              <span>{t(strings.messagesFileViewer.loadingPreview)}</span>
            </div>
          )}

          {!isLoading && status === "error" && (
            <div className="h-full overflow-auto px-4 py-4">
              <div className="mx-auto max-w-2xl rounded-2xl border border-red-500/30 bg-red-500/10 p-4 text-sm text-red-300">
                <div className="mb-2 flex items-center gap-2 font-medium">
                  <AlertTriangle className="h-4 w-4" />
                  <span>{t(strings.messagesFileViewer.unavailable)}</span>
                </div>
                <p>{error}</p>
                {requestedPath && (
                  <p className="mt-2 break-all text-xs text-red-200/80">
                    {t(strings.messagesFileViewer.requestedPrefix, { path: requestedPath })}
                  </p>
                )}
                {requestedPath && (
                  <button
                    type="button"
                    onClick={onReopen}
                    data-testid="file-preview-reopen"
                    className="mt-3 inline-flex items-center gap-1.5 rounded-lg border border-red-400/40 bg-red-500/10 px-2.5 py-1.5 text-xs font-medium text-red-200 transition hover:bg-red-500/20"
                  >
                    <RotateCw className="h-3.5 w-3.5" />
                    {t(strings.messagesFileViewer.reopen)}
                  </button>
                )}
              </div>
            </div>
          )}

          {!isLoading && (status === "ready" || status === "unsupported") && model && Renderer && (
            <Renderer model={model} text={text} onError={onRendererError} />
          )}
        </div>
      </div>
    </div>
  );
}
