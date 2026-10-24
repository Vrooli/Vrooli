import { create, type MessageInitShape } from "@bufbuild/protobuf";
import {
  ApplyRunSchema,
  ApplyStatus,
  BuildBaselineSchema,
  OperationKind,
  OperationSchema,
  PlanSchema,
  type ApplyRun,
  type BuildBaseline,
  type Operation,
  type Plan,
} from "@vrooli/proto-types/architecture-cartographer/v1/apply/apply_pb";

export const makeOperation = (
  overrides: MessageInitShape<typeof OperationSchema> = {},
): Operation =>
  create(OperationSchema, {
    id: "op-1",
    kind: OperationKind.MOVE_FILE,
    fromPath: "api/internal/foo/foo.go",
    toPath: "api/internal/bar/foo.go",
    payload: new Uint8Array(),
    resolvesConflictIds: [],
    ...overrides,
  });

export const makePlan = (
  overrides: MessageInitShape<typeof PlanSchema> = {},
): Plan =>
  create(PlanSchema, {
    id: "plan-1",
    scenario: "demo-scenario",
    domain: "foo",
    operations: [makeOperation()],
    conflictIds: ["c-1"],
    ...overrides,
  });

export const makeApplyRun = (
  overrides: MessageInitShape<typeof ApplyRunSchema> = {},
): ApplyRun =>
  create(ApplyRunSchema, {
    id: "run-1",
    planId: "plan-1",
    scenario: "demo-scenario",
    domain: "foo",
    status: ApplyStatus.PLANNED,
    buildLog: "",
    ...overrides,
  });

export const makeBuildBaseline = (
  overrides: MessageInitShape<typeof BuildBaselineSchema> = {},
): BuildBaseline =>
  create(BuildBaselineSchema, {
    scenario: "demo-scenario",
    green: true,
    toolchain: "go",
    log: "",
    ...overrides,
  });
