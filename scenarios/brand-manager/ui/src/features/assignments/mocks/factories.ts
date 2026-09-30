/**
 * Test data factories for the assignments domain. Co-located with the feature
 * so deleting `features/assignments/` takes the factories with it (no central
 * residue).
 *
 * The Assignment / ListAssignmentsResponse types are GENERATED proto messages;
 * factories use `create(<Schema>, overrides)` so field defaults match proto3
 * semantics and adding a proto field is instantly available without editing
 * this file.
 */
import { create, type MessageInitShape } from "@bufbuild/protobuf";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import {
  AssignBrandResponseSchema,
  AssignmentSchema,
  ListAssignmentsResponseSchema,
  ScenarioStatusSchema,
  type AssignBrandResponse,
  type Assignment,
  type ListAssignmentsResponse,
  type ScenarioStatus,
} from "@vrooli/proto-types/brand-manager/v1/assignments/assignments_pb";

export type { AssignBrandResponse, Assignment, ListAssignmentsResponse, ScenarioStatus };

export const makeAssignment = (overrides: MessageInitShape<typeof AssignmentSchema> = {}): Assignment =>
  create(AssignmentSchema, {
    id: "assignment-1",
    brandId: "brand-1",
    scenarioName: "web-console",
    brandVersion: 1,
    elements: [],
    appliedAt: timestampFromDate(new Date("2026-01-01T00:00:00.000Z")),
    ...overrides,
  });

export const makeListAssignmentsResponse = (
  overrides: MessageInitShape<typeof ListAssignmentsResponseSchema> = {},
): ListAssignmentsResponse =>
  create(ListAssignmentsResponseSchema, {
    assignments: [],
    ...overrides,
  });

export const makeScenarioStatus = (
  overrides: MessageInitShape<typeof ScenarioStatusSchema> = {},
): ScenarioStatus =>
  create(ScenarioStatusSchema, {
    scenario: "web-console",
    hasBrand: false,
    ...overrides,
  });

export const makeAssignBrandResponse = (
  overrides: MessageInitShape<typeof AssignBrandResponseSchema> = {},
): AssignBrandResponse =>
  create(AssignBrandResponseSchema, {
    assignment: makeAssignment(),
    ...overrides,
  });
