import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createPdfSession, type PdfCallbacks } from "../pdfRuntime";

const vendor = vi.hoisted(() => ({
  handlers: new Map<string, (event?: unknown) => void>(),
  dispatch: vi.fn(), options: vi.fn(), load: vi.fn(), disconnect: vi.fn(),
  resize: () => {},
  viewer: { pagesCount: 27, currentPageNumber: 1, currentScaleValue: "", setDocument: vi.fn() },
  link: { setViewer: vi.fn(), setDocument: vi.fn() },
  l10nDestroy: vi.fn(async () => {}),
}));
vi.mock("pdfjs-dist/legacy/build/pdf.worker.min.mjs?url", () => ({ default: "/assets/pdf-worker.mjs" }));
vi.mock("pdfjs-dist/legacy/build/pdf.mjs", () => ({
  GlobalWorkerOptions: {}, AnnotationMode: { ENABLE: 1 }, AnnotationEditorType: { DISABLE: -1 },
  PasswordResponses: { NEED_PASSWORD: 1, INCORRECT_PASSWORD: 2 }, getDocument: vendor.load,
}));
vi.mock("pdfjs-dist/legacy/web/pdf_viewer.mjs", () => ({
  EventBus: class {
    on(name: string, callback: (event?: unknown) => void) { vendor.handlers.set(name, callback); }
    off(name: string) { vendor.handlers.delete(name); }
    dispatch = vendor.dispatch;
  },
  PDFViewer: vi.fn(function (options: unknown) { vendor.options(options); return vendor.viewer; }),
  PDFLinkService: vi.fn(function () { return vendor.link; }),
  PDFFindController: vi.fn(function () {}),
  GenericL10n: class { destroy = vendor.l10nDestroy; },
}));

let complete: (document: object) => void = () => {};
let reject: (reason: Error) => void = () => {};
let task: { promise: Promise<object>; onPassword?: (update: (password: string) => void, reason: number) => void; destroy: ReturnType<typeof vi.fn> };
beforeEach(() => {
  vi.clearAllMocks(); vendor.handlers.clear();
  vendor.viewer.currentPageNumber = 1; vendor.viewer.currentScaleValue = "";
  task = { promise: new Promise((resolve, fail) => { complete = resolve; reject = fail; }), destroy: vi.fn(async () => {}) };
  vendor.load.mockReturnValue(task);
  vi.stubGlobal("ResizeObserver", class {
    constructor(callback: () => void) { vendor.resize = callback; }
    observe() {} disconnect = vendor.disconnect;
  });
});
afterEach(() => { vi.unstubAllGlobals(); });
function callbacks(): PdfCallbacks { return { ready: vi.fn(), page: vi.fn(), scale: vi.fn(), matches: vi.fn(), password: vi.fn(), error: vi.fn() }; }
async function open() {
  const events = callbacks(), container = document.createElement("div"), pages = document.createElement("div");
  const session = await createPdfSession(container, pages, "/file.pdf", events);
  return { session, events, pages };
}

describe("PDF adapter ownership", () => {
  it("loads local decoding assets, bounds rendering, and exposes navigation only after readiness", async () => {
    const { session, events } = await open();
    expect(vendor.load).toHaveBeenCalledWith(expect.objectContaining({ url: "/file.pdf", withCredentials: true, enableXfa: false, cMapPacked: true, wasmUrl: new URL("pdf-assets/wasm/", document.baseURI).href }));
    expect(vendor.options).toHaveBeenCalledWith(expect.objectContaining({ maxCanvasPixels: 4_000_000, maxCanvasDim: 8192, annotationMode: 1, annotationEditorMode: -1, enableAutoLinking: false }));
    session.page(12); session.zoom("2"); session.find("early");
    expect(vendor.viewer.currentPageNumber).toBe(1);
    expect(vendor.dispatch).not.toHaveBeenCalled();
    complete({}); await Promise.resolve();
    expect(vendor.viewer.setDocument).toHaveBeenCalledWith({});
    vendor.handlers.get("pagesinit")?.();
    expect(events.ready).toHaveBeenCalledWith(27);
    expect(vendor.viewer.currentScaleValue).toBe("page-width");
    session.page(100); expect(vendor.viewer.currentPageNumber).toBe(27);
    session.page(-4); expect(vendor.viewer.currentPageNumber).toBe(1);
    session.page(NaN); expect(vendor.viewer.currentPageNumber).toBe(1);
    session.zoom("page-fit"); vendor.resize(); expect(vendor.viewer.currentScaleValue).toBe("page-fit");
    vendor.handlers.get("pagechanging")?.({ pageNumber: 5 });
    vendor.handlers.get("scalechanging")?.({ scale: 0.75 });
    expect(events.page).toHaveBeenCalledWith(5);
    expect(events.scale).toHaveBeenCalledWith(0.75);
    session.destroy();
  });
  it("forwards document search and clears highlights when search closes", async () => {
    const { session, events } = await open(); vendor.handlers.get("pagesinit")?.();
    session.find("literal query", true, true);
    expect(vendor.dispatch).toHaveBeenCalledWith("find", expect.objectContaining({ query: "literal query", type: "again", findPrevious: true, highlightAll: true, caseSensitive: false }));
    vendor.handlers.get("updatefindcontrolstate")?.({ matchesCount: { current: 1, total: 3 }, state: 3 });
    expect(events.matches).toHaveBeenLastCalledWith(1, 3, true);
    vendor.handlers.get("updatefindmatchescount")?.({ matchesCount: { current: 2, total: 3 } });
    expect(events.matches).toHaveBeenLastCalledWith(2, 3, false);
    session.find(""); expect(vendor.dispatch).toHaveBeenCalledWith("findbarclose", expect.anything());
    session.destroy();
  });
  it("retries passwords and surfaces load/render failures without hiding recovery actions", async () => {
    const { session, events } = await open(); const update = vi.fn();
    task.onPassword?.(update, 1); expect(events.password).toHaveBeenLastCalledWith(false);
    session.password("entered"); expect(update).toHaveBeenCalledWith("entered");
    task.onPassword?.(update, 2); expect(events.password).toHaveBeenLastCalledWith(true);
    vendor.handlers.get("pagerendered")?.({ error: new Error("render failure") });
    reject(new Error("invalid PDF")); await Promise.resolve(); await Promise.resolve();
    expect(events.error).toHaveBeenCalledTimes(2);
    session.destroy();
  });
  it("cancels late work, document handlers, decoding transport and layout observers on close", async () => {
    const { session, events, pages } = await open();
    const options: unknown = vendor.options.mock.calls[0]?.[0];
    const pageChange = vendor.handlers.get("pagechanging"), password = task.onPassword;
    pages.append(document.createElement("canvas")); session.destroy();
    expect(options).toHaveProperty("abortSignal.aborted", true);
    expect(vendor.viewer.setDocument).toHaveBeenCalledWith(null);
    expect(vendor.link.setDocument).toHaveBeenCalledWith(null);
    expect(vendor.handlers.size).toBe(0);
    expect(vendor.disconnect).toHaveBeenCalled(); expect(task.destroy).toHaveBeenCalled(); expect(vendor.l10nDestroy).toHaveBeenCalled();
    expect(pages.childElementCount).toBe(0);
    pageChange?.({ pageNumber: 2 }); password?.(vi.fn(), 2); complete({}); await Promise.resolve();
    expect(events.page).not.toHaveBeenCalled(); expect(events.password).not.toHaveBeenCalled();
    expect(vendor.viewer.setDocument).toHaveBeenCalledTimes(1);
  });
});
