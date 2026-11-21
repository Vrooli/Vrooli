import { Suspense, useEffect, useMemo, useRef, useState, type PointerEvent, type ReactNode } from "react";
import { AlertTriangle, ArrowLeft, Check, FileText, FolderOpen, Loader2, Maximize2, Minimize2, RotateCw, Search } from "lucide-react";
import { IconButton } from "@vrooli/react-component-library/IconButton";
import { FullPageDrawer } from "@vrooli/react-component-library/FullPageDrawer/1";
import { FilePath } from "@vrooli/react-component-library/FilePath/1";

import { useFileViewerText } from "../i18n/context.js";
import { strings } from "../i18n/strings.js";
import { basename as pathBasename, dirname as pathDirname, pathCrumbs, type PathCrumb } from "../lib/paths.js";
import { ArchiveBreadcrumbs } from "../preview/renderers/ArchiveBreadcrumbs.js";
import { rendererForKind } from "../preview/renderers/index.js";
import { selectors } from "../selectors.js";
import type { PreviewModel, PreviewSource, PreviewTextContent } from "../source/types.js";
import type { FilePreviewController } from "./useFilePreviewController.js";

/** What a host's header actions can act on: the open target and its loaded text. */
export interface ViewerActionContext {
  /** The resolved path shown in the header (empty while resolving). */
  path: string;
  model: PreviewModel | null;
  text: PreviewTextContent | null;
}

export interface FileViewerProps {
  source: PreviewSource;
  /** From `useFilePreviewController(source)`; the viewer renders its state. */
  controller: FilePreviewController;
  /**
   * Extra buttons beside the title (a hand-off, a Run, a picker's Select).
   * Called on every render with the current target; return null for none.
   */
  renderHeaderActions?: (context: ViewerActionContext) => ReactNode;
  /**
   * A host band under the preview: an attachment strip, a comment bar. It
   * does not scroll, sits above the bottom safe-area inset in the drawer, and
   * receives the open target so it can act on, or pin notes to, that file.
   * Return null for none.
   */
  renderFooter?: (context: ViewerActionContext) => ReactNode;
  onClose: () => void;
  /** "drawer" (default) is a full-page drawer; "dock" is a resizable side panel. */
  presentation?: "drawer" | "dock";
  focusMode?: boolean;
  onFocus?: () => void;
  onRestore?: () => void;
  onResizeStart?: (event: PointerEvent<HTMLDivElement>) => void;
  /** Show the resolved path under the title. Default true. */
  showPath?: boolean;
}

