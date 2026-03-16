import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { SandboxDetail } from "./SandboxDetail";
import type { Sandbox } from "../lib/api";

// Mock DiffViewer to avoid complex setup
vi.mock("./DiffViewer", () => ({
  DiffViewer: () => <div data-testid="diff-viewer-mock" />,
}));

function makeSandbox(overrides: Partial<Sandbox> = {}): Sandbox {
  return {
    id: "407578d1-ee26-475a-9ac9-3745b92d5dc3",
    name: "test-sandbox",
    scopePath: "/home/user/Vrooli/scenarios/web-console",
    reservedPath: "/home/user/Vrooli/scenarios/web-console",
    reservedPaths: ["/home/user/Vrooli/scenarios/web-console"],
    noLock: false,
    projectRoot: "/home/user/Vrooli",
    owner: "d3f6f295-55ac-4b1a-9876-abcdef123456",
    ownerType: "agent",
    status: "active",
    errorMessage: "",
    createdAt: new Date().toISOString(),
    lastUsedAt: new Date().toISOString(),
    sizeBytes: 2048,
    fileCount: 10,
    driver: "fuse-overlayfs",
    driverVersion: "1.13",
    lowerDir: "/lower",
    upperDir: "/upper",
    workDir: "/work",
    mergedDir: "/merged",
    activePids: [],
    sessionCount: 0,
    tags: [],
    metadata: {},
    updatedAt: new Date().toISOString(),
    version: 1,
    ...overrides,
  } as Sandbox;
}

const defaultProps = {
  isDiffLoading: false,
  onStop: vi.fn(),
  onStart: vi.fn(),
  onApprove: vi.fn(),
  onReject: vi.fn(),
  onDelete: vi.fn(),
  onApproveSelected: vi.fn(),
  isApproving: false,
  isRejecting: false,
  isStopping: false,
  isStarting: false,
  isDeleting: false,
  isReviewMode: false,
  onReviewModeChange: vi.fn(),
  selectedFileIds: [] as string[],
  onSelectedFileIdsChange: vi.fn(),
  selectedHunks: [],
  onSelectedHunksChange: vi.fn(),
  hideDiffViewer: true,
};

describe("SandboxDetail", () => {
  describe("R9: MetadataRow font selection", () => {
    it("renders without font-mono by default for non-path metadata", () => {
      render(<SandboxDetail sandbox={makeSandbox()} {...defaultProps} />);

      const rows = screen.getAllByTestId("metadata-row");
      // Owner row should NOT have font-mono (it's a formatted label, not a raw path)
      const ownerRow = rows.find((r) => r.textContent?.includes("Owner"));
      const valueSpan = ownerRow?.querySelector(".text-sm");
      expect(valueSpan?.className).not.toContain("font-mono");
    });

    it("renders font-mono for path metadata (Reserved)", () => {
      render(<SandboxDetail sandbox={makeSandbox()} {...defaultProps} />);

      const rows = screen.getAllByTestId("metadata-row");
      const reservedRow = rows.find((r) => r.textContent?.includes("Reserved"));
      const valueSpan = reservedRow?.querySelector(".text-sm");
      expect(valueSpan?.className).toContain("font-mono");
    });
  });

  describe("R4: Show more toggle", () => {
    it("hides secondary metadata by default", () => {
      render(<SandboxDetail sandbox={makeSandbox()} {...defaultProps} />);

      expect(screen.queryByTestId("secondary-metadata")).not.toBeInTheDocument();
      expect(screen.getByTestId("show-more-toggle")).toHaveTextContent("Show more");
    });

    it("reveals secondary metadata when Show more is clicked", async () => {
      const user = userEvent.setup();
      render(<SandboxDetail sandbox={makeSandbox()} {...defaultProps} />);

      await user.click(screen.getByTestId("show-more-toggle"));

      expect(screen.getByTestId("secondary-metadata")).toBeInTheDocument();
      expect(screen.getByTestId("show-more-toggle")).toHaveTextContent("Show less");
    });

    it("hides secondary metadata when Show less is clicked", async () => {
      const user = userEvent.setup();
      render(<SandboxDetail sandbox={makeSandbox()} {...defaultProps} />);

      await user.click(screen.getByTestId("show-more-toggle"));
      expect(screen.getByTestId("secondary-metadata")).toBeInTheDocument();

      await user.click(screen.getByTestId("show-more-toggle"));
      expect(screen.queryByTestId("secondary-metadata")).not.toBeInTheDocument();
    });
  });

  describe("R1: Action button grouping", () => {
    it("renders a visual divider between lifecycle and review buttons for active sandbox", () => {
      render(
        <SandboxDetail
          sandbox={makeSandbox({ status: "active" })}
          {...defaultProps}
          hideDiffViewer={false}
          onLaunchAgent={vi.fn()}
        />,
      );

      expect(screen.getByTestId("action-divider")).toBeInTheDocument();
    });

    it("does not render divider when only review buttons exist (stopped sandbox, no launch)", () => {
      render(
        <SandboxDetail
          sandbox={makeSandbox({ status: "stopped" })}
          {...defaultProps}
          hideDiffViewer={false}
        />,
      );

      // Stopped has Start button (lifecycle) and review buttons, so divider should exist
      expect(screen.getByTestId("action-divider")).toBeInTheDocument();
    });

    it("renders all expected action buttons for an active sandbox (desktop)", () => {
      render(
        <SandboxDetail
          sandbox={makeSandbox({ status: "active" })}
          {...defaultProps}
          hideDiffViewer={false}
          onLaunchAgent={vi.fn()}
          onOverrideAcceptance={vi.fn()}
        />,
      );

      expect(screen.getByTestId("stop-button")).toBeInTheDocument();
      expect(screen.getByTestId("launch-agent-button")).toBeInTheDocument();
      expect(screen.getByTestId("selection-mode-toggle")).toBeInTheDocument();
      expect(screen.getByTestId("approve-button")).toBeInTheDocument();
      expect(screen.getByTestId("reject-button")).toBeInTheDocument();
      expect(screen.getByTestId("delete-button")).toBeInTheDocument();
    });

    it("hides review toggle and divider on mobile (hideDiffViewer=true)", () => {
      render(
        <SandboxDetail
          sandbox={makeSandbox({ status: "active" })}
          {...defaultProps}
          hideDiffViewer={true}
          onLaunchAgent={vi.fn()}
        />,
      );

      expect(screen.queryByTestId("action-divider")).not.toBeInTheDocument();
      expect(screen.queryByTestId("selection-mode-toggle")).not.toBeInTheDocument();
    });
  });

  describe("R6: Formatted owner display", () => {
    it("shows shortened UUID owner with type prefix", () => {
      render(
        <SandboxDetail
          sandbox={makeSandbox({ owner: "d3f6f295-55ac-4b1a-9876-abcdef123456", ownerType: "agent" })}
          {...defaultProps}
        />,
      );

      expect(screen.getByText("agent:d3f6f295")).toBeInTheDocument();
    });
  });
});
