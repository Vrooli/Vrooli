/**
 * Tests for OpsTriggerButton.
 *
 * Pins the contracts that matter for both layout consumers:
 *  - Always-shown — count zero never collapses the trigger.
 *  - Idle vs active styling — the count drives the visual state.
 *  - Plural-correct label — "1 agent" / "0 agents" / "5 agents".
 *  - Navigation — middle/modifier-click works because it's a `<Link>`.
 *  - Both variants render with the same `data-testid` so workflow tooling
 *    locates the trigger regardless of context.
 *  - Live updates — store mutations propagate through the selector.
 */

import { afterEach, describe, expect, it } from "vitest";
import { act, cleanup, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { OpsTriggerButton } from "./OpsTriggerButton";
import { selectors } from "../../consts/selectors";
import { useOperationsStore } from "../../stores/operations-store";
import type { OperationsView } from "../../types/operations";

function makeView(activitiesCount: number): OperationsView {
  const activities = Array.from({ length: activitiesCount }, (_, i) => ({
    activityId: `a-${i}`,
    runId: `run-${i}`,
    ownerType: "backlog",
    ownerName: `item-${i}`,
    purpose: "process",
    status: "running",
    requestedAt: "2026-05-02T01:00:00Z",
  }));

  return {
    lanes: [
      { lane: "investigate", active: 0, capacity: 6, queue: 0 },
      { lane: "execute", active: 0, capacity: 3, queue: 0 },
      { lane: "review", active: 0, capacity: 8, queue: 0 },
      { lane: "reconcile", active: 0, capacity: 2, queue: 0 },
    ],
    queue: { depth: 0, maxDepth: 0 },
    activities,
    recentlyFinished: [],
    generatedAt: "2026-05-02T01:00:00Z",
    windowSeconds: 10800,
  };
}

function setStoreCount(count: number): void {
  useOperationsStore.setState({ view: makeView(count) });
}

function renderButton(variant: "compact" | "hud", className?: string) {
  return render(
    <MemoryRouter>
      <OpsTriggerButton variant={variant} className={className} />
    </MemoryRouter>,
  );
}

afterEach(() => {
  cleanup();
  useOperationsStore.getState().reset();
});

describe("OpsTriggerButton", () => {
  describe("compact variant", () => {
    it("always renders even when no activities are running", () => {
      setStoreCount(0);
      renderButton("compact");

      const trigger = screen.getByTestId(selectors.layout.opsTriggerButton);
      expect(trigger).toBeInTheDocument();
      expect(trigger).toHaveTextContent("0 agents");
      expect(trigger.getAttribute("data-variant")).toBe("compact");
    });

    it("uses singular 'agent' when exactly one is active", () => {
      setStoreCount(1);
      renderButton("compact");

      expect(
        screen.getByTestId(selectors.layout.opsTriggerButton),
      ).toHaveTextContent("1 agent");
    });

    it("uses plural 'agents' for multiple active runs", () => {
      setStoreCount(7);
      renderButton("compact");

      expect(
        screen.getByTestId(selectors.layout.opsTriggerButton),
      ).toHaveTextContent("7 agents");
    });

    it("idles in slate when count is zero, glows emerald when active", () => {
      setStoreCount(0);
      renderButton("compact");

      const idle = screen.getByTestId(selectors.layout.opsTriggerButton);
      expect(idle.className).toContain("bg-slate-800/60");
      expect(idle.className).not.toContain("bg-emerald-500/15");

      act(() => {
        setStoreCount(2);
      });

      const active = screen.getByTestId(selectors.layout.opsTriggerButton);
      expect(active.className).toContain("bg-emerald-500/15");
    });

    it("links to /graph/plan and exposes an a11y label", () => {
      setStoreCount(3);
      renderButton("compact");

      const trigger = screen.getByTestId(selectors.layout.opsTriggerButton);
      expect(trigger.tagName).toBe("A");
      expect(trigger.getAttribute("href")).toBe("/graph/plan");
      expect(trigger.getAttribute("aria-label")).toBe(
        "Operations Center · 3 agents",
      );
    });
  });

  describe("hud variant", () => {
    it("always renders with the same selector as the compact variant", () => {
      setStoreCount(0);
      renderButton("hud");

      const trigger = screen.getByTestId(selectors.layout.opsTriggerButton);
      expect(trigger).toBeInTheDocument();
      expect(trigger.getAttribute("data-variant")).toBe("hud");
    });

    it("renders the count in a chip and a sm-and-up plural label", () => {
      setStoreCount(4);
      renderButton("hud");

      const trigger = screen.getByTestId(selectors.layout.opsTriggerButton);
      // The chip carries the bare count.
      expect(trigger).toHaveTextContent("4");
      // The plural-aware label sits next to it (hidden on small screens via CSS only).
      expect(trigger).toHaveTextContent("agents");
    });

    it("links to /graph/plan", () => {
      setStoreCount(2);
      renderButton("hud");

      const trigger = screen.getByTestId(selectors.layout.opsTriggerButton);
      expect(trigger.tagName).toBe("A");
      expect(trigger.getAttribute("href")).toBe("/graph/plan");
    });

    it("forwards className so callers can apply responsive visibility utilities", () => {
      setStoreCount(0);
      renderButton("hud", "md:hidden");

      const trigger = screen.getByTestId(selectors.layout.opsTriggerButton);
      expect(trigger.className).toContain("md:hidden");
    });
  });

  it("reflects store updates without a remount", () => {
    setStoreCount(0);
    renderButton("hud");

    expect(
      screen.getByTestId(selectors.layout.opsTriggerButton),
    ).toHaveTextContent("0");

    act(() => {
      setStoreCount(5);
    });

    expect(
      screen.getByTestId(selectors.layout.opsTriggerButton),
    ).toHaveTextContent("5");
  });
});
