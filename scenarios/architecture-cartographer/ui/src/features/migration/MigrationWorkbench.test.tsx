import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

vi.mock("../../api/migration", () => ({
  migrationClient: {
    listMigrations: vi.fn(),
    getMigrationStatus: vi.fn(),
    nextMigrationStep: vi.fn(),
    createMigration: vi.fn(),
    resolveFinding: vi.fn(),
    applyFinding: vi.fn(),
    reauditMigration: vi.fn(),
    closeMigration: vi.fn(),
  },
}));

import { migrationClient } from "../../api/migration";
import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { MigrationWorkbench } from "./MigrationWorkbench";
import { makeMigration, makeMigrationStatus, makeTrackedFinding } from "./flow/fixtures";

afterEach(() => {
  cleanup();
  vi.mocked(migrationClient.listMigrations).mockReset();
  vi.mocked(migrationClient.getMigrationStatus).mockReset();
  vi.mocked(migrationClient.nextMigrationStep).mockReset();
});

type ListResult = Awaited<ReturnType<typeof migrationClient.listMigrations>>;
type StatusResult = Awaited<ReturnType<typeof migrationClient.getMigrationStatus>>;
type NextResult = Awaited<ReturnType<typeof migrationClient.nextMigrationStep>>;

describe("MigrationWorkbench", () => {
  it("renders the migration list on the primary side and an empty detail prompt on the secondary side", async () => {
    vi.mocked(migrationClient.listMigrations).mockResolvedValue({
      migrations: [],
    } as unknown as ListResult);

    renderWithProviders(<MigrationWorkbench scenario="demo" />);

    await waitFor(() =>
      expect(screen.getByTestId(selectors.features.migration.workbench.root)).toBeInTheDocument(),
    );
    expect(
      screen.getByTestId(selectors.features.migration.workbench.emptyDetail),
    ).toBeInTheDocument();
  });

  it("renders the detail panel with a worklist when migrationId is provided", async () => {
    vi.mocked(migrationClient.listMigrations).mockResolvedValue({
      migrations: [makeMigration({ id: "m-1" })],
    } as unknown as ListResult);
    vi.mocked(migrationClient.getMigrationStatus).mockResolvedValue({
      status: makeMigrationStatus(),
    } as unknown as StatusResult);
    vi.mocked(migrationClient.nextMigrationStep).mockResolvedValue({
      findings: [makeTrackedFinding()],
    } as unknown as NextResult);

    renderWithProviders(<MigrationWorkbench scenario="demo" migrationId="m-1" />);

    await waitFor(() =>
      expect(screen.getByTestId(selectors.features.migration.detail.root)).toBeInTheDocument(),
    );
    expect(
      screen.getByTestId(
        selectors.features.migration.detail.findingCard({ stableId: "afid:abc12345" }),
      ),
    ).toBeInTheDocument();
  });
});
