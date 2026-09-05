import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils";
import { DocsPage } from "./DocsPage";
import { DocsSidebar } from "./components/DocsSidebar";
import { MarkdownViewer } from "./components/MarkdownViewer";
import type { DocsManifest } from "../../hooks/useDocs";

const docsManifest: DocsManifest = {
  version: "1.0",
  title: "Autoheal Docs",
  defaultDocument: "guide.md",
  sections: [
    {
      id: "guides",
      title: "Guides",
      visibility: "public",
      documents: [
        { path: "guide.md", title: "Guide", description: "Getting started" },
        { path: "reference.md", title: "Reference", description: "API reference" },
      ],
    },
    {
      id: "development",
      title: "Development",
      visibility: "developers-only",
      documents: [{ path: "dev.md", title: "Developer notes" }],
    },
  ],
};

const docsState = vi.hoisted(() => ({
  manifest: undefined as DocsManifest | undefined,
  manifestLoading: false,
  manifestError: null as Error | null,
  content: "# Guide",
  contentLoading: false,
  contentError: null as Error | null,
}));

vi.mock("../../hooks/useDocs", () => ({
  useDocsManifest: () => ({
    data: docsState.manifest,
    isLoading: docsState.manifestLoading,
    error: docsState.manifestError,
  }),
  useDocContent: () => ({
    data: docsState.content,
    isLoading: docsState.contentLoading,
    error: docsState.contentError,
  }),
}));

function setViewport(mobile: boolean) {
  window.matchMedia = vi.fn().mockImplementation(() => ({
    matches: mobile,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    addListener: vi.fn(),
    removeListener: vi.fn(),
  })) as typeof window.matchMedia;
}

describe("documentation surfaces", () => {
  beforeEach(() => {
    docsState.manifest = docsManifest;
    docsState.manifestLoading = false;
    docsState.manifestError = null;
    docsState.content = "# Guide";
    docsState.contentLoading = false;
    docsState.contentError = null;
    setViewport(false);
    window.location.hash = "";
    Object.assign(navigator, { clipboard: { writeText: vi.fn().mockResolvedValue(undefined) } });
  });

  it("renders and filters the document navigation", () => {
    const onSelect = vi.fn();
    const onSearch = vi.fn();
    renderWithProviders(
      <DocsSidebar
        manifest={docsManifest}
        selectedPath="guide.md"
        onSelectDoc={onSelect}
        searchQuery=""
        onSearchChange={onSearch}
      />,
    );

    expect(screen.getByText("Guides")).toBeInTheDocument();
    expect(screen.getByText("Developer notes")).toBeInTheDocument();
    fireEvent.click(screen.getByText("Reference"));
    expect(onSelect).toHaveBeenCalledWith("reference.md");
    fireEvent.click(screen.getByText("Guides"));
    expect(screen.queryByText("Reference")).not.toBeInTheDocument();
    fireEvent.change(screen.getByTestId("docs-search"), { target: { value: "missing" } });
    expect(onSearch).toHaveBeenCalledWith("missing");
  });

  it("supports the mobile navigation state", () => {
    setViewport(true);
    renderWithProviders(
      <DocsSidebar
        manifest={docsManifest}
        selectedPath={null}
        onSelectDoc={vi.fn()}
        searchQuery="missing"
        onSearchChange={vi.fn()}
      />,
    );
    const toggle = screen.getByRole("button", { name: /browse documents/i });
    expect(toggle).toHaveAttribute("aria-expanded", "false");
    fireEvent.click(toggle);
    expect(toggle).toHaveAttribute("aria-expanded", "true");
    expect(screen.getByText(/no documents match/i)).toBeInTheDocument();
  });

  it("renders the document page, handles hash navigation, and copies a path", async () => {
    window.location.hash = "#docs?path=reference.md";
    renderWithProviders(<DocsPage />);
    expect(screen.getByTestId("docs-page")).toBeInTheDocument();
    expect(screen.getAllByText("Guide").length).toBeGreaterThan(0);
    fireEvent.click(screen.getByTestId("docs-copy-path"));
    await waitFor(() => expect(screen.getByText("Copied!")).toBeInTheDocument());
    fireEvent.click(screen.getByText("Docs"));
  });

  it("renders loading and error states", () => {
    docsState.contentLoading = true;
    renderWithProviders(
      <MarkdownViewer content="" path="" isLoading onBack={vi.fn()} />,
    );
    expect(screen.getByText("Loading document...")).toBeInTheDocument();

    docsState.contentLoading = false;
    docsState.contentError = new Error("unavailable");
    renderWithProviders(
      <MarkdownViewer content="" path="guide.md" isLoading={false} error={docsState.contentError} onBack={vi.fn()} />,
    );
    expect(screen.getByText(/failed to load document/i)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /go back/i }));
  });

  it("renders rich markdown and the page fallback", async () => {
    renderWithProviders(
      <MarkdownViewer
        content={'# Heading\n\n## Sub\n\n**bold** *italic* `code`\n\n- item\n\n1. ordered\n\n[link](https://example.com)\n\n|a|b|\n|---|---|\n\n```text\nhello\n```\n\n```mermaid\ngraph TD\nA-->B\n```'}
        path="guide.md"
        isLoading={false}
        onBack={vi.fn()}
      />,
    );
    expect(screen.getByText("Heading")).toBeInTheDocument();
    await waitFor(() => expect(screen.getByTestId("docs-copy-path")).toBeInTheDocument());

    docsState.manifestError = new Error("docs unavailable");
    renderWithProviders(<DocsPage />);
    expect(screen.getByText(/documentation is available/i)).toBeInTheDocument();
  });
});
