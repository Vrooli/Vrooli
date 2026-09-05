import { afterEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  listFacets: vi.fn(),
  setPin: vi.fn(),
  assignFacet: vi.fn(),
  listPinProposals: vi.fn(),
  resolvePinProposal: vi.fn(),
  getFrontier: vi.fn(),
}));

vi.mock("@connectrpc/connect", () => ({
  createClient: vi.fn(() => ({
    listFacets: mocks.listFacets,
    setPin: mocks.setPin,
    assignFacet: mocks.assignFacet,
    listPinProposals: mocks.listPinProposals,
    resolvePinProposal: mocks.resolvePinProposal,
    getFrontier: mocks.getFrontier,
  })),
}));

import { assignFacet, getFrontier, listFacets, listPinProposals, resolvePinProposal, setPin } from "./operator";

describe("api/operator", () => {
  afterEach(() => vi.clearAllMocks());

  it("maps operator RPC requests and response collections", async () => {
    mocks.listFacets.mockResolvedValueOnce({ facets: ["facet"] });
    mocks.setPin.mockResolvedValueOnce({});
    mocks.assignFacet.mockResolvedValueOnce({});
    mocks.listPinProposals.mockResolvedValueOnce({ proposals: ["proposal"] });
    mocks.resolvePinProposal.mockResolvedValueOnce({});
    mocks.getFrontier.mockResolvedValueOnce({ nodes: ["node"], eligibleCount: 1, target: 2 });

    await expect(listFacets()).resolves.toEqual(["facet"]);
    await setPin("entry", true);
    await assignFacet("entry", "episode");
    await expect(listPinProposals()).resolves.toEqual(["proposal"]);
    await resolvePinProposal("proposal", false);
    await expect(getFrontier()).resolves.toEqual({ nodes: ["node"], eligibleCount: 1, target: 2 });

    expect(mocks.setPin).toHaveBeenCalledWith({ entryId: "entry", pinned: true });
    expect(mocks.assignFacet).toHaveBeenCalledWith({ entryId: "entry", facetId: "episode" });
    expect(mocks.resolvePinProposal).toHaveBeenCalledWith({ proposalId: "proposal", accept: false });
  });
});
