import { renderWithProviders as render } from "../../test-utils/index.js";
import { Suspense } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";

import { strings } from "../../i18n/strings.js";
import { renderers, rendererForKind } from "../renderers/index.js";
import { AudioPreview } from "../renderers/AudioPreview.js";
import { CsvPreview } from "../renderers/CsvPreview.js";
import { DiffPreview } from "../renderers/DiffPreview.js";
import { ImagePreview } from "../renderers/ImagePreview.js";
import { MarkdownPreview } from "../renderers/MarkdownPreview.js";
import { PdfPreview } from "../renderers/PdfPreview.js";
import { CodePreview } from "../renderers/TextRenderers.js";
import { UnsupportedPreview } from "../renderers/UnsupportedPreview.js";
import { VideoPreview } from "../renderers/VideoPreview.js";

import { NO_CAPABILITIES, type PreviewSource } from "../../source/types.js";
import type { PreviewKind, PreviewModel, PreviewRendererProps, PreviewTextContent } from "../types.js";

// Avoid pulling shiki/mermaid into jsdom for the markdown renderer smoke test.
vi.mock("../../markdown/index.js", () => ({
  MarkdownRenderer: ({ content }: { content: string }) => <div data-testid="mock-md">{content}</div>,
}));

const ALL_KINDS: PreviewKind[] = [
  "markdown",
  "code",
  "text",
  "svg",
  "image",
  "pdf",
  "audio",
  "video",
  "csv",
  "spreadsheet",
  "document",
  "presentation",
  "ebook",
  "model3d",
  "font",
  "database",
  "archive",
  "diff",
  "directory",
  "unsupported",
];

// The directory-navigation half of the renderer contract. Every non-directory
// renderer ignores these, which is what makes the contract additive.
const navProps = {
  listing: null,
  onNavigate: () => {},
  onLoadMore: () => {},
  onListOptionsChange: () => {},
  loadingMore: false,
} satisfies Omit<PreviewRendererProps, "model" | "text" | "onError">;

function model(overrides: Partial<PreviewModel> = {}): PreviewModel {
  return {
    previewId: "pv-1",
    inputPath: "/tmp/a",
    resolvedPath: "/tmp/a",
    basename: "a",
    kind: "image",
    mimeType: "application/octet-stream",
    sizeBytes: 100,
    canPreview: true,
    canDownload: true,
    textContentAvailable: false,
    listingAvailable: false,
    blobHref: "/api/v1/sessions/s/file-previews/pv-1/blob",
    expiresMs: Date.now() + 60_000,
    ...overrides,
  };
}

const stubSource: PreviewSource = {
  scope: "test",
  capabilities: NO_CAPABILITIES,
  resolve: () => Promise.reject(new Error("unused")),
  readText: () => Promise.reject(new Error("unused")),
  listDirectory: () => Promise.reject(new Error("unused")),
};

function text(content: string, kind: PreviewKind): PreviewTextContent {
  return { resolvedPath: "/tmp/a", kind, mimeType: "text/plain", content, truncated: false };
}

describe("renderer registry", () => {
  it("maps every preview kind to a renderer", () => {
    for (const kind of ALL_KINDS) {
      expect(renderers[kind]).toBeTruthy();
    }
  });
  it("loads a renderer on first use behind a Suspense boundary", async () => {
    const Lazy = rendererForKind("code");
    render(<Suspense fallback={<p>loading</p>}><Lazy model={model({ kind: "code", resolvedPath: "/tmp/a.ts" })} text={text("const a = 1;", "code")} onError={() => {}} {...navProps} /></Suspense>);
    expect(screen.getByText("loading")).toBeInTheDocument();
    expect(await screen.findByText("const a = 1;")).toBeInTheDocument();
  });
  it("falls back to the unsupported renderer for unknown kinds", () => {
    expect(rendererForKind("totally-unknown" as PreviewKind)).toBe(renderers.unsupported);
  });
});

describe("image renderer", () => {
  it("renders an img with alt text and blob src", () => {
    const Renderer = ImagePreview;
    render(<Renderer model={model({ basename: "logo.png", kind: "image", mimeType: "image/png" })} text={null} onError={() => {}} {...navProps} />);
    const img = screen.getByRole("img", { name: "logo.png" });
    expect(img).toHaveAttribute("src", "/api/v1/sessions/s/file-previews/pv-1/blob");
  });
});

