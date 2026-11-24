import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import {
  acceptWorkspaceSourceBaseline,
  compareWorkspaceSourceBaseline,
  createWorkspaceSourceBaseline,
  listWorkspaceAssets,
  listWorkspaceCaptureReceipts,
  listWorkspaceSourceBaselineComparisons,
  listWorkspaceSourceBaselines,
} from "../../api/workspaces";
import { renderWithProviders } from "../../test-utils";
import { strings } from "../../consts/strings";
import { VisualDriftPanel } from "./VisualDriftPanel";

vi.mock("../../api/workspaces", () => ({
  acceptWorkspaceSourceBaseline: vi.fn(),
  compareWorkspaceSourceBaseline: vi.fn(),
  createWorkspaceSourceBaseline: vi.fn(),
  listWorkspaceAssets: vi.fn(),
  listWorkspaceCaptureReceipts: vi.fn(),
  listWorkspaceSourceBaselineComparisons: vi.fn(),
  listWorkspaceSourceBaselines: vi.fn(),
  workspaceAssetUrl: ({ url }: { url: string }) => url,
  workspaceCaptureScreenshotUrl: (_workspaceId: string, _stateId: string, receiptId: string) =>
    `/capture/${receiptId}`,
  workspaceSourceVisualHeatmapUrl: (_workspaceId: string, _stateId: string, id: string) =>
    `/heatmap/${id}`,
}));

vi.mock("../../i18n", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../i18n")>();
  return { ...actual, useTranslation: () => ({ t: (key: string) => key }) };
});

const candidate = {
  id: "candidate-1",
  workspaceId: "workspace-1",
  stateId: "state-1",
  assetId: "asset-1",
  sha256: "hash",
  sourceLocator: "design/source.png",
  createdAt: "2026-11-20T12:00:00Z",
  acceptedAt: "",
  acceptedBy: "",
};
const accepted = {
  ...candidate,
  id: "accepted-1",
  acceptedAt: "2026-11-20T12:01:00Z",
  acceptedBy: "Design reviewer",
};
const receipt = {
  id: "receipt-current",
  workspaceId: "workspace-1",
  stateId: "state-1",
  attemptId: "attempt-1",
  captureId: "capture-1",
  capturedAt: "2026-11-20T12:00:00Z",
  sourceRevision: 4,
  evidence: {} as never,
};
const historicalReceipt = {
  ...receipt,
  id: "receipt-history",
  capturedAt: "2026-11-19T12:00:00Z",
  sourceRevision: 3,
};
const comparison = {
  comparisonId: "comparison-1",
  workspaceId: "workspace-1",
  stateId: "state-1",
  sourceBaselineId: "accepted-1",
  sourceAssetId: "asset-history",
  currentReceiptId: "receipt-history",
  createdAt: "2026-11-20T12:02:00Z",
  result: {
    jobId: "job-1",
    verdict: "changed",
    dimensionsMatch: true,
    width: 100,
    height: 200,
    changedPixels: "20",
    totalPixels: "100",
    changedFraction: 0.2,
    mae: 0.1,
    rmse: 0.2,
    psnr: 20,
    phashDistance: 2,
    phashSimilarity: 0.9,
    ssim: 0.8,
    heatmapRef: "diff.png",
    warnings: [],
  },
};

