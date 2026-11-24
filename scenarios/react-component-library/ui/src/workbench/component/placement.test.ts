import { describe, expect, it } from "vitest";

import { DEFAULT_TEMPLATE } from "../templates/library";
import { node, slotAccepts, slotNode, slotRole, slotsOf, type TemplateNode } from "../templates/spec";
import {
  assetPlacementProfile,
  estimateSlotBox,
  neutralizeSlots,
  placeComponent,
  slotLabel,
} from "./placement";

const page = (...children: TemplateNode[]) => node("page", "page", { pad: 28 }, children);

describe("typed slots", () => {
  it("types every authored slot and leaves legacy slots readable", () => {
    const slots = slotsOf(DEFAULT_TEMPLATE.spec);
    expect(slots.length).toBeGreaterThanOrEqual(1);
    for (const slot of slots) {
      expect(slotRole(slot)).toBeDefined();
      expect(slotAccepts(slot).length).toBeGreaterThan(0);
    }

    const legacy = node("legacy", "slot", { hint: "Old slot" });
    expect(slotRole(legacy)).toBeUndefined();
    expect(slotAccepts(legacy)).toEqual([]);
    expect(slotNode("typed", { role: "metric", accepts: ["metric"] })).toMatchObject({
      type: "slot",
      role: "metric",
      accepts: ["metric"],
    });
  });

  it("finds slots in document order and labels them from their hint", () => {
    const spec = page(
      node("row", "row", {}, [
        slotNode("first", { hint: "Header action", role: "action", accepts: ["button"] }),
      ]),
      slotNode("second", { hint: "KPI card", role: "metric", accepts: ["metric"] }),
    );
    expect(slotsOf(spec).map((slot) => slot.id)).toEqual(["first", "second"]);
    expect(slotLabel(slotsOf(spec)[1]!)).toBe("KPI card");
  });
});

describe("asset placement profile", () => {
  it("reads the accessible name before falling back to the generic card", () => {
    expect(assetPlacementProfile({ displayName: "MetricCard" }).kind).toBe("metric");
    expect(assetPlacementProfile({ displayName: "Revenue chart" }).kind).toBe("chart");
    expect(assetPlacementProfile({ displayName: "Primary Button" }).kind).toBe("button");
    expect(assetPlacementProfile({ displayName: "Monetization Account" }).kind).toBe("metric");
    expect(assetPlacementProfile({ displayName: "Something Unclassifiable" }).kind).toBe("card");
  });

  it("carries a family so a chart can also land in a metric slot", () => {
    const profile = assetPlacementProfile({ displayName: "Trend chart" });
    expect(profile.kind).toBe("chart");
    expect(profile.kinds).toContain("metric");
    expect(profile.size).toEqual({ w: 560, h: 280 });
  });
});

describe("estimateSlotBox", () => {
  it("splits the page width at grids and rows without the DOM", () => {
    const spec = page(
      node("grid", "grid", { cols: 2, gap: 14 }, [
        slotNode("left", { role: "metric", accepts: ["metric"] }),
        slotNode("right", { role: "chart", accepts: ["chart"] }),
      ]),
    );
    // 1180 - 2*28 = 1124; (1124 - 14) / 2 = 555
    expect(estimateSlotBox(spec, "left", 1180)).toEqual({ w: 555, h: 40 });
    expect(estimateSlotBox(spec, "missing", 1180)).toBeUndefined();
  });

  it("keeps the slot's declared minimum height", () => {
    const spec = page(slotNode("slot", { role: "hero", accepts: ["metric"], minH: 96 }));
    expect(estimateSlotBox(spec, "slot", 390)?.h).toBe(96);
  });
});

