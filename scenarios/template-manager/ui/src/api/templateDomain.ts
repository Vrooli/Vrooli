import { createClient } from "@connectrpc/connect";
import {
  DebtService,
  type DebtEntry,
  type ListDebtResponse,
} from "@vrooli/proto-types/template-manager/v1/debt/debt_pb";
import { MonitorService, type GetMonitorStatusResponse } from "@vrooli/proto-types/template-manager/v1/monitor/monitor_pb";
import {
  RegistryService,
  type ListTemplatesResponse,
  type TemplateRecord,
} from "@vrooli/proto-types/template-manager/v1/registry/registry_pb";
import {
  ValidationRunService,
  type DriftSnapshot,
  type ListDriftSnapshotsResponse,
  type ListValidationRunsResponse,
  type ValidationRun,
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

/**
 * Registry template detail: the governed record plus every run, drift snapshot,
 * and debt entry scoped to it. The three lists are server-filtered by
 * `template_id` so the detail view never over-fetches the whole fleet.
 */
export async function fetchTemplateDetail(templateId: string): Promise<{
  template: TemplateRecord;
  runs: ValidationRun[];
  drift: DriftSnapshot[];
  debt: DebtEntry[];
}> {
  const [template, runs, drift, debt] = await Promise.all([
    registryClient.getTemplate({ id: templateId }),
    validationClient.listValidationRuns({ templateId }),
    validationClient.listDriftSnapshots({ templateId }),
    debtClient.listDebt({ templateId }),
  ]);

  if (!template.template) {
    throw new Error(`Template '${templateId}' not found`);
  }

  return {
    template: template.template,
    runs: runs.runs,
    drift: drift.snapshots,
    debt: debt.entries,
  };
}

/** The full governed template inventory, for the browsable registry list. */
export async function fetchTemplateList(): Promise<TemplateRecord[]> {
  const response = await registryClient.listTemplates({});
  return response.templates;
}

/** Every persisted validation run across the fleet, for the run history list. */
export async function fetchValidationRunList(): Promise<ValidationRun[]> {
  const response = await validationClient.listValidationRuns({});
  return response.runs;
}

/** Single persisted validation run with its per-phase results and findings. */
export async function fetchValidationRun(id: string): Promise<ValidationRun> {
  const response = await validationClient.getValidationRun({ id });
  if (!response.run) {
    throw new Error(`Validation run '${id}' not found`);
  }
  return response.run;
}

/** Single debt entry (provenance, status, message). */
export async function fetchDebtEntry(key: string): Promise<DebtEntry> {
  const response = await debtClient.getDebt({ key });
  if (!response.entry) {
    throw new Error(`Debt entry '${key}' not found`);
  }
  return response.entry;
}

/**
 * The full debt ledger plus the template inventory, so the list view can offer
 * a template filter without a second round-trip per option.
 */
export async function fetchDebtLedger(): Promise<{
  entries: DebtEntry[];
  templates: TemplateRecord[];
}> {
  const [debt, templates] = await Promise.all([
    debtClient.listDebt({}),
    registryClient.listTemplates({}),
  ]);
  return { entries: debt.entries, templates: templates.templates };
}

export type {
  DebtEntry,
  DriftSnapshot,
  ListDebtResponse,
  ListDriftSnapshotsResponse,
  GetMonitorStatusResponse,
  ListTemplatesResponse,
  ListValidationRunsResponse,
  TemplateRecord,
  ValidationRun,
};
