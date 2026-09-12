import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { EventKind, RedemptionState } from "@vrooli/proto-types/token-economy/v1/access/access_pb";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { strings } from "../../consts/strings";
import { HolderHistoryPage } from "./HolderHistoryPage";
import { HolderHomePage } from "./HolderHomePage";
import { HolderRewardsPage } from "./HolderRewardsPage";

const api = vi.hoisted(() => ({
  viewEconomy: vi.fn(),
  browseCatalog: vi.fn(),
  requestRedemption: vi.fn(),
}));

const earningReason = "Kitchen completed";

vi.mock("../../api/tokenEconomy", () => ({
  holderClient: api,
  nextIdempotencyKey: () => "redeem-test-key",
}));

const economy = {
  holder: { id: "holder-1", displayName: "Sam", authenticatorSubject: "auth:sam" },
  balances: [{ tokenTypeId: "chores", amount: 10n }],
  events: [{ id: "event-1", tokenTypeId: "chores", amount: 10n, kind: EventKind.CREDIT, reason: earningReason, causeReference: "grant:grant-1" }],
  redemptions: [{ id: "redemption-1", tokenTypeId: "chores", catalogEntryId: "reward-1", holderId: "holder-1", grantId: "grant-1", amount: 4n, state: RedemptionState.PENDING_APPROVAL }],
};

describe("holder view", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("explains the balance with history and pending state accessibly", async () => {
    api.viewEconomy.mockResolvedValue(economy);
    const { container } = renderWithProviders(<HolderHomePage />);

    await waitFor(() => expect(screen.getByTestId("holder-balance-chores")).toHaveTextContent("10"));
    expect(screen.getByText(earningReason)).toBeInTheDocument();
    expect(screen.getByText(strings.holderView.rewardRequest)).toBeInTheDocument();
    await expectNoA11yViolations(container);
  });

  it("renders the append-only history as an ordered, labelled list", async () => {
    api.viewEconomy.mockResolvedValue(economy);
    const { container } = renderWithProviders(<HolderHistoryPage />);

    await waitFor(() => expect(screen.getByRole("list", { name: strings.holderView.historyListLabel })).toBeInTheDocument());
    expect(screen.getByText(earningReason)).toBeInTheDocument();
    await expectNoA11yViolations(container);
  });

  it("requests a reward through the generated holder client and shows a non-animated success signal", async () => {
    api.viewEconomy.mockResolvedValue(economy);
    api.browseCatalog.mockResolvedValue({ entries: [{ id: "reward-1", tokenTypeId: "chores", title: "Choose dinner", description: "Pick tonight's meal", costAmount: 4n }] });
    api.requestRedemption.mockResolvedValue({ redemption: { id: "redemption-2" } });
    const user = userEvent.setup();
    const { container } = renderWithProviders(<HolderRewardsPage />);

    await user.click(await screen.findByTestId("holder-redeem-reward-1"));
    await waitFor(() => expect(api.requestRedemption).toHaveBeenCalledWith(expect.objectContaining({ idempotencyKey: "redeem-test-key", redemption: { catalogEntryId: "reward-1", grantId: "grant-1" } })));
    expect(await screen.findByRole("status")).toHaveTextContent("holderView.redemptionRecorded");
    await expectNoA11yViolations(container);
  });
});
