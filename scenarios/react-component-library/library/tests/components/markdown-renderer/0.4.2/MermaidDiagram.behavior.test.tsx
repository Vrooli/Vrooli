import { fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("mermaid", () => ({
  default: {
    initialize: vi.fn(),
    render: vi
      .fn()
      .mockResolvedValue({ svg: "<svg aria-label='diagram'></svg>" }),
  },
}));

import { MermaidDiagram } from "@vrooli/react-component-library/markdown-renderer/0.4.2";
import { renderWithProviders } from "../../../../../ui/src/test-utils";

// Fixture source plus the label the mocked SVG carries, named once so the
// assertions survive a wording change in one place.
const mermaidSource = "graph TD; A-->B";
const diagramLabel = "diagram";

describe("MermaidDiagram", () => {
  // Restore the whole `navigator` object, not just mocks: replacing it with a
  // clipboard-only stub would otherwise leak into every later file in the
  // single-threaded test run.
  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it("renders a diagram and exposes source, copy, and open actions", async () => {
    const onMermaidOpen = vi.fn();
    const writeText = vi.fn().mockResolvedValue(undefined);
    vi.stubGlobal("navigator", { clipboard: { writeText } });
    renderWithProviders(
      <MermaidDiagram code={mermaidSource} onMermaidOpen={onMermaidOpen} />,
    );

    await waitFor(() =>
      expect(screen.getByLabelText(diagramLabel)).toBeInTheDocument(),
    );
    fireEvent.click(screen.getByRole("button", { name: "Source" }));
    expect(screen.getByText(mermaidSource)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Copy" }));
    await waitFor(() => expect(writeText).toHaveBeenCalledWith(mermaidSource));
    fireEvent.click(screen.getByRole("button", { name: "Open" }));
    expect(onMermaidOpen).toHaveBeenCalledWith(mermaidSource);
  });

  it("renders safely when no external open action is provided", async () => {
    renderWithProviders(<MermaidDiagram code={mermaidSource} />);
    fireEvent.click(screen.getByRole("button", { name: "Source" }));
    await waitFor(() =>
      expect(screen.getByText(mermaidSource)).toBeInTheDocument(),
    );
    expect(
      screen.queryByRole("button", { name: "Open" }),
    ).not.toBeInTheDocument();
  });
});
