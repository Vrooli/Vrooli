import { beforeEach, describe, expect, it, vi } from "vitest";

const { admin, ledger, clientIndex } = vi.hoisted(() => ({
  admin: {
    createFundingIntent: vi.fn(),
    listFundingIntents: vi.fn(),
    confirmFundingTransfer: vi.fn(),
    previewWithdrawal: vi.fn(),
    executeWithdrawal: vi.fn(),
  },
  ledger: { getLedgerDestination: vi.fn() },
  clientIndex: { value: 0 },
}));

vi.mock("@connectrpc/connect", () => ({
  createClient: vi.fn(() => (clientIndex.value++ === 0 ? admin : ledger)),
}));

import {
  confirmFundingTransfer,
  createFundingIntent,
  executeWithdrawal,
  getLedgerFundingDestination,
  listFundingIntents,
  previewWithdrawal,
} from "./funding";

describe("funding API", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    clientIndex.value = 0;
  });

  it("creates a valued funding intent and rejects an empty server response", async () => {
    vi.spyOn(globalThis.crypto, "randomUUID").mockReturnValue("123e4567-e89b-12d3-a456-426614174000");
    const intent = { id: "123e4567-e89b-12d3-a456-426614174000" } as never;
    admin.createFundingIntent.mockResolvedValueOnce({ intent });

    await expect(createFundingIntent({ budgetId: "budget", atomicAmount: "1000000", valueMinor: 250n })).resolves.toBe(intent);
    expect(admin.createFundingIntent).toHaveBeenCalledWith({
      id: "123e4567-e89b-12d3-a456-426614174000", budgetId: "budget", atomicAmount: "1000000",
      valuation: { amountMinor: 250n, currency: "USD", basisReference: "Operator-declared local-fork funding value" },
    });
    admin.createFundingIntent.mockResolvedValueOnce({});
    await expect(createFundingIntent({ budgetId: "budget", atomicAmount: "1", valueMinor: 1n })).rejects.toThrow("empty funding intent");
    vi.restoreAllMocks();
  });

  it("requires a configured ledger destination and explains its unconfigured state", async () => {
    const destination = { configured: true, bookName: "Research", accountName: "Cash" } as never;
    ledger.getLedgerDestination.mockResolvedValueOnce({ destination });
    await expect(getLedgerFundingDestination()).resolves.toBe(destination);
    ledger.getLedgerDestination.mockResolvedValueOnce({ destination: { configured: false, unconfiguredReason: "Choose a ledger account" } });
    await expect(getLedgerFundingDestination()).rejects.toThrow("Choose a ledger account");
    ledger.getLedgerDestination.mockResolvedValueOnce({ destination: { configured: true } });
    await expect(getLedgerFundingDestination()).rejects.toThrow("no verified Money Ledger destination");
  });

  it("lists intents and confirms only a returned funding intent", async () => {
    const intents = [{ id: "123e4567-e89b-12d3-a456-426614174000" }] as never;
    admin.listFundingIntents.mockResolvedValueOnce({ intents });
    await expect(listFundingIntents("budget")).resolves.toBe(intents);
    expect(admin.listFundingIntents).toHaveBeenCalledWith({ budgetId: "budget" });
    const intent = { id: "123e4567-e89b-12d3-a456-426614174000" } as never;
    admin.confirmFundingTransfer.mockResolvedValueOnce({ intent });
    await expect(confirmFundingTransfer("123e4567-e89b-12d3-a456-426614174000", "0xabc")).resolves.toBe(intent);
    expect(admin.confirmFundingTransfer).toHaveBeenCalledWith({ id: "123e4567-e89b-12d3-a456-426614174000", transactionHash: "0xabc" });
    admin.confirmFundingTransfer.mockResolvedValueOnce({});
    await expect(confirmFundingTransfer("123e4567-e89b-12d3-a456-426614174000", "0xabc")).rejects.toThrow("empty funding confirmation");
  });

  it("previews and executes a withdrawal with explicit amount, fee and idempotency", async () => {
    const quote = { id: "quote-1" } as never;
    admin.previewWithdrawal.mockResolvedValueOnce({ quote });
    await expect(previewWithdrawal({ budgetId: "budget", mandateId: "mandate", destination: "0xabc", atomicAmount: "500", maxFeeWei: "10", feeValuationMinor: 2n, valueMinor: 50n })).resolves.toBe(quote);
    expect(admin.previewWithdrawal).toHaveBeenCalledWith({
      budgetId: "budget", mandateId: "mandate", destination: "0xabc", atomicAmount: "500", maxFeeWei: "10", feeValuationMinor: 2n,
      valuation: { amountMinor: 50n, currency: "USD", basisReference: "Operator-declared local-fork withdrawal value" },
    });
    admin.previewWithdrawal.mockResolvedValueOnce({});
    await expect(previewWithdrawal({ budgetId: "budget", mandateId: "mandate", destination: "0xabc", atomicAmount: "500", maxFeeWei: "10", feeValuationMinor: 2n, valueMinor: 50n })).rejects.toThrow("empty withdrawal quote");
    const withdrawal = { id: "withdrawal-1" } as never;
    admin.executeWithdrawal.mockResolvedValueOnce({ withdrawal });
    await expect(executeWithdrawal("quote-1", "retry-1")).resolves.toBe(withdrawal);
    expect(admin.executeWithdrawal).toHaveBeenCalledWith({ quoteId: "quote-1", idempotencyKey: "retry-1" });
    admin.executeWithdrawal.mockResolvedValueOnce({});
    await expect(executeWithdrawal("quote-1", "retry-1")).rejects.toThrow("empty withdrawal");
  });
});
