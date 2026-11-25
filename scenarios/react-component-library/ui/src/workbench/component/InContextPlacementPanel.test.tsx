import { fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { cleanup } from "@testing-library/react";

import { InContextPlacementPanel } from "./InContextPlacementPanel";
import type { PlacementDecision, SlotScore } from "./placement";

function candidate(overrides: Partial<SlotScore> & Pick<SlotScore, "slotId" | "label">): SlotScore {
  return {
    role: "action",
    accepts: ["button"],
    compatible: true,
    score: 80,
    rolePoints: 34,
    sizePoints: 25,
    positionPoints: 15,
    roleMatch: "family",
    index: 0,
    box: { w: 300, h: 44 },
    ...overrides,
  };
}

function decision(overrides: Partial<PlacementDecision> = {}): PlacementDecision {
  const candidates = [
    candidate({ slotId: "action", label: "Header action" }),
    candidate({ slotId: "metric", label: "Metric region", role: "metric", roleMatch: "accepted", index: 1 }),
    candidate({ slotId: "list", label: "List region", role: "list", roleMatch: "accepted", index: 2 }),
    candidate({ slotId: "hero", label: "Hero region", role: "hero", roleMatch: "mismatch", compatible: false, index: 3 }),
  ];
  return {
    profile: { kind: "button", kinds: ["button", "action"], size: { w: 140, h: 44 } },
    authoredId: "action",
    auto: candidates[0],
    chosen: candidates[0],
    manual: false,
    candidates,
    ...overrides,
  };
}

afterEach(() => cleanup());

describe("InContextPlacementPanel", () => {
  it("shows the top three candidates and reveals the rest on request", () => {
    render(<InContextPlacementPanel decision={decision()} slotHeight={44} onSlotChoice={vi.fn()} />);
    expect(screen.queryByTestId("slot-candidate-hero")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "workbench.component.ctxShowAll" }));
    expect(screen.getByTestId("slot-candidate-hero")).toBeInTheDocument();
  });

  it("warns when a slot accepts the kind but is not the natural role", () => {
    const region = candidate({
      slotId: "region",
      label: "Live component",
      role: "hero",
      roleMatch: "accepted",
    });
    render(
      <InContextPlacementPanel
        decision={decision({ auto: region, chosen: region, candidates: [region] })}
        slotHeight={44}
        onSlotChoice={vi.fn()}
      />,
    );
    expect(screen.getByRole("status")).toHaveTextContent("workbench.component.ctxRoleWarning");
    expect(screen.getAllByTestId("slot-match")[0]).toHaveTextContent("workbench.component.ctxMatchAccepted");
  });

  it("does not warn when the role is a natural home for the kind", () => {
    render(<InContextPlacementPanel decision={decision()} slotHeight={44} onSlotChoice={vi.fn()} />);
    expect(screen.queryByTestId("slot-role-warning")).not.toBeInTheDocument();
    expect(screen.getAllByTestId("slot-match")[0]).toHaveTextContent("workbench.component.ctxMatchFamily");
  });
});
