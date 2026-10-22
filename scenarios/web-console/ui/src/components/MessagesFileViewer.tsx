import type { ArchiveView, ArchiveEntry } from "../api/filePreview";
import { useEffect, useMemo, useRef, useState } from "react";
import { AlertTriangle, ArrowLeft, FolderOpen, Loader2, Maximize2, Minimize2, RotateCw, Send } from "lucide-react";
import { useTranslation } from "react-i18next";

import { strings } from "../consts/strings";
import { selectors } from "../consts/selectors";
import { basename as pathBasename, dirname as pathDirname, pathCrumbs, type PathCrumb } from "../lib/paths";
import { IconButton } from "@vrooli/react-component-library/IconButton";
import { FullPageDrawer } from "@vrooli/react-component-library/FullPageDrawer/1";
import { FilePath } from "@vrooli/react-component-library/FilePath/1";
import { ArchiveBreadcrumbs } from "./file-preview/renderers/ArchivePreview";
import { rendererForKind } from "./file-preview/renderers";
import type { DirectorySort, PreviewState } from "./file-preview/types";
import { scriptRunPlan } from "../lib/fileExecution";

interface MessagesFileViewerProps {
  state: PreviewState;
 onArchiveFolder?: (folder:string)=>void;
 onArchiveEntry?: (entry:Pick<ArchiveEntry,"id"|"name">)=>void;
 onArchiveViewChange?: (view:Partial<ArchiveView>)=>void;
  /**
   * Hand this file to another session in the group.
   *
   * This is the operator's existing habit — open the plan from the Messages
   * view, read it, then pass it on — so it is where the handoff earns the
   * most. The resolved path is the payload; the viewer does not classify it.
   */
  onHandoff?: (resolvedPath: string) => void;
  onRun?: (resolvedPath: string, text: string) => void;
  onClose: () => void;
  onReopen: () => void;
  onRendererError: (message: string) => void;
  onNavigate: (path: string) => void;
  onNavigateBack: () => void;
  onLoadMore: () => void;
  onListOptionsChange: (options: { sort?: DirectorySort; showHidden?: boolean }) => void;
  presentation?: "drawer" | "dock";
  focusMode?: boolean;
  onFocus?: () => void;
  onRestore?: () => void;
  onResizeStart?: (event: React.PointerEvent<HTMLDivElement>) => void;
}

