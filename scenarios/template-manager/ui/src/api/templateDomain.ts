import { createClient } from "@connectrpc/connect";
import { DebtService, type ListDebtResponse } from "@vrooli/proto-types/template-manager/v1/debt/debt_pb";
import { MonitorService, type GetMonitorStatusResponse } from "@vrooli/proto-types/template-manager/v1/monitor/monitor_pb";
import {
  RegistryService,
  type ListTemplatesResponse,
} from "@vrooli/proto-types/template-manager/v1/registry/registry_pb";
import {
  ValidationRunService,
  type ListDriftSnapshotsResponse,
  type ListValidationRunsResponse,
} from "@vrooli/proto-types/template-manager/v1/validation/validation_pb";

import { transport } from "./client";

export const registryClient = createClient(RegistryService, transport);
export const validationClient = createClient(ValidationRunService, transport);
export const debtClient = createClient(DebtService, transport);
export const monitorClient = createClient(MonitorService, transport);

export async function fetchTemplateDashboard(): Promise<{
  templates: ListTemplatesResponse;
  runs: ListValidationRunsResponse;
  drift: ListDriftSnapshotsResponse;
  debt: ListDebtResponse;
  monitor: GetMonitorStatusResponse;
}> {
  const [templates, runs, drift, debt, monitor] = await Promise.all([
    registryClient.listTemplates({}),
    validationClient.listValidationRuns({}),
    validationClient.listDriftSnapshots({}),
    debtClient.listDebt({}),
    monitorClient.getMonitorStatus({}),
  ]);

  return { templates, runs, drift, debt, monitor };
}

export type {
  ListDebtResponse,
  ListDriftSnapshotsResponse,
  GetMonitorStatusResponse,
  ListTemplatesResponse,
  ListValidationRunsResponse,
};
