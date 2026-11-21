import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

vi.mock("../api/cases", () => ({
  casesClient: {
    listCases: vi.fn(),
    getCase: vi.fn(),
    listAnalystLabels: vi.fn(),
    exportPrivatePacket: vi.fn(),
    importCase: vi.fn(),
  },
}));

import { casesClient } from "../api/cases";
import { renderWithProviders } from "../test-utils";
import { CasesPage } from "./CasesPage";

describe("CasesPage Wormhole evidence rendering", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the retained Base-to-Arbitrum correlation and its limits", async () => {
    const from = "base:0x5e18f88f3a3b1589a4c517ee05e54b90cd3e86ac274a8c22ee659bfeb4007198:erc20:41";
    const to = "arbitrum:0x5f594142eed3a77387f81057e824afa534b14a7063e314b8db6ce5470e334085:erc20:2";
    const rule = "bridge.wormhole.token_bridge.transfer_redeemed_emitter_chain_address_sequence";
    const caveat = "Matches the Wormhole message tuple; token-unit continuity is not proved.";
    vi.mocked(casesClient.listCases).mockResolvedValue({ cases: [{ id: "wormhole-live-pair-20240911", title: "Wormhole evidence: Base to Arbitrum", private: true, sourceCount: 5, occurrenceCount: 2, edgeCount: 1, updatedAt: undefined }] } as never);
    vi.mocked(casesClient.getCase).mockResolvedValue({ case: {
      id: "wormhole-live-pair-20240911",
      title: "Wormhole evidence: Base to Arbitrum",
      summary: "Private local evidence from retained successful public chain receipts.",
      private: true,
      sources: [{ id: "base-source", name: "Base receipt", url: "https://mainnet.base.org", retrievedAt: "2026-11-14T06:09:02Z", sha256: "648d15de34fa23545e852bc5969e55c73f61609cdac023232c8719961382cf6e" }],
      claims: [],
      occurrences: [
        { id: from, chain: 3, transactionId: "0x5e18f88f3a3b1589a4c517ee05e54b90cd3e86ac274a8c22ee659bfeb4007198", eventIndex: 41, from: "0xsender", to: "0x8d2de8d2f73f1f4cab472ac9a881c9b123c79627", asset: "erc20:0xff0c62a4979400841efaa6faadb07ac7d5c98b27", atomicQuantity: "75966", scale: 0, kind: 1, sourceId: "base-source" },
        { id: to, chain: 5, transactionId: "0x5f594142eed3a77387f81057e824afa534b14a7063e314b8db6ce5470e334085", eventIndex: 2, from: "0xbridge", to: "0xe70946692e2e56ae47bfae2d93d31bd60952b090", asset: "erc20:0xe4728f3e48e94c6da2b53610e677cc241dafb134", atomicQuantity: "75966", scale: 0, kind: 1, sourceId: "arbitrum-destination" },
      ],
      edges: [{ id: "bridge:wormhole:token-bridge:30:emitter:86175", fromOccurrence: from, toOccurrence: to, grade: 2, rule, evidenceIds: [from, to], caveats: [caveat] }],
      gaps: ["No identity, ownership, intent, or exchange custody claim is made."],
      counterexamples: [],
      sourceDisagreements: [],
      createdAt: undefined,
      updatedAt: undefined,
      walletSnapshots: [],
      pumpInstructions: [],
      unresolvedPumpInstructions: [],
    } } as never);
    vi.mocked(casesClient.listAnalystLabels).mockResolvedValue({ labels: [] } as never);

    const user = userEvent.setup();
    renderWithProviders(<CasesPage />);
    await user.click(await screen.findByRole("button", { name: /Wormhole evidence: Base to Arbitrum/ }));

    expect(await screen.findByText(`2: ${rule}`)).toBeInTheDocument();
    expect(screen.getAllByText((_, node) => node?.textContent?.includes("0x5e18f88f3a3b1589a4c517ee05e54b90cd3e86ac274a8c22ee659bfeb4007198") === true && node.textContent.includes("0x5f594142eed3a77387f81057e824afa534b14a7063e314b8db6ce5470e334085"))).not.toHaveLength(0);
    expect(screen.getByText(`Caveat: ${caveat}`)).toBeInTheDocument();
    expect(screen.getByText("No identity, ownership, intent, or exchange custody claim is made.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Export private packet" })).toBeEnabled();
  });
});
