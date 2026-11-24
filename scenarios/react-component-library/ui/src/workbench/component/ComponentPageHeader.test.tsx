import { create } from "@bufbuild/protobuf";
import { ComponentSchema } from "@vrooli/proto-types/react-component-library/v1/components/components_pb";
import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { strings } from "../../consts/strings";
import { renderWithProviders } from "../../test-utils";
import type { WorkbenchStory } from "../data";
import { ComponentPageHeader } from "./ComponentPageHeader";

const asset = create(ComponentSchema, {
  id: "asset-1",
  libraryId: "react-component-library:Button",
  displayName: "Button",
  slug: "button",
  description: "A clear action",
  version: "1.2.0",
  latestVersion: "1.2.0",
  catalogId: "controls.button",
  slot: "action",
  metrics: { directAdoptionCount: 2, effectiveAdoptionCount: 5 },
});
const stories: WorkbenchStory[] = [
  {
    key: "button@1.2.0:default",
    id: "default",
    name: "Default",
    role: "primary",
    args: {},
    fieldsJson: "{}",
    version: "1.2.0",
    sourcePath: "Button.tsx",
    contractId: "button@1.2.0",
  },
  {
    key: "button@1.2.0:disabled",
    id: "disabled",
    name: "Disabled",
    role: "state",
    args: { disabled: true },
    fieldsJson: "{}",
    version: "1.2.0",
    sourcePath: "Button.tsx",
    contractId: "button@1.2.0",
  },
];

function renderHeader(overrides: Partial<Parameters<typeof ComponentPageHeader>[0]> = {}) {
  const props = {
    asset,
    latestVersion: "1.2.0",
    mode: "fixtures" as const,
    stories,
    selectedStory: stories[0],
    copied: false,
    openingWorkspace: false,
    canOpenWorkspace: true,
    runningEvidence: false,
    checksCount: 2,
    passed: 2,
    checkRatio: 1,
    onShare: vi.fn(),
    onOpenWorkspace: vi.fn(),
    onRunEvidence: vi.fn(),
    onOpenDrawer: vi.fn(),
    onChooseMode: vi.fn(),
    onChooseStory: vi.fn(),
    onRerunStory: vi.fn(),
    onToggleDrawer: vi.fn(),
    ...overrides,
  };
  renderWithProviders(<ComponentPageHeader {...props} />);
  return props;
}

describe("ComponentPageHeader", () => {
  it("routes story, mode, evidence, share, and drawer actions to their callbacks", () => {
    const props = renderHeader();

    fireEvent.change(screen.getByRole("combobox", { name: strings.workbench.component.story }), {
      target: { value: "disabled" },
    });
    expect(props.onChooseStory).toHaveBeenCalledWith(stories[1]);

    fireEvent.click(screen.getByRole("button", { name: strings.workbench.component.inContext }));
    expect(props.onChooseMode).toHaveBeenCalledWith("context");

    fireEvent.click(screen.getByRole("button", { name: strings.workbench.component.share }));
    fireEvent.click(screen.getByRole("button", { name: strings.workbench.component.runEvidence }));
    fireEvent.click(screen.getByRole("button", { name: strings.workbench.component.rerun }));
    fireEvent.click(screen.getByRole("button", { name: strings.workbench.component.assetData }));
    fireEvent.click(screen.getByRole("button", { name: strings.workbench.component.tests }));

    expect(props.onShare).toHaveBeenCalledOnce();
    expect(props.onRunEvidence).toHaveBeenCalledOnce();
    expect(props.onRerunStory).toHaveBeenCalledOnce();
    expect(props.onToggleDrawer).toHaveBeenCalledOnce();
    expect(props.onOpenDrawer).toHaveBeenCalledWith("tests");
  });

  it("reflects copied and busy states and prevents unusable workspace actions", () => {
    const props = renderHeader({ copied: true, openingWorkspace: true, canOpenWorkspace: false, runningEvidence: true });

    expect(screen.getByRole("button", { name: strings.workbench.component.copied })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: strings.workbench.component.useInDesign })).toBeDisabled();
    expect(screen.getByRole("button", { name: strings.workbench.component.running })).toBeDisabled();
    expect(screen.getByRole("heading", { name: asset.displayName })).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: strings.workbench.component.useInDesign }));
    expect(props.onOpenWorkspace).not.toHaveBeenCalled();
  });
});
