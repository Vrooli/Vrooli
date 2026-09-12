import { useMemo, useState } from "react";
import { AlertTriangle, ArrowLeft, Loader2, RotateCw, Send } from "lucide-react";
import { useTranslation } from "react-i18next";

import { strings } from "../consts/strings";
import { basename as pathBasename, pathCrumbs } from "../lib/paths";
import { IconButton } from "@vrooli/react-component-library/IconButton";
import { FullPageDrawer } from "@vrooli/react-component-library/FullPageDrawer/1";
import { FilePath } from "@vrooli/react-component-library/FilePath/1";
import { rendererForKind } from "./file-preview/renderers";
import type { DirectorySort, PreviewState } from "./file-preview/types";

interface MessagesFileViewerProps {
  state: PreviewState;
  /**
   * Hand this file to another session in the group.
   *
   * This is the operator's existing habit — open the plan from the Messages
   * view, read it, then pass it on — so it is where the handoff earns the
   * most. The resolved path is the payload; the viewer does not classify it.
   */
  onHandoff?: (resolvedPath: string) => void;
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
  onHandoff,
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
  const [titleExpanded, setTitleExpanded] = useState(false);

  const displayPath = model?.resolvedPath ?? requestedPath ?? "";
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

  // The handoff sits beside Back, and only for a file with a resolved path:
  // there is nothing to hand over while the preview is still resolving, and a
  // directory listing is not a payload the operator meant to send.
  const canHandoff = Boolean(onHandoff && displayPath && model?.kind !== "directory");

  const titleEl = (
    <span
      role="button"
      tabIndex={0}
      onClick={() => { setTitleExpanded((prev) => !prev); }}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          setTitleExpanded((prev) => !prev);
        }
      }}
      className={titleExpanded ? "block break-words" : "line-clamp-2"}
    >
      {basename}
    </span>
  );

  const headerActions = (canGoBack || canHandoff) ? (
    <div className="flex shrink-0 items-center gap-1.5">
      {canGoBack && (
        <IconButton
          onClick={onNavigateBack}
          data-testid="file-preview-back"
          aria-label={t(strings.messagesFileViewer.directoryBack)}
          surface="soft"
          shape="rounded"
          className="shrink-0"
        >
          <ArrowLeft />
        </IconButton>
      )}
      {canHandoff && (
        <button
          type="button"
          onClick={() => onHandoff?.(displayPath)}
          data-testid="handoff-file-viewer"
          aria-label={t(strings.messagesFileViewer.handOffTitle)}
          title={t(strings.messagesFileViewer.handOffTitle)}
          className="inline-flex shrink-0 items-center gap-1.5 rounded-lg border border-wc-default bg-wc-surface-input px-2 py-1.5 text-xs font-medium text-wc-text-secondary transition hover:bg-wc-surface-raised hover:text-wc-text-primary"
        >
          <Send className="h-3.5 w-3.5" />
          {t(strings.messagesFileViewer.handOff)}
        </button>
      )}
    </div>
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
                    onClick={() => { onNavigate(crumb.path); }}
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
        <div className="mt-1 w-full min-w-0">
          {displayPath ? (
            <FilePath
              path={displayPath}
              showCopyButton
              copyLabel={t(strings.messagesFileViewer.copyPath)}
              copiedLabel={t(strings.messagesFileViewer.copied)}
              testId="file-preview-path"
              className="w-full text-xs text-wc-text-muted"
            />
          ) : (
            <p className="min-w-0 truncate text-xs text-wc-text-muted">
              {t(strings.messagesFileViewer.loadingFile)}
            </p>
          )}
        </div>
      )}
    </>
  );

  return (
    <FullPageDrawer
      // Sized to the app's viewport like every overlay. No text entry, so the
      // keyboard never moves it.
      avoidKeyboard
      open={open}
      onClose={onClose}
      closeLabel={t(strings.messagesFileViewer.closeAriaLabel)}
      title={titleEl}
      headerActions={headerActions}
      headerExtra={headerExtra}
      testId="messages-file-viewer-panel"
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
    </FullPageDrawer>
  );
}
