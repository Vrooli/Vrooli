import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { ViewerPage } from "./ViewerPage";

vi.mock("../../shared/hooks/viewerHooks", () => ({
  useDocViewer: () => ({
    path: "scenarios/alpha/docs/README.md",
    setPath: vi.fn(),
    viewMode: "code",
    setViewMode: vi.fn(),
    content: { content: "# Title", path: "scenarios/alpha/docs/README.md" },
    meta: {
      path: "scenarios/alpha/docs/README.md",
      docTypeLabel: "readme",
      sizeLabel: "1 KB",
      modifiedLabel: "Today",
      canReset: false,
      resetDefaults: { maxAgeDays: 30, keepMinEntries: 3 },
    },
    isLoading: false,
    hasError: false,
    errorMessage: "",
    refresh: vi.fn(),
    resetResult: null,
    resetError: "",
    isResetting: false,
    runReset: vi.fn(),
  }),
}));

describe("ViewerPage", () => {
  it("renders viewer header and path input", () => {
    render(<ViewerPage onNavigate={vi.fn()} />);
    expect(screen.getByText("Document Viewer")).toBeDefined();
    expect(screen.getByPlaceholderText(/docs\/QUICKSTART.md/i)).toBeDefined();
  });
});
