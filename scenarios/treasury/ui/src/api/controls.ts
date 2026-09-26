import { createClient } from "@connectrpc/connect";
import type { Mandate } from "@vrooli/proto-types/treasury/v1/mandate/mandate_pb";
import { TreasuryAdmin } from "@vrooli/proto-types/treasury/v1/authorization/authorization_pb";

import { transport } from "./client";

const treasuryAdmin = createClient(TreasuryAdmin, transport);

function operatorHeaders(operatorToken: string): HeadersInit {
  return { Authorization: `Bearer ${operatorToken}` };
}

export async function listOperatorMandates(operatorToken: string): Promise<Mandate[]> {
  const response = await treasuryAdmin.listMandates({}, { headers: operatorHeaders(operatorToken) });
  return response.mandates;
}

export async function getScenarioFrozen(operatorToken: string): Promise<boolean> {
  return (await treasuryAdmin.getFreezeStatus({}, { headers: operatorHeaders(operatorToken) })).frozen;
}

export async function cancelStandingMandate(operatorToken: string, mandateId: string): Promise<Mandate> {
  const response = await treasuryAdmin.cancelStandingMandate({ mandateId }, { headers: operatorHeaders(operatorToken) });
  if (!response.mandate) throw new Error("Treasury returned an empty mandate cancellation");
  return response.mandate;
}

export async function setScenarioFrozen(operatorToken: string, frozen: boolean): Promise<boolean> {
  if (frozen) {
    return (await treasuryAdmin.freezeAll({}, { headers: operatorHeaders(operatorToken) })).frozen;
  }
  return (await treasuryAdmin.unfreezeAll({}, { headers: operatorHeaders(operatorToken) })).frozen;
}

export type { Mandate };
