import { beforeEach, describe, expect, it, vi } from "vitest";
import { exportCheckHistoryToCSV, exportTrendDataToCSV } from "./export";

describe("CSV exports", () => {
  beforeEach(() => {
    vi.stubGlobal("URL", { ...URL, createObjectURL: vi.fn(() => "blob:test"), revokeObjectURL: vi.fn() });
  });

  it("exports trends, transitions, and quoted values", () => {
    const click = vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => undefined);
    exportTrendDataToCSV({
      checkTrends: [{ checkId: "dns", total: 2, ok: 1, warning: 1, critical: 0, uptimePercent: 50, currentStatus: "warning", recentStatuses: ["ok", "warning"], lastChecked: "now" }],
      incidents: [{ timestamp: "now", checkId: "dns", fromStatus: "ok", toStatus: "warning", message: "comma, value" }],
      windowHours: 24,
      uptimePercentage: 99.5,
    });
    exportCheckHistoryToCSV("dns", [{ timestamp: "now", status: "ok", message: 'quote "value"' }]);
    expect(click).toHaveBeenCalledTimes(2);
  });
});
