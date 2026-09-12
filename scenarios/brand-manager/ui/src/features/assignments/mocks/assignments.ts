/**
 * Mock builders for `api/assignments` — the UI ↔ API assignments boundary.
 * Co-located with the assignments feature; deleting `features/assignments/`
 * takes these mocks with it. Canonical usage:
 *
 *   import { makeAssignmentsMocks } from "./mocks/assignments";
 *
 *   vi.mock("../../api/assignments", async (importOriginal) => {
 *     const actual = await importOriginal<typeof import("../../api/assignments")>();
 *     return { ...actual, ...makeAssignmentsMocks() };
 *   });
 */
import { vi } from "vitest";

import { makeAssignBrandResponse, makeListAssignmentsResponse, makeScenarioStatus } from "./factories";

export interface AssignmentsMocks {
  assignmentsClient: {
    listAssignments: ReturnType<typeof vi.fn>;
    assignBrand: ReturnType<typeof vi.fn>;
    getScenarioStatus: ReturnType<typeof vi.fn>;
    unassignScenario: ReturnType<typeof vi.fn>;
  };
}

export const makeAssignmentsMocks = (): AssignmentsMocks => ({
  assignmentsClient: {
    listAssignments: vi.fn().mockResolvedValue(makeListAssignmentsResponse()),
    assignBrand: vi.fn().mockResolvedValue(makeAssignBrandResponse()),
    getScenarioStatus: vi.fn().mockResolvedValue({ status: makeScenarioStatus() }),
    unassignScenario: vi.fn().mockResolvedValue({}),
  },
});
