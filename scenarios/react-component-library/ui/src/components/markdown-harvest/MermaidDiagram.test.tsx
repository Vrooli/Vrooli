import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("mermaid", () => ({
  default: {
    initialize: vi.fn(),
    render: vi.fn().mockResolvedValue({ svg: "<svg aria-label='diagram'></svg>" }),
  },
}));

import { MermaidDiagram } from "./MermaidDiagram";

describe("MermaidDiagram", () => {
  afterEach(() => vi.restoreAllMocks());

  it("renders a diagram and exposes source, copy, and open actions", async () => {
    const onMermaidOpen = vi.fn();
    const writeText = vi.fn().mockResolvedValue(undefined);
    vi.stubGlobal("navigator", { clipboard: { writeText } });
    render(<MermaidDiagram code="graph TD; A-->B" onMermaidOpen={onMermaidOpen} />);

    await waitFor(() => expect(screen.getByLabelText("diagram")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: "Source" }));
    expect(screen.getByText("graph TD; A-->B")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Copy" }));
    await waitFor(() => expect(writeText).toHaveBeenCalledWith("graph TD; A-->B"));
    fireEvent.click(screen.getByRole("button", { name: "Open" }));
    expect(onMermaidOpen).toHaveBeenCalledWith("graph TD; A-->B");
  });

  it("renders safely when no external open action is provided", async () => {
    render(<MermaidDiagram code="graph TD; A-->B" />);
    await waitFor(() => expect(screen.getByText("graph TD; A-->B")).toBeInTheDocument());
    expect(screen.queryByRole("button", { name: "Open" })).not.toBeInTheDocument();
  });
});
