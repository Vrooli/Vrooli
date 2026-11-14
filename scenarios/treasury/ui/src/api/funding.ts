import { createClient } from "@connectrpc/connect";
import { TreasuryAdmin } from "@vrooli/proto-types/treasury/v1/authorization/authorization_pb";
import type {
  FundingIntent,
  Withdrawal,
  WithdrawalQuote,
} from "@vrooli/proto-types/treasury/v1/funding/wallet_ops_pb";

import { transport } from "./client";

const treasuryAdmin = createClient(TreasuryAdmin, transport);

function operatorHeaders(operatorToken: string): HeadersInit {
  return { Authorization: `Bearer ${operatorToken}` };
}

export async function createFundingIntent(
  operatorToken: string,
  input: { budgetId: string; atomicAmount: string; ledgerBookId: string; ledgerAccountId: string; valueMinor: bigint },
): Promise<FundingIntent> {
  const response = await treasuryAdmin.createFundingIntent({
    id: crypto.randomUUID(),
    budgetId: input.budgetId,
    atomicAmount: input.atomicAmount,
    ledgerBookId: input.ledgerBookId,
    ledgerAccountId: input.ledgerAccountId,
    valuation: { amountMinor: input.valueMinor, currency: "USD", basisReference: "Operator-declared local-fork funding value" },
  }, { headers: operatorHeaders(operatorToken) });
  if (!response.intent) throw new Error("Treasury returned an empty funding intent");
  return response.intent;
}

export async function listFundingIntents(operatorToken: string, budgetId: string): Promise<FundingIntent[]> {
  const response = await treasuryAdmin.listFundingIntents({ budgetId }, { headers: operatorHeaders(operatorToken) });
  return response.intents;
}

export async function confirmFundingTransfer(operatorToken: string, id: string, transactionHash: string): Promise<FundingIntent> {
  const response = await treasuryAdmin.confirmFundingTransfer({ id, transactionHash }, { headers: operatorHeaders(operatorToken) });
  if (!response.intent) throw new Error("Treasury returned an empty funding confirmation");
  return response.intent;
}

export async function previewWithdrawal(
  operatorToken: string,
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
  }, { headers: operatorHeaders(operatorToken) });
  if (!response.quote) throw new Error("Treasury returned an empty withdrawal quote");
  return response.quote;
}

export async function executeWithdrawal(operatorToken: string, agentIdentityToken: string, quoteId: string, idempotencyKey: string): Promise<Withdrawal> {
  const headers = new Headers(operatorHeaders(operatorToken));
  headers.set("X-Treasury-Mandate-Identity-Token", agentIdentityToken);
  const response = await treasuryAdmin.executeWithdrawal({ quoteId, idempotencyKey }, { headers });
  if (!response.withdrawal) throw new Error("Treasury returned an empty withdrawal");
  return response.withdrawal;
}

export type { FundingIntent, Withdrawal, WithdrawalQuote };
