import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";

import { interp, renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import en from "../../i18n/locales/en.json";
import { FindingStatus } from "@vrooli/proto-types/web-search/v1/findings/findings_pb";

vi.mock("../../api/clients", () => ({
  findingsClient: {
    listDisputes: vi.fn(),
    resolveDispute: vi.fn(),
  },
  liveSearchClient: { search: vi.fn() },
  researchClient: { runL3: vi.fn() },
}));

import { findingsClient, researchClient } from "../../api/clients";
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

  it("removes a resolved dispute from the open queue without a reload", async () => {
    vi.mocked(findingsClient.listDisputes)
      .mockResolvedValueOnce({ findings: [disputed] } as never)
      .mockResolvedValue({ findings: [] } as never);
    vi.mocked(findingsClient.resolveDispute).mockResolvedValue({ finding: disputed } as never);

    renderWithProviders(<DisputesPanel />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.disputes.resolveButton)).toBeInTheDocument();
    });
    fireEvent.click(screen.getByTestId(selectors.disputes.resolveButton));
    fireEvent.submit(screen.getByTestId(selectors.disputes.resolveForm));

    // Resolution invalidates the queue query; the refetch (now empty) drops
    // the entry and the empty state appears in the same mount.
    await waitFor(() => {
      expect(screen.getByTestId(selectors.disputes.empty)).toBeInTheDocument();
    });
    expect(screen.queryAllByTestId(selectors.disputes.item)).toHaveLength(0);
  });

  it("removes a dismissed dispute from the open queue via the keep resolution", async () => {
    vi.mocked(findingsClient.listDisputes)
      .mockResolvedValueOnce({ findings: [disputed] } as never)
      .mockResolvedValue({ findings: [] } as never);
    vi.mocked(findingsClient.resolveDispute).mockResolvedValue({ finding: disputed } as never);

    renderWithProviders(<DisputesPanel />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.disputes.dismissButton)).toBeInTheDocument();
    });
    fireEvent.click(screen.getByTestId(selectors.disputes.dismissButton));

    // Dismiss is modeled as resolution=keep: the server clears the dispute and
    // returns the finding to active, so the entry leaves the open queue on the
    // same invalidate/refetch path as resolve.
    await waitFor(() => {
      expect(findingsClient.resolveDispute).toHaveBeenCalledWith({
        id: "d1",
        resolution: "keep",
        replacement: "",
        reason: "dismissed from review queue",
      });
    });
    await waitFor(() => {
      expect(screen.getByTestId(selectors.disputes.empty)).toBeInTheDocument();
    });
    expect(screen.queryAllByTestId(selectors.disputes.item)).toHaveLength(0);
  });

  it("spawns an L3 re-research run for an entry and surfaces the run id", async () => {
    // Real English so the inline status line's interpolated run id is visible
    // (cimode renders the bare key path without interpolation).
    await setLocale("en");
    vi.mocked(findingsClient.listDisputes).mockResolvedValue({ findings: [disputed] } as never);
    vi.mocked(researchClient.runL3).mockResolvedValue({ runId: "run-7" } as never);

    renderWithProviders(<DisputesPanel />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.disputes.reresearchButton)).toBeInTheDocument();
    });
    fireEvent.click(screen.getByTestId(selectors.disputes.reresearchButton));

    await waitFor(() => {
      expect(researchClient.runL3).toHaveBeenCalledWith({ query: disputed.claim });
    });
    expect(screen.getByTestId(selectors.disputes.reresearchStatus)).toHaveTextContent(
      interp(en.disputes.reresearchStarted, { runId: "run-7" }),
    );
    // The entry stays in the queue — re-research gathers evidence, it does not resolve.
    expect(screen.getAllByTestId(selectors.disputes.item)).toHaveLength(1);
  });

  it("renders a 50-entry dispute queue within 500ms", async () => {
    const findings = Array.from({ length: 50 }, (_, i) => ({ ...disputed, id: `d${i}` }));
    vi.mocked(findingsClient.listDisputes).mockResolvedValue({ findings } as never);

    const start = performance.now();
    renderWithProviders(<DisputesPanel />);
    await waitFor(() => {
      expect(screen.getAllByTestId(selectors.disputes.item)).toHaveLength(50);
    });
    expect(performance.now() - start).toBeLessThan(500);
  });
});
