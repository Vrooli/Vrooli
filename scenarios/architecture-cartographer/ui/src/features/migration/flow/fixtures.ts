/**
 * Migration fixtures for unit + integration tests. Pure data — no mocks here.
 *
 * Builders produce proto-shaped messages with sane defaults; callers override
 * only the fields they care about.
 */
import { create, type MessageInitShape } from "@bufbuild/protobuf";
import {
  MigrationLifecycle,
  MigrationSchema,
  MigrationStatusSchema,
  TrackedFindingSchema,
  TrackedFindingStatus,
  type Migration,
  type MigrationStatus,
  type TrackedFinding,
} from "@vrooli/proto-types/architecture-cartographer/v1/migration/migration_pb";

export const makeMigration = (
  overrides: MessageInitShape<typeof MigrationSchema> = {},
): Migration =>
  create(MigrationSchema, {
    id: "m-1",
    scenario: "demo-scenario",
    name: "big-refactor",
    status: MigrationLifecycle.OPEN,
    ...overrides,
  });

export const makeTrackedFinding = (
  overrides: MessageInitShape<typeof TrackedFindingSchema> = {},
): TrackedFinding =>
  create(TrackedFindingSchema, {
    stableId: "afid:abc12345",
    scenario: "demo-scenario",
    source: "architecture",
    code: "cycle/cross-domain",
    severity: "blocker",
    locations: ["api/a", "api/b"],
    domains: ["a", "b"],
    message: "import cycle between a and b",
    status: TrackedFindingStatus.DETECTED,
    ...overrides,
  });

export const makeMigrationStatus = (
  overrides: MessageInitShape<typeof MigrationStatusSchema> = {},
): MigrationStatus =>
  create(MigrationStatusSchema, {
    migration: makeMigration(),
    findings: [makeTrackedFinding()],
    total: 1,
    open: 1,
    resolved: 0,
    validated: 0,
    regressions: 0,
    ...overrides,
  });
