/**
 * Overview tests. The load-bearing assertion is the product's spine: a target
 * that was backed up successfully but never verified must render the
 * *unverified* chip — never a success state — so an operator can tell "backed
 * up" from "proven restorable" at a glance.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, within } from "@testing-library/react";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";

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
vi.mock("../api/discovery", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../api/discovery")>()),
  listTargetSuggestions: vi.fn(),
  listDestinationSuggestions: vi.fn(),
}));

import * as healthApi from "../api/health";
import * as targetsApi from "../api/targets";
import * as destinationsApi from "../api/destinations";
import * as runsApi from "../api/runs";
import * as discoveryApi from "../api/discovery";
import { SourceKind } from "../api/targets";
import { BackendKind, CapPolicy, UsageState } from "../api/destinations";
import { RunStatus } from "../api/runs";
import { OverviewPage } from "./OverviewPage";

const ts = (d: Date) => timestampFromDate(d);
const recently = new Date(Date.now() - 60 * 60 * 1000);

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(healthApi.fetchHealth).mockResolvedValue({
    status: "ok",
    service: "data-backup-manager",
    readiness: "ready",
  } as never);
  vi.mocked(targetsApi.listTargets).mockResolvedValue([
    { id: "t1", owner: "prompt-manager", name: "store", sourceKind: SourceKind.FILESYSTEM, locator: "store" },
    { id: "t2", owner: "prompt-manager", name: "cache", sourceKind: SourceKind.REDIS, locator: "c" },
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
  // Default: no suggestions, so existing assertions stay focused on the catalog.
  vi.mocked(discoveryApi.listTargetSuggestions).mockResolvedValue([] as never);
  vi.mocked(discoveryApi.listDestinationSuggestions).mockResolvedValue([] as never);
});

afterEach(() => cleanup());

describe("OverviewPage", () => {
  it("flags a backed-up-but-unverified target as unverified, not verified", async () => {
    vi.mocked(runsApi.listTargetStatus).mockResolvedValue([
      // t1: backed up, but NEVER verified (lastVerifiedAt unset).
      { targetId: "t1", lastSuccessAt: ts(recently), lastRunStatus: RunStatus.COMPLETED, lastVerifiedSnapshotId: "" },
      // t2: backed up AND recently verified.
      {
        targetId: "t2",
        lastSuccessAt: ts(recently),
        lastRunStatus: RunStatus.COMPLETED,
        lastVerifiedAt: ts(recently),
        lastVerifiedSnapshotId: "snap-1",
      },
    ] as never);

    renderWithProviders(<OverviewPage />);

    const t1Row = await screen.findByTestId(
      selectors.overview.coverageRow({ targetId: "t1" }),
      undefined,
      { timeout: 5000 },
    );
    expect(within(t1Row).getByText(strings.status.verified.unverified)).toBeInTheDocument();
    expect(within(t1Row).queryByText(strings.status.verified.verified)).not.toBeInTheDocument();

    const t2Row = screen.getByTestId(selectors.overview.coverageRow({ targetId: "t2" }));
    expect(within(t2Row).getByText(strings.status.verified.verified)).toBeInTheDocument();
  });

  it("renders the storage strip with destination usage", async () => {
    vi.mocked(runsApi.listTargetStatus).mockResolvedValue([] as never);
    renderWithProviders(<OverviewPage />);
    const storage = await screen.findByTestId(selectors.overview.storage);
    // Backend label resolves through the i18n registry (cimode → key text);
    // wait for the query to resolve past the loading skeleton.
    await screen.findByText(strings.status.backend.filesystem);
    expect(within(storage).getByRole("progressbar")).toBeInTheDocument();
  });

  it("shows the setup CTA when nothing is configured and no suggestions exist", async () => {
    vi.mocked(targetsApi.listTargets).mockResolvedValue([] as never);
    vi.mocked(destinationsApi.listDestinations).mockResolvedValue([] as never);
    vi.mocked(runsApi.listTargetStatus).mockResolvedValue([] as never);

    renderWithProviders(<OverviewPage />);
    expect(await screen.findByTestId(selectors.overview.setupCta)).toBeInTheDocument();
    expect(screen.queryByTestId(selectors.discovery.panel)).not.toBeInTheDocument();
  });

  it("leads with the onboarding Suggestions panel when the catalog is empty but suggestions exist", async () => {
    vi.mocked(targetsApi.listTargets).mockResolvedValue([] as never);
    vi.mocked(destinationsApi.listDestinations).mockResolvedValue([] as never);
    vi.mocked(runsApi.listTargetStatus).mockResolvedValue([] as never);
    vi.mocked(discoveryApi.listTargetSuggestions).mockResolvedValue([
      {
        id: "ts1",
        owner: "vrooli",
        name: "plans",
        sourceKind: SourceKind.FILESYSTEM,
        locator: "/home/u/.vrooli/plans",
        rationale: "Your Vrooli plans.",
        approxBytes: 4096n,
      },
    ] as never);

    renderWithProviders(<OverviewPage />);
    // The suggestions panel is the onboarding centerpiece; no manual setup CTA.
    expect(await screen.findByTestId(selectors.discovery.panel)).toBeInTheDocument();
    expect(screen.queryByTestId(selectors.overview.setupCta)).not.toBeInTheDocument();
  });
});