export default function MessagesFileViewer({
  state, onArchiveFolder, onArchiveEntry, onArchiveViewChange,
  onHandoff,
  onRun,
  onClose,
  onReopen,
  onRendererError,
  onNavigate,
  onNavigateBack,
  onLoadMore,
  onListOptionsChange,
  presentation = "drawer",
  focusMode = false,
  onFocus,
  onRestore,
  onResizeStart,
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

  const listingRefreshing = status === "loadingListing" && Boolean(listing);
  const isLoading = status === "resolving" || status === "loadingText" || (status === "loadingListing" && !listingRefreshing);
  const Renderer = model ? rendererForKind(model.kind) : null;
  const canGoBack = stack.length > 0;

  // The handoff sits beside Back, and only for a file with a resolved path:
  // there is nothing to hand over while the preview is still resolving, and a
  // directory listing is not a payload the operator meant to send.
  const canHandoff = Boolean(onHandoff && !model?.archiveMember && displayPath && model?.kind !== "directory");
  const canRun = Boolean(onRun && !model?.archiveMember && model?.kind === "code" && scriptRunPlan(displayPath, text?.content ?? ""));

  // "Open containing folder" lets a file that was opened directly reach its
  // directory and browse around it. navigateTo pushes the current file onto the
  // history stack, so Back returns to the file. It is offered for any on-disk
  // file with an addressable parent — not for a directory (which already shows
  // navigable breadcrumbs) nor an archive member (whose parent is inside the
  // archive, not the host filesystem).
  const parentDir = useMemo(
    () => (model?.kind !== "directory" && !model?.archiveMember ? pathDirname(displayPath) : null),
    [model?.kind, model?.archiveMember, displayPath],
  );

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

  const headerActions = (canHandoff || canRun) ? (
    <div className="flex shrink-0 items-center gap-1.5">
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
      {canRun && <button type="button" onClick={() => onRun?.(displayPath, text?.content ?? "")} data-testid="run-file-viewer" className="inline-flex shrink-0 items-center gap-1.5 rounded-lg bg-wc-accent px-2 py-1.5 text-xs font-medium text-white">Run</button>}
    </div>
  ) : null;

  const navigationRef = useRef<HTMLElement>(null), backFocusPending = useRef(false);
  useEffect(() => {
    if (!backFocusPending.current || isLoading) return;
    backFocusPending.current = false;
    const back = navigationRef.current?.querySelector<HTMLButtonElement>("[data-testid=file-preview-back]");
    if (back && !back.disabled) back.focus(); else navigationRef.current?.focus();
  }, [displayPath, isLoading, state.archiveView?.folder]);
  const navigation = (model?.kind === "directory" || model?.kind === "archive" || canGoBack || (isLoading && backFocusPending.current)) ? (
    <nav ref={navigationRef} tabIndex={-1} aria-label={t(strings.documentPreview.navigation)} data-testid={selectors.inspectionPreview.navigation}
      className="flex min-w-0 shrink-0 items-center gap-1 border-b border-wc-default bg-wc-surface-raised px-2 py-1 outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-wc-accent">
      <IconButton type="button" onClick={() => { backFocusPending.current = true; onNavigateBack(); }} disabled={!canGoBack} data-testid="file-preview-back" aria-label={t(strings.documentPreview.back)} title={t(strings.documentPreview.back)} surface="ghost" shape="rounded" className="shrink-0"><ArrowLeft className="rtl:rotate-180" /></IconButton>
      {model?.kind === "archive" ? <ArchiveBreadcrumbs folder={state.archiveView?.folder ?? ""} onNavigate={onArchiveFolder} /> : crumbs.length > 0 ? <PreviewBreadcrumbs crumbs={crumbs} onNavigate={onNavigate} /> : <span className="min-w-0 truncate px-2 text-xs text-wc-text-muted">{stack.at(-1)?.model.basename ?? basename}</span>}
    </nav>
  ) : null;
  const headerExtra = model?.kind === "directory" ? null : (
    <div className="mt-1 flex w-full min-w-0 items-center gap-1">{displayPath ? <>
      <FilePath path={displayPath} showCopyButton copyLabel={t(strings.messagesFileViewer.copyPath)} copiedLabel={t(strings.messagesFileViewer.copied)} testId="file-preview-path" className="min-w-0 flex-1 text-xs text-wc-text-muted" />
      {parentDir && <IconButton type="button" onClick={() => { onNavigate(parentDir); }} data-testid="file-preview-open-folder" aria-label={t(strings.messagesFileViewer.directoryUp)} title={t(strings.messagesFileViewer.directoryUp)} surface="ghost" shape="rounded" className="shrink-0"><FolderOpen /></IconButton>}
    </> :
      <p className="min-w-0 truncate text-xs text-wc-text-muted">{t(strings.messagesFileViewer.loadingFile)}</p>}</div>
  );

  const content = (
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
            <div className="mb-2 flex items-center gap-2 font-medium"><AlertTriangle className="h-4 w-4" /><span>{t(strings.messagesFileViewer.unavailable)}</span></div>
            <p>{error}</p>
            {requestedPath && <p className="mt-2 break-all text-xs text-red-200/80">{t(strings.messagesFileViewer.requestedPrefix, { path: requestedPath })}</p>}
            <div className="mt-3 flex flex-wrap gap-2">
              {requestedPath && <button type="button" onClick={onReopen} data-testid="file-preview-reopen" className="inline-flex items-center gap-1.5 rounded-lg border border-red-400/40 bg-red-500/10 px-2.5 py-1.5 text-xs font-medium text-red-200 transition hover:bg-red-500/20"><RotateCw className="h-3.5 w-3.5" />{t(strings.messagesFileViewer.reopen)}</button>}
              {model?.canDownload && model.blobHref && <a href={model.blobHref} download={model.basename} data-testid="file-preview-error-download" className="inline-flex min-h-11 items-center rounded-lg border border-red-400/40 px-3 text-xs font-medium text-red-200">{t(strings.messagesFileViewer.download)}</a>}
              {canGoBack && <button type="button" onClick={onNavigateBack} data-testid="file-preview-error-back" className="inline-flex items-center gap-1.5 rounded-lg border border-red-400/40 bg-red-500/10 px-2.5 py-1.5 text-xs font-medium text-red-200 transition hover:bg-red-500/20"><ArrowLeft className="h-3.5 w-3.5" />{t(strings.messagesFileViewer.directoryBack)}</button>}
            </div>
          </div>
        </div>
      )}

      {!isLoading && (status === "ready" || status === "unsupported" || listingRefreshing) && model && Renderer && (
        <Renderer archive={state.archive} archiveView={state.archiveView} onArchiveFolder={onArchiveFolder} onArchiveEntry={onArchiveEntry} onArchiveViewChange={onArchiveViewChange} model={model} text={text} listing={listing} listingRefreshing={listingRefreshing} onReopen={onReopen} onError={onRendererError} onNavigate={onNavigate} onLoadMore={onLoadMore} onListOptionsChange={onListOptionsChange} loadingMore={loadingMore} />
      )}
    </>
  );

  if (presentation === "dock") {
    return (
      <aside data-testid="file-preview-dock" data-focus-mode={focusMode ? "true" : "false"} className="relative flex h-full min-h-0 min-w-0 flex-col border-s border-wc-default bg-wc-surface-base">
        <div role="separator" aria-orientation="vertical" aria-label="Resize file viewer" onPointerDown={onResizeStart} className="absolute -start-1 top-0 z-10 h-full w-2 cursor-col-resize touch-none hover:bg-wc-accent/40" />
        <div className="flex min-h-0 flex-1 flex-col">
          <div className="shrink-0 border-b border-wc-default px-3 py-2">
            <div className="flex min-w-0 items-start justify-between gap-2">
              <div className="min-w-0 flex-1"><div className="text-sm font-semibold text-wc-text-primary">{titleEl}</div>{headerExtra}</div>
              <div className="flex shrink-0 items-center gap-1">{headerActions}
                <IconButton onClick={focusMode ? onRestore : onFocus} data-testid="file-preview-focus-toggle" aria-label={focusMode ? "Restore split view" : "Focus file viewer"} title={focusMode ? "Restore split view" : "Focus file viewer"} surface="soft" shape="rounded" className="shrink-0">{focusMode ? <Minimize2 /> : <Maximize2 />}</IconButton>
                <IconButton onClick={onClose} data-testid="file-preview-dock-close" aria-label={t(strings.messagesFileViewer.closeAriaLabel)} surface="soft" shape="rounded" className="shrink-0">×</IconButton>
              </div>
            </div>
          </div>
          {navigation}
          <div className="min-h-0 flex-1 overflow-hidden">{content}</div>
        </div>
      </aside>
    );
  }

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
      subheader={navigation}
      testId="messages-file-viewer-panel"
    >
      {content}
    </FullPageDrawer>
  );
}


