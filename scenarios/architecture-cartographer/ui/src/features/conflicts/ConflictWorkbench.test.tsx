import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

vi.mock("../../api/conflicts", () => ({
  conflictsClient: {
    listConflicts: vi.fn(),
    getConflict: vi.fn(),
    detectConflicts: vi.fn(),
    validateConflicts: vi.fn(),
    assignConflict: vi.fn(),
    resolveConflict: vi.fn(),
    reopenConflict: vi.fn(),
  },
}));

import { conflictsClient } from "../../api/conflicts";
import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { ConflictWorkbench } from "./ConflictWorkbench";
import { makeConflict } from "./flow/fixtures";

afterEach(() => {
  cleanup();
  vi.mocked(conflictsClient.listConflicts).mockReset();
  vi.mocked(conflictsClient.getConflict).mockReset();
});

type ListResult = Awaited<ReturnType<typeof conflictsClient.listConflicts>>;
type GetResult = Awaited<ReturnType<typeof conflictsClient.getConflict>>;

describe("ConflictWorkbench", () => {
  it("renders list on the primary side and an empty detail prompt on the secondary side", async () => {
    vi.mocked(conflictsClient.listConflicts).mockResolvedValue({
      conflicts: [],
      nextPageToken: "",
    } as unknown as ListResult);

    renderWithProviders(<ConflictWorkbench scenario="demo" />);

    await waitFor(() =>
      expect(screen.getByTestId(selectors.features.conflicts.workbench.root)).toBeInTheDocument(),
    );
    expect(
      screen.getByTestId(selectors.features.conflicts.workbench.emptyDetail),
    ).toBeInTheDocument();
  });

  it("renders the detail panel on the secondary side when conflictId is provided", async () => {
    vi.mocked(conflictsClient.listConflicts).mockResolvedValue({
      conflicts: [makeConflict({ id: "c-1" })],
      nextPageToken: "",
    } as unknown as ListResult);
    vi.mocked(conflictsClient.getConflict).mockResolvedValue({
      conflict: makeConflict({ id: "c-1" }),
    } as unknown as GetResult);

    renderWithProviders(<ConflictWorkbench scenario="demo" conflictId="c-1" />);

    await waitFor(() =>
      expect(screen.getByTestId(selectors.features.conflicts.detail.root)).toBeInTheDocument(),
    );
  });
});
