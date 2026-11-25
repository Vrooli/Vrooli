import { cleanup, fireEvent, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { setLocale } from "../../i18n";

import { renderWithProviders } from "../../test-utils";
import type { CatalogAsset, WorkbenchStory } from "../data";
import { FixtureBench } from "./FixtureBench";

vi.mock("./FixtureFrame", () => ({
  FixtureFrame: ({
    storyId,
    theme,
    onMeasure,
  }: {
    storyId: string;
    theme?: { id: string };
    onMeasure?: (paint: { x: number; y: number; width: number; height: number }) => void;
  }) => (
    <div data-testid={`fixture-${storyId}`} data-theme={theme?.id}>
      <button
        type="button"
        onClick={() => onMeasure?.({ x: 2, y: 3, width: 80, height: 24 })}
      >
        Measure {storyId}
      </button>
    </div>
  ),
}));

const asset = {
  id: "asset-1",
  displayName: "IconButton",
  catalogId: "controls.icon-button",
  slot: "ui-primitive",
  assetKind: "ASSET_KIND_COMPONENT",
} as unknown as CatalogAsset;

function story(
  id: string,
  name: string,
  role: string,
  extra: Partial<WorkbenchStory> = {},
): WorkbenchStory {
  return {
    key: `1.0.0:${id}`,
    id,
    name,
    role,
    args: {},
    fieldsJson: "{}",
    version: "1.0.0",
    sourcePath: "IconButton.tsx",
    contractId: "asset@1",
    specimen: { module: "./story.tsx", export: id },
    ...extra,
  };
}

const stories: WorkbenchStory[] = [
  story("default", "Ghost, circular", "anatomy", { mode: "live" }),
  story("surface-soft", "Soft", "axis", { mode: "live", axis: "surface" }),
  story("danger", "Danger", "boundary", { mode: "live" }),
  story("empty", "Empty label", "boundary", {
    mode: "pinned",
    specimen: { module: "./story.tsx", export: "default" },
  }),
];

function renderBench(overrides: Partial<Parameters<typeof FixtureBench>[0]> = {}) {
  const onSelect = vi.fn();
  renderWithProviders(
    <FixtureBench
      asset={asset}
      stories={stories}
      version="1.0.0"
      sourceRevision="rev-a"
      selectedId="default"
      onSelect={onSelect}
      overrides={{}}
      reloads={{}}
      statuses={{}}
      onStatus={vi.fn()}
      onRetry={vi.fn()}
      {...overrides}
    />,
  );
  return { onSelect };
}

beforeEach(async () => {
  await setLocale("en");
});

afterEach(() => {
  cleanup();
});

describe("FixtureBench", () => {
  it("groups the anatomy, axis and boundary questions", () => {
    renderBench();
    expect(screen.getByRole("heading", { name: "Anatomy" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Surface" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Boundaries" })).toBeInTheDocument();
    expect(screen.getAllByTestId("fixture-default")).toHaveLength(1);
  });

  it("selects a frame from its header", () => {
    const { onSelect } = renderBench();
    fireEvent.click(screen.getByRole("button", { name: "Select Danger" }));
    expect(onSelect).toHaveBeenCalledWith(expect.objectContaining({ id: "danger" }));
  });

  it("reports duplicate frames without mounting them until asked", () => {
    renderBench();
    const toggle = screen.getByRole("button", { name: /Show 1 redundant frame/ });
    expect(screen.queryByTestId("fixture-empty")).not.toBeInTheDocument();
    fireEvent.click(toggle);
    expect(screen.getByTestId("fixture-empty")).toBeInTheDocument();
  });

  it("switches the specimen surface and reports measured size", () => {
    renderBench();
    expect(screen.getAllByTestId("fixture-default")[0]).toHaveAttribute(
      "data-theme",
      "workbench-dark",
    );
    fireEvent.click(screen.getByTestId("fixture-surface-light"));
    expect(screen.getAllByTestId("fixture-default")[0]).toHaveAttribute(
      "data-theme",
      "workbench-light",
    );

    fireEvent.click(screen.getAllByRole("button", { name: "Measure default" })[0]!);
    expect(screen.getAllByTestId("bench-size")[0]).toHaveTextContent("80 × 24px");
  });

  it("offers a retry on a failed frame without waiting for the harness", () => {
    const onRetry = vi.fn();
    renderBench({ statuses: { default: "error" }, onRetry });
    expect(screen.getByRole("alert")).toHaveTextContent("did not render");
    fireEvent.click(screen.getByTestId("bench-retry"));
    expect(onRetry).toHaveBeenCalledWith("default");
  });

  it("explains the empty state when there is no story contract", () => {
    renderBench({ stories: [] });
    expect(screen.getByTestId("bench-empty")).toHaveTextContent("No declared stories");
  });
});
