const availableMealSlots = ["breakfast", "lunch", "dinner"] as const;

export function mealSlotsFromOnboardingDraft(draftJson?: string): string[] {
  if (!draftJson) return [];
  try {
    const draft = JSON.parse(draftJson) as { mealSlots?: unknown };
    return Array.isArray(draft.mealSlots)
      ? draft.mealSlots.filter((slot): slot is string => typeof slot === "string" && availableMealSlots.includes(slot as typeof availableMealSlots[number]))
      : [];
  } catch {
    return [];
  }
}

export function firstWeekSlots(dates: string[], selectedSlots: string[]) {
  const slots = selectedSlots.length ? selectedSlots : ["dinner"];
  return dates.flatMap((date) => slots.map((slotName) => ({ date, slotName, mode: "flexible", quantity: "1" })));
}
