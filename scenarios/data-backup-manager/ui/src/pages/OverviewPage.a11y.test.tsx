import { afterEach, beforeEach, describe, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";

import { renderWithProviders, expectNoA11yViolations } from "../test-utils";
import { selectors } from "../consts/selectors";

vi.mock("../api/health", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../api/health")>()),
  fetchHealth: vi.fn(),
}));
vi.mock("../api/targets", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../api/targets")>()),
  listTargets: vi.fn(),
}));
vi.mock("../api/destinations", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../api/destinations")>()),
  listDestinations: vi.fn(),
}));
vi.mock("../api/runs", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../api/runs")>()),
  listTargetStatus: vi.fn(),
}));

import * as healthApi from "../api/health";
import * as targetsApi from "../api/targets";
import * as destinationsApi from "../api/destinations";
import * as runsApi from "../api/runs";
import { SourceKind } from "../api/targets";
import { BackendKind, CapPolicy, UsageState } from "../api/destinations";
import { RunStatus } from "../api/runs";
import { OverviewPage } from "./OverviewPage";

const now = new Date();
const ts = timestampFromDate(now);

beforeEach(() => {
  vi.mocked(healthApi.fetchHealth).mockResolvedValue({
    status: "ok",
    service: "data-backup-manager",
    readiness: "ready",
  } as never);
  vi.mocked(targetsApi.listTargets).mockResolvedValue([
    { id: "t1", owner: "prompt-manager", name: "store", sourceKind: SourceKind.FILESYSTEM, locator: "store" },
  ] as never);
  vi.mocked(destinationsApi.listDestinations).mockResolvedValue([
    {
      id: "d1",
      name: "local",
      backendKind: BackendKind.FILESYSTEM,
      location: "/var/backups",
      capBytes: 100n,
      capPolicy: CapPolicy.ALERT_BLOCK,
      encryptionAlgorithm: "AES256-GCM",
      secretRef: "vault:dbm/d1",
      usageBytes: 10n,
      usageState: UsageState.WITHIN,
    },
  ] as never);
  vi.mocked(runsApi.listTargetStatus).mockResolvedValue([
    { targetId: "t1", lastSuccessAt: ts, lastRunStatus: RunStatus.COMPLETED, lastVerifiedSnapshotId: "" },
  ] as never);
});

afterEach(() => cleanup());

describe("OverviewPage a11y", () => {
  it("has no axe violations once loaded", async () => {
    const { container } = renderWithProviders(<OverviewPage />);
    // Wait for every async surface to settle (coverage data + the health-backed
    // posture readiness) so no state update lands outside act during the scan.
    await screen.findByTestId(selectors.overview.coverageRow({ targetId: "t1" }));
    await waitFor(() =>
      expect(screen.getByTestId(selectors.overview.posture)).toHaveTextContent("ready"),
    );
    await expectNoA11yViolations(container);
  });
});