/** The viewer shell: title, navigation, header actions and the renderer for the open target. */
export function FileViewer({
  source,
  controller,
  renderHeaderActions,
  renderFooter,
  onClose,
  presentation = "drawer",
  focusMode = false,
  onFocus,
  onRestore,
  onResizeStart,
  showPath = true,
}: FileViewerProps) {
  const {
    state,
    reopen: onReopen,
    reportError: onRendererError,
    navigateTo: onNavigate,
    navigateBack: onNavigateBack,
    loadMore: onLoadMore,
    setListOptions: onListOptionsChange,
    chooseCandidate: onChooseCandidate,
    openFoundMatch: onOpenFoundMatch,
    cancelSearch: onCancelSearch,
    navigateArchiveFolder: onArchiveFolder,
    navigateArchiveEntry: onArchiveEntry,
    setArchiveView: onArchiveViewChange,
  } = controller;
  const { t } = useFileViewerText();
  const { open, status, model, text, listing, error, requestedPath, stack, loadingMore } = state;
  const [titleExpanded, setTitleExpanded] = useState(false);
  // basisLabel names the base a candidate resolved against, for the ambiguity
  // picker. Bases the package does not know fall through to the raw value.
  const basisLabel = (basis: string): string => {
    switch (basis) {
      case "session_cwd": return t(strings.viewer.basisWorkingDirectory);
      case "project_root": return t(strings.viewer.basisProjectRoot);
      case "absolute": return t(strings.viewer.basisAbsolute);
      default: return basis;
    }
  };

  const displayPath = model?.resolvedPath ?? requestedPath ?? "";
  const basename = useMemo(() => {
    const fullPath = model?.resolvedPath ?? requestedPath ?? "";
    if (!fullPath) return t(strings.viewer.filePreviewFallback);
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

  // "Open containing folder" lets a file that was opened directly reach its
  // directory and browse around it. navigateTo pushes the current file onto the
  // history stack, so Back returns to the file. It is offered for any on-disk
  // file with an addressable parent — not for a directory (which already shows
  // navigable breadcrumbs) nor an archive member (whose parent is inside the
  // archive, not the host filesystem).
  const canOpenParent = source.capabilities.parentFolders;
  const parentDir = useMemo(
    () => (canOpenParent && model?.kind !== "directory" && !model?.archiveMember ? pathDirname(displayPath) : null),
    [canOpenParent, model?.kind, model?.archiveMember, displayPath],
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

  const actionContext: ViewerActionContext = { path: displayPath, model, text };
  const extraActions = renderHeaderActions?.(actionContext);
  const footerContent = renderFooter?.(actionContext);
  const footer = footerContent ? <div className="min-w-0 flex-1" data-testid="file-preview-footer">{footerContent}</div> : null;
  const headerActions = extraActions ? <div className="flex shrink-0 items-center gap-1.5">{extraActions}</div> : null;

  const navigationRef = useRef<HTMLElement>(null), backFocusPending = useRef(false);
  useEffect(() => {
    if (!backFocusPending.current || isLoading) return;
    backFocusPending.current = false;
    const back = navigationRef.current?.querySelector<HTMLButtonElement>("[data-testid=file-preview-back]");
    if (back && !back.disabled) back.focus(); else navigationRef.current?.focus();
  }, [displayPath, isLoading, state.archiveView?.folder]);
  const navigation = (model?.kind === "directory" || model?.kind === "archive" || canGoBack || (isLoading && backFocusPending.current)) ? (
    <nav ref={navigationRef} tabIndex={-1} aria-label={t(strings.documentPreview.navigation)} data-testid={selectors.inspectionPreview.navigation}
      className="flex min-w-0 shrink-0 items-center gap-1 border-b border-fv-default bg-fv-surface-raised px-2 py-1 outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-fv-accent">
      <IconButton type="button" onClick={() => { backFocusPending.current = true; onNavigateBack(); }} disabled={!canGoBack} data-testid="file-preview-back" aria-label={t(strings.documentPreview.back)} title={t(strings.documentPreview.back)} surface="ghost" shape="rounded" className="shrink-0"><ArrowLeft className="rtl:rotate-180" /></IconButton>
      {model?.kind === "archive" ? <ArchiveBreadcrumbs folder={state.archiveView?.folder ?? ""} onNavigate={onArchiveFolder} /> : crumbs.length > 0 ? <PreviewBreadcrumbs crumbs={crumbs} onNavigate={onNavigate} /> : <span className="min-w-0 truncate px-2 text-xs text-fv-text-muted">{stack.at(-1)?.model.basename ?? basename}</span>}
    </nav>
  ) : null;
  const headerExtra = model?.kind === "directory" || !showPath ? null : (
    <div className="mt-1 flex w-full min-w-0 items-center gap-1">{displayPath ? <>
      <FilePath path={displayPath} showCopyButton copyLabel={t(strings.viewer.copyPath)} copiedLabel={t(strings.viewer.copied)} testId="file-preview-path" className="min-w-0 flex-1 text-xs text-fv-text-muted" />
      {parentDir && <IconButton type="button" onClick={() => { onNavigate(parentDir); }} data-testid="file-preview-open-folder" aria-label={t(strings.viewer.directoryUp)} title={t(strings.viewer.directoryUp)} surface="ghost" shape="rounded" className="shrink-0"><FolderOpen /></IconButton>}
    </> :
      <p className="min-w-0 truncate text-xs text-fv-text-muted">{t(strings.viewer.loadingFile)}</p>}</div>
  );

  const content = (
    <>
      {isLoading && <LoadingPreview />}

      {!isLoading && status === "error" && (
        <div className="h-full overflow-auto px-4 py-4">
          <div className="mx-auto max-w-2xl rounded-2xl border border-red-500/30 bg-red-500/10 p-4 text-sm text-red-300">
            <div className="mb-2 flex items-center gap-2 font-medium"><AlertTriangle className="h-4 w-4" /><span>{t(strings.viewer.unavailable)}</span></div>
            <p>{error}</p>
            {requestedPath && <p className="mt-2 break-all text-xs text-red-200/80">{t(strings.viewer.requestedPrefix, { path: requestedPath })}</p>}
            <div className="mt-3 flex flex-wrap gap-2">
              {requestedPath && <button type="button" onClick={onReopen} data-testid="file-preview-reopen" className="inline-flex items-center gap-1.5 rounded-lg border border-red-400/40 bg-red-500/10 px-2.5 py-1.5 text-xs font-medium text-red-200 transition hover:bg-red-500/20"><RotateCw className="h-3.5 w-3.5" />{t(strings.viewer.reopen)}</button>}
              {model?.canDownload && model.blobHref && <a href={model.blobHref} download={model.basename} data-testid="file-preview-error-download" className="inline-flex min-h-11 items-center rounded-lg border border-red-400/40 px-3 text-xs font-medium text-red-200">{t(strings.viewer.download)}</a>}
              {canGoBack && <button type="button" onClick={onNavigateBack} data-testid="file-preview-error-back" className="inline-flex items-center gap-1.5 rounded-lg border border-red-400/40 bg-red-500/10 px-2.5 py-1.5 text-xs font-medium text-red-200 transition hover:bg-red-500/20"><ArrowLeft className="h-3.5 w-3.5" />{t(strings.viewer.directoryBack)}</button>}
            </div>
          </div>
        </div>
      )}

      {!isLoading && status === "choosing" && (
        <div className="h-full overflow-auto px-4 py-4">
          <div className="mx-auto max-w-2xl rounded-2xl border border-fv-default bg-fv-surface-raised p-4 text-sm">
            <div className="mb-1 flex items-center gap-2 font-medium text-fv-text-primary"><FolderOpen className="h-4 w-4" /><span>{t(strings.viewer.candidatesTitle)}</span></div>
            {requestedPath && <p className="mb-3 break-all text-xs text-fv-text-muted">{requestedPath}</p>}
            <ul className="flex flex-col gap-2">
              {(state.candidates ?? []).map((candidate) => (
                <li key={candidate.resolvedPath}>
                  <button type="button" onClick={() => onChooseCandidate?.(candidate.resolvedPath)} data-testid="file-preview-candidate"
                    className="flex w-full min-w-0 items-center gap-2 rounded-lg border border-fv-default bg-fv-surface-input px-3 py-2 text-start transition hover:bg-fv-surface-raised">
                    <FileText className="h-4 w-4 shrink-0 text-fv-text-muted" />
                    <span className="min-w-0 flex-1">
                      <span className="block truncate text-fv-text-primary" dir="auto">{candidate.resolvedPath}</span>
                      <span className="block text-xs text-fv-text-muted">{basisLabel(candidate.resolutionBasis)}</span>
                    </span>
                  </button>
                </li>
              ))}
            </ul>
          </div>
        </div>
      )}

      {!isLoading && (status === "ready" || status === "unsupported" || listingRefreshing) && model && Renderer && (
        <div className="flex h-full min-h-0 flex-col">
          {state.search && (status === "ready" || listingRefreshing) && (
            <div className="flex shrink-0 items-center gap-2 border-b border-fv-default bg-fv-surface-raised px-3 py-2 text-xs" data-testid="file-preview-search-banner">
              {state.search.foundPath ? <>
                <Check className="h-4 w-4 shrink-0 text-green-400" />
                <span className="min-w-0 flex-1 truncate text-fv-text-primary" dir="auto"><span className="font-medium">{state.search.relPath}</span> {t(strings.viewer.searchFound)}</span>
                <button type="button" onClick={onOpenFoundMatch} data-testid="file-preview-open-found" className="shrink-0 rounded-lg bg-fv-accent px-2.5 py-1 font-medium text-white">{t(strings.viewer.searchOpen)}</button>
              </> : <>
                <Search className="h-4 w-4 shrink-0 text-fv-text-muted" />
                <span className="min-w-0 flex-1 truncate text-fv-text-muted" dir="auto">{t(strings.viewer.searchBrowse)} <span className="font-medium text-fv-text-secondary">{state.search.relPath}</span></span>
                <button type="button" onClick={onCancelSearch} data-testid="file-preview-cancel-search" className="shrink-0 rounded-lg border border-fv-default px-2.5 py-1 text-fv-text-secondary">{t(strings.viewer.searchDismiss)}</button>
              </>}
            </div>
          )}
          <div className="min-h-0 flex-1 overflow-hidden">
            <Suspense fallback={<LoadingPreview />}>
              <Renderer archive={state.archive} archiveView={state.archiveView} onArchiveFolder={onArchiveFolder} onArchiveEntry={onArchiveEntry} onArchiveViewChange={onArchiveViewChange} model={model} text={text} listing={listing} listingRefreshing={listingRefreshing} onReopen={onReopen} onError={onRendererError} onNavigate={onNavigate} onLoadMore={onLoadMore} onListOptionsChange={onListOptionsChange} loadingMore={loadingMore} source={source} anchor={state.anchor} />
            </Suspense>
          </div>
        </div>
      )}
    </>
  );

  if (presentation === "dock") {
    return (
      <aside data-testid="file-preview-dock" data-focus-mode={focusMode ? "true" : "false"} className="relative flex h-full min-h-0 min-w-0 flex-col border-s border-fv-default bg-fv-surface-base">
        <div role="separator" aria-orientation="vertical" aria-label={t(strings.viewer.resizeViewer)} onPointerDown={onResizeStart} className="absolute -start-1 top-0 z-10 h-full w-2 cursor-col-resize touch-none hover:bg-fv-accent/40" />
        <div className="flex min-h-0 flex-1 flex-col">
          <div className="shrink-0 border-b border-fv-default px-3 py-2">
            <div className="flex min-w-0 items-start justify-between gap-2">
              <div className="min-w-0 flex-1"><div className="text-sm font-semibold text-fv-text-primary">{titleEl}</div>{headerExtra}</div>
              <div className="flex shrink-0 items-center gap-1">{headerActions}
                <IconButton onClick={focusMode ? onRestore : onFocus} data-testid="file-preview-focus-toggle" aria-label={focusMode ? t(strings.viewer.restoreSplit) : t(strings.viewer.focusViewer)} title={focusMode ? t(strings.viewer.restoreSplit) : t(strings.viewer.focusViewer)} surface="soft" shape="rounded" className="shrink-0">{focusMode ? <Minimize2 /> : <Maximize2 />}</IconButton>
                <IconButton onClick={onClose} data-testid="file-preview-dock-close" aria-label={t(strings.viewer.closeAriaLabel)} surface="soft" shape="rounded" className="shrink-0">×</IconButton>
              </div>
            </div>
          </div>
          {navigation}
          <div className="min-h-0 flex-1 overflow-hidden">{content}</div>
          {footer && <div className="shrink-0 border-t border-fv-default">{footer}</div>}
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
      closeLabel={t(strings.viewer.closeAriaLabel)}
      title={titleEl}
      headerActions={headerActions}
      headerExtra={headerExtra}
      subheader={navigation}
      footer={footer}
      testId="messages-file-viewer-panel"
    >
      {content}
    </FullPageDrawer>
  );
}


function LoadingPreview() {
  const { t } = useFileViewerText();
  return (
    <div className="flex h-full items-center justify-center gap-2 text-fv-text-muted">
      <Loader2 className="h-4 w-4 animate-spin" />
      <span>{t(strings.viewer.loadingPreview)}</span>
    </div>
  );
}

export function PreviewBreadcrumbs({ crumbs, onNavigate }: { crumbs: PathCrumb[]; onNavigate: (path: string) => void }) {
  const current = useRef<HTMLSpanElement>(null), path = crumbs.at(-1)?.path;
  // This component mounts with the actual portal content, after the drawer is
  // present. A parent effect can run before the portal attaches its children.
  useEffect(() => { current.current?.scrollIntoView({ block: "nearest", inline: "end" }); }, [path]);
  return <div data-testid="file-preview-breadcrumbs" className="flex min-w-0 flex-1 items-center gap-0.5 overflow-x-auto whitespace-nowrap text-xs text-fv-text-muted [scrollbar-width:thin]">
    {crumbs.map((crumb, index) => <span key={crumb.path} className="flex shrink-0 items-center gap-0.5">
      {index > 0 && <span aria-hidden="true" className="text-fv-text-faint">/</span>}
      {index === crumbs.length - 1 ? <span ref={current} aria-current="location" className="px-2 font-medium text-fv-text-primary" dir="auto">{crumb.label}</span> :
        <button type="button" onClick={() => { onNavigate(crumb.path); }} className="min-h-11 rounded px-2 outline-none hover:bg-fv-surface-input hover:text-fv-text-primary focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-fv-accent"><bdi>{crumb.label}</bdi></button>}
    </span>)}
  </div>;
}
