import { describe, expect, it } from "vitest";
import { reviewRoutine } from "./routineCapture";

describe("reviewRoutine", () => {
  it("preserves source, detects only stated meals, slots, foods and supplements, and leaves missing facts unknown", () => {
    const source = "Breakfast (daily): smoothie\nLunch (weekdays): tofu meal\nSnack (most afternoons): crunchy snacks\nSupplement: recurring, product and dose not recorded";
    const review = reviewRoutine(source);

    expect(review.sourceText).toBe(source);
    expect(review.meals.map(({ slot, recurrence, name, foods }) => ({ slot, recurrence, name, foods }))).toEqual([
      { slot: "Breakfast", recurrence: "daily", name: "smoothie", foods: ["smoothie"] },
      { slot: "Lunch", recurrence: "weekdays", name: "tofu meal", foods: ["tofu"] },
      { slot: "Snack", recurrence: "most afternoons", name: "crunchy snacks", foods: ["crunchy snacks"] },
    ]);
    expect(review.supplements).toEqual(["Supplement: recurring, product and dose not recorded"]);
    expect(review.questions).toHaveLength(4);
    expect(review.questions[0]).toContain("quantity and basis");
    expect(review.questions[3]).toContain("dose are not stated");
  });

  it("asks whether repeated meal names are duplicates", () => {
    const review = reviewRoutine("Breakfast: oats\nLunch: oats");
    expect(review.questions).toContain("Are repeated meal names separate meals or the same meal described twice?");
  });
});
