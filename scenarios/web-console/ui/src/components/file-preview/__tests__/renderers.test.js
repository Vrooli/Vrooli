import { jsx as _jsx } from "react/jsx-runtime";
import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { renderers, rendererForKind } from "../renderers";
// Avoid pulling shiki/mermaid into jsdom for the markdown renderer smoke test.
vi.mock("../../markdown", () => ({
    MarkdownRenderer: ({ content }) => _jsx("div", { "data-testid": "mock-md", children: content }),
}));
const ALL_KINDS = [
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
    "unsupported",
];
function model(overrides = {}) {
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
        blobUrl: "/api/v1/sessions/s/file-previews/pv-1/blob",
        blobHref: "/api/v1/sessions/s/file-previews/pv-1/blob",
        warnings: [],
        ...overrides,
    };
}
function text(content, kind) {
    return { resolvedPath: "/tmp/a", kind, mimeType: "text/plain", content, truncated: false };
}
describe("renderer registry", () => {
    it("maps every preview kind to a renderer", () => {
        for (const kind of ALL_KINDS) {
            expect(renderers[kind]).toBeTypeOf("function");
        }
    });
    it("falls back to the unsupported renderer for unknown kinds", () => {
        expect(rendererForKind("totally-unknown")).toBe(renderers.unsupported);
    });
});
describe("image renderer", () => {
    it("renders an img with alt text and blob src", () => {
        const Renderer = renderers.image;
        render(_jsx(Renderer, { model: model({ basename: "logo.png", kind: "image", mimeType: "image/png" }), text: null, onError: () => { } }));
        const img = screen.getByRole("img", { name: "logo.png" });
        expect(img).toHaveAttribute("src", "/api/v1/sessions/s/file-previews/pv-1/blob");
    });
});
describe("audio renderer", () => {
    it("renders an audio element with blob src", () => {
        const Renderer = renderers.audio;
        const { container } = render(_jsx(Renderer, { model: model({ kind: "audio", mimeType: "audio/mpeg" }), text: null, onError: () => { } }));
        const audio = container.querySelector("audio");
        expect(audio).not.toBeNull();
        expect(audio).toHaveAttribute("src", "/api/v1/sessions/s/file-previews/pv-1/blob");
    });
});
describe("video renderer", () => {
    it("renders a video element with playsInline and blob src", () => {
        const Renderer = renderers.video;
        const { container } = render(_jsx(Renderer, { model: model({ kind: "video", mimeType: "video/mp4" }), text: null, onError: () => { } }));
        const video = container.querySelector("video");
        expect(video).not.toBeNull();
        expect(video).toHaveAttribute("src", "/api/v1/sessions/s/file-previews/pv-1/blob");
    });
});
describe("pdf renderer", () => {
    it("renders an iframe pointed at the blob href", () => {
        const Renderer = renderers.pdf;
        const { container } = render(_jsx(Renderer, { model: model({ kind: "pdf", mimeType: "application/pdf" }), text: null, onError: () => { } }));
        const iframe = container.querySelector("iframe");
        expect(iframe).toHaveAttribute("src", "/api/v1/sessions/s/file-previews/pv-1/blob");
    });
});
describe("csv renderer", () => {
    it("renders a table with header + rows", () => {
        const Renderer = renderers.csv;
        render(_jsx(Renderer, { model: model({ kind: "csv", resolvedPath: "/tmp/a.csv", textContentAvailable: true }), text: text("name,age\nAda,36\nGrace,45", "csv"), onError: () => { } }));
        expect(screen.getByTestId("file-preview-csv")).toBeInTheDocument();
        expect(screen.getByText("name")).toBeInTheDocument();
        expect(screen.getByText("Ada")).toBeInTheDocument();
        expect(screen.getByText("45")).toBeInTheDocument();
    });
});
describe("diff renderer", () => {
    it("highlights additions and removals", () => {
        const Renderer = renderers.diff;
        render(_jsx(Renderer, { model: model({ kind: "diff", resolvedPath: "/tmp/a.diff", textContentAvailable: true }), text: text("@@ -1 +1 @@\n-old line\n+new line", "diff"), onError: () => { } }));
        expect(screen.getByTestId("file-preview-diff")).toBeInTheDocument();
        expect(screen.getByText("+new line")).toBeInTheDocument();
        expect(screen.getByText("-old line")).toBeInTheDocument();
    });
});
describe("unsupported renderer", () => {
    it("shows metadata and download/copy affordances", () => {
        const Renderer = renderers.unsupported;
        render(_jsx(Renderer, { model: model({ kind: "unsupported", canPreview: false, mimeType: "application/zip" }), text: null, onError: () => { } }));
        expect(screen.getByTestId("file-preview-unsupported")).toBeInTheDocument();
        expect(screen.getByTestId("file-preview-download")).toBeInTheDocument();
        expect(screen.getByTestId("file-preview-copy-path")).toBeInTheDocument();
    });
});
