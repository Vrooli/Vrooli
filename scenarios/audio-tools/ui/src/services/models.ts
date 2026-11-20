import { createClient } from "@connectrpc/connect";
import {
  CommercialUse,
  ModelRegistryService,
  Readiness,
  type Benchmarks,
  type Hardware,
  type ModelRecord,
} from "@vrooli/proto-types/audio-tools/v1/models/models_pb";
import { makeApiError, transport, type ApiError } from "../api/client";
import { tryCall, type Result } from "./result";
import { normalizeConnectError } from "./settings";

const client = createClient(ModelRegistryService, transport);

/** Runtime-honesty state as surfaced by the registry. */
export type ReadinessLevel =
  | "available"
  | "candidate"
  | "experimental"
  | "watch"
  | "unspecified";

/** Licence-derived commercial eligibility. UNSPECIFIED means "unknown". */
export type CommercialEligibility =
  | "allowed"
  | "disallowed"
  | "unknown"
  | "unspecified";

export interface ModelHardware {
  cpuCapable: boolean;
  gpuRequired: boolean;
  minVramGb: number;
  minRamGb: number;
  osArch: string[];
}

export interface ModelBenchmarks {
  wer: number;
  sim: number;
  rtf: number;
  notes: string;
}

/**
 * ModelRow is the UI-facing projection of a registry record. Enum values are
 * normalised to string unions so components never compare raw proto numbers.
 */
export interface ModelRow {
  id: string;
  name: string;
  providerIds: string[];
  engineIds: string[];
  capability: string;
  runtime: string;
  languages: string[];
  sizeMbApprox: number;
  params: string;
  license: string;
  commercialUse: CommercialEligibility;
  licenseNotes: string;
  sourceUrl: string;
  sourceRepo: string;
  readiness: ReadinessLevel;
  roles: string[];
  defaultEligible: boolean;
  isDefault: boolean;
  allowDefault: boolean;
  hardware: ModelHardware;
  benchmarks: ModelBenchmarks;
  notes: string;
}

export interface ListModelsFilters {
  capability?: string;
  language?: string;
  commercialOnly?: boolean;
  onlyReady?: boolean;
}

export interface ResolveModelInput {
  capability: string;
  language?: string;
  commercial?: boolean;
}

export interface ResolvedModel {
  model: ModelRow;
  explanation: string;
}

/**
 * ResolveModel reports NOT_FOUND as a Connect code (numeric "5" once
 * normalised) or as the REST envelope's snake_case code. Accept both so the
 * page can render a friendly empty message instead of an error dump.
 */
const NOT_FOUND_CODES = new Set(["not_found", "5"]);

export function isNotFound(error: ApiError): boolean {
  return NOT_FOUND_CODES.has(error.code);
}

/** "en-US" -> "en". The registry matches on the primary subtag. */
export function primarySubtag(language: string): string {
  const [first] = language.trim().toLowerCase().split("-");
  return first ?? "";
}

export async function listModels(
  filters: ListModelsFilters = {},
): Promise<Result<ModelRow[]>> {
  return tryCall(async () => {
    let resp;
    try {
      resp = await client.listModels({
        capability: filters.capability ?? "",
        language: filters.language ?? "",
        commercialOnly: filters.commercialOnly ?? false,
        onlyReady: filters.onlyReady ?? false,
      });
    } catch (e) {
      throw normalizeConnectError(e);
    }
    return resp.models.map(toModelRow);
  });
}

export async function resolveModel(
  input: ResolveModelInput,
): Promise<Result<ResolvedModel>> {
  return tryCall(async () => {
    let resp;
    try {
      resp = await client.resolveModel({
        capability: input.capability,
        language: input.language ?? "",
        commercial: input.commercial ?? false,
      });
    } catch (e) {
      throw normalizeConnectError(e);
    }
    if (!resp.model) {
      throw makeApiError("not_found", "no eligible model for this request", 404);
    }
    return { model: toModelRow(resp.model), explanation: resp.explanation };
  });
}

function toModelRow(record: ModelRecord): ModelRow {
  return {
    id: record.id,
    name: record.name,
    providerIds: record.providerIds,
    engineIds: record.engineIds,
    capability: record.capability,
    runtime: record.runtime,
    languages: record.languages,
    sizeMbApprox: record.sizeMbApprox,
    params: record.params,
    license: record.license,
    commercialUse: commercialOf(record.commercialUse),
    licenseNotes: record.licenseNotes,
    sourceUrl: record.sourceUrl,
    sourceRepo: record.sourceRepo,
    readiness: readinessOf(record.readiness),
    roles: record.roles,
    defaultEligible: record.defaultEligible,
    isDefault: record.isDefault,
    allowDefault: record.allowDefault,
    hardware: toHardware(record.hardware),
    benchmarks: toBenchmarks(record.benchmarks),
    notes: record.notes,
  };
}

function readinessOf(value: Readiness): ReadinessLevel {
  switch (value) {
    case Readiness.AVAILABLE:
      return "available";
    case Readiness.CANDIDATE:
      return "candidate";
    case Readiness.EXPERIMENTAL:
      return "experimental";
    case Readiness.WATCH:
      return "watch";
    default:
      return "unspecified";
  }
}

function commercialOf(value: CommercialUse): CommercialEligibility {
  switch (value) {
    case CommercialUse.ALLOWED:
      return "allowed";
    case CommercialUse.DISALLOWED:
      return "disallowed";
    case CommercialUse.UNKNOWN:
      return "unknown";
    default:
      return "unspecified";
  }
}

function toHardware(hardware: Hardware | undefined): ModelHardware {
  return {
    cpuCapable: hardware?.cpuCapable ?? false,
    gpuRequired: hardware?.gpuRequired ?? false,
    minVramGb: hardware?.minVramGb ?? 0,
    minRamGb: hardware?.minRamGb ?? 0,
    osArch: hardware?.osArch ?? [],
  };
}

function toBenchmarks(benchmarks: Benchmarks | undefined): ModelBenchmarks {
  return {
    wer: benchmarks?.wer ?? 0,
    sim: benchmarks?.sim ?? 0,
    rtf: benchmarks?.rtf ?? 0,
    notes: benchmarks?.notes ?? "",
  };
}
