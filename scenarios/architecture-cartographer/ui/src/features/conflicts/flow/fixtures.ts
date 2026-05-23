/**
 * Conflict fixtures for unit + integration tests. Pure data — no mocks here.
 *
 * `makeConflict()` builds a proto-shaped Conflict with sane defaults; callers
 * override the fields they care about. Defaults are picked so the most common
 * test path is `makeConflict()` with no args.
 */
import { create, type MessageInitShape } from "@bufbuild/protobuf";
import {
  ConflictSchema,
  ResolutionStatus,
  Severity,
  type Conflict,
} from "@vrooli/proto-types/architecture-cartographer/v1/conflicts/conflicts_pb";

export const makeConflict = (
  overrides: MessageInitShape<typeof ConflictSchema> = {},
): Conflict =>
  create(ConflictSchema, {
    id: "c-1",
    scenario: "demo-scenario",
    detector: "cycle",
    type: "cycle",
    subtype: "type-only",
    severity: Severity.ERROR,
    locations: ["pkg/foo/foo.go", "pkg/bar/bar.go"],
    domains: ["foo", "bar"],
    status: ResolutionStatus.DETECTED,
    assignedDomain: "",
    resolutionNote: "",
    snapshotId: "snap-1",
    ...overrides,
  });
