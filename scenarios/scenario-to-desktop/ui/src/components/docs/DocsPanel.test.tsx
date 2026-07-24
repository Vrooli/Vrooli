import { fireEvent, render, screen, waitFor } from "@/test-utils";
import { describe, expect, it, beforeEach, vi } from "vitest";
import { DocsPanel } from "./DocsPanel";

const { fetchDocsManifestMock, fetchDocContentMock, writeToClipboardMock } = vi.hoisted(() => ({
  fetchDocsManifestMock: vi.fn(),
  fetchDocContentMock: vi.fn(),
  writeToClipboardMock: vi.fn(),
}));

vi.mock("../../lib/api", () => ({
  fetchDocsManifest: fetchDocsManifestMock,
  fetchDocContent: fetchDocContentMock,
}));

vi.mock("../../lib/browser", () => ({
  writeToClipboard: writeToClipboardMock,
}));

const manifest = {
  version: "1",
  title: "Scenario documentation",
  defaultDocument: "guides/getting-started.md",
  sections: [
    {
      id: "guides",
      title: "Guides",
      documents: [
        { path: "guides/getting-started.md", title: "Getting started", description: "Set up the desktop wrapper" },
        { path: "guides/runbook.md", title: "Operator runbook", description: "Operate a build" },
      ],
    },
  ],
};

describe("DocsPanel", () => {
  beforeEach(() => {
    fetchDocsManifestMock.mockResolvedValue(manifest);
    fetchDocContentMock.mockImplementation(async (path: string) => ({ path, content: "" }));
    writeToClipboardMock.mockResolvedValue({ success: true });
  });

  it("selects the manifest default and tells the host which document is active", async () => {
    const onPathChange = vi.fn();

    render(<DocsPanel onPathChange={onPathChange} />);

    expect(await screen.findByRole("button", { name: /Getting started/i })).toBeInTheDocument();
    await waitFor(() => {
      expect(fetchDocContentMock).toHaveBeenCalledWith("guides/getting-started.md");
    });
    expect(onPathChange).toHaveBeenLastCalledWith("guides/getting-started.md");
    expect(screen.getByText("guides/getting-started.md")).toBeInTheDocument();
  });

  it("filters the document list and lets an operator select and copy a document path", async () => {
    render(<DocsPanel />);

    await screen.findByRole("button", { name: /Getting started/i });
    fireEvent.change(screen.getByPlaceholderText("Search docs..."), { target: { value: "runbook" } });

    expect(screen.getByRole("button", { name: /Operator runbook/i })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Getting started/i })).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /Operator runbook/i }));
    await waitFor(() => {
      expect(fetchDocContentMock).toHaveBeenCalledWith("guides/runbook.md");
    });

    fireEvent.click(screen.getByRole("button", { name: "Copy path" }));
    expect(writeToClipboardMock).toHaveBeenCalledWith("guides/runbook.md");
  });
});
