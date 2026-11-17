import { create } from "@bufbuild/protobuf";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { AgentBudgetHeadroomSchema } from "@vrooli/proto-types/treasury/v1/authorization/authorization_pb";
import { LedgerDestinationSchema } from "@vrooli/proto-types/treasury/v1/ledger/ledger_pb";
import {
  BudgetWalletSchema,
  FundingIntentSchema,
  WithdrawalQuoteSchema,
  WithdrawalSchema,
} from "@vrooli/proto-types/treasury/v1/funding/wallet_ops_pb";

import en from "../i18n/locales/en.json";
import { setLocale } from "../i18n";
import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { FundPage } from "./FundPage";

const paymentUri = "ethereum:0x833589fcd6edb6e08f4c7c32d4f71b54bda02913@8453/transfer?address=0x0000000000000000000000000000000000000001&uint256=2500000";
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
  notificationState: "sent",
  receiptDeliveryId: "receipt-delivery-1",
});
const deliveringIntent = create(FundingIntentSchema, {
  ...intent,
  state: "delivering",
  transactionHash: "0xlocal-receipt",
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
  getOperatorBudgetHeadroom: vi.fn(),
  getLedgerFundingDestination: vi.fn(),
  previewWithdrawal: vi.fn(),
  getOperatorSession: vi.fn(),
  refreshOperatorSession: vi.fn(),
  signInOperator: vi.fn(),
  signOutOperator: vi.fn(),
  basePay: vi.fn(),
  basePayStatus: vi.fn(),
}));

vi.mock("@base-org/account", () => ({ pay: mocks.basePay, getPaymentStatus: mocks.basePayStatus }));

vi.mock("../api/funding", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../api/funding")>()),
  ...mocks,
}));

vi.mock("../api/operatorSession", () => ({
  getOperatorSession: mocks.getOperatorSession,
  refreshOperatorSession: mocks.refreshOperatorSession,
  signInOperator: mocks.signInOperator,
  signOutOperator: mocks.signOutOperator,
  getOperatorBudgetHeadroom: mocks.getOperatorBudgetHeadroom,
}));

beforeEach(async () => {
  await setLocale("en");
  mocks.createFundingIntent.mockResolvedValue(intent);
  mocks.listFundingIntents.mockResolvedValue([]);
  mocks.getOperatorSession.mockResolvedValue({ authenticated: true, subject: "operator@example.test" });
  mocks.refreshOperatorSession.mockResolvedValue({ authenticated: true, subject: "operator@example.test" });
  mocks.signInOperator.mockResolvedValue({ authenticated: true });
  mocks.signOutOperator.mockResolvedValue(undefined);
  mocks.getOperatorBudgetHeadroom.mockResolvedValue(create(AgentBudgetHeadroomSchema, { availableMinor: 100n, fundingCreditedMinor: 0n, fundingRemainingMinor: 0n, currency: "USD" }));
  mocks.getLedgerFundingDestination.mockResolvedValue(create(LedgerDestinationSchema, {
    configured: true,
    bookId: "secret-book-uuid",
    accountId: "secret-account-uuid",
    bookName: "Research",
    accountName: "Base deposits",
    currency: "USD",
  }));
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
  vi.restoreAllMocks();
});

