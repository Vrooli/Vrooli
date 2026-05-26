import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";

vi.mock("../api/restores", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../api/restores")>()),
  listRestores: vi.fn(),
  verifyTarget: vi.fn(),
  restoreTarget: vi.fn(),
}));
vi.mock("../api/runs", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../api/runs")>()),
  listRuns: vi.fn(),
}));

import * as restoresApi from "../api/restores";
import * as runsApi from "../api/runs";
import { RunStatus, TargetOutcomeStatus, TriggerSource } from "../api/runs";
import { RestoresPage } from "./RestoresPage";

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(restoresApi.listRestores).mockResolvedValue([] as never);
  vi.mocked(restoresApi.verifyTarget).mockResolvedValue({ id: "rr1" } as never);
  vi.mocked(restoresApi.restoreTarget).mockResolvedValue({ id: "rr2" } as never);
  // One successful run outcome → one selectable snapshot in the flow.
  vi.mocked(runsApi.listRuns).mockResolvedValue([
    {
      id: "r1",
      planId: "p1",
      trigger: TriggerSource.MANUAL,
      status: RunStatus.COMPLETED,
      outcomes: [
        { targetId: "t1", destinationId: "d1", status: TargetOutcomeStatus.SUCCEEDED, snapshotId: "snap-abc123def456", bytes: 1024n, error: "" },
      ],
    },
  ] as never);
});

afterEach(() => cleanup());

describe("RestoresPage", () => {
  it("verifies a snapshot in one click (non-destructive)", async () => {
    const user = userEvent.setup();
    renderWithProviders(<RestoresPage />);
    await user.click(screen.getByTestId(selectors.restores.startButton));
    await user.selectOptions(screen.getByRole("combobox"), "t1|d1|snap-abc123def456");
    await user.click(screen.getByTestId(selectors.restores.verifyButton));
    expect(restoresApi.verifyTarget).toHaveBeenCalledWith("t1", "d1", "snap-abc123def456");
  });

  it("gates restore behind a confirmation step", async () => {
    const user = userEvent.setup();
    renderWithProviders(<RestoresPage />);
    await user.click(screen.getByTestId(selectors.restores.startButton));
    await user.selectOptions(screen.getByRole("combobox"), "t1|d1|snap-abc123def456");
    await user.type(screen.getByTestId(selectors.restores.restoreLocation), "/var/restore/t1");
    await user.click(screen.getByTestId(selectors.restores.restoreButton));

    // Restore must NOT fire until the confirmation is accepted.
    expect(restoresApi.restoreTarget).not.toHaveBeenCalled();
    expect(screen.getByTestId(selectors.restores.restoreConfirm)).toBeInTheDocument();

    await user.click(screen.getByTestId(selectors.restores.restoreConfirmButton));
    expect(restoresApi.restoreTarget).toHaveBeenCalledWith("t1", "d1", "snap-abc123def456", "/var/restore/t1");
  });
});
