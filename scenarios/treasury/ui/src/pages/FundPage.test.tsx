import { create } from "@bufbuild/protobuf";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  FundingIntentSchema,
  WithdrawalQuoteSchema,
  WithdrawalSchema,
} from "@vrooli/proto-types/treasury/v1/funding/wallet_ops_pb";

import en from "../i18n/locales/en.json";
import { setLocale } from "../i18n";
import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { FundPage } from "./FundPage";

const paymentUri = "ethereum:8453/transfer?address=0x0000000000000000000000000000000000000001&uint256=2500000";
const intent = create(FundingIntentSchema, {
  id: "intent-1",
  budgetId: "research-budget",
  atomicAmount: "2500000",
  paymentUri,
  state: "awaiting_transfer",
});
const acceptedIntent = create(FundingIntentSchema, {
  ...intent,
  state: "accepted",
  receiptDeliveryId: "receipt-delivery-1",
});
const quote = create(WithdrawalQuoteSchema, {
  id: "quote-1",
  budgetId: "research-budget",
  destination: "0x0000000000000000000000000000000000000002",
  atomicAmount: "1000000",
  network: "eip155:8453",
  maxFeeWei: "1000000000",
  valueMinor: 100n,
  feeValuationMinor: 5n,
  currency: "USD",
  estimatedSeconds: 15,
  expiresAt: { seconds: 1_800_000_000n },
  executionMode: "local-fork",
  mandateId: "mandate-1",
});
const withdrawal = create(WithdrawalSchema, {
  id: "withdrawal-1",
  quoteId: quote.id,
  state: "settled",
  transactionHash: "0xabc123",
});

const mocks = vi.hoisted(() => ({
  createFundingIntent: vi.fn(),
  confirmFundingTransfer: vi.fn(),
  executeWithdrawal: vi.fn(),
  listFundingIntents: vi.fn(),
  previewWithdrawal: vi.fn(),
}));

vi.mock("../api/funding", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../api/funding")>()),
  ...mocks,
}));

beforeEach(async () => {
  await setLocale("en");
  mocks.createFundingIntent.mockResolvedValue(intent);
  mocks.listFundingIntents.mockResolvedValue([]);
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
  vi.restoreAllMocks();
});

