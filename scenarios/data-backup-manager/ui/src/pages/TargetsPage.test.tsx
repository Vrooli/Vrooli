import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";

vi.mock("../api/targets", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../api/targets")>()),
  listTargets: vi.fn(),
  registerTarget: vi.fn(),
  deregisterTarget: vi.fn(),
}));
vi.mock("../api/runs", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../api/runs")>()),
  listTargetStatus: vi.fn(),
}));
vi.mock("../api/restores", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../api/restores")>()),
  listRestores: vi.fn(),
}));

import * as targetsApi from "../api/targets";
import * as runsApi from "../api/runs";
import * as restoresApi from "../api/restores";
import { SourceKind } from "../api/targets";
import { RunStatus } from "../api/runs";
import { TargetsPage } from "./TargetsPage";

const recently = timestampFromDate(new Date(Date.now() - 60 * 60 * 1000));

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(targetsApi.listTargets).mockResolvedValue([
    { id: "t1", owner: "prompt-manager", name: "store", sourceKind: SourceKind.FILESYSTEM, locator: "store/teams" },
  ] as never);
  // Backed up, never verified — the spine case.
  vi.mocked(runsApi.listTargetStatus).mockResolvedValue([
    { targetId: "t1", lastSuccessAt: recently, lastRunStatus: RunStatus.COMPLETED, lastVerifiedSnapshotId: "" },
  ] as never);
  vi.mocked(restoresApi.listRestores).mockResolvedValue([] as never);
  vi.mocked(targetsApi.registerTarget).mockResolvedValue({ id: "t2" } as never);
  vi.mocked(targetsApi.deregisterTarget).mockResolvedValue(true);
});

afterEach(() => cleanup());

describe("TargetsPage", () => {
  it("shows a backed-up-but-unverified target as unverified in the table", async () => {
    renderWithProviders(<TargetsPage />);
    const row = await screen.findByTestId(selectors.targets.row({ id: "t1" }));
    expect(within(row).getByText(strings.status.verified.unverified)).toBeInTheDocument();
  });

  it("validates required fields before registering", async () => {
    const user = userEvent.setup();
    renderWithProviders(<TargetsPage />);
    await user.click(screen.getByTestId(selectors.targets.registerButton));
    await user.click(screen.getByTestId(selectors.targets.formSubmit));
    expect(screen.getByText(strings.targets.ownerRequired)).toBeInTheDocument();
    expect(targetsApi.registerTarget).not.toHaveBeenCalled();
  });

  it("registers a target when the form is valid", async () => {
    const user = userEvent.setup();
    renderWithProviders(<TargetsPage />);
    await user.click(screen.getByTestId(selectors.targets.registerButton));
    await user.type(screen.getByTestId(selectors.targets.formOwner), "swarm-manager");
    await user.type(screen.getByTestId(selectors.targets.formName), "db");
    await user.type(screen.getByTestId(selectors.targets.formLocator), "swarm.db");
    await user.click(screen.getByTestId(selectors.targets.formSubmit));
    expect(targetsApi.registerTarget).toHaveBeenCalledWith(
      expect.objectContaining({ owner: "swarm-manager", name: "db", locator: "swarm.db" }),
    );
  });

  it("deregisters after confirmation", async () => {
    const user = userEvent.setup();
    renderWithProviders(<TargetsPage />);
    await screen.findByTestId(selectors.targets.row({ id: "t1" }));
    await user.click(screen.getByTestId(selectors.targets.deregisterButton));
    await user.click(screen.getByTestId(selectors.targets.deregisterConfirm));
    expect(targetsApi.deregisterTarget).toHaveBeenCalledWith("prompt-manager", "store");
  });
});
