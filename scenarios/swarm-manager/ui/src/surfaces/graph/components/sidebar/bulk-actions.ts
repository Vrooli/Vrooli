export interface BulkOutcome {
  id: string;
  label: string;
  status: "success" | "failed" | "skipped";
  message?: string;
}

export async function runBulkAction<T>(
  items: T[],
  options: {
    concurrency?: number;
    getId: (item: T) => string;
    getLabel: (item: T) => string;
    run: (item: T) => Promise<unknown>;
  },
): Promise<BulkOutcome[]> {
  const concurrency = Math.max(1, options.concurrency ?? 3);
  const outcomes: BulkOutcome[] = [];
  let cursor = 0;

  async function worker() {
    while (cursor < items.length) {
      const item = items[cursor];
      cursor += 1;
      if (!item) continue;
      try {
        await options.run(item);
        outcomes.push({
          id: options.getId(item),
          label: options.getLabel(item),
          status: "success",
        });
      } catch (error) {
        outcomes.push({
          id: options.getId(item),
          label: options.getLabel(item),
          status: "failed",
          message: error instanceof Error ? error.message : "Action failed",
        });
      }
    }
  }

  await Promise.all(Array.from({ length: Math.min(concurrency, items.length) }, () => worker()));
  return outcomes;
}

export function summarizeBulkOutcomes(outcomes: BulkOutcome[]): string {
  const succeeded = outcomes.filter((outcome) => outcome.status === "success").length;
  const failed = outcomes.filter((outcome) => outcome.status === "failed").length;
  const skipped = outcomes.filter((outcome) => outcome.status === "skipped").length;
  const parts = [`${succeeded} succeeded`];
  if (failed > 0) parts.push(`${failed} failed`);
  if (skipped > 0) parts.push(`${skipped} skipped`);
  return parts.join(", ");
}

export function failedOutcomeIds(outcomes: BulkOutcome[]): Set<string> {
  return new Set(outcomes.filter((outcome) => outcome.status === "failed").map((outcome) => outcome.id));
}

