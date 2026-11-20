import { create } from "@bufbuild/protobuf";
import {
  CommercialUse,
  ModelRecordSchema,
  Readiness,
  type ModelRecord,
} from "@vrooli/proto-types/audio-tools/v1/models/models_pb";

interface HardwareFacts {
  cpuCapable: boolean;
  gpuRequired: boolean;
  minVramGb: number;
  minRamGb: number;
  osArch: string[];
}

interface BenchmarkFacts {
  wer: number;
  sim: number;
  rtf: number;
  notes: string;
}

/**
 * Explicit override shape (rather than MessageInitShape) because the record
 * has nested messages: a partial nested object is easier to spread and keeps
 * call sites from having to know about proto `$typeName` internals.
 */
export interface ModelRecordOverrides {
  id?: string;
  name?: string;
  providerIds?: string[];
  engineIds?: string[];
  capability?: string;
  runtime?: string;
  languages?: string[];
  sizeMbApprox?: number;
  params?: string;
  license?: string;
  commercialUse?: CommercialUse;
  licenseNotes?: string;
  sourceUrl?: string;
  sourceRepo?: string;
  readiness?: Readiness;
  roles?: string[];
  defaultEligible?: boolean;
  isDefault?: boolean;
  allowDefault?: boolean;
  hardware?: Partial<HardwareFacts>;
  benchmarks?: Partial<BenchmarkFacts>;
  notes?: string;
}

const DEFAULT_HARDWARE: HardwareFacts = {
  cpuCapable: true,
  gpuRequired: false,
  minVramGb: 0,
  minRamGb: 4,
  osArch: ["linux/amd64"],
};

const DEFAULT_BENCHMARKS: BenchmarkFacts = {
  wer: 0.05,
  sim: 0,
  rtf: 0.2,
  notes: "Internal multilingual eval.",
};

/**
 * Registry-record factory. Domain-local so deleting this feature takes its
 * test doubles along. Defaults describe a ready, commercially usable local
 * STT model; pass overrides for other capabilities/states.
 */
export const makeModelRecord = (
  overrides: ModelRecordOverrides = {},
): ModelRecord =>
  create(ModelRecordSchema, {
    id: overrides.id ?? "whisper-large-v3",
    name: overrides.name ?? "Whisper Large v3",
    providerIds: overrides.providerIds ?? ["whisper-stt"],
    engineIds: overrides.engineIds ?? ["whisper-local"],
    capability: overrides.capability ?? "stt",
    runtime: overrides.runtime ?? "faster-whisper",
    languages: overrides.languages ?? ["en", "es", "fr", "de"],
    sizeMbApprox: overrides.sizeMbApprox ?? 3090,
    params: overrides.params ?? "1.55B",
    license: overrides.license ?? "MIT",
    commercialUse: overrides.commercialUse ?? CommercialUse.ALLOWED,
    licenseNotes:
      overrides.licenseNotes ??
      "MIT permits commercial use with attribution.",
    sourceUrl: overrides.sourceUrl ?? "https://example.test/openai/whisper",
    sourceRepo: overrides.sourceRepo ?? "openai/whisper",
    readiness: overrides.readiness ?? Readiness.AVAILABLE,
    roles: overrides.roles ?? ["default", "accurate"],
    defaultEligible: overrides.defaultEligible ?? true,
    isDefault: overrides.isDefault ?? true,
    allowDefault: overrides.allowDefault ?? false,
    hardware: { ...DEFAULT_HARDWARE, ...overrides.hardware },
    benchmarks: { ...DEFAULT_BENCHMARKS, ...overrides.benchmarks },
    notes: overrides.notes ?? "Primary local STT model.",
  });
