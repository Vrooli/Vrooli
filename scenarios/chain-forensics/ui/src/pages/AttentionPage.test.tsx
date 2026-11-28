import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";

vi.mock("../api/cases", () => ({
  casesClient: {
    listCases: vi.fn(),
    listAttentionDeliveries: vi.fn(),
    exportPrivatePacket: vi.fn(),
    deliverPrivatePacketToAttention: vi.fn(),
  },
}));

import { casesClient } from "../api/cases";
import { selectors } from "../consts/selectors";
import { renderWithProviders } from "../test-utils";
import { AttentionPage } from "./AttentionPage";

const packetJSON = new TextEncoder().encode(JSON.stringify({
  schema_version: "chain-forensics.packet/v1",
  case: {
    id: "private-rug-case",
    title: "Private rug case",
    summary: "Candidate path with explicit unresolved exchange and bridge joins.",
    private: true,
    edges: [{ id: "edge-1", grade: "hypothesis", rule: "candidate bridge correlation", caveats: ["Message identity is not matched."] }],
    gaps: ["No source-matched destination bridge message."],
    counterexamples: ["Shared exchange routing remains plausible."],
    sourceDisagreements: [],
  },
}));
const packetHash = "a".repeat(64);

describe("AttentionPage", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("shows a private local receipt with packet hash, evidence grade, and gaps", async () => {
    vi.mocked(casesClient.listCases).mockResolvedValue({ cases: [{ id: "private-rug-case", title: "Private rug case", private: true }] } as never);
    vi.mocked(casesClient.listAttentionDeliveries).mockResolvedValue({ deliveries: [{ caseId: "private-rug-case", packetSha256: packetHash, packetJson: packetJSON, deliveredAt: timestampFromDate(new Date("2026-11-21T12:00:00Z")) }] } as never);

    renderWithProviders(<AttentionPage />);

    expect(await screen.findByText("Private local packet")).toBeInTheDocument();
    expect(screen.getByText("hypothesis")).toBeInTheDocument();
    expect(screen.getByText("No source-matched destination bridge message.")).toBeInTheDocument();
    expect(screen.getByText("Shared exchange routing remains plausible.")).toBeInTheDocument();
    expect(screen.getByText(`Packet SHA-256 · ${packetHash}`)).toBeInTheDocument();
    expect(screen.getByText(/does not publish, notify, or contact anyone/i)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /publish|submit tip|contact/i })).not.toBeInTheDocument();
  });

  it("exports then delivers a selected private case and refreshes the local receipt list", async () => {
    vi.mocked(casesClient.listCases).mockResolvedValue({ cases: [{ id: "private-rug-case", title: "Private rug case", private: true }] } as never);
    vi.mocked(casesClient.listAttentionDeliveries).mockResolvedValue({ deliveries: [] } as never);
    vi.mocked(casesClient.exportPrivatePacket).mockResolvedValue({ sha256: packetHash } as never);
    vi.mocked(casesClient.deliverPrivatePacketToAttention).mockResolvedValue({ delivery: { caseId: "private-rug-case", packetSha256: packetHash, packetJson: packetJSON } } as never);
    const user = userEvent.setup();

    renderWithProviders(<AttentionPage />);

    await user.selectOptions(await screen.findByLabelText("Private case"), "private-rug-case");
    await user.click(screen.getByTestId(selectors.pages.attentionDeliverButton));

    expect(casesClient.exportPrivatePacket).toHaveBeenCalledWith({ id: "private-rug-case" });
    expect(casesClient.deliverPrivatePacketToAttention).toHaveBeenCalledWith({ caseId: "private-rug-case", packetSha256: packetHash });
    expect(await screen.findByText("Deliver to local attention")).toBeInTheDocument();
  });
});
