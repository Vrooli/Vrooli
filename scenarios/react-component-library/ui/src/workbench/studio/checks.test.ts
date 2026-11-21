import { describe, expect, it } from "vitest";

import { node } from "../templates/spec";
import {
  TOUCH_TARGET_MIN,
  buttonEstimates,
  captureEvidenceVerdict,
  declaredButtonHeight,
  emptyAudit,
  layoutVerdicts,
  paintedShare,
  primaryActionVerdict,
  stateAudit,
  underTargetButtons,
} from "./checks";

const spec = node("page", "page", {}, [
  node("stat1", "stat", { label: "Active accounts", value: "12,480" }),
  node("stat2", "stat", { label: "Expansion", value: "31%", states: "empty,loading" }),
  node("slot", "slot", { hint: "Primary action" }),
  node("button", "button", { text: "Open", size: "sm" }),
  node("big", "button", { text: "Present", size: "lg" }),
  node("copy", "text", { text: "Body" }),
]);

describe("layout checks", () => {
  it("reports data layers with no declared states", () => {
    const audit = stateAudit(spec);
    expect(audit.total).toBe(5);
    expect(audit.covered).toBe(1);
    expect(audit.missing).toEqual(["Active accounts", "Primary action", "Open", "Present"]);
  });

  it("estimates button heights from the declared size", () => {
    expect(declaredButtonHeight("sm")).toBe(32);
    expect(declaredButtonHeight("lg")).toBe(48);
    expect(declaredButtonHeight(undefined)).toBe(40);
    expect(buttonEstimates(spec).map((button) => button.height)).toEqual([32, 48]);
    expect(underTargetButtons(spec).map((button) => button.label)).toEqual(["Open"]);
    expect(TOUCH_TARGET_MIN).toBe(44);
  });

  it("treats unmeasured frames honestly", () => {
    const frames = [{ id: "desktop", name: "Desktop", device: "desktop" as const, spec }];
    const audit = emptyAudit(frames);
    expect(audit.desktop).toEqual({ overflow: 0, buttons: null, painted: null, measured: false });
    expect(layoutVerdicts(audit.desktop, 2)).toEqual({
      overflow: "unmeasured",
      targets: "unmeasured",
      states: "warn",
    });
  });

  it("fails overflow, warns on small targets, and passes clean frames", () => {
    expect(
      layoutVerdicts(
        { overflow: 12, buttons: { total: 2, under: 0 }, painted: null, measured: true },
        0,
      ),
    ).toEqual({ overflow: "fail", targets: "pass", states: "pass" });
    expect(
      layoutVerdicts(
        { overflow: 0, buttons: { total: 2, under: 1 }, painted: null, measured: true },
        3,
      ),
    ).toEqual({ overflow: "pass", targets: "warn", states: "warn" });
  });

  it("reports painted-area share only from measured frames", () => {
    expect(paintedShare({})).toBeNull();
    expect(
      paintedShare({
        desktop: { overflow: 0, buttons: null, painted: { rcl: 30, total: 100 }, measured: true },
        phone: { overflow: 0, buttons: null, painted: { rcl: 10, total: 100 }, measured: true },
      }),
    ).toEqual({ rcl: 40, total: 200, ratio: 0.2 });
    expect(
      paintedShare({
        desktop: { overflow: 0, buttons: null, painted: { rcl: 0, total: 0 }, measured: true },
      }),
    ).toBeNull();
  });

  it("never passes a claim the design does not evidence", () => {
    expect(primaryActionVerdict(0)).toBe("warn");
    expect(primaryActionVerdict(1)).toBe("pass");
    expect(captureEvidenceVerdict(false)).toBe("unmeasured");
    expect(captureEvidenceVerdict(true)).toBe("pass");
  });
});