function PreviewBreadcrumbs({ crumbs, onNavigate }: { crumbs: PathCrumb[]; onNavigate: (path: string) => void }) {
  const current = useRef<HTMLSpanElement>(null), path = crumbs.at(-1)?.path;
  // This component mounts with the actual portal content, after the drawer is
  // present. A parent effect can run before the portal attaches its children.
  useEffect(() => { current.current?.scrollIntoView({ block: "nearest", inline: "end" }); }, [path]);
  return <div data-testid="file-preview-breadcrumbs" className="flex min-w-0 flex-1 items-center gap-0.5 overflow-x-auto whitespace-nowrap text-xs text-wc-text-muted [scrollbar-width:thin]">
    {crumbs.map((crumb, index) => <span key={crumb.path} className="flex shrink-0 items-center gap-0.5">
      {index > 0 && <span aria-hidden="true" className="text-wc-text-faint">/</span>}
      {index === crumbs.length - 1 ? <span ref={current} aria-current="location" className="px-2 font-medium text-wc-text-primary" dir="auto">{crumb.label}</span> :
        <button type="button" onClick={() => { onNavigate(crumb.path); }} className="min-h-11 rounded px-2 outline-none hover:bg-wc-surface-input hover:text-wc-text-primary focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-wc-accent"><bdi>{crumb.label}</bdi></button>}
    </span>)}
  </div>;
}