describe("media load failures", () => {
  it("replaces a broken image with a load-error notice", () => {
    const Renderer = ImagePreview;
    render(<Renderer model={model({ basename: "logo.png", kind: "image", mimeType: "image/png" })} text={null} onError={() => {}} {...navProps} />);
    fireEvent.error(screen.getByRole("img", { name: "logo.png" }));
    expect(screen.getByText(strings.viewer.mediaLoadError)).toBeInTheDocument();
  });

  it("shows a codec hint when audio fails to load", () => {
    const Renderer = AudioPreview;
    const { container } = render(<Renderer model={model({ kind: "audio", mimeType: "audio/mpeg" })} text={null} onError={() => {}} {...navProps} />);
    const audio = container.querySelector("audio");
    if (!audio) throw new Error("Expected an audio preview");
    fireEvent.error(audio);
    expect(screen.getByText(strings.viewer.codecHint)).toBeInTheDocument();
  });

  it("shows a codec hint when video fails to load", () => {
    const Renderer = VideoPreview;
    const { container } = render(<Renderer model={model({ kind: "video", mimeType: "video/mp4" })} text={null} onError={() => {}} {...navProps} />);
    const video = container.querySelector("video");
    if (!video) throw new Error("Expected a video preview");
    fireEvent.error(video);
    expect(screen.getByText(strings.viewer.codecHint)).toBeInTheDocument();
  });
});

describe("audio renderer", () => {
  it("renders an audio element with blob src", () => {
    const Renderer = AudioPreview;
    const { container } = render(
      <Renderer model={model({ kind: "audio", mimeType: "audio/mpeg" })} text={null} onError={() => {}} {...navProps} />,
    );
    const audio = container.querySelector("audio");
    expect(audio).not.toBeNull();
    expect(audio).toHaveAttribute("src", "/api/v1/sessions/s/file-previews/pv-1/blob");
  });
});

describe("video renderer", () => {
  it("renders a video element with playsInline and blob src", () => {
    const Renderer = VideoPreview;
    const { container } = render(
      <Renderer model={model({ kind: "video", mimeType: "video/mp4" })} text={null} onError={() => {}} {...navProps} />,
    );
    const video = container.querySelector("video");
    expect(video).not.toBeNull();
    expect(video).toHaveAttribute("src", "/api/v1/sessions/s/file-previews/pv-1/blob");
  });
});

describe("pdf renderer", () => {
  it("provides reading controls and a same-origin download fallback", () => {
    const Renderer = PdfPreview;
    render(
      <Renderer model={model({ kind: "pdf", mimeType: "application/pdf" })} text={null} onError={() => {}} {...navProps} />,
    );
    expect(screen.getByRole("button", { name: strings.artifactPreview.nextPage })).toBeDisabled();
    expect(screen.getByTestId("file-preview-download")).toHaveAttribute("href", "/api/v1/sessions/s/file-previews/pv-1/blob");
  });
});

describe("csv renderer", () => {
  it("renders a table with header + rows", () => {
    const Renderer = CsvPreview;
    render(
      <Renderer
        model={model({ kind: "csv", resolvedPath: "/tmp/a.csv", textContentAvailable: true })}
        text={text("name,age\nAda,36\nGrace,45", "csv")}
        onError={() => {}} {...navProps}
      />,
    );
    expect(screen.getByTestId("file-preview-csv")).toBeInTheDocument();
    expect(screen.getByText("name")).toBeInTheDocument();
    expect(screen.getByText("Ada")).toBeInTheDocument();
    expect(screen.getByText("45")).toBeInTheDocument();
  });
});

describe("diff renderer", () => {
  it("highlights additions and removals", () => {
    const Renderer = DiffPreview;
    render(
      <Renderer
        model={model({ kind: "diff", resolvedPath: "/tmp/a.diff", textContentAvailable: true })}
        text={text("@@ -1 +1 @@\n-old line\n+new line", "diff")}
        onError={() => {}} {...navProps}
      />,
    );
    expect(screen.getByTestId("file-preview-diff")).toBeInTheDocument();
    expect(screen.getByText("+new line")).toBeInTheDocument();
    expect(screen.getByText("-old line")).toBeInTheDocument();
  });
});

