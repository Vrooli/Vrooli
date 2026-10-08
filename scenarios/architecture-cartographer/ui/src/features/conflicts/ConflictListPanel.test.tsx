import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

vi.mock("../../api/conflicts", () => ({
  conflictsClient: {
    listConflicts: vi.fn(),
    detectConflicts: vi.fn(),
    validateConflicts: vi.fn(),
  },
}));

import { conflictsClient } from "../../api/conflicts";
import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { ConflictListPanel } from "./ConflictListPanel";
import { makeConflict } from "./fixtures";
import { Severity } from "@vrooli/proto-types/architecture-cartographer/v1/shared/shared_pb";

afterEach(() => {
  cleanup();
  vi.mocked(conflictsClient.listConflicts).mockReset();
  vi.mocked(conflictsClient.detectConflicts).mockReset();
  vi.mocked(conflictsClient.validateConflicts).mockReset();
});

type ListResult = Awaited<ReturnType<typeof conflictsClient.listConflicts>>;

describe("ConflictListPanel", () => {
  it("renders the empty state when the API returns no conflicts", async () => {
    vi.mocked(conflictsClient.listConflicts).mockResolvedValue({
      conflicts: [],
      nextPageToken: "",
    } as unknown as ListResult);

    renderWithProviders(<ConflictListPanel scenario="demo" />);

    await waitFor(() =>
      expect(screen.getByTestId(selectors.features.conflicts.list.empty)).toBeInTheDocument(),
    );
  });

  it("renders one row per conflict with an open link to the detail page", async () => {
    vi.mocked(conflictsClient.listConflicts).mockResolvedValue({
      conflicts: [
        makeConflict({ id: "c-1", type: "cycle", severity: Severity.BLOCKER, domains: ["foo"] }),
        makeConflict({ id: "c-2", type: "mislocated_file" }),
      ],
      nextPageToken: "",
    } as unknown as ListResult);

    renderWithProviders(<ConflictListPanel scenario="demo" />);

    await waitFor(() =>
      expect(screen.getByTestId(selectors.features.conflicts.list.root)).toBeInTheDocument(),
    );
    expect(
      screen.getByTestId(selectors.features.conflicts.list.openButton({ id: "c-1" })),
    ).toHaveAttribute("href", "/targets/demo/conflicts/c-1");
    expect(
      screen.getByTestId(selectors.features.conflicts.list.openButton({ id: "c-2" })),
    ).toBeInTheDocument();
  });

  it("fires detect when the Run detection button is pressed", async () => {
    vi.mocked(conflictsClient.listConflicts).mockResolvedValue({
      conflicts: [],
      nextPageToken: "",
    } as unknown as ListResult);
    vi.mocked(conflictsClient.detectConflicts).mockResolvedValue({
      conflicts: [],
    } as unknown as Awaited<ReturnType<typeof conflictsClient.detectConflicts>>);

    const user = userEvent.setup();
    renderWithProviders(<ConflictListPanel scenario="demo" />);

    await waitFor(() =>
      expect(screen.getByTestId(selectors.features.conflicts.list.detectButton)).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId(selectors.features.conflicts.list.detectButton));
    expect(conflictsClient.detectConflicts).toHaveBeenCalledWith({
      scenario: "demo",
      snapshotId: "",
      idempotencyKey: "",
    });
  });

  it("shows the error state when the list query fails", async () => {
    vi.mocked(conflictsClient.listConflicts).mockRejectedValue(new Error("boom"));

    renderWithProviders(<ConflictListPanel scenario="demo" />);

    await waitFor(() =>
      expect(screen.getByTestId(selectors.features.conflicts.list.error)).toBeInTheDocument(),
    );
  });
});
