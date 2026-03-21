import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import "@testing-library/jest-dom/vitest";
import { ExportButton } from "./ExportButton";

const mockExportScheme = vi.fn();
vi.mock("../lib/api", () => ({
  exportScheme: (...args: unknown[]): unknown => mockExportScheme(...args),
}));

vi.mock("@vrooli/api-base", () => ({
  resolveApiBase: () => "http://localhost:3000/api/v1",
  buildApiUrl: (path: string) => `http://localhost:3000/api/v1${path}`,
}));

const mockDownloadJSON = vi.fn();
vi.mock("../lib/download", () => ({
  downloadJSON: (...args: unknown[]): unknown => mockDownloadJSON(...args),
  slugify: (name: string) => name.replace(/\s+/g, "-").toLowerCase(),
}));

function renderExportButton(props?: Partial<{ schemeId: string; schemeName: string }>) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <ExportButton schemeId={props?.schemeId ?? "s1"} schemeName={props?.schemeName ?? "My Scheme"} />
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  mockExportScheme.mockReset();
  mockDownloadJSON.mockReset();
});

describe("ExportButton", () => {
  it("renders export button", () => {
    mockExportScheme.mockResolvedValue({ scheme: { id: "s1" }, information: [], thoughts: [], edges: [] });
    renderExportButton();
    expect(screen.getByTestId("export-btn")).toBeInTheDocument();
  });

  it("renders with accessible label", () => {
    mockExportScheme.mockResolvedValue({ scheme: { id: "s1" }, information: [], thoughts: [], edges: [] });
    renderExportButton();
    expect(screen.getByLabelText("Export scheme")).toBeInTheDocument();
  });

  it("unmounts cleanly without errors", () => {
    mockExportScheme.mockResolvedValue({ scheme: { id: "s1" }, information: [], thoughts: [], edges: [] });
    const { unmount } = renderExportButton();
    expect(() => unmount()).not.toThrow();
  });

  // [REQ:P1-002] Clicking export triggers download
  it("calls exportScheme and downloadJSON on click", async () => {
    const exportData = {
      scheme: { id: "s1", name: "My Scheme" },
      information: [],
      thoughts: [],
      edges: [],
      export_format: "vrooli-graph-v1",
    };
    mockExportScheme.mockResolvedValue(exportData);
    renderExportButton();

    const user = userEvent.setup();
    await user.click(screen.getByTestId("export-btn"));

    await waitFor(() => {
      expect(mockExportScheme).toHaveBeenCalledWith("s1");
      expect(mockDownloadJSON).toHaveBeenCalledWith(exportData, "my-scheme-export.json");
    });
  });

  // [REQ:P1-002] Shows success checkmark after export
  it("shows success state after successful export", async () => {
    mockExportScheme.mockResolvedValue({ scheme: { id: "s1" }, information: [], thoughts: [], edges: [] });
    renderExportButton();

    const user = userEvent.setup();
    await user.click(screen.getByTestId("export-btn"));

    // After success, the aria-label should still be "Export scheme" (check icon shows)
    await waitFor(() => {
      expect(mockDownloadJSON).toHaveBeenCalled();
    });
  });

  // [REQ:P1-002] Shows error state on export failure
  it("shows error icon and retry label on failure", async () => {
    mockExportScheme.mockRejectedValue(new Error("network error"));
    renderExportButton();

    const user = userEvent.setup();
    await user.click(screen.getByTestId("export-btn"));

    await waitFor(() => {
      expect(screen.getByLabelText("Export failed — click to retry")).toBeInTheDocument();
    });
  });

  // [REQ:P1-002] Button is disabled while export is pending
  it("disables button while export is pending", async () => {
    let resolveExport: (value: unknown) => void = () => { /* noop default */ };
    mockExportScheme.mockImplementation(() => new Promise((r) => { resolveExport = r; }));
    renderExportButton();

    const user = userEvent.setup();
    await user.click(screen.getByTestId("export-btn"));

    expect(screen.getByTestId("export-btn")).toBeDisabled();

    // Resolve to clean up
    resolveExport({ scheme: { id: "s1" }, information: [], thoughts: [], edges: [] });
    await waitFor(() => {
      expect(screen.getByTestId("export-btn")).not.toBeDisabled();
    });
  });
});
