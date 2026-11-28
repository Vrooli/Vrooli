import { MermaidDiagram, renderers, type PreviewModel, type PreviewRendererProps } from "@vrooli/file-viewer";
import { screen, waitFor } from "@testing-library/react";
import type { ComponentType } from "react";
import { describe, expect, it, vi } from "vitest";

import { renderWithProviders as render } from "../../test-utils";

// The shared viewer bundles neither engine; Web Console's host supplies them.
vi.mock("mermaid", () => ({
  default: { initialize: vi.fn(), render: vi.fn().mockResolvedValue({ svg: '<svg data-testid="diagram-drawn"></svg>' }) },
}));
vi.mock("@vrooli/file-viewer/spreadsheet", () => ({
  spreadsheetEngine: { parse: vi.fn().mockResolvedValue({ error: "encrypted" }) },
}));

const workbook: PreviewModel = {
  previewId: "wb", inputPath: "/book.xlsx", resolvedPath: "/book.xlsx", basename: "book.xlsx", kind: "spreadsheet",
  mimeType: "application/vnd.ms-excel", sizeBytes: 10, canPreview: true, canDownload: true, textContentAvailable: false,
  listingAvailable: false, blobHref: "/blob/wb", expiresMs: 0,
};

describe("FileViewerHost engines", () => {
  it("draws Mermaid diagrams with the app's mermaid", async () => {
    render(<MermaidDiagram code="graph TD; A-->B" />);
    expect(await screen.findByTestId("diagram-drawn")).toBeInTheDocument();
  });

  it("previews spreadsheets with the shared spreadsheet engine instead of reporting them unsupported", async () => {
    const Spreadsheet = renderers.spreadsheet as ComponentType<PreviewRendererProps>;
    // The spreadsheet renderer reads only the model; the rest of its contract is unused here.
    render(<Spreadsheet {...({ model: workbook } as PreviewRendererProps)} />);
    await waitFor(() => { expect(screen.getByRole("status")).toHaveTextContent("inspectionPreview.workbookEncrypted"); });
  });
});
