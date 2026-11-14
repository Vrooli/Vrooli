import workerUrl from "pdfjs-dist/legacy/build/pdf.worker.min.mjs?url";
import "pdfjs-dist/web/pdf_viewer.css";

export interface PdfCallbacks {
  ready: (pages: number) => void;
  page: (page: number) => void;
  scale: (scale: number) => void;
  matches: (current: number, total: number, pending: boolean) => void;
  password: (incorrect: boolean) => void;
  error: () => void;
}
export interface PdfSession {
  page: (page: number) => void;
  zoom: (value: string) => void;
  find: (query: string, previous?: boolean, again?: boolean) => void;
  password: (password: string) => void;
  destroy: () => void;
}

function record(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

export async function createPdfSession(container: HTMLDivElement, pages: HTMLDivElement, href: string, callbacks: PdfCallbacks): Promise<PdfSession> {
  // The component bundle reads pdfjsLib installed by the core; preserve ordering.
  const pdf = await import("pdfjs-dist/legacy/build/pdf.mjs");
  const { EventBus, PDFViewer, PDFLinkService, PDFFindController, GenericL10n } = await import("pdfjs-dist/legacy/web/pdf_viewer.mjs");
  pdf.GlobalWorkerOptions.workerSrc = workerUrl;
  const events = new EventBus();
  const abort = new AbortController();
  const link = new PDFLinkService({ eventBus: events, externalLinkTarget: 2, externalLinkRel: "noopener noreferrer" });
  const find = new PDFFindController({ eventBus: events, linkService: link });
  const l10n = new GenericL10n("en-US");
  const options = {
    container, viewer: pages, eventBus: events, linkService: link, findController: find,
    annotationMode: pdf.AnnotationMode.ENABLE, annotationEditorMode: pdf.AnnotationEditorType.DISABLE,
    maxCanvasPixels: 4_000_000, maxCanvasDim: 8192, enableDetailCanvas: false,
    enableAutoLinking: false, l10n, abortSignal: abort.signal,
  };
  const viewer = new PDFViewer(options);
  link.setViewer(viewer);
  const asset = new URL("pdf-assets/", document.baseURI).href;
  const task = pdf.getDocument({
    url: href, withCredentials: true, enableXfa: false,
    cMapUrl: `${asset}cmaps/`, cMapPacked: true,
    standardFontDataUrl: `${asset}standard_fonts/`, wasmUrl: `${asset}wasm/`, iccUrl: `${asset}iccs/`,
  });
  let updatePassword: ((value: string) => void) | null = null;
  let ready = false;
  task.onPassword = (update: (value: string) => void, reason: number) => {
    if (abort.signal.aborted) return;
    updatePassword = update;
    callbacks.password(reason === pdf.PasswordResponses.INCORRECT_PASSWORD);
  };
  const handlers = {
    pagesinit: () => { if (!abort.signal.aborted) { ready = true; viewer.currentScaleValue = "page-width"; callbacks.ready(viewer.pagesCount); } },
    pagechanging: (event: unknown) => { if (!abort.signal.aborted && record(event) && typeof event.pageNumber === "number") callbacks.page(event.pageNumber); },
    scalechanging: (event: unknown) => { if (!abort.signal.aborted && record(event) && typeof event.scale === "number") callbacks.scale(event.scale); },
    updatefindmatchescount: (event: unknown) => {
      if (!abort.signal.aborted && record(event) && record(event.matchesCount)) {
        const { current, total } = event.matchesCount;
        if (typeof current === "number" && typeof total === "number") callbacks.matches(current, total, false);
      }
    },
    updatefindcontrolstate: (event: unknown) => {
      if (abort.signal.aborted || !record(event)) return;
      const matches = record(event.matchesCount) ? event.matchesCount : {};
      callbacks.matches(typeof matches.current === "number" ? matches.current : 0, typeof matches.total === "number" ? matches.total : 0, event.state === 3);
    },
    pagerendered: (event: unknown) => { if (!abort.signal.aborted && record(event) && event.error) callbacks.error(); },
  };
  for (const [name, handler] of Object.entries(handlers)) events.on(name, handler);
  void task.promise.then((document) => {
    if (abort.signal.aborted) return;
    link.setDocument(document);
    viewer.setDocument(document);
  }).catch(() => { if (!abort.signal.aborted) callbacks.error(); });
  const resize = new ResizeObserver(() => {
    const value = viewer.currentScaleValue;
    if (ready && !abort.signal.aborted && ["page-width", "page-fit"].includes(value)) viewer.currentScaleValue = value;
  });
  resize.observe(container);
  return {
    page: (page) => { if (ready && Number.isFinite(page)) viewer.currentPageNumber = Math.max(1, Math.min(viewer.pagesCount, Math.round(page))); },
    zoom: (value) => { if (ready) viewer.currentScaleValue = value; },
    find: (query, previous = false, again = false) => {
      if (!ready) return;
      events.dispatch("find", { source: container, type: again ? "again" : "", query, phraseSearch: true, caseSensitive: false, entireWord: false, highlightAll: true, findPrevious: previous, matchDiacritics: false });
      if (!query) events.dispatch("findbarclose", { source: container });
    },
    password: (value) => { updatePassword?.(value); },
    destroy: () => {
      abort.abort();
      resize.disconnect();
      for (const [name, handler] of Object.entries(handlers)) events.off(name, handler);
      // Reset the public document owner before releasing the worker: this also
      // cancels render/search callbacks and removes document event listeners.
      viewer.setDocument(null);
      link.setDocument(null);
      void l10n.destroy().catch(() => {});
      void task.destroy().catch(() => {});
      pages.replaceChildren();
      updatePassword = null;
    },
  };
}
