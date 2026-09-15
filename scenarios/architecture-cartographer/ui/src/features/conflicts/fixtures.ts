/**
 * Conflict fixtures for unit + integration tests. Pure data — no mocks here.
 *
 * `makeConflict()` builds a proto-shaped Conflict with sane defaults; callers
 * override the fields they care about. The conflicts domain is detection-only,
 * so the envelope carries no lifecycle fields (status / assigned domain live
 * in the campaign domain's CampaignItem).
 */
import { create, type MessageInitShape } from "@bufbuild/protobuf";
import {
  ConflictSchema,
  Severity,
  type Conflict,
} from "@vrooli/proto-types/architecture-cartographer/v1/shared/shared_pb";

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
    snapshotId: "snap-1",
    ...overrides,
  });
