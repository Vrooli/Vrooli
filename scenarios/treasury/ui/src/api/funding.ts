import { createClient } from "@connectrpc/connect";
import { FundingAdminService } from "@vrooli/proto-types/treasury/v1/funding/wallet_ops_pb";
import { LedgerService } from "@vrooli/proto-types/treasury/v1/ledger/ledger_pb";
import type {
  FundingIntent,
  Withdrawal,
  WithdrawalQuote,
} from "@vrooli/proto-types/treasury/v1/funding/wallet_ops_pb";
import type { LedgerDestination } from "@vrooli/proto-types/treasury/v1/ledger/ledger_pb";

import { transport } from "./client";

const treasuryAdmin = createClient(FundingAdminService, transport);
const treasuryLedger = createClient(LedgerService, transport);

export async function createFundingIntent(
  input: { budgetId: string; atomicAmount: string; valueMinor: bigint },
): Promise<FundingIntent> {
  const response = await treasuryAdmin.createFundingIntent({
    id: crypto.randomUUID(),
    budgetId: input.budgetId,
    atomicAmount: input.atomicAmount,
    valuation: { amountMinor: input.valueMinor, currency: "USD", basisReference: "Operator-declared local-fork funding value" },
  });
  if (!response.intent) throw new Error("Treasury returned an empty funding intent");
  return response.intent;
}

export async function getLedgerFundingDestination(): Promise<LedgerDestination> {
  const response = await treasuryLedger.getLedgerDestination({});
  const destination = response.destination;
  if (!destination?.configured || !destination.bookName || !destination.accountName) {
    throw new Error(destination?.unconfiguredReason || "Treasury has no verified Money Ledger destination configured");
  }
  return destination;
}

export async function listFundingIntents(budgetId: string): Promise<FundingIntent[]> {
  const response = await treasuryAdmin.listFundingIntents({ budgetId });
  return response.intents;
}

export async function confirmFundingTransfer(id: string, transactionHash: string): Promise<FundingIntent> {
  const response = await treasuryAdmin.confirmFundingTransfer({ id, transactionHash });
  if (!response.intent) throw new Error("Treasury returned an empty funding confirmation");
  return response.intent;
}

export async function previewWithdrawal(
  input: { budgetId: string; mandateId: string; destination: string; atomicAmount: string; maxFeeWei: string; feeValuationMinor: bigint; valueMinor: bigint },
): Promise<WithdrawalQuote> {
  const response = await treasuryAdmin.previewWithdrawal({
    budgetId: input.budgetId,
    mandateId: input.mandateId,
    destination: input.destination,
    atomicAmount: input.atomicAmount,
    maxFeeWei: input.maxFeeWei,
    feeValuationMinor: input.feeValuationMinor,
    valuation: { amountMinor: input.valueMinor, currency: "USD", basisReference: "Operator-declared local-fork withdrawal value" },
  });
  if (!response.quote) throw new Error("Treasury returned an empty withdrawal quote");
  return response.quote;
}

export async function executeWithdrawal(quoteId: string, idempotencyKey: string): Promise<Withdrawal> {
  const response = await treasuryAdmin.executeWithdrawal({ quoteId, idempotencyKey });
  if (!response.withdrawal) throw new Error("Treasury returned an empty withdrawal");
  return response.withdrawal;
}

export type { FundingIntent, LedgerDestination, Withdrawal, WithdrawalQuote };
