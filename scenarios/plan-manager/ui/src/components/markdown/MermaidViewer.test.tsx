import { describe, expect, it, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";

const renderMermaid = vi.fn();

vi.mock("mermaid", () => ({
  default: { initialize: vi.fn(), render: (...args: unknown[]) => renderMermaid(...args) },
}));

import { MermaidViewer } from "./MermaidViewer";

describe("MermaidViewer", () => {
  it("renders Mermaid SVG in the overlay", async () => {
    renderMermaid.mockResolvedValueOnce({ svg: "<svg><title>flowchart</title></svg>" });
    render(<MermaidViewer code="flowchart LR\nA --> B" onClose={vi.fn()} />);
    await waitFor(() => expect(screen.getByTestId("mermaid-viewer-svg").querySelector("svg")).toBeInTheDocument());
  });

  it("shows the original source when rendering fails", async () => {
    renderMermaid.mockRejectedValueOnce(new Error("invalid diagram"));
    render(<MermaidViewer code="not valid Mermaid" onClose={vi.fn()} />);
    expect(await screen.findByText("invalid diagram")).toBeInTheDocument();
    expect(screen.getByText("not valid Mermaid")).toBeInTheDocument();
    expect(screen.queryByTestId("mermaid-viewer-svg")).not.toBeInTheDocument();
  });
});
