import { describe, expect, it } from "vitest";
import { itemActionsFromNextAction } from "./backlog-queue-utils";

const item = { kind: "idea" as const, name: "test-item", status: "ready" as const, dependsOn: [] };

describe("itemActionsFromNextAction", () => {
  it("uses the server projection instead of deriving a runnable CTA locally", () => {
    const result = itemActionsFromNextAction(
      item,
      {
        id: "accept_plan",
        compactLabel: "Accept plan",
        expandedLabel: "Accept the plan",
        enabled: true,
        blockers: [],
      },
    );

    expect(result.canRun).toBe(false);
    expect(result.primaryCta).toBeNull();
  });

  it("keeps a server-projected run disabled with its reason", () => {
    const result = itemActionsFromNextAction(
      item,
      {
        id: "run",
        compactLabel: "Run",
        expandedLabel: "Run now",
        enabled: false,
        reason: "A dependency must finish first.",
        blockers: [{ code: "dependency_unmet", message: "A dependency must finish first.", forceable: false }],
      },
    );

    expect(result.runDisabled).toBe(true);
    expect(result.disabledReason).toMatch(/dependency/i);
  });
});
