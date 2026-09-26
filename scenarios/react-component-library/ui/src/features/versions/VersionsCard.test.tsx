import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { create } from "@bufbuild/protobuf";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import {
  DiffOp,
  makeDiffCell,
  makeDiffRow,
  makeDiffVersionsResponse,
  makeListVersionsResponse,
  makeVersion,
} from "./mocks/factories";
import { makeVersionsMocks } from "./mocks/versions";
import { makeAdoption } from "../adoptions/mocks/factories";
import { ListAdoptionsResponseSchema } from "@vrooli/proto-types/react-component-library/v1/adoptions/adoptions_pb";
import {
  ListRetireCandidatesResponseSchema,
  RetireCandidateSchema,
} from "@vrooli/proto-types/react-component-library/v1/versions/lifecycle_pb";

vi.mock("../../api/versions", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/versions")>();
  return { ...actual, ...makeVersionsMocks() };
});

vi.mock("../../api/versionLedger", () => ({
  listVersionLedger: vi.fn().mockResolvedValue([]),
  versionLifecycleClient: {
    listRetireCandidates: vi.fn().mockResolvedValue({ candidates: [] }),
    retireVersion: vi.fn().mockResolvedValue({}),
    archiveVersion: vi.fn().mockResolvedValue({}),
    materializeVersion: vi.fn().mockResolvedValue({}),
  },
}));

vi.mock("../../api/componentTests", () => ({
  listComponentTestReports: vi.fn().mockResolvedValue([]),
}));

vi.mock("../../api/adoptions", () => ({
  adoptionsClient: {
    listAdoptions: vi.fn().mockResolvedValue({ adoptions: [] }),
  },
}));

import { VersionsCard } from "./VersionsCard";

