import { fireEvent, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { AssetKind } from "@vrooli/proto-types/react-component-library/v1/components/components_pb";
import { i18n, setLocale } from "../../i18n";
import { strings } from "../../consts/strings";
import { renderWithProviders } from "../../test-utils";
import { DEFAULT_TEMPLATE } from "../templates/library";
import { node } from "../templates/spec";
import { emptyAudit } from "./checks";
import { StudioInspector } from "./StudioInspector";
import { seedFrames } from "./model";
import { buildStudioInspectorCheckSummary } from "./studioInspectorCheckSummary";

function renderInspector(overrides: Partial<React.ComponentProps<typeof StudioInspector>> = {}) {
  const frames = seedFrames(DEFAULT_TEMPLATE).slice(0, 2);
  const frame = frames[0]!;
  const props: React.ComponentProps<typeof StudioInspector> = {
    tab: "inspect",
    onTab: vi.fn(),
    design: {
      id: "d",
      name: "Design",
      updatedAt: "",
      revision: 1,
      frames,
      theme: DEFAULT_TEMPLATE.palette,
      options: { radius: DEFAULT_TEMPLATE.radius, display: DEFAULT_TEMPLATE.display },
    },
    frame,
    theme: DEFAULT_TEMPLATE.palette,
    selected: node("copy", "text", { label: "Confirm purchase", text: "Pay now" }),
    onPropChange: vi.fn(),
    onReplaceNode: vi.fn(),
    catalogCandidates: [],
    catalogLoading: false,
    catalogFailed: false,
    pendingCatalogAssetId: "",
    onReplaceCatalogAsset: vi.fn(),
    coverage: { rcl: 2, custom: 3, total: 5, ratio: 0.4 },
    claims: [],
    frames,
    audits: emptyAudit(frames),
    stateGap: { total: 0, covered: 0, missing: [] },
    aiScope: "layer",
    onAiScope: vi.fn(),
    onSelectFrame: vi.fn(),
    onApplyProposal: () => ({ ok: true }),
    onApplied: vi.fn(),
    assist: false,
    ...overrides,
  };
  return renderWithProviders(<StudioInspector {...props} />);
}

describe("StudioInspector guided next steps", () => {
  beforeEach(async () => {
    await setLocale("en");
  });
  it("offers the existing best-match replacement for a selected custom layer", () => {
    const frames = seedFrames(DEFAULT_TEMPLATE).slice(0, 1);
    const frame = frames[0]!;
    const onReplaceNode = vi.fn();
    renderWithProviders(
      <StudioInspector
        tab="checks"
        onTab={vi.fn()}
        design={{
          id: "d",
          name: "Design",
          updatedAt: "",
          revision: 1,
          frames,
          theme: DEFAULT_TEMPLATE.palette,
          options: { radius: DEFAULT_TEMPLATE.radius, display: DEFAULT_TEMPLATE.display },
        }}
        frame={frame}
        theme={DEFAULT_TEMPLATE.palette}
        selected={node("copy", "text", { label: "Confirm purchase", text: "Pay now" })}
        onPropChange={vi.fn()}
        onReplaceNode={onReplaceNode}
        catalogCandidates={[]}
        catalogLoading={false}
        catalogFailed={false}
        pendingCatalogAssetId=""
        onReplaceCatalogAsset={vi.fn()}
        coverage={{ rcl: 2, custom: 3, total: 5, ratio: 0.4 }}
        claims={[]}
        frames={frames}
        audits={emptyAudit(frames)}
        stateGap={{ total: 0, covered: 0, missing: [] }}
        aiScope="layer"
        onAiScope={vi.fn()}
        onSelectFrame={vi.fn()}
        onApplyProposal={() => ({ ok: true })}
        onApplied={vi.fn()}
        assist={false}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Replace with Button" }));
    expect(onReplaceNode).toHaveBeenCalledWith("button");
  });

  it("shows frame details and lets the user switch frames when nothing is selected", () => {
    const onSelectFrame = vi.fn();
    const frames = seedFrames(DEFAULT_TEMPLATE).slice(0, 2);
    renderInspector({ selected: null, onSelectFrame });

    expect(document.querySelector(".studio-inspector-title b")?.textContent).toBe(
      `${frames[0]!.name} frame`,
    );
    expect(document.querySelector(".studio-inspector-title small")?.textContent).toBe(
      "desktop · 1180 × 780",
    );
    fireEvent.click(screen.getByRole("button", { name: frames[1]!.name }));
    expect(onSelectFrame).toHaveBeenCalledWith(frames[1]!.id);
  });

  it("identifies a selected imported catalog slot and offers catalog replacements", () => {
    const onReplaceCatalogAsset = vi.fn();
    const candidate = {
      id: "component-42",
      name: "Revenue summary",
      displayName: "Revenue summary",
      description: "A compact revenue card",
      assetKind: AssetKind.COMPONENT,
      tags: [],
    } as unknown as React.ComponentProps<typeof StudioInspector>["catalogCandidates"][number];
    const importedSlot = node("imported-slot", "slot", {
      label: "Revenue summary",
      hint: "Imported from catalog",
      componentId: candidate.id,
    });
    renderInspector({
      selected: importedSlot,
      catalogCandidates: [candidate],
      onReplaceCatalogAsset,
    });

    expect(document.querySelector(".studio-inspector-title small")?.textContent).toBe(
      "RCL · Revenue summary",
    );
    expect(screen.getByLabelText(i18n.t(strings.workbench.studio.hint))).toHaveValue(
      "Imported from catalog",
    );
    fireEvent.click(
      screen.getByRole("button", {
        name: i18n.t(strings.workbench.studio.replaceWithCatalogComponent, {
          candidate: "Revenue summary",
        }),
      }),
    );
    expect(onReplaceCatalogAsset).toHaveBeenCalledWith(candidate.id);
  });

  it("shows measured check failures and opens the affected frame or layer inspector", () => {
    const onSelectFrame = vi.fn();
    const onTab = vi.fn();
    const frames = seedFrames(DEFAULT_TEMPLATE).slice(0, 2);
    const audits = emptyAudit(frames);
    audits[frames[1]!.id] = {
      measured: true,
      overflow: 12,
      buttons: { total: 2, under: 1 },
      painted: null,
    };
    renderInspector({
      tab: "checks",
      frames,
      audits,
      stateGap: { total: 2, covered: 0, missing: ["Primary action", "Search"] },
      selected: node("action", "button", { label: "Primary action" }),
      onSelectFrame,
      onTab,
    });

    expect(
      screen.getByText(
        i18n.t(strings.workbench.studio.checkOverflowFail, { names: frames[1]!.name }),
      ),
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        i18n.t(strings.workbench.studio.checkTargetsDetail, { under: 1, total: 2, min: 44 }),
      ),
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        i18n.t(strings.workbench.studio.checkStatesGap, {
          count: 2,
          names: "Primary action, Search",
        }),
      ),
    ).toBeInTheDocument();
    fireEvent.click(
      screen.getByRole("button", { name: i18n.t(strings.workbench.studio.openFrame) }),
    );
    fireEvent.click(
      screen.getByRole("button", { name: i18n.t(strings.workbench.studio.inspectLayers) }),
    );
    expect(onSelectFrame).toHaveBeenCalledWith(frames[1]!.id);
    expect(onTab).toHaveBeenCalledWith("inspect");
  });
});

describe("StudioInspector check summary", () => {
  it("uses the same measured findings for the tab badge and panel summary", () => {
    const frames = seedFrames(DEFAULT_TEMPLATE).slice(0, 2);
    const audits = emptyAudit(frames);
    audits[frames[1]!.id] = {
      measured: true,
      overflow: 12,
      buttons: { total: 2, under: 1 },
      painted: null,
    };

    const summary = buildStudioInspectorCheckSummary({
      coverage: { rcl: 2, custom: 3, total: 5, ratio: 0.4 },
      audits,
      frames,
      stateGap: { total: 2, covered: 0, missing: ["Primary action"] },
      selected: node("action", "button", { label: "Primary action" }),
      claims: [],
    });

    expect(summary.checks.needsReview).toBe(3);
    expect(summary.checks.total).toBe(3);
    expect(summary.overflowFrames.map(({ id }) => id)).toEqual([frames[1]!.id]);
    expect(summary.targetTotal).toBe(2);
    expect(summary.targetUnder).toBe(1);
    expect(summary.overflowVerdict).toBe("fail");
    expect(summary.targetVerdict).toBe("warn");
    expect(summary.stateVerdict).toBe("warn");
  });
});
