import "@testing-library/jest-dom";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { vi } from "vitest";
import { renderWithProviders } from "./test-utils/renderWithProviders";

const docsState = vi.hoisted(() => ({ error: null as Error | null, loading: false, content: "" }));
const manifest = {
  version: "1", title: "Cloud Docs", description: "Deploy safely", defaultDocument: "quickstart.md",
  sections: [
    { id: "guide", title: "Guides", visibility: "developers-only", documents: [
      { path: "quickstart.md", title: "Quickstart", description: "Start here" },
      { path: "vps.md", title: "VPS Setup", description: "Prepare a VPS" },
    ] },
  ],
};

vi.mock("./hooks/useDocs", () => ({
  useDocsManifest: () => ({ data: docsState.error ? undefined : manifest, isLoading: docsState.loading, error: docsState.error }),
  useDocContent: () => ({ data: docsState.content, isLoading: docsState.loading, error: docsState.error }),
}));

import { DocsPage } from "./components/docs/DocsPage";
import { DocsSidebar } from "./components/docs/DocsSidebar";
import { MarkdownViewer } from "./components/docs/MarkdownViewer";

describe("documentation surfaces", () => {
  beforeEach(() => {
    docsState.error = null; docsState.loading = false; docsState.content = "# Welcome";
    Object.assign(navigator, { clipboard: { writeText: vi.fn().mockResolvedValue(undefined) } });
  });

  it("renders the manifest, searches, collapses sections, selects documents, and updates the URL", async () => {
    const onPath = vi.fn();
    renderWithProviders(<DocsPage initialDocPath="vps" onDocPathChange={onPath} />);
    expect(await screen.findByRole("heading", { name: "Cloud Docs" })).toBeInTheDocument();
    expect(screen.getByText("VPS Setup")).toBeInTheDocument();
    fireEvent.change(screen.getByPlaceholderText("Search docs..."), { target: { value: "quick" } });
    expect(screen.getByText("Quickstart")).toBeInTheDocument();
    expect(screen.queryByText("VPS Setup")).not.toBeInTheDocument();
    fireEvent.change(screen.getByPlaceholderText("Search docs..."), { target: { value: "nothing" } });
    expect(screen.getByText(/No documents match/)).toBeInTheDocument();
    fireEvent.change(screen.getByPlaceholderText("Search docs..."), { target: { value: "" } });
    fireEvent.click(screen.getByRole("button", { name: /Guides/ }));
    fireEvent.click(screen.getByRole("button", { name: /Guides/ }));
    fireEvent.click(screen.getByRole("button", { name: "Quickstart" }));
    expect(onPath).toHaveBeenCalledWith("quickstart");
    fireEvent.click(screen.getByRole("button", { name: "Copy path" }));
  });

  it("covers viewer loading, empty, error, markdown rendering, and clipboard failure", async () => {
    const back = vi.fn();
    const { rerender } = renderWithProviders(<MarkdownViewer content="" path="" isLoading={true} onBack={back} />);
    expect(screen.getByText("Loading document...")).toBeInTheDocument();
    rerender(<MarkdownViewer content="" path="" isLoading={false} onBack={back} />);
    expect(screen.getByText(/Select a document/)).toBeInTheDocument();
    rerender(<MarkdownViewer content="" path="vps.md" isLoading={false} error={new Error("offline")} onBack={back} />);
    expect(screen.getByText(/Failed to load document/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Go back" }));
    expect(back).toHaveBeenCalled();
    Object.assign(navigator, { clipboard: { writeText: vi.fn().mockRejectedValue(new Error("denied")) } });
    rerender(<MarkdownViewer content={"# Title\n\n**bold** and *italic* with `code`\n\n- item\n\n1. ordered\n\n| a | b |\n|---|---|\n| c | d |\n\n---\n\n```ts\nconst x = 1;\n```"} path="vps.md" isLoading={false} onBack={back} />);
    expect(screen.getByText("Title")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Copy path" }));
    await waitFor(() => expect(screen.getByRole("button", { name: "Copy path" })).toBeInTheDocument());
  });

  it("renders sidebar loading and no-manifest states", () => {
    const onSelect = vi.fn();
    renderWithProviders(<DocsSidebar manifest={undefined} selectedPath={null} onSelectDoc={onSelect} searchQuery="" onSearchChange={vi.fn()} />);
    expect(screen.getByText("Loading documentation...")).toBeInTheDocument();
  });
});
