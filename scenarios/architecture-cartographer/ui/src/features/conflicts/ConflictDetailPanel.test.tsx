import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

vi.mock("../../api/conflicts", () => ({
  conflictsClient: {
    getConflict: vi.fn(),
    listConflicts: vi.fn(),
    assignConflict: vi.fn(),
    resolveConflict: vi.fn(),
    reopenConflict: vi.fn(),
    validateConflicts: vi.fn(),
  },
}));

import { conflictsClient } from "../../api/conflicts";
import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { ConflictDetailPanel } from "./ConflictDetailPanel";
import { makeConflict } from "./flow/fixtures";
import { ResolutionStatus } from "@vrooli/proto-types/architecture-cartographer/v1/conflicts/conflicts_pb";

afterEach(() => {
  cleanup();
  vi.mocked(conflictsClient.getConflict).mockReset();
  vi.mocked(conflictsClient.resolveConflict).mockReset();
  vi.mocked(conflictsClient.reopenConflict).mockReset();
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

  it("renders only the legal action buttons for a detected conflict", async () => {
    vi.mocked(conflictsClient.getConflict).mockResolvedValue({
      conflict: makeConflict({ status: ResolutionStatus.DETECTED }),
    } as unknown as GetResult);

    renderWithProviders(<ConflictDetailPanel scenario="demo" conflictId="c-1" />);
    await waitFor(() =>
      expect(screen.getByTestId(selectors.features.conflicts.detail.actions)).toBeInTheDocument(),
    );

    // Legal events from "detected": assign, split, resolve, force_resolve.
    for (const event of ["assign", "split", "resolve", "force_resolve"] as const) {
      expect(
        screen.getByTestId(selectors.features.conflicts.detail.actionButton({ event })),
      ).toBeInTheDocument();
    }
    // Illegal from "detected": validate, commit, reopen.
    for (const event of ["validate", "commit", "reopen"] as const) {
      expect(
        screen.queryByTestId(selectors.features.conflicts.detail.actionButton({ event })),
      ).not.toBeInTheDocument();
    }
  });

  it("calls resolveConflict (non-force) when the Resolve button is pressed", async () => {
    vi.mocked(conflictsClient.getConflict).mockResolvedValue({
      conflict: makeConflict({ status: ResolutionStatus.DETECTED }),
    } as unknown as GetResult);
    vi.mocked(conflictsClient.resolveConflict).mockResolvedValue({
      conflict: undefined,
      dryRun: false,
      applyDeferred: false,
    } as unknown as Awaited<ReturnType<typeof conflictsClient.resolveConflict>>);

    const user = userEvent.setup();
    renderWithProviders(<ConflictDetailPanel scenario="demo" conflictId="c-1" />);

    await waitFor(() =>
      expect(
        screen.getByTestId(selectors.features.conflicts.detail.actionButton({ event: "resolve" })),
      ).toBeInTheDocument(),
    );
    await user.click(
      screen.getByTestId(selectors.features.conflicts.detail.actionButton({ event: "resolve" })),
    );
    expect(conflictsClient.resolveConflict).toHaveBeenCalledWith({
      id: "c-1",
      note: "",
      force: false,
      dryRun: false,
    });
  });

  it("shows the not-found state when the API returns an undefined conflict", async () => {
    vi.mocked(conflictsClient.getConflict).mockResolvedValue({ conflict: undefined } as unknown as GetResult);
    renderWithProviders(<ConflictDetailPanel scenario="demo" conflictId="c-1" />);
    await waitFor(() =>
      expect(screen.getByTestId(selectors.features.conflicts.detail.notFound)).toBeInTheDocument(),
    );
  });
});
