import { useCallback, useEffect, useRef, useState } from "react";

import { useFileViewerText } from "../i18n/context.js";
import { strings } from "../i18n/strings.js";
import { splitHtmlFileReference } from "../lib/fileReferences.js";
import { archiveStatusKey } from "../preview/archiveData.js";
import { DEFAULT_ARCHIVE_VIEW, type ArchiveView } from "../preview/types.js";
import {
  supportsArchives,
  type ArchiveSource,
  type DirectorySort,
  type PreviewListing,
  type PreviewModel,
  type PreviewSource,
  type PreviewSourceContext,
} from "../source/types.js";
import { IDLE_PREVIEW_STATE, type PreviewFrame, type PreviewSearch, type PreviewState } from "./state.js";

// MAX_STACK_DEPTH bounds the navigation history. A long walk keeps working;
// the oldest frame is dropped rather than growing without limit.
const MAX_STACK_DEPTH = 32;

const DEFAULT_SORT: DirectorySort = "dirs_first_name";

// useFilePreviewController owns the file-preview viewer's state machine over a
// PreviewSource. It is the single seam a host calls to open a path; the viewer
// component only renders the returned state.
//
// Every async landing is guarded by a request id. A navigation issued while a
// resolve, a text fetch, or a page fetch is in flight invalidates that
// response rather than letting it land in the wrong frame — which is what
// makes directory walking safe to drive from rapid clicks.
export interface OpenPreviewOptions {
  /**
   * A heading to scroll to once the document is open, by GitHub-style slug
   * (`plan`, `plan-1` for a second "Plan"). Applies to Markdown and Word
   * documents, or a URL fragment for an HTML preview.
   */
  anchor?: string;
}

