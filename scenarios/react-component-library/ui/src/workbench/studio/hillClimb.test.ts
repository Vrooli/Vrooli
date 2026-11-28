import { describe, expect, it } from "vitest";
import { hillClimbSteps } from "./hillClimb";
import { emptyAudit } from "./checks";
import { node } from "../templates/spec";
import { frameCoverage } from "./rcl";

describe("hillClimbSteps", () => {
  it("ranks measured usability findings and keeps unavailable evidence explicit", () => {
    const frames = [{ id: "phone", name: "Phone" }];
    const audits = emptyAudit(frames as never);
    audits.phone = { measured: true, overflow: 12, buttons: { total: 2, under: 1 }, painted: null };
    const steps = hillClimbSteps({
      coverage: frameCoverage(
        node("page", "page", {}, [node("copy", "text", { text: "Confirm" })]),
      ),
      area: null,
      audits,
      frameNames: frames,
      selectedIsCustom: true,
      stateGap: { total: 1, covered: 0, missing: ["Search"] },
      claims: [{ id: "capture", verdict: "unmeasured", detail: "No capture" }],
    });
    expect(steps.map((step) => step.id)).toEqual([
      "overflow",
      "targets",
      "states",
      "rcl",
      "evidence-capture",
    ]);
    expect(steps[0]).toMatchObject({ action: "overflow-frame", frameId: "phone" });
    expect(steps.find((step) => step.id === "rcl")?.action).toBe("replace");
  });

  it("does not infer painted share or issue a quality score when measurements are absent", () => {
    const frames = [{ id: "wide", name: "Wide" }];
    const steps = hillClimbSteps({
      coverage: { rcl: 1, custom: 1, total: 2, ratio: 0.5 },
      area: null,
      audits: emptyAudit(frames as never),
      frameNames: frames,
      selectedIsCustom: false,
      stateGap: { total: 0, covered: 0, missing: [] },
      claims: [],
    });
    expect(steps.map((step) => step.id)).toEqual(["rcl"]);
    expect(steps[0]?.detail).toContain("Select a custom layer");
  });
});