describe("Fund page", () => {
  it("renders the exact server payment URI as an accessible scannable QR", async () => {
    const user = userEvent.setup();
    renderWithProviders(<FundPage />);

    await user.type(screen.getByTestId(selectors.funding.operatorToken), "operator-secret");
    await user.type(screen.getByTestId(selectors.funding.budgetId), "research-budget");
    await user.type(screen.getByTestId(selectors.funding.atomicAmount), "2500000");
    await user.type(screen.getByTestId(selectors.funding.declaredValue), "2.50");
    await user.type(screen.getByTestId(selectors.funding.ledgerBookId), "book-1");
    await user.type(screen.getByTestId(selectors.funding.ledgerAccountId), "account-1");
    await user.click(screen.getByTestId(selectors.funding.createIntentButton));

    expect(mocks.createFundingIntent).toHaveBeenCalledWith("operator-secret", {
      budgetId: "research-budget",
      atomicAmount: "2500000",
      valueMinor: 250n,
      ledgerBookId: "book-1",
      ledgerAccountId: "account-1",
    });
    expect(await screen.findByRole("img", { name: en.fund.paymentQrLabel })).toBeVisible();
    expect(screen.getByTestId(selectors.funding.paymentQr)).toBeVisible();
    expect(screen.getByTestId(selectors.funding.paymentUri)).toHaveValue(paymentUri);
    expect(screen.getByRole("link", { name: en.fund.openWallet })).toHaveAttribute("href", paymentUri);
  });

  it("refreshes recent intents, selects one, and copies its exact wallet URI", async () => {
    const user = userEvent.setup();
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", { configurable: true, value: { writeText } });
    mocks.listFundingIntents.mockResolvedValue([intent]);
    renderWithProviders(<FundPage />);

    await user.type(screen.getByTestId(selectors.funding.operatorToken), "operator-secret");
    await user.type(screen.getByTestId(selectors.funding.budgetId), "research-budget");
    await user.click(screen.getByTestId(selectors.funding.refreshButton));
    await user.click(await screen.findByRole("button", { name: /intent-1/ }));
    await user.click(screen.getByTestId(selectors.funding.copyLinkButton));

    expect(mocks.listFundingIntents).toHaveBeenCalledWith("operator-secret", "research-budget");
    expect(writeText).toHaveBeenCalledWith(paymentUri);
    expect(await screen.findByRole("status")).toHaveTextContent(en.fund.copiedNotice);
  });

  it("confirms the entered receipt hash and announces accepted allocation", async () => {
    const user = userEvent.setup();
    mocks.createFundingIntent.mockResolvedValue(intent);
    mocks.confirmFundingTransfer.mockResolvedValue(acceptedIntent);
    renderWithProviders(<FundPage />);

    await user.type(screen.getByTestId(selectors.funding.operatorToken), "operator-secret");
    await user.type(screen.getByTestId(selectors.funding.budgetId), "research-budget");
    await user.type(screen.getByTestId(selectors.funding.atomicAmount), "2500000");
    await user.type(screen.getByTestId(selectors.funding.declaredValue), "2.50");
    await user.type(screen.getByTestId(selectors.funding.ledgerBookId), "book-1");
    await user.type(screen.getByTestId(selectors.funding.ledgerAccountId), "account-1");
    await user.click(screen.getByTestId(selectors.funding.createIntentButton));
    await user.type(screen.getByLabelText(en.fund.transactionHash), "0xlocal-receipt");
    await user.click(screen.getByRole("button", { name: en.fund.confirmAllocate }));

    expect(mocks.confirmFundingTransfer).toHaveBeenCalledWith("operator-secret", intent.id, "0xlocal-receipt");
    expect(await screen.findByRole("status")).toHaveTextContent("receipt-delivery-1");
  });

  it("previews a withdrawal, then executes only the explicit local-fork quote", async () => {
    const user = userEvent.setup();
    mocks.previewWithdrawal.mockResolvedValue(quote);
    mocks.executeWithdrawal.mockResolvedValue(withdrawal);
    renderWithProviders(<FundPage />);

    await user.type(screen.getByTestId(selectors.funding.operatorToken), "operator-secret");
    await user.type(screen.getByTestId(selectors.funding.agentToken), "agent-token");
    await user.type(screen.getByTestId(selectors.funding.budgetId), "research-budget");
    await user.type(screen.getByTestId(selectors.funding.withdrawalDestination), quote.destination);
    await user.type(screen.getByTestId(selectors.funding.mandateId), quote.mandateId);
    await user.type(screen.getAllByLabelText(en.fund.atomicAmount)[1]!, quote.atomicAmount);
    await user.type(screen.getAllByLabelText(en.fund.declaredValue)[1]!, "1.00");
    await user.clear(screen.getByLabelText(en.fund.maxFee));
    await user.type(screen.getByLabelText(en.fund.maxFee), quote.maxFeeWei);
    await user.clear(screen.getByLabelText(en.fund.feeValue));
    await user.type(screen.getByLabelText(en.fund.feeValue), "0.05");
    await user.click(screen.getByTestId(selectors.funding.withdrawalPreviewButton));

    expect(mocks.previewWithdrawal).toHaveBeenCalledWith("operator-secret", {
      budgetId: "research-budget",
      mandateId: "mandate-1",
      destination: quote.destination,
      atomicAmount: "1000000",
      valueMinor: 100n,
      maxFeeWei: "1000000000",
      feeValuationMinor: 5n,
    });
    await user.click(await screen.findByRole("button", { name: en.fund.executeFork }));
    expect(mocks.executeWithdrawal).toHaveBeenCalledWith("operator-secret", "agent-token", quote.id, expect.any(String));
    expect(await screen.findByText(en.fund.withdrawNotice.replace("{{state}}", "settled"))).toBeVisible();
  });

  it("keeps execution disabled for non-fork quotes and reports preview failures", async () => {
    const user = userEvent.setup();
    mocks.previewWithdrawal.mockResolvedValue(create(WithdrawalQuoteSchema, { ...quote, executionMode: "mainnet" }));
    renderWithProviders(<FundPage />);

    await user.type(screen.getByTestId(selectors.funding.operatorToken), "operator-secret");
    await user.type(screen.getByTestId(selectors.funding.budgetId), "research-budget");
    await user.type(screen.getByTestId(selectors.funding.withdrawalDestination), quote.destination);
    await user.type(screen.getByTestId(selectors.funding.mandateId), quote.mandateId);
    await user.type(screen.getAllByLabelText(en.fund.atomicAmount)[1]!, quote.atomicAmount);
    await user.type(screen.getAllByLabelText(en.fund.declaredValue)[1]!, "1.00");
    await user.click(screen.getByTestId(selectors.funding.withdrawalPreviewButton));
    expect(await screen.findByRole("button", { name: en.fund.executeDisabled })).toBeDisabled();

    mocks.previewWithdrawal.mockRejectedValueOnce(new Error("approval expired"));
    await user.click(screen.getByTestId(selectors.funding.withdrawalPreviewButton));
    expect(await screen.findByRole("alert")).toHaveTextContent("approval expired");
    expect(screen.queryByText(/Quote quote-1/)).not.toBeInTheDocument();
  });
});