export function useFilePreviewController(source: PreviewSource) {
  const { t } = useFileViewerText();
  // Callbacks read the latest source through a ref so a host may build a new
  // source object per render; only a different scope resets the viewer.
  const sourceRef = useRef(source);
  sourceRef.current = source;
  const scope = source.scope;
  const [state, setState] = useState<PreviewState>(IDLE_PREVIEW_STATE);
  const reqIdRef = useRef(0);
  const requestAbortRef = useRef<AbortController | null>(null);

  const cancelRequest = useCallback(() => { requestAbortRef.current?.abort(); reqIdRef.current++; }, []);

  // A host can keep the viewer mounted while the thing it browses changes (a
  // pane repinned to another session, another decision opened). Reset when the
  // scope changes so a previous scope's preview can never flash in the next.
  useEffect(() => {
    requestAbortRef.current?.abort();
    reqIdRef.current++;
    stateRef.current = IDLE_PREVIEW_STATE;
    setState(IDLE_PREVIEW_STATE);
    return cancelRequest;
  }, [scope, cancelRequest]);

  // stateRef mirrors the latest state so event handlers can read it without
  // doing work inside a setState updater — updaters must stay pure, since
  // React may invoke them more than once for a single update.
  const stateRef = useRef<PreviewState>(IDLE_PREVIEW_STATE);
  const apply = useCallback((next: PreviewState | ((prev: PreviewState) => PreviewState)) => {
    const value = typeof next === "function" ? next(stateRef.current) : next;
    stateRef.current = value;
    setState(value);
  }, []);

  const messageFromError = useCallback(
    (err: unknown): string => {
      if (err instanceof Error && err.message) return err.message;
      return t(strings.viewer.openFailed);
    },
    [t],
  );

  // probeMatch asks whether `dir` contains the relative path the user is trying
  // to locate, returning the resolved absolute path when it does. It drives the
  // "Found <relpath> → Open" affordance as the operator browses folders.
  const probeMatch = useCallback(
    async (dir: string, relPath: string): Promise<string | null> => {
      const joined = `${dir.replace(/\/+$/, "")}/${relPath.replace(/^[.]?\/+/, "")}`;
      try {
        const result = await sourceRef.current.resolve(joined);
        return result.outcome === "resolved" ? result.model.resolvedPath : null;
      } catch {
        return null;
      }
    },
    [],
  );

  // load resolves `path` and fetches whatever its kind needs. `stack` is the
  // history the new target should sit on top of, so the same routine serves a
  // fresh open (empty stack), a navigation (current frame pushed), and a Back
  // that had to re-resolve (frame popped).
  const load = useCallback(
    async (
      path: string,
      stack: PreviewFrame[],
      { context, listOptions, archiveSource, archiveView, searchCtx, anchor }: {
        context?: PreviewSourceContext;
        listOptions?: { sort: DirectorySort; showHidden: boolean };
        archiveSource?: ArchiveSource;
        archiveView?: ArchiveView;
        searchCtx?: { relPath: string } | null;
        anchor?: string;
      } = {},
    ) => {
      requestAbortRef.current?.abort();
      const request = new AbortController();
      requestAbortRef.current = request;
      const reqId = ++reqIdRef.current;
      apply({
        ...IDLE_PREVIEW_STATE,
        status: "resolving",
        open: true,
        requestedPath: path,
        stack, archiveSource, archiveView, anchor,
      });

      const src = sourceRef.current;
      try {
        let model: PreviewModel;
        if (archiveSource) {
          if (!supportsArchives(src)) throw new Error(t(strings.viewer.openFailed));
          const opened = await src.openArchiveEntry(archiveSource, request.signal);
          if (!opened.target) throw new Error(t(archiveStatusKey(opened.status)));
          model = opened.target;
        } else {
          let outcome = await src.resolve(path, { context, signal: request.signal });
          if (reqId !== reqIdRef.current) return;
          // Without the locate capability an ambiguous match opens its first
          // candidate and an unresolved path is simply unavailable.
          if (outcome.outcome === "ambiguous" && (searchCtx || !src.capabilities.locate) && outcome.candidates[0]) {
            outcome = await src.resolve(outcome.candidates[0].resolvedPath, { context, signal: request.signal });
            if (reqId !== reqIdRef.current) return;
          }
          if (outcome.outcome === "unresolved") {
            // A relative path that matched nowhere: drop into a folder browser
            // rooted at a base directory (prefer the project root) so the
            // operator can navigate to where the file actually lives.
            const bases = outcome.searchBases;
            const base = !searchCtx && src.capabilities.locate ? (bases.find((b) => b.resolutionBasis === "project_root") ?? bases[0]) : undefined;
            if (!base) {
              apply((s) => ({ ...s, status: "error", error: t(strings.viewer.openFailed) }));
              return;
            }
            void load(base.resolvedPath, [], { listOptions, searchCtx: { relPath: outcome.inputPath }, anchor });
            return;
          }
          if (outcome.outcome === "ambiguous") {
            // A relative path that matched more than one location: let the
            // user choose rather than silently picking the first.
            apply((s) => ({ ...s, status: "choosing", candidates: outcome.candidates, search: null }));
            return;
          }
          model = outcome.model;
        }
        if (reqId !== reqIdRef.current) return;

        if (model.kind === "archive") {
          // A source that cannot open archives leaves them download-only.
          if (!supportsArchives(src)) {
            apply((s) => ({ ...s, status: "unsupported", model: { ...model, kind: "unsupported" } }));
            return;
          }
          apply(s => ({ ...s, status: "loadingListing", model }));
          const archive = await src.listArchive(model, request.signal);
          if (reqId !== reqIdRef.current) return;
          apply(s => ({ ...s, status: "ready", model, archive, archiveView: archiveView ?? { ...DEFAULT_ARCHIVE_VIEW } }));
          return;
        }
        if (model.listingAvailable) {
          apply((s) => ({ ...s, status: "loadingListing", model }));
          const sort = listOptions?.sort ?? DEFAULT_SORT;
          const showHidden = listOptions?.showHidden ?? false;
          const listing = await src.listDirectory(model, { sort, showHidden, signal: request.signal });
          if (reqId !== reqIdRef.current) return;
          let search: PreviewSearch | null = null;
          if (searchCtx) {
            const foundPath = await probeMatch(model.resolvedPath, searchCtx.relPath);
            if (reqId !== reqIdRef.current) return;
            search = { relPath: searchCtx.relPath, foundPath };
          }
          apply((s) => ({ ...s, status: "ready", model, listing, search }));
          return;
        }

        if (!model.canPreview && !model.textContentAvailable) {
          // Resolvable but no dedicated renderer — show metadata + download.
          apply((s) => ({ ...s, status: "unsupported", model }));
          return;
        }

        if (model.textContentAvailable) {
          apply((s) => ({ ...s, status: "loadingText", model }));
          const text = await src.readText(model, request.signal);
          if (reqId !== reqIdRef.current) return;
          apply((s) => ({ ...s, status: "ready", model, text }));
          return;
        }

        // Blob-rendered kind (svg/image/pdf/audio/video) — bytes load lazily in
        // the renderer via model.blobHref.
        apply((s) => ({
          ...s,
          status: model.canPreview ? "ready" : "unsupported",
          model,
          text: null,
        }));
      } catch (err) {
        if (reqId !== reqIdRef.current) return;
        apply((s) => ({ ...s, status: "error", error: messageFromError(err) }));
      }
    },
    [apply, messageFromError, probeMatch, t],
  );

  // openPreview starts a fresh viewing session from a message link or chip,
  // optionally scrolled to a heading.
  // It clears any navigation history: the new target is the new root.
  const openPreview = useCallback(
    (path: string, source?: PreviewSourceContext, options?: OpenPreviewOptions) => {
      const target = source === "message_link" ? splitHtmlFileReference(path) : { path };
      return load(target.path, [], { context: source, anchor: options?.anchor ?? target.anchor });
    },
    [load],
  );

  // navigateTo opens a path from inside the viewer (a directory row, or a
  // breadcrumb segment), pushing the current target onto the history.
  const navigateTo = useCallback(
    (path: string) => {
      const current = stateRef.current;
      const frame = frameFrom(current);
      const stack = frame ? [...current.stack, frame].slice(-MAX_STACK_DEPTH) : current.stack;
      // Stay in search mode while browsing so each new directory is re-probed.
      const searchCtx = current.search ? { relPath: current.search.relPath } : null;
      void load(path, stack, { searchCtx, anchor: searchCtx ? current.anchor : undefined });
    },
    [load],
  );

  // chooseCandidate opens one of the ambiguous matches the picker offered.
  const chooseCandidate = useCallback((path: string) => { void load(path, [], { anchor: stateRef.current.anchor }); }, [load]);

  // openFoundMatch opens the file located during a folder-browse search.
  const openFoundMatch = useCallback(() => {
    const found = stateRef.current.search?.foundPath;
    if (found) void load(found, [], { anchor: stateRef.current.anchor });
  }, [load]);

  // cancelSearch keeps the folder browser open but dismisses the locate banner.
  const cancelSearch = useCallback(() => { apply((s) => ({ ...s, search: null })); }, [apply]);

  // navigateBack pops one frame. The restored frame is reused as-is while its
  // preview id is still live; once expired the path is re-resolved so Back
  // never lands on a dead blob href or a stale listing token.
  const navigateBack = useCallback(() => {
    const current = stateRef.current;
    const previous = current.stack[current.stack.length - 1];
    if (!previous) return;
    const stack = current.stack.slice(0, -1);

    if (isExpired(previous.model)) {
      void load(previous.requestedPath, stack, { ...previous, listOptions: listOptionsOf(previous.listing) });
      return;
    }

    requestAbortRef.current?.abort();
    // Invalidate anything still in flight so it cannot land on the frame we
    // are restoring.
    reqIdRef.current++;
    apply({
      status: "ready",
      open: true,
      requestedPath: previous.requestedPath,
      anchor: previous.anchor,
      model: previous.model,
      text: previous.text,
      listing: previous.listing,
      archive: previous.archive, archiveView: previous.archiveView, archiveSource: previous.archiveSource,
      error: null,
      loadingMore: false,
      stack,
    });
  }, [apply, load]);

  // loadMore appends the next page of the open directory. It carries the
  // request id so a page that lands after the user has navigated away is
  // discarded instead of being appended to a different directory.
  const loadMore = useCallback(() => {
    const current = stateRef.current;
    if (current.loadingMore || !current.model || !current.listing) return;
    if (!current.listing.nextPageToken) return;

    const reqId = reqIdRef.current;
    const model = current.model;
    const listing = current.listing;
    apply((s) => ({ ...s, loadingMore: true }));

    void (async () => {
      try {
        const page = await sourceRef.current.listDirectory(model, {
          sort: listing.sort,
          showHidden: listing.showHidden,
          pageToken: listing.nextPageToken,
        });
        if (reqId !== reqIdRef.current) return;
        apply((s) =>
          s.listing
            ? {
              ...s,
              loadingMore: false,
              listing: { ...page, entries: [...s.listing.entries, ...page.entries] },
            }
            : { ...s, loadingMore: false },
        );
      } catch (err) {
        if (reqId !== reqIdRef.current) return;
        apply((s) => ({ ...s, loadingMore: false, status: "error", error: messageFromError(err) }));
      }
    })();
  }, [apply, messageFromError]);

  // setListOptions re-lists the open directory from page one. Sort and filter
  // changes cannot be applied to an in-progress paged read — the server binds
  // both into its continuation token — so this always restarts the listing.
  const setListOptions = useCallback(
    (options: { sort?: DirectorySort; showHidden?: boolean; }) => {
      const current = stateRef.current;
      if (!current.model || !current.listing) return;

      const reqId = ++reqIdRef.current;
      const model = current.model;
      const sort = options.sort ?? current.listing.sort;
      const showHidden = options.showHidden ?? current.listing.showHidden;
      // Clear loadingMore: a page fetch in flight right now will be discarded
      // by the request-id guard and never gets to clear the flag itself, which
      // would leave "Load more" disabled for good.
      apply((s) => ({ ...s, status: "loadingListing", loadingMore: false }));

      void (async () => {
        try {
          const listing = await sourceRef.current.listDirectory(model, { sort, showHidden });
          if (reqId !== reqIdRef.current) return;
          apply((s) => ({ ...s, status: "ready", listing }));
        } catch (err) {
          if (reqId !== reqIdRef.current) return;
          apply((s) => ({ ...s, status: "error", error: messageFromError(err) }));
        }
      })();
    },
    [apply, messageFromError],
  );

  // reopen re-runs the current target, preserving the history beneath it and
  // the directory's ordering. It is the recovery action for both an error card
  // and the "directory changed" notice.
  const reopen = useCallback(() => {
    const current = stateRef.current;
    if (!current.requestedPath) return;
    void load(current.requestedPath, current.stack, { ...current, listOptions: listOptionsOf(current.listing) });
  }, [load]);


  const navigateArchiveFolder = useCallback((folder: string) => {
    const current = stateRef.current, frame = frameFrom(current);
    if (!frame || !current.archive || !current.archiveView || current.archiveView.folder === folder) return;
    reqIdRef.current++;
    apply({ ...current, archiveView: { ...current.archiveView, folder, query: "", page: 1 }, stack: [...current.stack, frame].slice(-MAX_STACK_DEPTH) });
  }, [apply]);
  const navigateArchiveEntry = useCallback((entry: { id: string; name: string; }) => {
    const current = stateRef.current, frame = frameFrom(current);
    if (!frame || !current.model || !current.archive) return;
    void load(`${current.model.resolvedPath}!/${entry.name}`, [...current.stack, frame].slice(-MAX_STACK_DEPTH), {
      archiveSource: { previewId: current.model.previewId, entryId: entry.id },
    });
  }, [load]);
  const setArchiveView = useCallback((view: Partial<ArchiveView>) => {
    apply(current => current.archiveView ? { ...current, archiveView: { ...current.archiveView, ...view } } : current);
  }, [apply]);

  const reportError = useCallback(
    (message: string) => {
      apply((s) => ({ ...s, status: "error", error: message }));
    },
    [apply],
  );

  const close = useCallback(() => {
    requestAbortRef.current?.abort();
    reqIdRef.current++;
    apply(IDLE_PREVIEW_STATE);
  }, [apply]);

  return {
    state,
    openPreview,
    navigateArchiveFolder, navigateArchiveEntry, setArchiveView,
    navigateTo,
    navigateBack,
    loadMore,
    setListOptions,
    reopen,
    reportError,
    close,
    chooseCandidate,
    openFoundMatch,
    cancelSearch,
  };
}

// frameFrom snapshots the current target as a history frame, or null when
// there is nothing worth returning to (an unresolved or errored view).
function frameFrom(state: PreviewState): PreviewFrame | null {
  if (!state.model || !state.requestedPath) return null;
  if (state.status !== "ready" && state.status !== "unsupported" && !(state.status === "loadingListing" && state.listing)) return null;
  return {
    requestedPath: state.requestedPath,
    anchor: state.anchor,
    model: state.model,
    text: state.text,
    listing: state.listing,
    archive: state.archive, archiveView: state.archiveView, archiveSource: state.archiveSource,
  };
}

// listOptionsOf carries a directory's ordering across a reload so reopening
// does not silently reset the view the operator chose.
function listOptionsOf(
  listing: PreviewListing | null,
): { sort: DirectorySort; showHidden: boolean; } | undefined {
  return listing ? { sort: listing.sort, showHidden: listing.showHidden } : undefined;
}

// isExpired reports whether a preview id has outlived its server-side TTL, in
// which case every handle it issued (blob href, listing token) is dead too.
function isExpired(model: PreviewModel): boolean {
  return model.expiresMs > 0 && model.expiresMs <= Date.now();
}
export type FilePreviewController = ReturnType<typeof useFilePreviewController>;
