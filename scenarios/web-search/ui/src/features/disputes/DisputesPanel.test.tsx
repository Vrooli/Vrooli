import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { FindingStatus } from "@vrooli/proto-types/web-search/v1/findings/findings_pb";

vi.mock("../../api/clients", () => ({
  findingsClient: {
    listDisputes: vi.fn(),
    resolveDispute: vi.fn(),
  },
  liveSearchClient: { search: vi.fn() },
}));

import { findingsClient } from "../../api/clients";
import { DisputesPanel } from "./DisputesPanel";

const disputed = {
  id: "d1",
  claim: "The capital of Australia is Sydney",
  briefId: "",
  confidence: 0.4,
  status: FindingStatus.DISPUTED,
  query: "australia capital",
  supersededBy: "",
  disputeNote: "Other sources say Canberra",
  source: 1,
  citations: [],
};

describe("DisputesPanel", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("lists disputed findings from the queue", async () => {
    vi.mocked(findingsClient.listDisputes).mockResolvedValue({ findings: [disputed] } as never);

    renderWithProviders(<DisputesPanel />);

    await waitFor(() => {
      expect(screen.getAllByTestId(selectors.disputes.item)).toHaveLength(1);
    });
    expect(findingsClient.listDisputes).toHaveBeenCalledWith({ limit: 100 });
    expect(screen.getByTestId(selectors.disputes.note)).toBeInTheDocument();
  });

  it("shows the empty state when there are no disputes", async () => {
    vi.mocked(findingsClient.listDisputes).mockResolvedValue({ findings: [] } as never);

    renderWithProviders(<DisputesPanel />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.disputes.empty)).toBeInTheDocument();
    });
  });

  it("resolves a dispute with the keep resolution", async () => {
    vi.mocked(findingsClient.listDisputes).mockResolvedValue({ findings: [disputed] } as never);
    vi.mocked(findingsClient.resolveDispute).mockResolvedValue({ finding: disputed } as never);

    renderWithProviders(<DisputesPanel />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.disputes.resolveButton)).toBeInTheDocument();
    });
    fireEvent.click(screen.getByTestId(selectors.disputes.resolveButton));
    fireEvent.submit(screen.getByTestId(selectors.disputes.resolveForm));

    await waitFor(() => {
      expect(findingsClient.resolveDispute).toHaveBeenCalledWith({
        id: "d1",
        resolution: "keep",
        replacement: "",
        reason: "",
      });
    });
  });
});
