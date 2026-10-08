import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { FindingStatus } from "@vrooli/proto-types/web-search/v1/findings/findings_pb";

vi.mock("../../api/clients", () => ({
  findingsClient: {
    listFindings: vi.fn(),
    pruneFindings: vi.fn(),
    addFinding: vi.fn(),
    editFinding: vi.fn(),
    supersedeFinding: vi.fn(),
    flagFinding: vi.fn(),
  },
  liveSearchClient: { search: vi.fn() },
}));

import { findingsClient } from "../../api/clients";
import { FindingsPanel } from "./FindingsPanel";

const finding = {
  id: "f1",
  claim: "Water boils at 100C at sea level",
  briefId: "",
  confidence: 0.9,
  status: FindingStatus.ACTIVE,
  query: "boiling point",
  supersededBy: "",
  disputeNote: "",
  source: 1,
  citations: [],
};

describe("FindingsPanel", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("lists findings from the client", async () => {
    vi.mocked(findingsClient.listFindings).mockResolvedValue({ findings: [finding] } as never);

    renderWithProviders(<FindingsPanel />);

    await waitFor(() => {
      expect(screen.getAllByTestId(selectors.findings.item)).toHaveLength(1);
    });
    expect(findingsClient.listFindings).toHaveBeenCalledWith({
      status: FindingStatus.UNSPECIFIED,
      includeArchived: false,
      limit: 100,
    });
  });

  it("filtering to active requests the ACTIVE status", async () => {
    vi.mocked(findingsClient.listFindings).mockResolvedValue({ findings: [] } as never);

    renderWithProviders(<FindingsPanel />);
    fireEvent.click(screen.getByText(strings.findings.filterActive));

    await waitFor(() => {
      expect(findingsClient.listFindings).toHaveBeenCalledWith({
        status: FindingStatus.ACTIVE,
        includeArchived: false,
        limit: 100,
      });
    });
  });

  it("the superseded filter forces include_archived", async () => {
    vi.mocked(findingsClient.listFindings).mockResolvedValue({ findings: [] } as never);

    renderWithProviders(<FindingsPanel />);
    fireEvent.click(screen.getByText(strings.findings.filterSuperseded));

    await waitFor(() => {
      expect(findingsClient.listFindings).toHaveBeenCalledWith({
        status: FindingStatus.SUPERSEDED,
        includeArchived: true,
        limit: 100,
      });
    });
  });

  it("shows the empty state when no findings match", async () => {
    vi.mocked(findingsClient.listFindings).mockResolvedValue({ findings: [] } as never);
    renderWithProviders(<FindingsPanel />);
    expect(await screen.findByTestId(selectors.findings.empty)).toBeInTheDocument();
  });

  it("dry-run prune calls with dry_run=true and reports the count", async () => {
    vi.mocked(findingsClient.listFindings).mockResolvedValue({ findings: [] } as never);
    vi.mocked(findingsClient.pruneFindings).mockResolvedValue({
      pruned: 2,
      findingIds: ["a", "b"],
    } as never);

    renderWithProviders(<FindingsPanel />);
    fireEvent.click(screen.getByTestId(selectors.findings.pruneDryRun));

    await waitFor(() => {
      expect(findingsClient.pruneFindings).toHaveBeenCalledWith({ dryRun: true });
    });
    expect(await screen.findByTestId(selectors.findings.pruneResult)).toBeInTheDocument();
  });

  it("renders an error state when listing fails", async () => {
    vi.mocked(findingsClient.listFindings).mockRejectedValue(new Error("boom"));
    renderWithProviders(<FindingsPanel />);
    expect(await screen.findByTestId(selectors.findings.error)).toBeInTheDocument();
  });
});
