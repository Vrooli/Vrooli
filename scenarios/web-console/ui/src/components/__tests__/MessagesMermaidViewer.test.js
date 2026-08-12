import { jsx as _jsx } from "react/jsx-runtime";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import MessagesMermaidViewer from "../MessagesMermaidViewer";
vi.mock("react-i18next", () => ({
    useTranslation: () => ({
        t: (key, vars) => (vars ? `${key} ${JSON.stringify(vars)}` : key),
    }),
}));
const { mockUseMermaidSvg } = vi.hoisted(() => ({
    mockUseMermaidSvg: vi.fn(),
}));
vi.mock("../markdown/hooks/useMermaidSvg", () => ({
    useMermaidSvg: (code) => mockUseMermaidSvg(code),
}));
const CODE = "graph TD; A-->B";
function ready() {
    mockUseMermaidSvg.mockReturnValue({
        svgHtml: '<svg viewBox="0 0 100 80"><rect /></svg>',
        error: null,
        loading: false,
    });
}
describe("MessagesMermaidViewer", () => {
    beforeEach(() => {
        mockUseMermaidSvg.mockReset();
        Object.assign(navigator, { clipboard: { writeText: vi.fn().mockResolvedValue(undefined) } });
    });
    afterEach(() => vi.clearAllMocks());
    it("renders nothing when closed", () => {
        ready();
        const { container } = render(_jsx(MessagesMermaidViewer, { open: false, code: CODE, onClose: vi.fn() }));
        expect(container.querySelector('[data-testid="messages-mermaid-viewer-panel"]')).toBeNull();
    });
    it("renders the diagram surface, title, badges, and zoom controls when open", () => {
        ready();
        render(_jsx(MessagesMermaidViewer, { open: true, code: CODE, onClose: vi.fn() }));
        expect(screen.getByTestId("messages-mermaid-viewer-panel")).toBeInTheDocument();
        expect(screen.getByTestId("mermaid-zoom-surface")).toBeInTheDocument();
        expect(screen.getByText("mermaid.viewerTitle")).toBeInTheDocument();
        expect(screen.getByText("mermaid.badgeMessageDiagram")).toBeInTheDocument();
        expect(screen.getByLabelText("mermaid.zoomIn")).toBeInTheDocument();
        expect(screen.getByLabelText("mermaid.zoomOut")).toBeInTheDocument();
        expect(screen.getByLabelText("mermaid.fitToScreen")).toBeInTheDocument();
        expect(screen.getByLabelText("mermaid.resetZoom")).toBeInTheDocument();
        expect(screen.getByTestId("mermaid-zoom-level")).toHaveTextContent("%");
    });
    it("toggles to source view and back, hiding zoom controls in source mode", () => {
        ready();
        render(_jsx(MessagesMermaidViewer, { open: true, code: CODE, onClose: vi.fn() }));
        fireEvent.click(screen.getByLabelText("mermaid.showSource"));
        expect(screen.getByTestId("mermaid-viewer-source")).toHaveTextContent(CODE);
        expect(screen.queryByLabelText("mermaid.zoomIn")).toBeNull();
        fireEvent.click(screen.getByLabelText("mermaid.showDiagram"));
        expect(screen.getByTestId("mermaid-zoom-surface")).toBeInTheDocument();
    });
    it("copies the source through the clipboard", () => {
        ready();
        render(_jsx(MessagesMermaidViewer, { open: true, code: CODE, onClose: vi.fn() }));
        fireEvent.click(screen.getByLabelText("mermaid.copySource"));
        expect(navigator.clipboard.writeText).toHaveBeenCalledWith(CODE);
    });
    it("shows the render error with a source fallback and hides zoom controls", () => {
        mockUseMermaidSvg.mockReturnValue({ svgHtml: null, error: "Parse error on line 1", loading: false });
        render(_jsx(MessagesMermaidViewer, { open: true, code: CODE, onClose: vi.fn() }));
        expect(screen.getByText("mermaid.renderError")).toBeInTheDocument();
        expect(screen.getByText("Parse error on line 1")).toBeInTheDocument();
        expect(screen.queryByTestId("mermaid-zoom-surface")).toBeNull();
        expect(screen.queryByLabelText("mermaid.zoomIn")).toBeNull();
    });
    it("shows a loading state while rendering", () => {
        mockUseMermaidSvg.mockReturnValue({ svgHtml: null, error: null, loading: true });
        render(_jsx(MessagesMermaidViewer, { open: true, code: CODE, onClose: vi.fn() }));
        expect(screen.getByText("mermaid.rendering")).toBeInTheDocument();
    });
    it("closes via the close button and Escape", () => {
        ready();
        const onClose = vi.fn();
        render(_jsx(MessagesMermaidViewer, { open: true, code: CODE, onClose: onClose }));
        fireEvent.click(screen.getByLabelText("mermaid.closeViewer"));
        expect(onClose).toHaveBeenCalledTimes(1);
        fireEvent.keyDown(window, { key: "Escape" });
        expect(onClose).toHaveBeenCalledTimes(2);
    });
});
