import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

vi.mock("../../api/conflicts", () => ({
  conflictsClient: {
    getConflict: vi.fn(),
    listConflicts: vi.fn(),
    validateConflicts: vi.fn(),
  },
}));

import { conflictsClient } from "../../api/conflicts";
import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { ConflictDetailPanel } from "./ConflictDetailPanel";
import { makeConflict } from "./fixtures";

afterEach(() => {
  cleanup();
  vi.mocked(conflictsClient.getConflict).mockReset();
});

type GetResult = Awaited<ReturnType<typeof conflictsClient.getConflict>>;

describe("ConflictDetailPanel", () => {
  it("renders the detail with locations, evidence count, and fix count", async () => {
    vi.mocked(conflictsClient.getConflict).mockResolvedValue({
      conflict: makeConflict({
        id: "c-1",
        locations: ["a.go", "b.go"],
        evidence: [{ kind: "scc_member", summary: "cycle", locator: "a.go" }],
        suggestedFixes: [
          { id: "fix-1", kind: 1, resolver: "moveFile", summary: "Move a.go", payload: new Uint8Array(), confidence: 0.9 },
        ],
      }),
    } as unknown as GetResult);

    renderWithProviders(<ConflictDetailPanel scenario="demo" conflictId="c-1" />);

    await waitFor(() =>
      expect(screen.getByTestId(selectors.features.conflicts.detail.root)).toBeInTheDocument(),
    );
    expect(screen.getByTestId(selectors.features.conflicts.detail.locations).textContent).toContain("a.go");
    expect(screen.getByTestId(selectors.features.conflicts.detail.evidence).textContent).toContain("scc_member");
    expect(
      screen.getByTestId(selectors.features.conflicts.detail.fixItem({ id: "fix-1" })),
    ).toBeInTheDocument();
  });

  it("is read-only: it renders no lifecycle action controls", async () => {
    vi.mocked(conflictsClient.getConflict).mockResolvedValue({
      conflict: makeConflict({ id: "c-1" }),
    } as unknown as GetResult);

    renderWithProviders(<ConflictDetailPanel scenario="demo" conflictId="c-1" />);
    await waitFor(() =>
      expect(screen.getByTestId(selectors.features.conflicts.detail.root)).toBeInTheDocument(),
    );
    expect(
      screen.queryByTestId(selectors.features.conflicts.detail.actions),
    ).not.toBeInTheDocument();
  });

  it("shows the not-found state when the API returns an undefined conflict", async () => {
    vi.mocked(conflictsClient.getConflict).mockResolvedValue({ conflict: undefined } as unknown as GetResult);
    renderWithProviders(<ConflictDetailPanel scenario="demo" conflictId="c-1" />);
    await waitFor(() =>
      expect(screen.getByTestId(selectors.features.conflicts.detail.notFound)).toBeInTheDocument(),
    );
  });
});
