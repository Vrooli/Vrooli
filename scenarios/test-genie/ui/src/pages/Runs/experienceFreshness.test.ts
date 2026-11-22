import { describe, expect, it } from "vitest";
import { experienceFreshness, experienceVerdictTone, formatExperienceStatus } from "./experienceFreshness";

describe("experienceFreshness", () => {
  it("labels a missing verdict as never run", () => {
    expect(experienceFreshness("never_run")).toBe("Never run");
  });

  it("marks a stored verdict older than the latest UI file change", () => {
    expect(experienceFreshness("older_than_last_ui_change")).toBe("Older than the last UI change");
  });

  it("uses the backend freshness classification for current verdicts", () => {
    expect(experienceFreshness("evaluated_after_last_ui_change")).toBe("Evaluated after the last UI change");
  });

  it("fails closed when freshness metadata is missing or UI time is unknown", () => {
    expect(experienceFreshness("ui_change_time_unavailable")).toBe("UI change time unavailable");
    expect(experienceFreshness()).toBe("Freshness unavailable");
    expect(experienceFreshness("ui_surface_unavailable")).toBe("UI surface unavailable");
  });
});

describe("formatExperienceStatus", () => {
  it("turns contract status values into readable labels", () => {
    expect(formatExperienceStatus("not_evaluated")).toBe("Not evaluated");
    expect(formatExperienceStatus("passed")).toBe("Passed");
    expect(formatExperienceStatus()).toBe("Unknown result");
  });
});

describe("experienceVerdictTone", () => {
  it("uses success styling only for a fresh passed verdict", () => {
    expect(experienceVerdictTone("evaluated_after_last_ui_change", "passed")).toBe("success");
  });

  it("keeps failed and not-evaluated verdicts non-successful even when their age is current", () => {
    expect(experienceVerdictTone("evaluated_after_last_ui_change", "failed")).toBe("failure");
    expect(experienceVerdictTone("evaluated_after_last_ui_change", "not_evaluated")).toBe("warning");
  });

  it("does not style a stale pass as success", () => {
    expect(experienceVerdictTone("older_than_last_ui_change", "passed")).toBe("warning");
  });
});
