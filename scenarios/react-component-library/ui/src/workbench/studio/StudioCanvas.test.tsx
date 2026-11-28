import { fireEvent, screen } from "@testing-library/react";
import { createRef } from "react";
import { describe, expect, it, vi } from "vitest";

import { renderWithProviders } from "../../test-utils";
import { DEFAULT_TEMPLATE } from "../templates/library";
import { seedFrames } from "./model";
import { emptyAudit } from "./checks";
import { StudioCanvas } from "./StudioCanvas";
import type { DesignDocument } from "../design-types";

describe("StudioCanvas coverage lens", () => {
  it("toggles the RCL/custom map on the rendered frame", () => {
    const frames = seedFrames(DEFAULT_TEMPLATE).filter((frame) => frame.device === "phone");
    const phone = frames[0];
    if (!phone) throw new Error("The default template has no phone frame.");
    const design: DesignDocument = {
      id: "design-test",
      name: "Test",
      updatedAt: "2026-01-01T00:00:00Z",
      revision: 1,
      frames,
      theme: DEFAULT_TEMPLATE.palette,
      options: { radius: DEFAULT_TEMPLATE.radius, display: DEFAULT_TEMPLATE.display },
    };
    renderWithProviders(
      <StudioCanvas
        design={design}
        frames={frames}
        activeFrameId={phone.id}
        selectedId={null}
        scales={{ [phone.id]: 1 }}
        guides={false}
        onToggleGuides={vi.fn()}
        zoomOverride={null}
        onZoomChange={vi.fn()}
        fitScale={1}
        audit={emptyAudit(frames)}
        slot={null}
        slotEditable={false}
        onPlacementChange={vi.fn()}
        onSelectFrame={vi.fn()}
        onSelectNode={vi.fn()}
        onMoveLayer={vi.fn()}
        onAddFrame={vi.fn()}
        onRemoveFrame={vi.fn()}
        scrollRef={createRef<HTMLDivElement>()}
        saving={false}
      />,
    );

    const toggle = screen.getByRole("button", { name: "workbench.studio.coverageOverlay" });
    expect(toggle).toHaveAttribute("aria-pressed", "false");
    fireEvent.click(toggle);

    expect(toggle).toHaveAttribute("aria-pressed", "true");
    expect(
      screen.getByRole("group", { name: "workbench.studio.coverageLegend" }),
    ).toBeInTheDocument();
    expect(document.querySelector('.tpl[data-coverage-mode="true"]')).not.toBeNull();
  });
});
