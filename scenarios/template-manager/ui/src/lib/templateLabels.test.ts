import { describe, expect, it } from "vitest";

import {
  debtStatusTone,
  driftTone,
  kindLabel,
  modeLabel,
  runStatusTone,
  severityTone,
} from "./templateLabels";

describe("templateLabels", () => {
  it("maps template kinds", () => {
    expect(kindLabel(1)).toBe("scenario");
    expect(kindLabel(2)).toBe("design");
    expect(kindLabel(3)).toBe("resource");
    expect(kindLabel(0)).toBe("template");
  });

  it("maps validation modes", () => {
    expect(modeLabel(1)).toBe("shallow");
    expect(modeLabel(2)).toBe("deep");
    expect(modeLabel(3)).toBe("drift");
    expect(modeLabel(99)).toBe("validation");
  });

  it("tones run statuses", () => {
    expect(runStatusTone("passed")).toBe("success");
    expect(runStatusTone("failed")).toBe("danger");
    expect(runStatusTone("running")).toBe("info");
    expect(runStatusTone("weird")).toBe("warning");
  });

  it("tones debt statuses", () => {
    expect(debtStatusTone("open")).toBe("danger");
    expect(debtStatusTone("resolved")).toBe("success");
    expect(debtStatusTone("acknowledged")).toBe("warning");
    expect(debtStatusTone("unknown")).toBe("neutral");
  });

  it("tones severities", () => {
    expect(severityTone("critical")).toBe("danger");
    expect(severityTone("medium")).toBe("warning");
    expect(severityTone("low")).toBe("info");
    expect(severityTone("")).toBe("neutral");
  });

  it("tones drift by count", () => {
    expect(driftTone(0)).toBe("success");
    expect(driftTone(3)).toBe("warning");
  });
});
