import { createClient } from "@connectrpc/connect";
import { BudgetAdminService } from "@vrooli/proto-types/treasury/v1/budget/budget_pb";
import { MandateAdminService } from "@vrooli/proto-types/treasury/v1/mandate/mandate_pb";
import type { Mandate } from "@vrooli/proto-types/treasury/v1/mandate/mandate_pb";

import { transport } from "./client";

const budgetAdmin = createClient(BudgetAdminService, transport);
const mandateAdmin = createClient(MandateAdminService, transport);

function operatorHeaders(operatorToken: string): HeadersInit {
  return { Authorization: `Bearer ${operatorToken}` };
}

export async function listOperatorMandates(operatorToken: string): Promise<Mandate[]> {
  const response = await mandateAdmin.listMandates({}, { headers: operatorHeaders(operatorToken) });
  return response.mandates;
}

export async function getScenarioFrozen(operatorToken: string): Promise<boolean> {
  return (await budgetAdmin.getFreezeStatus({}, { headers: operatorHeaders(operatorToken) })).frozen;
}

export async function cancelStandingMandate(operatorToken: string, mandateId: string): Promise<Mandate> {
  const response = await mandateAdmin.cancelStandingMandate({ mandateId }, { headers: operatorHeaders(operatorToken) });
  if (!response.mandate) throw new Error("Treasury returned an empty mandate cancellation");
  return response.mandate;
}

export async function setScenarioFrozen(operatorToken: string, frozen: boolean): Promise<boolean> {
  if (frozen) {
    return (await budgetAdmin.freezeAll({}, { headers: operatorHeaders(operatorToken) })).frozen;
  }
  return (await budgetAdmin.unfreezeAll({}, { headers: operatorHeaders(operatorToken) })).frozen;
}

export type { Mandate };
