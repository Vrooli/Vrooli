import { create } from "@bufbuild/protobuf";
import { ComponentSchema } from "@vrooli/proto-types/react-component-library/v1/components/components_pb";
import { fireEvent, screen } from "@testing-library/react";
import { useState } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { setLocale } from "../../i18n";
import { renderWithProviders } from "../../test-utils";
import type { WorkbenchStory } from "../data";
import { DEFAULT_TEMPLATE } from "../templates/library";
import { ComponentPageInspector } from "./ComponentPageInspector";
import type { InContextReport } from "./InContextStage";

const asset = create(ComponentSchema, {
  id: "asset-1",
  libraryId: "react-component-library:Button",
  displayName: "Button",
  slug: "button",
  description: "A clear action",
  version: "1.2.0",
  latestVersion: "1.2.0",
  catalogId: "controls.button",
  sourcePath: "library/components/Button/Button.tsx",
});
const story: WorkbenchStory = {
  key: "button@1.2.0:default",
  id: "default",
  name: "Default",
  role: "primary",
  args: { disabled: false },
  fieldsJson: JSON.stringify({ fields: [{ path: "disabled", kind: "boolean" }] }),
  version: "1.2.0",
  sourcePath: "Button.tsx",
  contractId: "button@1.2.0",
};

const chosenSlot = {
  slotId: "actions",
  label: "Action area",
  role: "action" as const,
  accepts: ["button"],
  compatible: true,
  score: 92,
  rolePoints: 45,
  sizePoints: 25,
  positionPoints: 15,
  index: 0,
  box: { x: 0, y: 0, w: 180, h: 48 },
};
const contextReport: InContextReport = {
  rows: [],
  passed: 1,
  scored: 2,
  decision: {
    profile: { kind: "button", kinds: ["button"], size: { w: 140, h: 44 } },
    authoredId: "actions",
    auto: chosenSlot,
    chosen: chosenSlot,
    manual: false,
    candidates: [chosenSlot],
  },
  device: "desktop",
  viewport: 1280,
  status: "ready",
  instruments: { guides: true, overflow: true, targets: true, contrast: true, blur: false },
  storyId: story.id,
  hasStory: true,
};

function InspectorHarness({ mode = "fixtures" }: { mode?: "fixtures" | "context" }) {
  const [tab, setTab] = useState<"checks" | "props" | "usage">("checks");
  const [overrides, setOverrides] = useState<Record<string, Record<string, unknown> | null>>({});
  const onOpenWorkspace = vi.fn();
  return (
    <>
      <ComponentPageInspector
        asset={asset}
        versionsCount={3}
        mode={mode}
        selectedStory={story}
        checks={[
          {
            claim: { id: "claim", statement: "Supports keyboard input" },
            verdict: "pass",
            checkedAt: "2026-11-21T12:00:00Z",
          },
        ]}
        checkRatio={1}
        contextRatio={0.5}
        contextScored={2}
        contextRows={[
          {
            id: "readable",
            label: "Readable label",
            detail: "measured at desktop width",
            verdict: "pass",
            evidence: "measured",
            scored: true,
          },
        ]}
        contextReport={contextReport}
        inspectorTab={tab}
        setInspectorTab={setTab}
        overrides={overrides}
        setOverrides={setOverrides}
        statuses={{ default: "ready" }}
        template={DEFAULT_TEMPLATE}
        openingWorkspace={false}
        canOpenWorkspace
        onOpenWorkspace={onOpenWorkspace}
      />
      <output data-testid="override-state">{JSON.stringify(overrides.default ?? null)}</output>
    </>
  );
}

describe("ComponentPageInspector", () => {
  beforeEach(async () => {
    // The RCL Tabs usage contract limits visible labels to 24 characters.
    // cimode renders translation keys as labels, which are not representative
    // of the UI and can exceed that contract. Exercise this component with its
    // actual English copy, as other component behavior suites do.
    await setLocale("en");
  });

  it("shows fixture checks, edits a selected story, and reports fixture metadata", () => {
    renderWithProviders(<InspectorHarness />);

    expect(document.querySelector(".wb-checkrow")).toHaveTextContent("Supports keyboard input");
    fireEvent.click(screen.getByRole("tab", { name: "Props" }));
    fireEvent.change(screen.getByLabelText(/^disabled$/), { target: { value: "true" } });
    fireEvent.click(screen.getByRole("button", { name: "Apply props" }));
    expect(screen.getByTestId("override-state")).toHaveTextContent('{"disabled":true}');

    fireEvent.click(screen.getByRole("tab", { name: "Usage" }));
    expect(document.querySelectorAll(".wb-mono")[0]).toHaveTextContent(asset.catalogId);
    expect(document.querySelectorAll(".wb-mono")[1]).toHaveTextContent(asset.sourcePath);
    expect(screen.getByText("3")).toBeInTheDocument();
  });

  it("shows in-context placement guidance and keeps studio access explicit", () => {
    renderWithProviders(<InspectorHarness mode="context" />);

    expect(screen.getByRole("heading", { name: asset.displayName })).toBeInTheDocument();
    expect(document.querySelector(".ic-check")).toHaveClass("is-pass");
    expect(document.querySelector(".ic-check")).toHaveTextContent("measured at desktop width");
    fireEvent.click(screen.getByRole("tab", { name: "Page" }));
    expect(document.querySelector(".wb-inspector-body")).toHaveTextContent("Action area");
    expect(document.querySelector(".wb-inspector-body")).toHaveTextContent("92%");
    expect(document.querySelector(".wb-inspector-body")).toHaveTextContent("1280 · desktop");
    expect(screen.getByRole("button", { name: "Take into studio" })).toBeEnabled();
    expect(screen.getByRole("link", { name: "Open design studio" })).toHaveAttribute(
      "href",
      "/design",
    );
  });
});
