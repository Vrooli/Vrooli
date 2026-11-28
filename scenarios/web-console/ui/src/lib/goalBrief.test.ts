import { describe, expect, it } from "vitest";
import { buildGoalBrief, pinnedStanceError, validateGoalBrief } from "@vrooli/goal-brief";

describe("shared goal brief rules", () => {
  it("preserves the full stance and emits a bounded kernel with a charter", () => {
    const stance = "the operator's exact words stay intact";
    const draft = buildGoalBrief({ destination: "Finish the workflow", stance, charterPath: "work/CHARTER.md" });
    expect(validateGoalBrief(draft)).toEqual([]);
    expect(draft.stanceMessage).toContain(stance);
    expect(Array.from(draft.kernel).length).toBeLessThanOrEqual(600);
    expect(draft.kernel).toContain("work/CHARTER.md");
  });

  it("rejects a changed stance and overlong kernel", () => {
    const draft = buildGoalBrief({ destination: "😀".repeat(700), stance: "keep these words", charterPath: "work/CHARTER.md" });
    expect(validateGoalBrief({ ...draft, stanceMessage: "changed" })).toContain("The stance message changed the exact stance.");
    expect(validateGoalBrief(draft)).toContain("Kernel is longer than 600 characters.");
  });

  it("checks exact pin payload against the transport byte limit", () => {
    expect(pinnedStanceError("Keep the exact line.", 4096)).toBeNull();
    expect(pinnedStanceError("\nKeep the exact line.", 4096)).toContain("one line");
    expect(pinnedStanceError("😀".repeat(1025), 4096)).toContain("4096-byte");
  });
});