describe("VersionsCard", () => {
  beforeEach(async () => {
    await setLocale("en");
  });
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders empty state when no versions are recorded", async () => {
    const { versionsClient } = await import("../../api/versions");
    vi.mocked(versionsClient.listVersions).mockResolvedValueOnce(makeListVersionsResponse());
    renderWithProviders(<VersionsCard componentId="cmp-1" />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.versions.empty)).toBeInTheDocument();
    });
    expect(screen.queryByTestId(selectors.versions.list)).not.toBeInTheDocument();
  });

  it("renders a row per version with version/sha/recorded labels", async () => {
    const { versionsClient } = await import("../../api/versions");
    vi.mocked(versionsClient.listVersions).mockResolvedValueOnce(
      makeListVersionsResponse({
        versions: [
          makeVersion({ id: "v1", version: "1.0.0", contentSha256: "aaa111bbb222ccc" }),
          makeVersion({ id: "v2", version: "1.0.1", contentSha256: "ddd333eee444fff" }),
        ],
      }),
    );
    renderWithProviders(<VersionsCard componentId="cmp-1" />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.versions.list)).toBeInTheDocument();
    });
    const items = screen.getAllByTestId("data-display-version-row");
    expect(items).toHaveLength(2);
    expect(items[0]?.textContent ?? "").toContain("v1.0.0");
    expect(items[1]?.textContent ?? "").toContain("v1.0.1");
    const shas = items.map((n) => n.textContent);
    expect(shas[0] ?? "").toContain("aaa111bbb222");
    expect(shas[1] ?? "").toContain("ddd333eee444");
  });

  it("labels an evicted version and materializes it on request", async () => {
    const { versionsClient } = await import("../../api/versions");
    const { versionLifecycleClient } = await import("../../api/versionLedger");
    vi.mocked(versionsClient.listVersions).mockResolvedValueOnce(
      makeListVersionsResponse({
        versions: [makeVersion({ id: "v-cold", version: "0.9.0", presence: "evicted" })],
      }),
    );
    renderWithProviders(<VersionsCard componentId="cmp-1" />);
    await waitFor(() =>
      expect(screen.getByTestId(`${selectors.versions.presenceBadge}-0.9.0`)).toBeInTheDocument(),
    );
    expect(screen.getByTestId(`${selectors.versions.presenceBadge}-0.9.0`)).toHaveTextContent(
      "Evicted",
    );
    await userEvent
      .setup()
      .click(screen.getByTestId(`${selectors.versions.materializeButton}-0.9.0`));
    await waitFor(() => {
      expect(vi.mocked(versionLifecycleClient.materializeVersion)).toHaveBeenCalledWith({
        componentId: "cmp-1",
        version: "0.9.0",
      });
    });
  });

  it("forwards from/to into diffVersions and renders the response rows", async () => {
    const { versionsClient } = await import("../../api/versions");
    vi.mocked(versionsClient.listVersions).mockResolvedValueOnce(
      makeListVersionsResponse({
        versions: [
          makeVersion({ id: "v1", version: "1.0.0" }),
          makeVersion({ id: "v2", version: "1.0.1" }),
        ],
      }),
    );
    vi.mocked(versionsClient.diffVersions).mockResolvedValueOnce(
      makeDiffVersionsResponse({
        fromLabel: "1.0.0",
        toLabel: "1.0.1",
        additions: 1,
        removals: 1,
        rows: [
          makeDiffRow(
            { lineNumber: 1, text: "alpha", op: DiffOp.EQUAL },
            { lineNumber: 1, text: "alpha", op: DiffOp.EQUAL },
          ),
          makeDiffRow(
            { lineNumber: 2, text: "beta", op: DiffOp.REMOVE },
            makeDiffCell({ op: DiffOp.EMPTY }),
          ),
          makeDiffRow(makeDiffCell({ op: DiffOp.EMPTY }), {
            lineNumber: 2,
            text: "BETA",
            op: DiffOp.ADD,
          }),
        ],
      }),
    );

    const user = userEvent.setup();
    renderWithProviders(<VersionsCard componentId="cmp-1" />);
    // Wait for the listVersions response to land so the diff selects
    // have populated options beyond the "—" sentinel.
    await waitFor(() => {
      expect(screen.getAllByTestId("data-display-version-row")).toHaveLength(2);
    });
    await user.selectOptions(screen.getByTestId(selectors.versions.diff.fromSelect), "1.0.0");
    await user.selectOptions(screen.getByTestId(selectors.versions.diff.toSelect), "1.0.1");
    await user.click(screen.getByTestId(selectors.versions.diff.runButton));

    await waitFor(() => {
      expect(screen.getByTestId(selectors.versions.diff.summary)).toBeInTheDocument();
    });
    expect(vi.mocked(versionsClient.diffVersions)).toHaveBeenCalledWith({
      componentId: "cmp-1",
      from: "1.0.0",
      to: "1.0.1",
    });
    const summary = screen.getByTestId(selectors.versions.diff.summary).textContent;
    expect(summary).toContain("+1");
    expect(summary).toContain("-1");
    expect(screen.getByTestId("data-display.diff-viewer")).toBeInTheDocument();
    // Spot-check that both sides of the server response remain visible.
    const table = screen.getByTestId(selectors.versions.diff.table);
    expect(table.textContent).toContain("alpha");
    expect(table.textContent).toContain("beta");
    expect(table.textContent).toContain("BETA");
  });

  it("hands a completed comparison to the enclosing Files workspace", async () => {
    const { versionsClient } = await import("../../api/versions");
    const diff = makeDiffVersionsResponse({ fromLabel: "1.0.0", toLabel: "1.0.1" });
    vi.mocked(versionsClient.listVersions).mockResolvedValueOnce(
      makeListVersionsResponse({
        versions: [
          makeVersion({ id: "v1", version: "1.0.0" }),
          makeVersion({ id: "v2", version: "1.0.1" }),
        ],
      }),
    );
    vi.mocked(versionsClient.diffVersions).mockResolvedValueOnce(diff);
    const onCompare = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(<VersionsCard componentId="cmp-1" onCompare={onCompare} />);

    await waitFor(() => expect(screen.getAllByTestId("data-display-version-row")).toHaveLength(2));
    await user.selectOptions(screen.getByTestId(selectors.versions.diff.fromSelect), "1.0.0");
    await user.selectOptions(screen.getByTestId(selectors.versions.diff.toSelect), "1.0.1");
    await user.click(screen.getByTestId(selectors.versions.diff.runButton));

    await waitFor(() => expect(onCompare).toHaveBeenCalledWith(diff));
    expect(screen.queryByTestId(selectors.versions.diff.table)).not.toBeInTheDocument();
  });

  it("run button is disabled until both sides are chosen", async () => {
    const { versionsClient } = await import("../../api/versions");
    vi.mocked(versionsClient.listVersions).mockResolvedValueOnce(
      makeListVersionsResponse({ versions: [makeVersion({ version: "1.0.0" })] }),
    );
    renderWithProviders(<VersionsCard componentId="cmp-1" />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.versions.diff.runButton)).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.versions.diff.runButton)).toBeDisabled();
  });

  it("selects a historical version for the enclosing detail view", async () => {
    const { versionsClient } = await import("../../api/versions");
    vi.mocked(versionsClient.listVersions).mockResolvedValueOnce(
      makeListVersionsResponse({ versions: [makeVersion({ version: "1.0.0" })] }),
    );
    const onSelectVersion = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(<VersionsCard componentId="cmp-1" onSelectVersion={onSelectVersion} />);
    await screen.findByTestId(selectors.versions.list);
    await user.click(screen.getByRole("button", { name: "View version 1.0.0" }));
    expect(onSelectVersion).toHaveBeenCalledWith("1.0.0");
  });

  it("renders the ledger contract, unknown test health, and required-token findings", async () => {
    const { versionsClient } = await import("../../api/versions");
    const { listVersionLedger } = await import("../../api/versionLedger");
    vi.mocked(versionsClient.listVersions).mockResolvedValueOnce(
      makeListVersionsResponse({
        versions: [
          makeVersion({
            id: "ledger-version",
            version: "2.0.0",
            sourcePath: "components/button/versions/2.0.0/Button.tsx",
            requiredTokens: ["--color-primary"],
          }),
        ],
      }),
    );
    vi.mocked(listVersionLedger).mockResolvedValueOnce([
      {
        libraryId: "react-component-library:Button",
        version: "2.0.0",
        createdAt: "2026-05-13T00:00:00Z",
        releasedAt: "2026-05-14T00:00:00Z",
        retiredAt: "",
        lifecycleState: "released",
        gatePassCount: 4,
        gateFailCount: 0,
        testRuns: 0,
        testPassRate: 0,
        adoptionCurrent: 3,
        adoptionPeak: 8,
        fileCount: 2,
        linesOfCode: 140,
        dependencyCount: 3,
        presence: "materialized",
      },
    ]);

    renderWithProviders(<VersionsCard componentId="cmp-1" />);

    await waitFor(() => expect(screen.getAllByTestId("data-display-version-row")).toHaveLength(1));
    expect(screen.getByTestId("version-health-2.0.0")).toHaveTextContent("unknown");
    expect(screen.getByTestId("version-required-tokens-2.0.0")).toHaveTextContent(
      "--color-primary",
    );
    expect(screen.getByTestId("data-display-verdict-summary")).toHaveTextContent("4 pass");
    expect(screen.getByTestId("versions-test-health")).toHaveTextContent("unknown");
    expect(screen.getByTestId("data-display-version-row")).toHaveTextContent("140 LOC");
  });

  it("joins reports and adopters into an expanded version row", async () => {
    const { versionsClient } = await import("../../api/versions");
    const { listComponentTestReports } = await import("../../api/componentTests");
    const { adoptionsClient } = await import("../../api/adoptions");
    vi.mocked(versionsClient.listVersions).mockResolvedValueOnce(
      makeListVersionsResponse({
        versions: [
          makeVersion({ id: "v2", version: "2.0.0", changelogMd: "Normalized control spacing." }),
          makeVersion({ id: "v1", version: "1.0.0" }),
        ],
      }),
    );
    vi.mocked(listComponentTestReports).mockResolvedValueOnce([
      {
        id: "report-2",
        rootLibraryId: "react-component-library:Button",
        rootVersion: "2.0.0",
        includeClosure: true,
        verdict: "failed",
        results: [
          {
            stage: "experience",
            assetLibraryId: "react-component-library:Button",
            version: "2.0.0",
            subject: "content-not-clipped",
            verdict: "failed",
            message: "The control content is clipped.",
            remediation: "Reduce the glyph size.",
          },
        ],
        artifacts: [
          {
            kind: "bas-screenshot",
            label: "primary:screenshot",
            storyId: "primary",
            assetLibraryId: "react-component-library:Button",
            version: "2.0.0",
            reference: "artifact://button.png",
          },
        ],
      },
    ]);
    vi.mocked(adoptionsClient.listAdoptions).mockResolvedValueOnce(
      create(ListAdoptionsResponseSchema, {
        adoptions: [
          makeAdoption({
            id: "ad-2",
            componentId: "cmp-1",
            scenario: "web-console",
            adoptedVersion: "2.0.0",
            forkStatus: "forked",
          }),
        ],
      }),
    );
    vi.mocked(versionsClient.diffVersions).mockResolvedValueOnce(
      makeDiffVersionsResponse({ fromLabel: "1.0.0", toLabel: "2.0.0", additions: 4, removals: 2 }),
    );

    const user = userEvent.setup();
    renderWithProviders(<VersionsCard componentId="cmp-1" />);
    await waitFor(() => expect(screen.getAllByTestId("data-display-version-row")).toHaveLength(2));
    await user.click(screen.getAllByRole("button", { name: "Show version details" })[0]!);

    await waitFor(() => expect(screen.getByTestId("version-expanded-2.0.0")).toBeInTheDocument());
    expect(screen.getByTestId("data-display-finding-list")).toHaveTextContent(
      "content-not-clipped",
    );
    expect(screen.getByTestId("version-expanded-2.0.0")).toHaveTextContent("web-console");
    expect(screen.getByTestId("version-expanded-2.0.0")).toHaveTextContent("forked");
    expect(screen.getByTestId("version-expanded-2.0.0")).toHaveTextContent("primary");
    await waitFor(() =>
      expect(screen.getByTestId("version-diff-summary-2.0.0")).toHaveTextContent("+4"),
    );
    expect(vi.mocked(listComponentTestReports)).toHaveBeenCalledWith({
      componentId: "cmp-1",
      limit: 0,
    });
    expect(vi.mocked(adoptionsClient.listAdoptions)).toHaveBeenCalledWith({
      componentId: "cmp-1",
      limit: 0,
    });
  });

  it("surfaces retire candidates and keeps the undo action available", async () => {
    const { versionsClient } = await import("../../api/versions");
    const { versionLifecycleClient } = await import("../../api/versionLedger");
    vi.mocked(versionsClient.listVersions).mockResolvedValueOnce(
      makeListVersionsResponse({ versions: [makeVersion({ version: "1.0.0" })] }),
    );
    vi.mocked(versionLifecycleClient.listRetireCandidates).mockResolvedValueOnce(
      create(ListRetireCandidatesResponseSchema, {
        candidates: [
          create(RetireCandidateSchema, {
            componentId: "cmp-1",
            libraryId: "react-component-library:Button",
            version: "0.9.0",
            status: "safe",
          }),
        ],
      }),
    );

    renderWithProviders(<VersionsCard componentId="cmp-1" />);
    await waitFor(() =>
      expect(screen.getByTestId("versions-retire-candidates")).toBeInTheDocument(),
    );
    expect(screen.getByText(/0.9.0/)).toBeInTheDocument();
    expect(screen.getAllByRole("button", { name: "Review retire actions" })[0]).toBeInTheDocument();
  });
});
