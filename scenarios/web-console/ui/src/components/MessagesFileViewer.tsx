import { useMemo, useState } from "react";
import { AlertTriangle, ArrowLeft, Check, Copy, Loader2, RotateCw } from "lucide-react";
import { useTranslation } from "react-i18next";

import { strings } from "../consts/strings";
import { basename as pathBasename, pathCrumbs } from "../lib/paths";
import { DrawerShell } from "./DrawerShell";
import { rendererForKind } from "./file-preview/renderers";
import type { DirectorySort, PreviewState } from "./file-preview/types";

interface MessagesFileViewerProps {
  state: PreviewState;
  onClose: () => void;
  onReopen: () => void;
  onRendererError: (message: string) => void;
  onNavigate: (path: string) => void;
  onNavigateBack: () => void;
  onLoadMore: () => void;
  onListOptionsChange: (options: { sort?: DirectorySort; showHidden?: boolean }) => void;
}

export default function MessagesFileViewer({
  state,
  onClose,
  onReopen,
  onRendererError,
  onNavigate,
  onNavigateBack,
  onLoadMore,
  onListOptionsChange,
}: MessagesFileViewerProps) {
  const { t } = useTranslation();
  const { open, status, model, text, listing, error, requestedPath, stack, loadingMore } = state;

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
    return pathBasename(fullPath) || fullPath;
  }, [model?.resolvedPath, requestedPath, t]);

  // Breadcrumbs only appear for a directory: for a file the resolved path is
  // already shown in full below the title, and a second path rendering would
  // be noise rather than navigation.
  const crumbs = useMemo(
    () => (model?.kind === "directory" && model.resolvedPath ? pathCrumbs(model.resolvedPath) : []),
    [model?.kind, model?.resolvedPath],
  );

  const isLoading = status === "resolving" || status === "loadingText" || status === "loadingListing";
  const Renderer = model ? rendererForKind(model.kind) : null;
  const canGoBack = stack.length > 0;

  const headerActions = canGoBack ? (
    <button
      type="button"
      onClick={onNavigateBack}
      data-testid="file-preview-back"
      aria-label={t(strings.messagesFileViewer.directoryBack)}
      title={t(strings.messagesFileViewer.directoryBack)}
      className="shrink-0 rounded-lg border border-wc-default bg-wc-surface-input p-1.5 text-wc-text-secondary transition hover:bg-wc-surface-raised hover:text-wc-text-primary"
    >
      <ArrowLeft className="h-4 w-4" />
    </button>
  ) : null;

  const headerExtra = (
    <>
      {crumbs.length > 0 ? (
        // Scrolls horizontally rather than wrapping so a deep path never
        // consumes the sheet's vertical space on a phone.
        <nav
          data-testid="file-preview-breadcrumbs"
          aria-label={t(strings.messagesFileViewer.directoryPreview)}
          className="mt-1 flex items-center gap-0.5 overflow-x-auto whitespace-nowrap text-xs text-wc-text-muted"
        >
          {crumbs.map((crumb, i) => {
            const isLast = i === crumbs.length - 1;
            return (
              <span key={crumb.path} className="flex shrink-0 items-center gap-0.5">
                {i > 0 && <span className="text-wc-text-faint">/</span>}
                {isLast ? (
                  <span className="px-1 font-medium text-wc-text-primary">{crumb.label}</span>
                ) : (
                  <button
                    type="button"
                    onClick={() => onNavigate(crumb.path)}
                    className="rounded px-1 py-0.5 transition hover:bg-wc-surface-input hover:text-wc-text-primary"
                  >
                    {crumb.label}
                  </button>
                )}
              </span>
            );
          })}
        </nav>
      ) : (
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
      )}
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
    </>
  );

  return (
    <DrawerShell
      open={open}
      onClose={onClose}
      closeAriaLabel={t(strings.messagesFileViewer.closeAriaLabel)}
      title={basename}
      headerActions={headerActions}
      headerExtra={headerExtra}
      panelTestId="messages-file-viewer-panel"
    >
      <>
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
              <div className="mt-3 flex flex-wrap gap-2">
                {requestedPath && (
                  <button
                    type="button"
                    onClick={onReopen}
                    data-testid="file-preview-reopen"
                    className="inline-flex items-center gap-1.5 rounded-lg border border-red-400/40 bg-red-500/10 px-2.5 py-1.5 text-xs font-medium text-red-200 transition hover:bg-red-500/20"
                  >
                    <RotateCw className="h-3.5 w-3.5" />
                    {t(strings.messagesFileViewer.reopen)}
                  </button>
                )}
                {canGoBack && (
                  <button
                    type="button"
                    onClick={onNavigateBack}
                    data-testid="file-preview-error-back"
                    className="inline-flex items-center gap-1.5 rounded-lg border border-red-400/40 bg-red-500/10 px-2.5 py-1.5 text-xs font-medium text-red-200 transition hover:bg-red-500/20"
                  >
                    <ArrowLeft className="h-3.5 w-3.5" />
                    {t(strings.messagesFileViewer.directoryBack)}
                  </button>
                )}
              </div>
            </div>
          </div>
        )}

        {!isLoading && (status === "ready" || status === "unsupported") && model && Renderer && (
          <Renderer
            model={model}
            text={text}
            listing={listing}
            onError={onRendererError}
            onNavigate={onNavigate}
            onLoadMore={onLoadMore}
            onListOptionsChange={onListOptionsChange}
            loadingMore={loadingMore}
          />
        )}
      </>
    </DrawerShell>
  );
}
