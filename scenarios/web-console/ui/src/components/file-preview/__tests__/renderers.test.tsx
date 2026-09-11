import { renderWithProviders as render } from "../../../test-utils";
import { describe, expect, it, vi } from "vitest";
import { fireEvent, screen } from "@testing-library/react";

import { strings } from "../../../consts/strings";
import { renderers, rendererForKind } from "../renderers";
import type { PreviewKind, PreviewModel, PreviewRendererProps, PreviewTextContent } from "../types";

// Avoid pulling shiki/mermaid into jsdom for the markdown renderer smoke test.
vi.mock("../../markdown", () => ({
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
    resolutionBasis: "absolute",
    kind: "image",
    mimeType: "application/octet-stream",
    sizeBytes: 100,
    canPreview: true,
    canDownload: true,
    supportsRange: true,
    textContentAvailable: false,
    listingAvailable: false,
    blobUrl: "/api/v1/sessions/s/file-previews/pv-1/blob",
    blobHref: "/api/v1/sessions/s/file-previews/pv-1/blob",
    expiresMs: Date.now() + 60_000,
    warnings: [],
    ...overrides,
  };
}

function text(content: string, kind: PreviewKind): PreviewTextContent {
  return { resolvedPath: "/tmp/a", kind, mimeType: "text/plain", content, truncated: false };
}

describe("renderer registry", () => {
  it("maps every preview kind to a renderer", () => {
    for (const kind of ALL_KINDS) {
      expect(renderers[kind]).toBeTypeOf("function");
    }
  });
  it("falls back to the unsupported renderer for unknown kinds", () => {
    expect(rendererForKind("totally-unknown" as PreviewKind)).toBe(renderers.unsupported);
  });
});

describe("image renderer", () => {
  it("renders an img with alt text and blob src", () => {
    const Renderer = renderers.image;
    render(<Renderer model={model({ basename: "logo.png", kind: "image", mimeType: "image/png" })} text={null} onError={() => {}} {...navProps} />);
    const img = screen.getByRole("img", { name: "logo.png" });
    expect(img).toHaveAttribute("src", "/api/v1/sessions/s/file-previews/pv-1/blob");
  });
});

describe("audio renderer", () => {
  it("renders an audio element with blob src", () => {
    const Renderer = renderers.audio;
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
    const Renderer = renderers.video;
    const { container } = render(
      <Renderer model={model({ kind: "video", mimeType: "video/mp4" })} text={null} onError={() => {}} {...navProps} />,
    );
    const video = container.querySelector("video");
    expect(video).not.toBeNull();
    expect(video).toHaveAttribute("src", "/api/v1/sessions/s/file-previews/pv-1/blob");
  });
});

describe("pdf renderer", () => {
  it("renders an iframe pointed at the blob href", () => {
    const Renderer = renderers.pdf;
    const { container } = render(
      <Renderer model={model({ kind: "pdf", mimeType: "application/pdf" })} text={null} onError={() => {}} {...navProps} />,
    );
    const iframe = container.querySelector("iframe");
    expect(iframe).toHaveAttribute("src", "/api/v1/sessions/s/file-previews/pv-1/blob");
  });
});

describe("csv renderer", () => {
  it("renders a table with header + rows", () => {
    const Renderer = renderers.csv;
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
    const Renderer = renderers.diff;
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
    const Renderer = renderers.unsupported;
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
  it.each(["report.html", "REPORT.HTM"])("renders %s in an isolated frame and lets users inspect source", (basename) => {
    const Renderer = renderers.code;
    const content = "<!doctype html><style>h1 { color: red }</style><h1>Report</h1><script>document.title = 'Report'</script>";
    render(<Renderer model={model({ kind: "code", basename, resolvedPath: `/tmp/${basename}` })} text={text(content, "code")} onError={() => {}} {...navProps} />);
    const frame = screen.getByTitle(strings.messagesFileViewer.htmlPreview);
    expect(frame).toHaveAttribute("sandbox", "allow-scripts");
    expect(frame).toHaveAttribute("referrerpolicy", "no-referrer");
    expect(frame.getAttribute("srcdoc")).toContain(content);
    expect(frame.getAttribute("srcdoc")).toContain("Content-Security-Policy");
    expect(frame.getAttribute("srcdoc")).toContain('<base href="about:srcdoc">');
    fireEvent.click(screen.getByRole("button", { name: strings.messagesFileViewer.htmlSource }));
    expect(screen.getByTestId("file-preview-code")).toBeInTheDocument();
    expect(screen.queryByTitle(strings.messagesFileViewer.htmlPreview)).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: strings.messagesFileViewer.htmlRendered }));
    expect(screen.getByTitle(strings.messagesFileViewer.htmlPreview)).toBeInTheDocument();
  });

  it("keeps line-linked and truncated HTML in source view initially", () => {
    const Renderer = renderers.text;
    render(<Renderer model={model({ kind: "text", basename: "report.htm", resolvedPath: "/tmp/report.htm", line: 1 })} text={{ ...text("<h1>Report</h1>", "text"), truncated: true }} onError={() => {}} {...navProps} />);
    expect(screen.getByTestId("file-preview-code")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: strings.messagesFileViewer.htmlRendered }));
    expect(screen.getByTestId("file-preview-notice")).toHaveTextContent(/truncated/i);
  });

  it("keeps ordinary code as source", () => {
    const Renderer = renderers.code;
    render(<Renderer model={model({ kind: "code", resolvedPath: "/tmp/a.ts" })} text={text("const markup = '<h1>Hello</h1>';", "code")} onError={() => {}} {...navProps} />);
    expect(screen.getByTestId("file-preview-code")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: strings.messagesFileViewer.htmlRendered })).not.toBeInTheDocument();
  });
});
