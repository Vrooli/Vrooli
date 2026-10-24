export type CapturedMeal = {
  id: string;
  slot: string;
  recurrence: string;
  name: string;
  sourceLine: string;
  foods: string[];
};

export type RoutineCaptureReview = {
  sourceText: string;
  meals: CapturedMeal[];
  supplements: string[];
  questions: string[];
};

const SLOT_LINE = /^(breakfast|morning|lunch|midday|snack|afternoon snack|dinner|supper|evening)(?:\s*\(([^)]+)\))?\s*[:—-]\s*(.+)$/i;
const FOOD_WORDS = [
  "smoothie", "tofu", "beans", "bean", "lentils", "lentil", "rice", "pasta",
  "oats", "oatmeal", "yogurt", "eggs", "egg", "chicken", "fish", "salad",
  "soup", "bread", "nuts", "fruit", "vegetables", "crunchy snacks",
];

function slotName(value: string): string {
  const normalized = value.toLocaleLowerCase();
  if (normalized === "morning") return "Breakfast";
  if (normalized === "midday") return "Lunch";
  if (normalized === "afternoon snack") return "Snack";
  if (normalized === "supper" || normalized === "evening") return "Dinner";
  return normalized.charAt(0).toLocaleUpperCase() + normalized.slice(1);
}

function detectedFoods(line: string): string[] {
  const lower = line.toLocaleLowerCase();
  return FOOD_WORDS.filter((food) => lower.includes(food));
}

export function reviewRoutine(sourceText: string): RoutineCaptureReview {
  const sourceLines = sourceText.split(/\r?\n/).map((line) => line.trim()).filter(Boolean);
  const meals: CapturedMeal[] = [];
  const supplements: string[] = [];

  for (const line of sourceLines) {
    const match = SLOT_LINE.exec(line);
    if (match) {
      const rawSlot = match[1];
      const rawRecurrence = match[2];
      const description = match[3];
      if (!rawSlot || !description) continue;
      meals.push({
        id: `meal-${meals.length + 1}`,
        slot: slotName(rawSlot),
        recurrence: rawRecurrence?.trim() || "recurrence not stated",
        name: description.trim(),
        sourceLine: line,
        foods: detectedFoods(description),
      });
      continue;
    }
    if (/\b(supplement|vitamin|mineral|capsule|tablet)\b/i.test(line)) supplements.push(line);
  }

  const questions = meals.map((meal) => `What quantity and basis should ${meal.slot.toLocaleLowerCase()} use?`);
  const names = new Map<string, number>();
  for (const meal of meals) names.set(meal.name.toLocaleLowerCase(), (names.get(meal.name.toLocaleLowerCase()) || 0) + 1);
  if ([...names.values()].some((count) => count > 1)) {
    questions.push("Are repeated meal names separate meals or the same meal described twice?");
  }
  if (supplements.length) {
    questions.push("Supplement product and dose are not stated; keep the schedule unconfigured until you provide them.");
  }

  return { sourceText, meals, supplements, questions };
}
