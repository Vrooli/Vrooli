import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi } from "vitest";
import "@testing-library/jest-dom/vitest";
import { ExportButton } from "./ExportButton";

vi.mock("../lib/api", () => ({
  exportScheme: vi.fn().mockResolvedValue({
    scheme: { id: "s1", name: "Test" },
    information: [],
    thoughts: [],
    edges: [],
    export_format: "vrooli-graph-v1",
  }),
}));

vi.mock("@vrooli/api-base", () => ({
  resolveApiBase: () => "http://localhost:3000/api/v1",
  buildApiUrl: (path: string) => `http://localhost:3000/api/v1${path}`,
}));

function renderExportButton() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <ExportButton schemeId="s1" schemeName="My Scheme" />
    </QueryClientProvider>,
  );
}

describe("ExportButton", () => {
  it("renders export button", () => {
    renderExportButton();
    expect(screen.getByTestId("export-btn")).toBeInTheDocument();
  });

  it("renders with accessible label", () => {
    renderExportButton();
    expect(screen.getByLabelText("Export scheme")).toBeInTheDocument();
  });

  it("unmounts cleanly without errors", () => {
    const { unmount } = renderExportButton();
    // Should not throw on unmount even if timer is pending
    expect(() => unmount()).not.toThrow();
  });
});
