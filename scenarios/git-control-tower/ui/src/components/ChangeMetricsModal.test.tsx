import "@testing-library/jest-dom";
import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ChangeMetricsModal } from "./ChangeMetricsModal";
import type { DiffStats, RepoFileStats } from "../lib/api";

// Mock useIsMobile to test both modes
let mockIsMobile = false;
vi.mock("../hooks", () => ({
  useIsMobile: () => mockIsMobile,
}));

const fileStats: DiffStats = {
  additions: 42,
  deletions: 15,
  files: 1,
  net_lines: 27,
  hunk_count: 3,
  largest_hunk: 28,
  density: 0.053,
  is_binary: false,
  is_rename: false,
};

const renameStats: DiffStats = {
  additions: 5,
  deletions: 2,
  files: 1,
  net_lines: 3,
  hunk_count: 1,
  largest_hunk: 7,
  density: 0.143,
  is_rename: true,
  old_path: "old-name.ts",
};

const aggregateFileStats: RepoFileStats = {
  staged: {
    "a.ts": { additions: 10, deletions: 3, files: 1, net_lines: 7 },
    "img.png": { additions: 0, deletions: 0, files: 1, is_binary: true },
  },
  unstaged: {
    "b.ts": { additions: 5, deletions: 2, files: 1, net_lines: 3 },
  },
  untracked: {
    "c.ts": { additions: 8, deletions: 0, files: 1, net_lines: 8 },
  },
};

describe("ChangeMetricsModal", () => {
  beforeEach(() => {
    mockIsMobile = false;
  });

  it("does not render when isOpen is false", () => {
    render(
      <ChangeMetricsModal
        isOpen={false}
        onClose={() => {}}
        mode="file"
        stats={fileStats}
      />,
    );
    expect(screen.queryByTestId("metrics-modal")).not.toBeInTheDocument();
  });

  it("renders file-level metrics", () => {
    render(
      <ChangeMetricsModal
        isOpen={true}
        onClose={() => {}}
        mode="file"
        stats={fileStats}
        filePath="src/api.ts"
      />,
    );
    expect(screen.getByTestId("metrics-modal")).toBeInTheDocument();
    expect(screen.getByTestId("metric-net-lines")).toHaveTextContent("net +27");
    expect(screen.getByTestId("metric-hunk-count")).toHaveTextContent("3");
    expect(screen.getByTestId("metric-largest-hunk")).toHaveTextContent(
      "28 lines",
    );
    expect(screen.getByTestId("density-bar")).toBeInTheDocument();
  });

  it("shows rename info when applicable", () => {
    render(
      <ChangeMetricsModal
        isOpen={true}
        onClose={() => {}}
        mode="file"
        stats={renameStats}
        filePath="new-name.ts"
      />,
    );
    const rename = screen.getByTestId("metric-rename");
    expect(rename).toHaveTextContent("old-name.ts");
    expect(rename).toHaveTextContent("new-name.ts");
  });

  it("shows binary indicator when applicable", () => {
    render(
      <ChangeMetricsModal
        isOpen={true}
        onClose={() => {}}
        mode="file"
        stats={{ ...fileStats, is_binary: true }}
      />,
    );
    expect(screen.getByTestId("metric-binary")).toHaveTextContent("Binary file");
  });

  it("renders aggregate metrics with breakdown", () => {
    render(
      <ChangeMetricsModal
        isOpen={true}
        onClose={() => {}}
        mode="aggregate"
        fileStats={aggregateFileStats}
      />,
    );
    expect(screen.getByTestId("metrics-modal")).toBeInTheDocument();
    expect(screen.getByTestId("metric-total-files")).toBeInTheDocument();
    // Should show category rows
    expect(screen.getByText(/Staged/)).toBeInTheDocument();
    expect(screen.getByText(/Unstaged/)).toBeInTheDocument();
    expect(screen.getByText(/Untracked/)).toBeInTheDocument();
  });

  it("closes on escape key", () => {
    const onClose = vi.fn();
    render(
      <ChangeMetricsModal
        isOpen={true}
        onClose={onClose}
        mode="file"
        stats={fileStats}
      />,
    );
    fireEvent.keyDown(document, { key: "Escape" });
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("closes on backdrop click (desktop)", () => {
    const onClose = vi.fn();
    render(
      <ChangeMetricsModal
        isOpen={true}
        onClose={onClose}
        mode="file"
        stats={fileStats}
      />,
    );
    // Click the backdrop (outermost div with role=dialog)
    const dialog = screen.getByRole("dialog");
    fireEvent.click(dialog);
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("renders mobile layout when on mobile", () => {
    mockIsMobile = true;
    render(
      <ChangeMetricsModal
        isOpen={true}
        onClose={() => {}}
        mode="file"
        stats={fileStats}
        filePath="src/api.ts"
      />,
    );
    // Mobile layout has the slide-in class
    const dialog = screen.getByRole("dialog");
    expect(dialog.className).toContain("slide-in-from-bottom");
  });

  it("renders custom title when provided", () => {
    render(
      <ChangeMetricsModal
        isOpen={true}
        onClose={() => {}}
        mode="aggregate"
        fileStats={aggregateFileStats}
        title="Staged (My Group)"
      />,
    );
    expect(screen.getByText("Staged (My Group)")).toBeInTheDocument();
  });

  it("does not show density bar when density is zero", () => {
    render(
      <ChangeMetricsModal
        isOpen={true}
        onClose={() => {}}
        mode="file"
        stats={{ ...fileStats, density: 0 }}
      />,
    );
    expect(screen.queryByTestId("density-bar")).not.toBeInTheDocument();
  });
});
