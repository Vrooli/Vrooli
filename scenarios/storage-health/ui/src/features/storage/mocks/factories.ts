/**
 * Test data factories for storage-health domains. Each `make<Domain>` returns a
 * stable plain object (the proto messages are structural TS types, so a literal
 * with the right camelCase fields satisfies them via a single `as` cast at the
 * boundary). Overrides are shallow-merged for the common case.
 */
import { ValidationStatus } from "@vrooli/proto-types/scenario-validation/v1/validation_pb";
import type {
  ScanFleetResponse,
  FleetScenarioEntry,
  AdviseEnginesResponse,
  EngineCandidate,
  AnalyzeMigrationsResponse,
  MigrationHygiene,
  ValidateScenarioResponse,
  FixResponse,
} from "../../../api/storage";

export const makeFleetEntry = (
  overrides: Partial<FleetScenarioEntry> = {},
): FleetScenarioEntry =>
  ({
    scenario: "demo",
    engines: ["sqlite"],
    primaryEngine: "sqlite",
    language: "go",
    storageStage: "greenfield",
    isolationReady: true,
    isolationReason: "",
    namespaceAdopted: true,
    hasBackupTarget: true,
    findingCount: 0,
    errorCount: 0,
    autofixableCount: 0,
    ...overrides,
  }) as FleetScenarioEntry;

export const makeScanFleetResponse = (
  overrides: Partial<ScanFleetResponse> = {},
): ScanFleetResponse =>
  ({
    entries: [],
    engineDistribution: [],
    stageDistribution: [],
    scenarioCount: 0,
    isolationUnreadyCount: 0,
    noBackupCount: 0,
    findingCount: 0,
    errors: [],
    scannedAt: "",
    ...overrides,
  }) as ScanFleetResponse;

export const makeEngineCandidate = (
  overrides: Partial<EngineCandidate> = {},
): EngineCandidate =>
  ({
    scenario: "demo",
    currentEngine: "postgres",
    recommendedEngine: "sqlite",
    fitnessScore: 0.8,
    rationale: "Single-writer workload fits SQLite.",
    autofixable: false,
    blockers: [],
    ...overrides,
  }) as EngineCandidate;

export const makeAdviseEnginesResponse = (
  overrides: Partial<AdviseEnginesResponse> = {},
): AdviseEnginesResponse =>
  ({
    candidates: [],
    scenarioCount: 0,
    errors: [],
    ...overrides,
  }) as AdviseEnginesResponse;

export const makeMigrationHygiene = (
  overrides: Partial<MigrationHygiene> = {},
): MigrationHygiene =>
  ({
    scenario: "demo",
    storageStage: "production",
    hasMigrations: true,
    hasAlterInSchema: false,
    nonIdempotentSchema: false,
    migrationDebt: 2,
    notes: ["schema.sql is not idempotent"],
    ...overrides,
  }) as MigrationHygiene;

export const makeAnalyzeMigrationsResponse = (
  overrides: Partial<AnalyzeMigrationsResponse> = {},
): AnalyzeMigrationsResponse =>
  ({
    entries: [],
    scenarioCount: 0,
    withMigrationsCount: 0,
    debtCount: 0,
    errors: [],
    ...overrides,
  }) as AnalyzeMigrationsResponse;

export const makeValidateResponse = (
  overrides: Partial<ValidateScenarioResponse> & {
    findings?: unknown[];
    findingsBySeverity?: Record<string, number>;
  } = {},
): ValidateScenarioResponse => {
  const { findings, findingsBySeverity, ...rest } = overrides;
  return {
    scenario: "demo",
    status: ValidationStatus.PASSED,
    assessment: {
      findings: findings ?? [],
      findingsBySeverity: findingsBySeverity ?? {},
      autofixableCount: 0,
    },
    ...rest,
  } as ValidateScenarioResponse;
};

export const makeFinding = (overrides: Record<string, unknown> = {}) => ({
  code: "STORAGE_001",
  severity: "SEVERITY_ERROR",
  title: "Routed test-isolation seams unwired",
  message: "",
  location: "api/internal/db/db.go",
  remediation: "Wire the routed DB seam in the test harness.",
  autofixAvailable: false,
  fixClass: "",
  ...overrides,
});

export const makeFixResponse = (overrides: Partial<FixResponse> = {}): FixResponse =>
  ({
    scenario: "demo",
    applied: false,
    candidates: [],
    messages: [],
    ...overrides,
  }) as FixResponse;
