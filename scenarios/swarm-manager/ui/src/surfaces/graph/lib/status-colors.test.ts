import { describe, it, expect } from "vitest";
import { getStatusColorClasses, isActionableBacklogStatus, ACTIONABLE_BACKLOG_STATUSES, STATUS_GROUP_INFO } from "./status-colors";

describe("getStatusColorClasses", () => {
  it("returns neutral colors for undefined status", () => {
    const colors = getStatusColorClasses(undefined);
    expect(colors.background).toContain("slate");
    expect(colors.border).toContain("slate");
  });

  it("returns neutral colors for unknown status", () => {
    const colors = getStatusColorClasses("some_unknown_status");
    expect(colors.background).toContain("slate");
  });

  it.each(["pending", "backlog", "unknown", "stopped", "unspecified"])(
    "returns neutral colors for %s",
    (status) => {
      const colors = getStatusColorClasses(status);
      expect(colors.background).toContain("slate");
    },
  );

  it.each(["in_progress", "running", "starting", "researching", "classifying", "validating"])(
    "returns active (cyan) colors for %s",
    (status) => {
      const colors = getStatusColorClasses(status);
      expect(colors.background).toContain("cyan");
      expect(colors.border).toContain("cyan");
    },
  );

  it.each(["queued", "needs_review", "needs_fixup", "needs_review_run", "ready"])(
    "returns waiting (amber) colors for %s",
    (status) => {
      const colors = getStatusColorClasses(status);
      expect(colors.background).toContain("amber");
    },
  );

  it.each(["completed", "complete", "classified"])(
    "returns done (emerald) colors for %s",
    (status) => {
      const colors = getStatusColorClasses(status);
      expect(colors.background).toContain("emerald");
    },
  );

  it.each(["failed", "error"])(
    "returns error (red) colors for %s",
    (status) => {
      const colors = getStatusColorClasses(status);
      expect(colors.background).toContain("red");
    },
  );

  it.each(["archived", "canceled", "cancelled"])(
    "returns terminal colors for %s",
    (status) => {
      const colors = getStatusColorClasses(status);
      expect(colors.border).toContain("slate");
      // Terminal uses a more muted shade
      expect(colors.text).toContain("slate-400");
    },
  );

  it("returns all three class fields", () => {
    const colors = getStatusColorClasses("running");
    expect(colors.background).toBeTruthy();
    expect(colors.border).toBeTruthy();
    expect(colors.text).toBeTruthy();
  });
});

describe("isActionableBacklogStatus", () => {
  it.each(["backlog", "researching", "ready", "queued", "in_progress", "failed"])(
    "returns true for actionable status %s",
    (status) => {
      expect(isActionableBacklogStatus(status)).toBe(true);
    },
  );

  it.each(["completed", "archived", "running", "cancelled", undefined])(
    "returns false for non-actionable status %s",
    (status) => {
      expect(isActionableBacklogStatus(status)).toBe(false);
    },
  );

  it("ACTIONABLE_BACKLOG_STATUSES contains exactly 6 statuses", () => {
    expect(ACTIONABLE_BACKLOG_STATUSES.size).toBe(6);
  });
});

describe("STATUS_GROUP_INFO", () => {
  it("has 6 groups", () => {
    expect(STATUS_GROUP_INFO).toHaveLength(6);
  });

  it("each entry has group, label, exampleStatuses, and classes", () => {
    for (const info of STATUS_GROUP_INFO) {
      expect(info.group).toBeTruthy();
      expect(info.label).toBeTruthy();
      expect(info.exampleStatuses.length).toBeGreaterThan(0);
      expect(info.classes.background).toBeTruthy();
      expect(info.classes.border).toBeTruthy();
      expect(info.classes.text).toBeTruthy();
    }
  });
});
