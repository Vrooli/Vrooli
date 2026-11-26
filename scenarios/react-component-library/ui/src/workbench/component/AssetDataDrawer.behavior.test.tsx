import { cleanup, fireEvent, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { ComponentVersionStatus } from "@vrooli/proto-types/react-component-library/v1/components/components_pb";
import { i18n, setLocale } from "../../i18n";
import { strings } from "../../consts/strings.generated";
import { renderWithProviders } from "../../test-utils";
import type { ComponentBundle } from "../data";

const api = vi.hoisted(() => ({ getAssetRelationships: vi.fn() }));
vi.mock("../../api/catalogGraph", () => ({ getAssetRelationships: api.getAssetRelationships }));
vi.mock("./SourceWorkbench", () => ({
  SourceWorkbench: ({
    version,
    onVersion,
    onSaved,
  }: {
    version: string;
    onVersion: (version: string) => void;
    onSaved?: (version: string, path: string, sha: string) => void;
  }) => (
    <div>
      <span data-testid="source-version">{version}</span>
      <button data-testid="select-source-version" onClick={() => onVersion("1.0.0")} />
      <button data-testid="save-source" onClick={() => onSaved?.(version, "Button.tsx", "sha")} />
    </div>
  ),
}));

import { AssetDataDrawer } from "./AssetDataDrawer";

function makeBundle(overrides: Partial<ComponentBundle> = {}): ComponentBundle {
  return {
    asset: {
      id: "asset-1",
      libraryId: "library:Button",
      catalogId: "catalog.button",
      slug: "button",
      displayName: "Button",
      latestVersion: "2.0.0",
      version: "2.0.0",
      draftVersion: "",
      sourcePath: "Button.tsx",
      manifestPath: "component.json",
      slot: "control",
      category: "forms",
      catalogKind: "component",
      catalogDomain: "ui",
      dependencies: [],
      tags: [],
      designStyles: [],
    },
    versions: [],
    stories: [],
    testReports: [],
    partial: [],
    ...overrides,
  } as unknown as ComponentBundle;
}

function renderTab(
  tab: "overview" | "source" | "versions" | "experience" | "relationships",
  bundle: ComponentBundle,
  props: Partial<Parameters<typeof AssetDataDrawer>[0]> = {},
) {
  return renderWithProviders(
    <AssetDataDrawer
      bundle={bundle}
      tab={tab}
      onTab={vi.fn()}
      version="2.0.0"
      onVersion={vi.fn()}
      {...props}
    />,
  );
}

beforeEach(async () => {
  await setLocale("en");
  api.getAssetRelationships.mockResolvedValue(undefined);
});
afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("AssetDataDrawer sections", () => {
  it("shows overview values and explicit empty metadata states", () => {
    const data = makeBundle({ partial: ["versions"] });
    renderTab("overview", data);
    expect(screen.getByText(data.asset.libraryId)).toBeInTheDocument();
    expect(screen.getByText(data.asset.slot)).toBeInTheDocument();
    expect(screen.getByText(i18n.t(strings.workbench.component.noTags))).toBeInTheDocument();
    expect(screen.getByText(i18n.t(strings.workbench.component.noAffinities))).toBeInTheDocument();
    expect(screen.getByText(i18n.t(strings.workbench.component.noDeps))).toBeInTheDocument();
    expect(
      screen.getByText(i18n.t(strings.workbench.component.partial, { parts: "versions" })),
    ).toBeInTheDocument();
  });

  it("renders populated overview metadata and version status branches", () => {
    const base = makeBundle();
    const tagLabel = "accessible";
    const styleLabel = "rounded";
    const dependencyLabel = "icons";
    const row = (version: string, status: ComponentVersionStatus) => ({
      id: version,
      version,
      status,
      sourcePath: `${version}/Button.tsx`,
      files: [],
      releasedAt: undefined,
      indexedAt: undefined,
    });
    const populated = {
      ...base,
      asset: {
        ...base.asset,
        tags: [tagLabel],
        designStyles: [{ styleId: styleLabel }],
        dependencies: [{ libraryId: dependencyLabel, version: "1.2" }],
      },
      versions: [
        row("3.0.0", ComponentVersionStatus.RELEASED),
        row("2.0.0", ComponentVersionStatus.DRAFT),
        row("1.0.0", ComponentVersionStatus.UNSPECIFIED),
      ],
    } as unknown as ComponentBundle;
    renderTab("overview", populated);
    expect(screen.getByText(tagLabel)).toBeInTheDocument();
    expect(screen.getByText(styleLabel)).toBeInTheDocument();
    expect(screen.getByText(dependencyLabel)).toBeInTheDocument();
    cleanup();
    const onVersion = vi.fn();
    renderTab("versions", populated, { onVersion });
    expect(screen.getByText(i18n.t(strings.workbench.component.released))).toBeInTheDocument();
    expect(screen.getByText(i18n.t(strings.workbench.component.draft))).toBeInTheDocument();
    expect(screen.getByText(i18n.t(strings.workbench.component.indexed))).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /3\.0\.0/ }));
    expect(onVersion).toHaveBeenCalledWith("3.0.0");
  });

  it("shows unavailable versions and experience contracts", () => {
    renderTab("versions", makeBundle());
    expect(screen.getByText(i18n.t(strings.workbench.component.noVersions))).toBeInTheDocument();
    cleanup();
    renderTab("experience", makeBundle());
    expect(screen.getByText(i18n.t(strings.workbench.component.noExperience))).toBeInTheDocument();
    cleanup();
    renderTab(
      "experience",
      makeBundle({
        experience: {
          evidenceStatus: "stale",
          evidenceMessage: "Evidence needs refresh",
          states: [],
          claims: [
            {
              id: "claim-pass",
              statement: "Has a name",
              type: "accessibility",
              tier: "core",
              states: ["default"],
            },
            {
              id: "claim-fail",
              statement: "Supports focus",
              type: "interaction",
              tier: "core",
              states: [],
            },
            {
              id: "claim-empty",
              statement: "Has docs",
              type: "documentation",
              tier: "optional",
              states: [],
            },
          ],
          evidence: [
            { claimId: "claim-pass", verdict: "pass" },
            { claimId: "claim-fail", verdict: "fail" },
            { claimId: "claim-fail", verdict: "unknown" },
          ],
        },
      } as unknown as Partial<ComponentBundle>),
    );
    const verdictLabels = ["pass · 1", "fail · 1", "unmeasured · 1"];
    verdictLabels.forEach((label) => expect(screen.getByText(label)).toBeInTheDocument());
    expect(screen.getByText(/Evidence needs refresh/)).toBeInTheDocument();
    expect(screen.getAllByText(/unmeasured/).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/no states/)).toHaveLength(2);
  });

  it("forwards source version selection and save events", () => {
    const base = makeBundle({
      versions: [{ version: "1.0.0" }],
    } as unknown as Partial<ComponentBundle>);
    const onVersion = vi.fn();
    const onSourceSaved = vi.fn();
    renderTab("source", base, { version: "missing-version", onVersion, onSourceSaved });
    expect(screen.getByTestId("source-version")).toHaveTextContent("1.0.0");
    fireEvent.click(screen.getByTestId("select-source-version"));
    fireEvent.click(screen.getByTestId("save-source"));
    expect(onVersion).toHaveBeenCalledWith("1.0.0");
    expect(onSourceSaved).toHaveBeenCalledWith("1.0.0", "Button.tsx", "sha");
  });

  it("shows relationships loading, empty, populated, and error states", async () => {
    api.getAssetRelationships.mockReturnValueOnce(new Promise(() => undefined));
    const { unmount } = renderTab("relationships", makeBundle());
    expect(document.querySelector(".wb-skeleton")).toBeInTheDocument();
    unmount();

    api.getAssetRelationships.mockResolvedValueOnce(null);
    renderTab("relationships", makeBundle());
    expect(
      await screen.findByText(i18n.t(strings.workbench.component.noRelationships)),
    ).toBeInTheDocument();
    cleanup();

    const dependency = { assetId: "a", name: "Icons", kind: "package", rungName: "shared" };
    const dependent = { assetId: "b", name: "Button demo" };
    api.getAssetRelationships.mockResolvedValueOnce({
      directDependencies: [dependency],
      directDependents: [dependent],
      transitiveDependentCount: 3,
    });
    renderTab("relationships", makeBundle());
    expect(await screen.findByText(dependency.name)).toBeInTheDocument();
    expect(screen.getByText(dependency.rungName)).toBeInTheDocument();
    expect(screen.getByText(dependent.name)).toBeInTheDocument();
    cleanup();

    api.getAssetRelationships.mockRejectedValueOnce(new Error("graph unavailable"));
    renderTab("relationships", makeBundle());
    expect(
      await screen.findByText(i18n.t(strings.workbench.component.relationshipsError)),
    ).toBeInTheDocument();
  });
});
