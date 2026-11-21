import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { FindingStatus } from "@vrooli/proto-types/web-search/v1/findings/findings_pb";

vi.mock("../../api/clients", () => ({
  findingsClient: {
    listEffectiveness: vi.fn(),
  },
  liveSearchClient: { search: vi.fn() },
}));

import { findingsClient } from "../../api/clients";
import { EffectivenessPanel } from "./EffectivenessPanel";

function finding(id: string, claim: string) {
  return {
    id,
    claim,
    briefId: "",
    confidence: 0.9,
    status: FindingStatus.ACTIVE,
    query: "",
    supersededBy: "",
    disputeNote: "",
    source: 1,
    citations: [],
  };
}

describe("EffectivenessPanel", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders surfaced/never-surfaced rows with the blended score", async () => {
    vi.mocked(findingsClient.listEffectiveness).mockResolvedValue({
      items: [
        {
          finding: finding("f1", "surfaced claim"),
          surfacedCount: 3,
          usedCount: 1,
          lastSurfacedAt: timestampFromDate(new Date("2026-06-01T00:00:00Z")),
          effectiveConfidence: 0.9,
          usageFactor: 1,
          effectiveScore: 0.9,
        },
        {
          finding: finding("f2", "never surfaced claim"),
          surfacedCount: 0,
          usedCount: 0,
          lastSurfacedAt: undefined,
          effectiveConfidence: 0.6,
          usageFactor: 0.5,
          effectiveScore: 0.3,
        },
      ],
    } as never);

    renderWithProviders(<EffectivenessPanel />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.findings.effectivenessList)).toBeInTheDocument();
    });
    const rows = screen.getAllByTestId(selectors.findings.effectivenessItem);
    expect(rows).toHaveLength(2);
    // Rows carry their finding id as a data attribute and the surfaced row sorts
    // first (the API returns highest-effective-score first).
    expect(rows[0]).toHaveAttribute("data-finding", "f1");
    expect(rows[1]).toHaveAttribute("data-finding", "f2");
    // The claim text (raw data, not i18n-interpolated) renders in each row.
    expect(rows[0]).toHaveTextContent("surfaced claim");
    expect(rows[1]).toHaveTextContent("never surfaced claim");
  });

  it("shows the empty state when there are no findings", async () => {
    vi.mocked(findingsClient.listEffectiveness).mockResolvedValue({ items: [] } as never);
    renderWithProviders(<EffectivenessPanel />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.findings.effectivenessEmpty)).toBeInTheDocument();
    });
  });
});
