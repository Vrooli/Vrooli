/**
 * AssignmentsCard tests — focused on the assignments-card surface only. Renders
 * <AssignmentsCard /> directly so failures point at assignments-feature
 * behaviour, not shell composition. Follows the canonical mock-builder pattern.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { makeAssignment, makeListAssignmentsResponse } from "./mocks/factories";
import { makeAssignmentsMocks } from "./mocks/assignments";

vi.mock("../../api/assignments", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/assignments")>();
  return { ...actual, ...makeAssignmentsMocks() };
});

import { AssignmentsCard } from "./AssignmentsCard";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";

describe("AssignmentsCard", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the empty state when listAssignments resolves with none", async () => {
    const { assignmentsClient } = await import("../../api/assignments");
    vi.mocked(assignmentsClient.listAssignments).mockResolvedValueOnce(makeListAssignmentsResponse());

    renderWithProviders(<AssignmentsCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.assignments.empty)).toBeInTheDocument();
    });
    expect(screen.queryByTestId(selectors.assignments.list)).not.toBeInTheDocument();
  });

  it("renders scenario, brand, and pinned version when assignments exist", async () => {
    const { assignmentsClient } = await import("../../api/assignments");
    vi.mocked(assignmentsClient.listAssignments).mockResolvedValueOnce(
      makeListAssignmentsResponse({
        assignments: [
          makeAssignment({
            id: "a",
            scenarioName: "web-console",
            brandId: "brand-x",
            brandVersion: 3,
            elements: ["logo", "colors"],
          }),
          makeAssignment({ id: "b", scenarioName: "audio-tools", brandId: "brand-y", brandVersion: 1 }),
        ],
      }),
    );

    renderWithProviders(<AssignmentsCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.assignments.list)).toBeInTheDocument();
    });
    const list = screen.getByTestId(selectors.assignments.list);
    expect(list.textContent).toContain("web-console");
    expect(list.textContent).toContain("audio-tools");
    expect(list.textContent).toContain("brand-x");
    expect(list.textContent).toContain("logo, colors");
    expect(screen.getAllByTestId(selectors.assignments.version)[0]?.textContent).toContain("3");
  });
});