describe("placeComponent", () => {
  const spec = page(
    slotNode("metricSlot", { hint: "KPI card", role: "metric", accepts: ["metric", "chart"] }),
    slotNode("actionSlot", { hint: "Header action", role: "action", accepts: ["button"] }),
  );

  it("picks the declared slot that accepts the component", () => {
    const decision = placeComponent({ spec, asset: { displayName: "MetricCard" }, viewportWidth: 1180 });
    expect(decision.chosen?.slotId).toBe("metricSlot");
    expect(decision.chosen?.compatible).toBe(true);
    expect(decision.authoredId).toBe("metricSlot");
    expect(decision.manual).toBe(false);
    expect(decision.candidates.find((row) => row.slotId === "actionSlot")?.compatible).toBe(false);
  });

  it("is deterministic for the same asset, page and width", () => {
    const first = placeComponent({ spec, asset: { displayName: "MetricCard" }, viewportWidth: 834 });
    const second = placeComponent({ spec, asset: { displayName: "MetricCard" }, viewportWidth: 834 });
    expect(second.candidates).toEqual(first.candidates);
    expect(second.chosen).toEqual(first.chosen);
  });

  it("ranks a compatible slot above an incompatible one even when later", () => {
    const decision = placeComponent({ spec, asset: { displayName: "Button" }, viewportWidth: 1180 });
    const action = decision.candidates.find((row) => row.slotId === "actionSlot");
    const metric = decision.candidates.find((row) => row.slotId === "metricSlot");
    expect(action?.compatible).toBe(true);
    expect(metric?.compatible).toBe(false);
    expect(action!.score).toBeGreaterThan(metric!.score);
  });

  it("honours a manual choice without losing the automatic best", () => {
    const decision = placeComponent({
      spec,
      asset: { displayName: "MetricCard" },
      viewportWidth: 1180,
      preferredSlotId: "actionSlot",
    });
    expect(decision.chosen?.slotId).toBe("actionSlot");
    expect(decision.manual).toBe(true);
    expect(decision.auto?.slotId).toBe("metricSlot");
  });

  it("treats a pin on the best slot as manual so it can be reset", () => {
    const decision = placeComponent({
      spec,
      asset: { displayName: "MetricCard" },
      viewportWidth: 1180,
      preferredSlotId: "metricSlot",
    });
    expect(decision.chosen?.slotId).toBe("metricSlot");
    expect(decision.manual).toBe(true);
  });

  it("falls back to auto when the manual slot is gone", () => {
    const decision = placeComponent({
      spec,
      asset: { displayName: "MetricCard" },
      viewportWidth: 1180,
      preferredSlotId: "deletedSlot",
    });
    expect(decision.chosen?.slotId).toBe("metricSlot");
    expect(decision.manual).toBe(false);
  });

  it("reports no slot rather than inventing one", () => {
    const empty = page(node("copy", "text", { text: "No slot here" }));
    const decision = placeComponent({ spec: empty, asset: { displayName: "MetricCard" }, viewportWidth: 1180 });
    expect(decision.chosen).toBeUndefined();
    expect(decision.candidates).toEqual([]);
  });

  it("prefers the slot that names the component's role when the total ties", () => {
    // Role specificity wins when geometry estimates tie.
    const spec = page(
      slotNode("actionSlot", { hint: "Header action", role: "action", accepts: ["button"] }),
      node("stats", "grid", { cols: 4, gap: 12 }, [
        slotNode("metricSlot", { hint: "KPI card", role: "metric", accepts: ["metric"] }),
      ]),
    );

    for (const viewportWidth of [390, 834, 1180]) {
      const decision = placeComponent({
        spec,
        asset: { displayName: "MonetizationAccount", description: "monthly revenue metric" },
        viewportWidth,
      });
      expect(decision.chosen?.slotId, `width ${viewportWidth}`).toBe("metricSlot");
      expect(decision.chosen?.role).toBe("metric");
    }
  });
});

describe("neutralizeSlots", () => {
  it("leaves exactly one live slot for the shared renderer to fill", () => {
    const spec = page(
      slotNode("first", { hint: "A", role: "metric", accepts: ["metric"] }),
      slotNode("second", { hint: "B", role: "action", accepts: ["button"] }),
    );
    const rendered = neutralizeSlots(spec, "second");
    expect(slotsOf(rendered).map((slot) => slot.id)).toEqual(["second"]);
    const spacer = rendered.children?.[0];
    expect(spacer?.id).toBe("first");
    expect(spacer?.type).toBe("spacer");
    expect(spacer?.props.h).toBe(40);
  });
});
