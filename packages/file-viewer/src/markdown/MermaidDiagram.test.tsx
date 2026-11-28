import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { DiagramEngine, FileViewerLoaders } from "../loaders.js";
import { strings } from "../i18n/strings.js";
import { renderWithProviders as render } from "../test-utils/index.js";
import { MermaidDiagram } from "./index.js";

const SOURCE = "graph TD; A-->B";

function engine(svg = "<svg data-testid=\"drawn\"></svg>") {
  const value = { initialize: vi.fn(), render: vi.fn<DiagramEngine["render"]>().mockResolvedValue({ svg }) } satisfies DiagramEngine;
  const load = vi.fn<() => Promise<DiagramEngine>>().mockResolvedValue(value);
  return { value, load, loaders: { diagram: load } satisfies FileViewerLoaders };
}

describe("Mermaid diagrams through the host engine", () => {
  it("draws the diagram with the host's engine, initialised once with strict security", async () => {
    const { value, load, loaders } = engine();
    render(<><MermaidDiagram code={SOURCE} /><MermaidDiagram code="graph LR; C-->D" /></>, { host: { loaders } });
    await waitFor(() => { expect(screen.getAllByTestId("drawn")).toHaveLength(2); });
    expect(load).toHaveBeenCalledTimes(1);
    expect(value.initialize).toHaveBeenCalledTimes(1);
    expect(value.initialize).toHaveBeenCalledWith(expect.objectContaining({ startOnLoad: false, securityLevel: "strict" }));
    expect(value.render).toHaveBeenCalledWith(expect.stringMatching(/^mermaid-/), SOURCE);
  });

  it("shows the source with a translated note when the host supplied no engine", () => {
    render(<MermaidDiagram code={SOURCE} />);
    expect(screen.getByText(strings.markdown.diagram.unavailable)).toBeInTheDocument();
    expect(screen.getByText(SOURCE)).toBeInTheDocument();
    expect(screen.queryByTestId("drawn")).toBeNull();
  });

  it("reports an engine that fails to load or render, keeping the source readable", async () => {
    const load = vi.fn<() => Promise<DiagramEngine>>().mockRejectedValue(new Error("mermaid is not installed"));
    render(<MermaidDiagram code={SOURCE} />, { host: { loaders: { diagram: load } } });
    await waitFor(() => { expect(screen.getByText("mermaid is not installed")).toBeInTheDocument(); });
    expect(screen.getByText(SOURCE)).toBeInTheDocument();
    const { value, loaders } = engine(); value.render.mockRejectedValueOnce(new Error("Parse error on line 1"));
    render(<MermaidDiagram code="not a diagram" />, { host: { loaders } });
    await waitFor(() => { expect(screen.getByText("Parse error on line 1")).toBeInTheDocument(); });
  });

  it("forgets a failed load so a later diagram can try again", async () => {
    const { value } = engine();
    const load = vi.fn<() => Promise<DiagramEngine>>().mockRejectedValueOnce(new Error("offline")).mockResolvedValue(value);
    const loaders = { diagram: load };
    const first = render(<MermaidDiagram code={SOURCE} />, { host: { loaders } });
    await waitFor(() => { expect(screen.getByText("offline")).toBeInTheDocument(); });
    first.unmount();
    render(<MermaidDiagram code={SOURCE} />, { host: { loaders } });
    await waitFor(() => { expect(value.render).toHaveBeenCalled(); });
    expect(load).toHaveBeenCalledTimes(2);
  });
});