describe("VisualDriftPanel source baseline workflow", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(listWorkspaceCaptureReceipts).mockResolvedValue([receipt]);
    vi.mocked(listWorkspaceSourceBaselines).mockResolvedValue({
      candidates: [candidate as never],
      accepted: null,
    });
    vi.mocked(listWorkspaceAssets).mockResolvedValue([
      {
        id: "asset-1",
        workspaceId: "workspace-1",
        name: "source.png",
        mime: "image/png",
        bytes: 10,
        width: 100,
        height: 200,
        fidelity: "reference",
        addedAt: "",
        url: "/asset/source.png",
      },
    ]);
    vi.mocked(listWorkspaceSourceBaselineComparisons).mockResolvedValue([]);
    vi.mocked(createWorkspaceSourceBaseline).mockResolvedValue(candidate as never);
    vi.mocked(acceptWorkspaceSourceBaseline).mockResolvedValue(accepted as never);
    vi.mocked(compareWorkspaceSourceBaseline).mockResolvedValue(comparison as never);
  });

  it("distinguishes candidates from accepted sources and requires an explicit attributed acceptance", async () => {
    renderWithProviders(<VisualDriftPanel workspaceId="workspace-1" stateId="state-1" />);
    expect(
      await screen.findByText(strings.workbench.workspace.sourceCandidateNotAccepted),
    ).toBeInTheDocument();
    expect(screen.getByText(strings.workbench.workspace.sourceNoAccepted)).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: strings.workbench.workspace.sourceAccept }),
    ).toBeDisabled();
    fireEvent.change(screen.getByLabelText(strings.workbench.workspace.sourceAcceptanceLabel), {
      target: { value: "Design reviewer" },
    });
    fireEvent.click(screen.getByRole("button", { name: strings.workbench.workspace.sourceAccept }));
    await waitFor(() =>
      expect(acceptWorkspaceSourceBaseline).toHaveBeenCalledWith(
        "workspace-1",
        "state-1",
        "candidate-1",
        "Design reviewer",
      ),
    );
    expect(
      screen.getByText(strings.workbench.workspace.sourceAcceptanceProvenance),
    ).toBeInTheDocument();
  });

  it("compares the accepted source anchor with the selected capture receipt", async () => {
    vi.mocked(listWorkspaceSourceBaselines).mockResolvedValue({
      candidates: [],
      accepted: accepted as never,
    });
    renderWithProviders(<VisualDriftPanel workspaceId="workspace-1" stateId="state-1" />);
    fireEvent.click(
      await screen.findByRole("button", { name: strings.workbench.workspace.sourceCompareButton }),
    );
    await waitFor(() =>
      expect(compareWorkspaceSourceBaseline).toHaveBeenCalledWith(
        "workspace-1",
        "state-1",
        "receipt-current",
      ),
    );
    expect(
      await screen.findByAltText(strings.workbench.workspace.sourceHeatmapAlt),
    ).toHaveAttribute("src", "/heatmap/comparison-1");
    expect(screen.getByText("20.00%")).toBeInTheDocument();
  });

  it("adds the selected reference asset only as an unaccepted candidate", async () => {
    renderWithProviders(<VisualDriftPanel workspaceId="workspace-1" stateId="state-1" />);
    fireEvent.click(
      await screen.findByRole("button", { name: strings.workbench.workspace.sourceAddCandidate }),
    );
    await waitFor(() =>
      expect(createWorkspaceSourceBaseline).toHaveBeenCalledWith(
        "workspace-1",
        "state-1",
        "asset-1",
      ),
    );
    expect(acceptWorkspaceSourceBaseline).not.toHaveBeenCalled();
  });

  it("shows the source image and capture recorded by the selected comparison", async () => {
    vi.mocked(listWorkspaceSourceBaselines).mockResolvedValue({
      candidates: [],
      accepted: accepted as never,
    });
    vi.mocked(listWorkspaceCaptureReceipts).mockResolvedValue([receipt, historicalReceipt]);
    vi.mocked(listWorkspaceAssets).mockResolvedValue([
      {
        id: "asset-1",
        workspaceId: "workspace-1",
        name: "current-source.png",
        mime: "image/png",
        bytes: 10,
        width: 100,
        height: 200,
        fidelity: "reference",
        addedAt: "",
        url: "/asset/current-source.png",
      },
      {
        id: "asset-history",
        workspaceId: "workspace-1",
        name: "historical-source.png",
        mime: "image/png",
        bytes: 10,
        width: 100,
        height: 200,
        fidelity: "reference",
        addedAt: "",
        url: "/asset/historical-source.png",
      },
    ]);
    vi.mocked(listWorkspaceSourceBaselineComparisons).mockResolvedValue([comparison as never]);
    renderWithProviders(<VisualDriftPanel workspaceId="workspace-1" stateId="state-1" />);
    expect(
      await screen.findByAltText(strings.workbench.workspace.sourceComparisonSourceImage),
    ).toHaveAttribute("src", "/asset/historical-source.png");
    expect(screen.getByAltText(/revision 3/i)).toHaveAttribute("src", "/capture/receipt-history");
  });
});