describe("unsupported renderer", () => {
  it("shows metadata and download/copy affordances", () => {
    const Renderer = UnsupportedPreview;
    render(
      <Renderer
        model={model({ kind: "unsupported", canPreview: false, mimeType: "application/zip" })}
        text={null}
        onError={() => {}} {...navProps}
      />,
    );
    expect(screen.getByTestId("file-preview-unsupported")).toBeInTheDocument();
    expect(screen.getByTestId("file-preview-download")).toBeInTheDocument();
    expect(screen.getByTestId("file-preview-copy-path")).toBeInTheDocument();
  });
});


describe("HTML text preview", () => {
  let resizeStage: (width: number, height: number) => void;
  const disconnect = vi.fn();
  const observe = vi.fn();
  beforeEach(() => {
    observe.mockClear();
    vi.stubGlobal("ResizeObserver", class {
      constructor(callback: ResizeObserverCallback) {
        resizeStage = (width, height) => { act(() => { callback([{ contentRect: { width, height } } as ResizeObserverEntry], this as unknown as ResizeObserver); }); };
      }
      observe = observe;
      disconnect = disconnect;
    });
  });
  afterEach(() => { vi.unstubAllGlobals(); });

  it("fits a desktop viewport on a phone and resizes without changing its CSS width", () => {
    const Renderer = CodePreview;
    const { unmount } = render(<Renderer model={model({ kind: "code", resolvedPath: "/tmp/a.html" })} text={text("<h1>Hello</h1>", "code")} onError={() => {}} {...navProps} />);
    resizeStage(360, 600);
    const frame = screen.getByTitle(strings.viewer.htmlPreview);
    fireEvent.change(screen.getByRole("combobox", { name: strings.viewer.htmlViewport }), { target: { value: "desktop" } });
    expect(frame).toHaveStyle({ width: "1440px", height: "2400px", transform: "scale(0.25)" });
    resizeStage(720, 600);
    expect(frame).toHaveStyle({ width: "1440px", height: "1200px", transform: "scale(0.5)" });
    fireEvent.change(screen.getByRole("combobox", { name: strings.viewer.htmlViewport }), { target: { value: "phone" } });
    expect(frame).toHaveStyle({ width: "390px", transform: "scale(1)" });
    unmount();
    expect(disconnect).toHaveBeenCalled();
  });

  it("zooms independently of viewport width, clamps steps, and can return to Fit", () => {
    const Renderer = CodePreview;
    render(<Renderer model={model({ kind: "code", resolvedPath: "/tmp/a.html" })} text={text("<h1>Hello</h1>", "code")} onError={() => {}} {...navProps} />);
    resizeStage(360, 600);
    fireEvent.change(screen.getByRole("combobox", { name: strings.viewer.htmlViewport }), { target: { value: "tablet" } });
    const zoom = screen.getByRole("combobox", { name: strings.viewer.htmlZoom });
    const frame = screen.getByTitle(strings.viewer.htmlPreview);
    fireEvent.change(zoom, { target: { value: "1" } });
    expect(frame).toHaveStyle({ width: "768px", height: "600px", transform: "scale(1)" });
    fireEvent.click(screen.getByRole("button", { name: strings.viewer.htmlZoomIn }));
    expect(frame).toHaveStyle({ transform: "scale(1.25)" });
    fireEvent.click(screen.getByRole("button", { name: strings.viewer.htmlZoomOut }));
    expect(frame).toHaveStyle({ transform: "scale(1)" });
    fireEvent.change(zoom, { target: { value: "2" } });
    expect(screen.getByRole("button", { name: strings.viewer.htmlZoomIn })).toBeDisabled();
    fireEvent.change(zoom, { target: { value: "0.1" } });
    expect(screen.getByRole("button", { name: strings.viewer.htmlZoomOut })).toBeDisabled();
    fireEvent.change(zoom, { target: { value: "fit" } });
    expect(frame).toHaveStyle({ transform: "scale(0.46875)" });
  });

  it("preserves zoom and viewport across source switches and resets them for another file", () => {
    const Renderer = CodePreview;
    const props = { model: model({ kind: "code", resolvedPath: "/tmp/a.html" }), text: text("<h1>Hello</h1>", "code"), onError: () => {}, ...navProps };
    const { rerender } = render(<Renderer {...props} />);
    resizeStage(360, 600);
    fireEvent.change(screen.getByRole("combobox", { name: strings.viewer.htmlViewport }), { target: { value: "desktop" } });
    fireEvent.change(screen.getByRole("combobox", { name: strings.viewer.htmlZoom }), { target: { value: "0.5" } });
    fireEvent.click(screen.getByRole("button", { name: strings.viewer.htmlSource }));
    expect(screen.getByRole("combobox", { name: strings.viewer.htmlZoom })).toBeDisabled();
    fireEvent.click(screen.getByRole("button", { name: strings.viewer.htmlRendered }));
    resizeStage(360, 600);
    expect(screen.getByRole("combobox", { name: strings.viewer.htmlViewport })).toHaveValue("desktop");
    expect(screen.getByTitle(strings.viewer.htmlPreview)).toHaveStyle({ width: "1440px", transform: "scale(0.5)" });
    rerender(<Renderer {...props} model={{ ...props.model, previewId: "pv-2", resolvedPath: "/tmp/b.html" }} />);
    expect(screen.getByRole("combobox", { name: strings.viewer.htmlViewport })).toHaveValue("responsive");
    expect(screen.getByRole("combobox", { name: strings.viewer.htmlZoom })).toHaveValue("fit");
  });

  it.each(["report.html", "REPORT.HTM"])("renders %s in an isolated frame and lets users inspect source", (basename) => {
    const Renderer = CodePreview;
    const content = "<!doctype html><style>h1 { color: red }</style><h1>Report</h1><script>document.title = 'Report'</script>";
    render(<Renderer model={model({ kind: "code", basename, resolvedPath: `/tmp/${basename}` })} text={text(content, "code")} onError={() => {}} {...navProps} />);
    const frame = screen.getByTitle(strings.viewer.htmlPreview);
    expect(frame).toHaveAttribute("sandbox", "allow-scripts");
    expect(frame).toHaveAttribute("referrerpolicy", "no-referrer");
    expect(frame.getAttribute("srcdoc")).toContain(content);
    expect(frame.getAttribute("srcdoc")).toContain("Content-Security-Policy");
    expect(frame.getAttribute("srcdoc")).toContain('<base href="about:srcdoc">');
    fireEvent.click(screen.getByRole("button", { name: strings.viewer.htmlSource }));
    expect(screen.getByTestId("file-preview-code")).toBeInTheDocument();
    expect(screen.queryByTitle(strings.viewer.htmlPreview)).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: strings.viewer.htmlRendered }));
    expect(screen.getByTitle(strings.viewer.htmlPreview)).toBeInTheDocument();
  });

  it("shows a download fallback rather than an empty file for unavailable code text", () => {
    const Renderer = CodePreview;
    render(<Renderer model={model({ kind: "code", sizeBytes: 3_759_455, textContentAvailable: false })} text={null} onError={() => {}} {...navProps} />);
    expect(screen.queryByTestId("file-preview-empty")).not.toBeInTheDocument();
    expect(screen.getByText(strings.viewer.textUnavailable)).toBeInTheDocument();
    expect(screen.getByTestId("file-preview-download")).toHaveAttribute("href", "/api/v1/sessions/s/file-previews/pv-1/blob");
  });

  it("renders HTML above the inline-text limit from bounded blob bytes and keeps source inspection", async () => {
    const content = "<h1>Research review</h1>";
    const fetch = vi.fn().mockResolvedValueOnce(new Response(content));
    vi.stubGlobal("fetch", fetch);
    const Renderer = CodePreview;
    render(<Renderer model={model({ kind: "code", resolvedPath: "/tmp/report.html", sizeBytes: 3_759_455, textContentAvailable: false })} text={null} onError={() => {}} {...navProps} />);
    expect(screen.queryByTestId("file-preview-empty")).not.toBeInTheDocument();
    await waitFor(() => { expect(screen.getByTitle(strings.viewer.htmlPreview).getAttribute("srcdoc")).toContain(content); });
    await waitFor(() => { expect(observe).toHaveBeenCalled(); });
    resizeStage(390, 600);
    expect(screen.getByTitle(strings.viewer.htmlPreview)).toHaveStyle({ width: "390px", height: "600px" });
    expect(fetch).toHaveBeenCalledWith("/api/v1/sessions/s/file-previews/pv-1/blob", expect.objectContaining({ credentials: "same-origin" }));
    fireEvent.click(screen.getByRole("button", { name: strings.viewer.htmlSource }));
    expect(screen.getByTestId("file-preview-code")).toBeInTheDocument();
  });

  it("does not call an oversize HTML document empty or execute a partial preview", () => {
    const Renderer = CodePreview;
    render(<Renderer model={model({ kind: "code", resolvedPath: "/tmp/report.html", sizeBytes: 17 * 1024 * 1024, textContentAvailable: false })} text={null} onError={() => {}} {...navProps} />);
    expect(screen.queryByTestId("file-preview-empty")).not.toBeInTheDocument();
    expect(screen.queryByTitle(strings.viewer.htmlPreview)).not.toBeInTheDocument();
    expect(screen.getByText(strings.viewer.textUnavailable)).toBeInTheDocument();
  });

  it("installs local pinch handling inside the isolated HTML document", () => {
    const Renderer = CodePreview;
    render(<Renderer model={model({ kind: "code", resolvedPath: "/tmp/a.html" })} text={text("<h1>Hello</h1>", "code")} onError={() => {}} {...navProps} />);
    const document = new DOMParser().parseFromString(screen.getByTitle(strings.viewer.htmlPreview).getAttribute("srcdoc") ?? "", "text/html");
    expect(document.querySelector("script[data-fv-html-pinch]")).not.toBeNull();
    expect(screen.getByRole("combobox", { name: strings.viewer.htmlZoom })).toHaveValue("fit");
  });

  it("initializes the isolated document fragment before the author's router and keeps it as text", () => {
    const anchor = 'review-2"></script><script id="unexpected">&';
    render(<CodePreview model={model({ kind: "code", resolvedPath: "/tmp/report.html" })} anchor={anchor}
      text={text('<script id="router">document.title = location.hash</script>', "code")} onError={() => {}} {...navProps} />);
    const frame = screen.getByTitle(strings.viewer.htmlPreview);
    const doc = new DOMParser().parseFromString(frame.getAttribute("srcdoc") ?? "", "text/html");
    expect(frame).toHaveAttribute("sandbox", "allow-scripts");
    expect(doc.scripts[0]?.dataset.fragment).toBe(anchor);
    expect(doc.scripts[0]?.textContent).toContain("about:srcdoc#");
    expect(doc.getElementById("router")).not.toBeNull();
    expect(doc.getElementById("unexpected")).toBeNull();
  });

  it("keeps line-linked and truncated HTML in source view initially", () => {
    const Renderer = CodePreview;
    render(<Renderer model={model({ kind: "text", basename: "report.htm", resolvedPath: "/tmp/report.htm", line: 1 })} text={{ ...text("<h1>Report</h1>", "text"), truncated: true }} onError={() => {}} {...navProps} />);
    expect(screen.getByTestId("file-preview-code")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: strings.viewer.htmlRendered }));
    expect(screen.getByTestId("file-preview-notice")).toHaveTextContent(/truncated/i);
  });

  it("uses the directory-aware runtime for interactive HTML previews when the source serves one", () => {
    const Renderer = CodePreview;
    render(
      <Renderer
        model={model({ kind: "text", basename: "index.html", resolvedPath: "/tmp/site/index.html", runtimeHref: "/api/v1/sessions/s/file-previews/pv-1/runtime/" })}
        text={text("<script src='render.js'></script>", "text")}
        onError={() => {}}
        source={{ ...stubSource, capabilities: { ...NO_CAPABILITIES, htmlRuntime: true } }}
        {...navProps}
      />,
    );
    const iframe = screen.getByTitle(strings.viewer.htmlPreview);
    expect(iframe).not.toHaveAttribute("src");
    expect(iframe.getAttribute("srcdoc")).toMatch(/<base href="http:\/\/localhost(?::\d+)?\/api\/v1\/sessions\/s\/file-previews\/pv-1\/runtime\/">/);
    expect(iframe.getAttribute("srcdoc")).toContain("script-src 'unsafe-inline' http://localhost");
    expect(iframe).toHaveAttribute("sandbox", "allow-scripts");
    expect(iframe).not.toHaveAttribute("allow-same-origin");
  });

  it("keeps the page inert when the source declares no html runtime", () => {
    render(
      <CodePreview
        model={model({ kind: "text", basename: "index.html", resolvedPath: "/tmp/site/index.html", runtimeHref: "/api/v1/sessions/s/file-previews/pv-1/runtime/" })}
        text={text("<script src='render.js'></script>", "text")}
        onError={() => {}}
        source={stubSource}
        {...navProps}
      />,
    );
    const srcdoc = screen.getByTitle(strings.viewer.htmlPreview).getAttribute("srcdoc") ?? "";
    expect(srcdoc).toContain('<base href="about:srcdoc">');
    expect(srcdoc).not.toContain("/runtime/");
  });

  it("keeps ordinary code as source", () => {
    const Renderer = CodePreview;
    render(<Renderer model={model({ kind: "code", resolvedPath: "/tmp/a.ts" })} text={text("const markup = '<h1>Hello</h1>';", "code")} onError={() => {}} {...navProps} />);
    expect(screen.getByTestId("file-preview-code")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: strings.viewer.htmlRendered })).not.toBeInTheDocument();
  });
});

