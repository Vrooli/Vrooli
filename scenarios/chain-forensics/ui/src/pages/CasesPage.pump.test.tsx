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

describe("CasesPage Pump instruction evidence", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders decoded launch facts with source/version and leaves unknown variants unresolved", async () => {
    const signature = "4orDLftSdVP3Y1AmygWzTxkEZgTU9M5iSfJskmbp2H8JkvXFgdhYtbfup9iCQj2WajGbheNdTGftegEf782Jwxwk";
    const version = "pump-public-docs/idl/pump.json@639b87fb45c3adeffa6f7c4a1bf0cb85e392bf22#create_v2";
    const unresolvedReason = "Pump discriminator is not in this decoder version";
    vi.mocked(casesClient.listCases).mockResolvedValue({ cases: [{ id: "pump-create-private", title: "Private Pump launch evidence", private: true, sourceCount: 1, occurrenceCount: 0, edgeCount: 0, updatedAt: undefined }] } as never);
    vi.mocked(casesClient.getCase).mockResolvedValue({ case: {
      id: "pump-create-private",
      title: "Private Pump launch evidence",
      summary: "One finalized transaction with an exact IDL match and one unresolved Pump instruction.",
      private: true,
      sources: [{ id: "solana-transaction-source", name: "Solana finalized transaction", url: "https://api.mainnet-beta.solana.com", retrievedAt: "2026-11-14T00:00:00Z", sha256: "9970f16e3e88579f23de73b6aad6780e480bce9288577e2467300d0e50894da2" }],
      claims: [],
      occurrences: [],
      edges: [],
      gaps: [],
      counterexamples: [],
      sourceDisagreements: [],
      walletSnapshots: [],
      pumpInstructions: [{
        id: `solana:${signature}:pump:2`, transactionId: signature, slot: 451779205n, instructionIndex: 2,
        programId: "6EF8rrecthR5Dkzon8Nwu78hRvfCKubJ14M5uBEwF6P", sourceId: "solana-transaction-source",
        kind: "create_v2", decoderVersion: version, mint: "9aqmJjCnnMQv42TXLk921ceUkN35nea2QP969n1caqjj",
        user: "AV7PjXHL5JXZ1YoYRoN9Dsstg1x2UciBupMCXcJP8gUz", creator: "AV7PjXHL5JXZ1YoYRoN9Dsstg1x2UciBupMCXcJP8gUz",
        name: "Super Intelligence", symbol: "SI", metadataUri: "https://metadata.example/captured.json", mayhemMode: false,
        cashbackEnabled: false, creatorFeeBps: 0n, holderReward: false,
      }],
      unresolvedPumpInstructions: [{
        id: `solana:${signature}:pump:4`, transactionId: signature, slot: 451779205n, instructionIndex: 4,
        programId: "6EF8rrecthR5Dkzon8Nwu78hRvfCKubJ14M5uBEwF6P", sourceId: "solana-transaction-source",
        discriminator: "c2ab1c46684d5b2f", reason: unresolvedReason,
      }],
    } } as never);
    vi.mocked(casesClient.listAnalystLabels).mockResolvedValue({ labels: [] } as never);

    const user = userEvent.setup();
    renderWithProviders(<CasesPage />);
    await user.click(await screen.findByRole("button", { name: /Private Pump launch evidence/ }));

    expect(await screen.findByTestId("pump-instruction")).toHaveTextContent("Super Intelligence (SI)");
    expect(screen.getAllByText((_, node) => node?.textContent?.includes(version) === true).length).toBeGreaterThan(0);
    expect(screen.getByText("Mayhem mode: false · cashback: false · creator fee bps: 0 · holder reward: false")).toBeInTheDocument();
    expect(screen.getByTestId("unresolved-pump-instruction")).toHaveTextContent("c2ab1c46684d5b2f");
    expect(screen.getByText(unresolvedReason)).toBeInTheDocument();
  });
});