describe("Fund page", () => {
  it("gives an unfunded operator a Base purchase and transfer path while protecting the fork-only address", () => {
    renderWithProviders(<FundPage />);

    expect(screen.getByText(en.fund.onrampTitle)).toBeVisible();
    expect(screen.getByText(en.fund.onrampIntro)).toHaveTextContent(/no Coinbase fee.*network or payment-provider costs may apply/i);
    expect(screen.getByText(en.fund.onrampBuy)).toBeVisible();
    expect(screen.getByText(en.fund.onrampTransfer)).toBeVisible();
    expect(screen.getByText(en.fund.onrampGas)).toBeVisible();
    expect(screen.getByText(en.fund.onrampSafety)).toBeVisible();
    expect(screen.getByRole("link", { name: en.fund.onrampBuyHelp })).toHaveAttribute("href", "https://help.coinbase.com/en/wallet/managing-account/payment-method");
    expect(screen.getByRole("link", { name: en.fund.onrampUSDCFeesHelp })).toHaveAttribute("href", "https://help.coinbase.com/en/wallet/managing-account/usdc-coinbase-wallet");
    expect(screen.getByRole("link", { name: en.fund.onrampTransferHelp })).toHaveAttribute("href", "https://help.coinbase.com/en/wallet/sending-and-receiving/how-do-i-move-assets-between-my-coinbase-wallet-and-my-coinbase");
  });


  it("signs in with an MFA challenge through the cookie session and never exposes bearer token fields", async () => {
    const user = userEvent.setup();
    mocks.getOperatorSession.mockResolvedValueOnce({ authenticated: false }).mockResolvedValueOnce({ authenticated: true, subject: "operator-1" });
    mocks.refreshOperatorSession.mockRejectedValueOnce(new Error("no session"));
    mocks.signInOperator
      .mockResolvedValueOnce({ authenticated: false, mfa_required: true, mfa_challenge: "challenge-1" })
      .mockResolvedValueOnce({ authenticated: true });
    renderWithProviders(<FundPage />);

    await user.type(await screen.findByTestId(selectors.funding.sessionEmail), "operator@example.test");
    await user.type(screen.getByTestId(selectors.funding.sessionPassword), "correct horse battery staple");
    await user.click(screen.getByTestId(selectors.funding.signInButton));
    expect(await screen.findByTestId(selectors.funding.sessionMfaCode)).toBeVisible();
    expect(screen.queryByTestId(selectors.funding.sessionPassword)).not.toBeInTheDocument();
    await user.type(screen.getByTestId(selectors.funding.sessionMfaCode), "123456");
    await user.click(screen.getByTestId(selectors.funding.signInButton));

    expect(await screen.findByTestId(selectors.funding.sessionIdentity)).toHaveTextContent("operator-1");
    expect(mocks.signInOperator).toHaveBeenNthCalledWith(1, expect.objectContaining({ email: "operator@example.test", password: "correct horse battery staple" }));
    expect(mocks.signInOperator).toHaveBeenNthCalledWith(2, expect.objectContaining({ email: "operator@example.test", password: "", mfaChallenge: "challenge-1", totpCode: "123456" }));
    expect(screen.queryByLabelText(/token/i)).not.toBeInTheDocument();
    expect(screen.queryByTestId("funding-operator-token")).not.toBeInTheDocument();
    expect(screen.queryByTestId("funding-agent-token")).not.toBeInTheDocument();
  });

  it("restores a refreshed cookie session and resolves the verified subject", async () => {
    mocks.getOperatorSession.mockResolvedValueOnce({ authenticated: false }).mockResolvedValueOnce({ authenticated: true, subject: "restored-operator" });
    mocks.refreshOperatorSession.mockResolvedValueOnce({ authenticated: true });
    renderWithProviders(<FundPage />);

    expect(await screen.findByTestId(selectors.funding.sessionIdentity)).toHaveTextContent("restored-operator");
    expect(mocks.refreshOperatorSession).toHaveBeenCalledOnce();
    expect(mocks.getOperatorSession).toHaveBeenCalledTimes(2);
  });

  it("clears operator-only funding state when the HttpOnly session signs out", async () => {
    const user = userEvent.setup();
    renderWithProviders(<FundPage />);
    await screen.findByTestId(selectors.funding.sessionIdentity);

    await user.click(screen.getByTestId(selectors.funding.signOutButton));

    expect(mocks.signOutOperator).toHaveBeenCalledOnce();
    expect(await screen.findByTestId(selectors.funding.sessionEmail)).toBeVisible();
    expect(screen.queryByTestId(selectors.funding.sessionIdentity)).not.toBeInTheDocument();
  });

  it("keeps sign-in available when session refresh returns signed out and reports login failure", async () => {
    const user = userEvent.setup();
    mocks.getOperatorSession.mockResolvedValueOnce({ authenticated: false }).mockResolvedValueOnce({ authenticated: false });
    mocks.refreshOperatorSession.mockResolvedValueOnce({ authenticated: false });
    mocks.signInOperator.mockRejectedValueOnce(new Error("credentials rejected"));
    renderWithProviders(<FundPage />);

    await user.type(await screen.findByTestId(selectors.funding.sessionEmail), "operator@example.test");
    await user.type(screen.getByTestId(selectors.funding.sessionPassword), "incorrect password");
    await user.click(screen.getByTestId(selectors.funding.signInButton));

    expect(await screen.findByRole("alert")).toHaveTextContent("credentials rejected");
    expect(mocks.refreshOperatorSession).toHaveBeenCalledOnce();
  });

  it("shows a configured-destination error without enabling receipt creation", async () => {
    mocks.getLedgerFundingDestination.mockRejectedValueOnce(new Error("destination is not configured"));
    renderWithProviders(<FundPage />);

    expect(await screen.findByText("destination is not configured")).toBeVisible();
    expect(screen.getByTestId(selectors.funding.createIntentButton)).toBeDisabled();
  });

  it("keeps the Fund surface available when recent-intent refresh fails", async () => {
    const user = userEvent.setup();
    mocks.listFundingIntents.mockRejectedValueOnce(new Error("funding intent service unavailable"));
    renderWithProviders(<FundPage />);
    await screen.findByTestId(selectors.funding.sessionIdentity);

    await user.type(screen.getByTestId(selectors.funding.budgetId), "research-budget");
    await user.click(screen.getByTestId(selectors.funding.refreshButton));

    expect(await screen.findByRole("alert")).toHaveTextContent("funding intent service unavailable");
  });

  it("reports sign-out failure while retaining the authenticated operator session", async () => {
    const user = userEvent.setup();
    mocks.signOutOperator.mockRejectedValueOnce(new Error("session revocation unavailable"));
    renderWithProviders(<FundPage />);
    await screen.findByTestId(selectors.funding.sessionIdentity);

    await user.click(screen.getByTestId(selectors.funding.signOutButton));

    expect(await screen.findByRole("alert")).toHaveTextContent("session revocation unavailable");
    expect(screen.getByTestId(selectors.funding.sessionIdentity)).toBeVisible();
  });

  it("separates accepted funded reserve from unchanged spend authorization", async () => {
    const user = userEvent.setup();
    mocks.confirmFundingTransfer.mockResolvedValue(acceptedIntent);
    mocks.getOperatorBudgetHeadroom
      .mockResolvedValueOnce(create(AgentBudgetHeadroomSchema, { availableMinor: 100n, fundingCreditedMinor: 0n, fundingRemainingMinor: 0n }))
      .mockResolvedValueOnce(create(AgentBudgetHeadroomSchema, { availableMinor: 100n, fundingCreditedMinor: 260n, fundingRemainingMinor: 260n }));
    renderWithProviders(<FundPage />);
    await screen.findByTestId(selectors.funding.sessionIdentity);

    await user.type(screen.getByTestId(selectors.funding.budgetId), "research-budget");
    await user.type(screen.getByTestId(selectors.funding.atomicAmount), "2.60");
    await user.type(screen.getByTestId(selectors.funding.declaredValue), "2.60");
    await screen.findByText(/Research · Base deposits/);
    await user.click(screen.getByTestId(selectors.funding.createIntentButton));
    expect(await screen.findByTestId(selectors.funding.remainingMinor)).toHaveTextContent("$0.00");

    await user.type(screen.getByLabelText(en.fund.transactionHash), "0xlocal-receipt");
    await user.click(screen.getByRole("button", { name: en.fund.confirmAllocate }));
    expect(await screen.findByTestId(selectors.funding.creditedMinor)).toHaveTextContent("$2.60");
    expect(screen.getByTestId(selectors.funding.remainingMinor)).toHaveTextContent("$2.60");
    expect(screen.getByTestId(selectors.funding.spendAvailableMinor)).toHaveTextContent("$1.00");
  });

  it("offers exact-amount Base Pay only for an approved mainnet intent and lets Treasury verify the returned hash", async () => {
    const user = userEvent.setup();
    const mainnetIntent = create(FundingIntentSchema, {
      ...intent,
      id: "mainnet-intent",
      fundingMode: "base-mainnet",
      paymentValueMinor: 260n,
      paymentCurrency: "USD",
      atomicAmount: "2600000",
      wallet: create(BudgetWalletSchema, { address: "0x0000000000000000000000000000000000000001" }),
    });
    const txHash = `0x${"a".repeat(64)}`;
    mocks.createFundingIntent.mockResolvedValueOnce(mainnetIntent);
    mocks.basePay.mockResolvedValueOnce({ id: txHash });
    mocks.basePayStatus.mockResolvedValueOnce({ status: "completed", amount: "2.60", recipient: "0x0000000000000000000000000000000000000001" });
    mocks.confirmFundingTransfer.mockResolvedValueOnce(acceptedIntent);
    renderWithProviders(<FundPage />);
    await screen.findByTestId(selectors.funding.sessionIdentity);
    await user.type(screen.getByTestId(selectors.funding.budgetId), "research-budget");
    await user.type(screen.getByTestId(selectors.funding.atomicAmount), "2.60");
    await user.type(screen.getByTestId(selectors.funding.declaredValue), "2.60");
    await screen.findByText(/Research · Base deposits/);
    await user.click(screen.getByTestId(selectors.funding.createIntentButton));

    await user.click(await screen.findByTestId(selectors.funding.basePayButton));

    expect(mocks.basePay).toHaveBeenCalledWith({ amount: "2.60", to: "0x0000000000000000000000000000000000000001", testnet: false });
    expect(mocks.basePayStatus).toHaveBeenCalledWith({ id: txHash, testnet: false });
    expect(mocks.confirmFundingTransfer).toHaveBeenCalledWith("mainnet-intent", txHash);
  });

  it("renders an accessible scannable QR without exposing the raw payment URI", async () => {
    const user = userEvent.setup();
    renderWithProviders(<FundPage />);
    await screen.findByTestId(selectors.funding.sessionIdentity);

    await user.type(screen.getByTestId(selectors.funding.budgetId), "research-budget");
    await user.type(screen.getByTestId(selectors.funding.atomicAmount), "2.50");
    await user.type(screen.getByTestId(selectors.funding.declaredValue), "2.50");
    await screen.findByText(/Research · Base deposits/);
    await user.click(screen.getByTestId(selectors.funding.createIntentButton));

    expect(mocks.createFundingIntent).toHaveBeenCalledWith({
      budgetId: "research-budget",
      atomicAmount: "2500000",
      valueMinor: 250n,
    });
    expect(await screen.findByRole("img", { name: en.fund.paymentQrLabel })).toBeVisible();
    expect(screen.getByTestId(selectors.funding.paymentQr)).toBeVisible();
    expect(screen.getByTestId(selectors.funding.paymentUri)).toHaveTextContent(en.fund.paymentLinkHelp);
    expect(screen.getByTestId(selectors.funding.paymentUri)).not.toHaveTextContent(paymentUri);
    expect(screen.getByTestId(selectors.funding.openWalletLink)).toHaveAttribute("href", paymentUri);
    expect(screen.getByTestId("funding-ledger-destination")).not.toHaveTextContent("secret-book-uuid");
    expect(screen.getByTestId("funding-ledger-destination")).not.toHaveTextContent("secret-account-uuid");
  });

  it("refreshes recent intents, selects one, and copies its exact wallet URI", async () => {
    const user = userEvent.setup();
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", { configurable: true, value: { writeText } });
    mocks.listFundingIntents.mockResolvedValue([intent]);
    renderWithProviders(<FundPage />);
    await screen.findByTestId(selectors.funding.sessionIdentity);

    await user.type(screen.getByTestId(selectors.funding.budgetId), "research-budget");
    await user.click(screen.getByTestId(selectors.funding.refreshButton));
    await user.click(await screen.findByTestId(selectors.funding.recentIntent));
    await user.click(screen.getByTestId(selectors.funding.copyLinkButton));

    expect(mocks.listFundingIntents).toHaveBeenCalledWith("research-budget");
    expect(writeText).toHaveBeenCalledWith(paymentUri);
    expect(await screen.findByText(en.fund.copiedNotice)).toBeVisible();
  });

  it("keeps the receipt visible when clipboard access is denied", async () => {
    const user = userEvent.setup();
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: { writeText: vi.fn().mockRejectedValue(new Error("permission denied")) },
    });
    mocks.listFundingIntents.mockResolvedValue([intent]);
    renderWithProviders(<FundPage />);
    await screen.findByTestId(selectors.funding.sessionIdentity);

    await user.type(screen.getByTestId(selectors.funding.budgetId), "research-budget");
    await user.click(screen.getByTestId(selectors.funding.refreshButton));
    await user.click(await screen.findByTestId(selectors.funding.recentIntent));
    await user.click(screen.getByTestId(selectors.funding.copyLinkButton));

    expect(await screen.findByRole("alert")).toHaveTextContent(en.fund.copyError);
    expect(screen.getByTestId(selectors.funding.paymentUri)).not.toHaveTextContent(paymentUri);
  });

  it("reports malformed funding amounts without creating an intent", async () => {
    const user = userEvent.setup();
    renderWithProviders(<FundPage />);
    await screen.findByTestId(selectors.funding.sessionIdentity);

    await user.type(screen.getByTestId(selectors.funding.budgetId), "research-budget");
    await user.type(screen.getByTestId(selectors.funding.atomicAmount), "0");
    await user.type(screen.getByTestId(selectors.funding.declaredValue), "2.50");
    await screen.findByText(/Research · Base deposits/);
    await user.click(screen.getByTestId(selectors.funding.createIntentButton));

    expect(await screen.findByRole("alert")).toHaveTextContent(en.fund.amountError);
    expect(mocks.createFundingIntent).not.toHaveBeenCalled();
  });

  it("shows why headroom is unavailable when the agent identity lookup fails", async () => {
    const user = userEvent.setup();
    mocks.getOperatorBudgetHeadroom.mockRejectedValueOnce(new Error("operator session rejected"));
    renderWithProviders(<FundPage />);
    await screen.findByTestId(selectors.funding.sessionIdentity);

    await user.type(screen.getByTestId(selectors.funding.budgetId), "research-budget");
    await user.click(screen.getByTestId(selectors.funding.refreshButton));

    expect(await screen.findByRole("alert")).toHaveTextContent("operator session rejected");
  });

  it("confirms the entered receipt hash and announces accepted allocation", async () => {
    const user = userEvent.setup();
    mocks.createFundingIntent.mockResolvedValue(intent);
    mocks.confirmFundingTransfer.mockResolvedValue(acceptedIntent);
    renderWithProviders(<FundPage />);
    await screen.findByTestId(selectors.funding.sessionIdentity);

    await user.type(screen.getByTestId(selectors.funding.budgetId), "research-budget");
    await user.type(screen.getByTestId(selectors.funding.atomicAmount), "2.50");
    await user.type(screen.getByTestId(selectors.funding.declaredValue), "2.50");
    await screen.findByText(/Research · Base deposits/);
    await user.click(screen.getByTestId(selectors.funding.createIntentButton));
    await user.type(screen.getByLabelText(en.fund.transactionHash), "0xlocal-receipt");
    await user.click(screen.getByRole("button", { name: en.fund.confirmAllocate }));

    expect(mocks.confirmFundingTransfer).toHaveBeenCalledWith(intent.id, "0xlocal-receipt");
    expect(await screen.findByText(/receipt-delivery-1/)).toBeVisible();
  });

  it("keeps the active receipt state current while the watcher delivers it", async () => {
    const user = userEvent.setup();
    mocks.createFundingIntent.mockResolvedValue(deliveringIntent);
    mocks.listFundingIntents.mockResolvedValueOnce([deliveringIntent]).mockResolvedValue([acceptedIntent]);
    renderWithProviders(<FundPage />);
    await screen.findByTestId(selectors.funding.sessionIdentity);

    await user.type(screen.getByTestId(selectors.funding.budgetId), "research-budget");
    await user.type(screen.getByTestId(selectors.funding.atomicAmount), "2.50");
    await user.type(screen.getByTestId(selectors.funding.declaredValue), "2.50");
    await screen.findByText(/Research · Base deposits/);
    await user.click(screen.getByTestId(selectors.funding.createIntentButton));

    const state = await screen.findByTestId(selectors.funding.activeIntentState);
    expect(state).toHaveTextContent("delivering");
    await vi.waitFor(() => expect(state).toHaveTextContent("accepted"), { timeout: 4_000 });
    expect(state).toHaveAttribute("aria-live", "polite");
    expect(screen.getByTestId(selectors.funding.notificationState)).toHaveTextContent("sent");
    expect(screen.getByTestId(selectors.funding.notificationState)).toHaveAttribute("aria-live", "polite");
    expect(mocks.listFundingIntents).toHaveBeenCalledTimes(2);
  });

  it("previews a withdrawal, then executes only the explicit local-fork quote", async () => {
    const user = userEvent.setup();
    mocks.previewWithdrawal.mockResolvedValue(quote);
    mocks.executeWithdrawal
      .mockResolvedValueOnce(create(WithdrawalSchema, { ...withdrawal, state: "pending_approval", transactionHash: "" }))
      .mockResolvedValueOnce(create(WithdrawalSchema, { ...withdrawal, state: "pending_approval", transactionHash: "" }))
      .mockResolvedValueOnce(withdrawal);
    renderWithProviders(<FundPage />);
    await screen.findByTestId(selectors.funding.sessionIdentity);

    await user.type(screen.getByTestId(selectors.funding.budgetId), "research-budget");
    await user.type(screen.getByTestId(selectors.funding.withdrawalDestination), quote.destination);
    await user.type(screen.getByTestId(selectors.funding.mandateId), quote.mandateId);
    await user.type(screen.getAllByLabelText(en.fund.tokenAmount)[1]!, "1.00");
    await user.type(screen.getAllByLabelText(en.fund.declaredValue)[1]!, "1.00");
    await user.clear(screen.getByLabelText(en.fund.maxFee));
    await user.type(screen.getByLabelText(en.fund.maxFee), quote.maxFeeWei);
    await user.clear(screen.getByLabelText(en.fund.feeValue));
    await user.type(screen.getByLabelText(en.fund.feeValue), "0.05");
    await user.click(screen.getByTestId(selectors.funding.withdrawalPreviewButton));

    expect(mocks.previewWithdrawal).toHaveBeenCalledWith({
      budgetId: "research-budget",
      mandateId: "mandate-1",
      destination: quote.destination,
      atomicAmount: "1000000",
      valueMinor: 100n,
      maxFeeWei: "1000000000",
      feeValuationMinor: 5n,
    });
    await user.click(await screen.findByRole("button", { name: en.fund.executeFork }));
    expect(screen.getByTestId(selectors.funding.withdrawalExecuteButton)).toBeVisible();
    const idempotencyKey = mocks.executeWithdrawal.mock.calls[0]?.[1];
    expect(idempotencyKey).toBeTruthy();
    expect(screen.getByTestId(selectors.funding.withdrawalExecuteButton)).toBeDisabled();
    await vi.waitFor(() => expect(screen.getByTestId(selectors.funding.withdrawalStatus)).toHaveTextContent("settled"), { timeout: 5_000 });
    expect(mocks.executeWithdrawal).toHaveBeenNthCalledWith(1, quote.id, idempotencyKey);
    expect(mocks.executeWithdrawal).toHaveBeenNthCalledWith(2, quote.id, idempotencyKey);
    expect(mocks.executeWithdrawal).toHaveBeenNthCalledWith(3, quote.id, idempotencyKey);
    expect(screen.getByTestId(selectors.funding.withdrawalStatus)).toHaveTextContent("settled");
    expect(await screen.findByText(en.fund.withdrawNotice.replace("{{state}}", "settled"))).toBeVisible();
  });

  it("keeps execution disabled for non-fork quotes and reports preview failures", async () => {
    const user = userEvent.setup();
    mocks.previewWithdrawal.mockResolvedValue(create(WithdrawalQuoteSchema, { ...quote, executionMode: "mainnet", estimatedSeconds: 61 }));
    renderWithProviders(<FundPage />);
    await screen.findByTestId(selectors.funding.sessionIdentity);

    await user.type(screen.getByTestId(selectors.funding.budgetId), "research-budget");
    await user.type(screen.getByTestId(selectors.funding.withdrawalDestination), quote.destination);
    await user.type(screen.getByTestId(selectors.funding.mandateId), quote.mandateId);
    await user.type(screen.getAllByLabelText(en.fund.tokenAmount)[1]!, "1.00");
    await user.type(screen.getAllByLabelText(en.fund.declaredValue)[1]!, "1.00");
    await user.click(screen.getByTestId(selectors.funding.withdrawalPreviewButton));
    expect(await screen.findByRole("button", { name: en.fund.executeDisabled })).toBeDisabled();
    expect(screen.getByText("2 min")).toBeVisible();

    mocks.previewWithdrawal.mockRejectedValueOnce(new Error("approval expired"));
    await user.click(screen.getByTestId(selectors.funding.withdrawalPreviewButton));
    expect(await screen.findByRole("alert")).toHaveTextContent("approval expired");
    expect(screen.queryByText(/Quote quote-1/)).not.toBeInTheDocument();
  });

  it("reports a failed local-fork withdrawal without losing the preview", async () => {
    const user = userEvent.setup();
    mocks.previewWithdrawal.mockResolvedValue(quote);
    mocks.executeWithdrawal.mockRejectedValueOnce(new Error("withdrawal authorization unavailable"));
    renderWithProviders(<FundPage />);
    await screen.findByTestId(selectors.funding.sessionIdentity);

    await user.type(screen.getByTestId(selectors.funding.budgetId), "research-budget");
    await user.type(screen.getByTestId(selectors.funding.withdrawalDestination), quote.destination);
    await user.type(screen.getByTestId(selectors.funding.mandateId), quote.mandateId);
    await user.type(screen.getAllByLabelText(en.fund.tokenAmount)[1]!, "1.00");
    await user.type(screen.getAllByLabelText(en.fund.declaredValue)[1]!, "1.00");
    await user.click(screen.getByTestId(selectors.funding.withdrawalPreviewButton));
    await user.click(await screen.findByRole("button", { name: en.fund.executeFork }));

    expect(await screen.findByRole("alert")).toHaveTextContent("withdrawal authorization unavailable");
    expect(screen.getByRole("button", { name: en.fund.executeFork })).toBeVisible();
  });
});