describe("markdown text renderer", () => {
  it("renders markdown content through the shared renderer", () => {
    const Renderer = MarkdownPreview;
    render(
      <Renderer
        model={model({ kind: "markdown", resolvedPath: "/tmp/a.md", textContentAvailable: true })}
        text={text("# Heading", "markdown")}
        onError={() => {}}
        {...navProps}
      />,
    );
    expect(screen.getByTestId("file-preview-markdown")).toBeInTheDocument();
    expect(screen.getByTestId("mock-md")).toHaveTextContent("# Heading");
  });

  it("shows the empty-file state for whitespace-only markdown", () => {
    const Renderer = MarkdownPreview;
    render(
      <Renderer
        model={model({ kind: "markdown", resolvedPath: "/tmp/a.md", textContentAvailable: true })}
        text={text("   \n  ", "markdown")}
        onError={() => {}}
        {...navProps}
      />,
    );
    expect(screen.getByTestId("file-preview-empty")).toBeInTheDocument();
  });
});

describe("code line preview font controls", () => {
  it("increases and decreases the font size within bounds", () => {
    const Renderer = CodePreview;
    render(<Renderer model={model({ kind: "code", resolvedPath: "/tmp/a.ts" })} text={text("line one\nline two", "code")} onError={() => {}} {...navProps} />);
    fireEvent.click(screen.getByRole('button', { name: strings.inspectionPreview.options }));
    expect(screen.getByText("13px")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: strings.viewer.increaseFontSize }));
    expect(screen.getByText("14px")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: strings.viewer.decreaseFontSize }));
    fireEvent.click(screen.getByRole("button", { name: strings.viewer.decreaseFontSize }));
    expect(screen.getByText("12px")).toBeInTheDocument();
  });

  it("toggles word wrap on the code viewer", () => {
    const Renderer = CodePreview;
    render(<Renderer model={model({ kind: "code", resolvedPath: "/tmp/a.ts" })} text={text("a very long line", "code")} onError={() => {}} {...navProps} />);
    fireEvent.click(screen.getByRole('button', { name: strings.inspectionPreview.options }));
    const wrapButton = screen.getByRole("button", { name: strings.viewer.enableWordWrap });
    expect(wrapButton).toHaveAttribute("aria-pressed", "false");
    fireEvent.click(wrapButton);
    expect(screen.getByRole("button", { name: strings.viewer.disableWordWrap })).toHaveAttribute("aria-pressed", "true");
  });
});
