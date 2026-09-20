import { describe, expect, it, vi } from "vitest";

const { getForecast } = vi.hoisted(() => ({ getForecast: vi.fn() }));
vi.mock("@connectrpc/connect", () => ({ createClient: () => ({ getForecast }) }));

import { fetchForecast } from "./forecasts";

describe("forecasts api", () => {
  it("requests a bounded deterministic outlook", async () => {
    getForecast.mockResolvedValueOnce({ forecast: { resultState: "feasible_in_scenario" } });
    await expect(fetchForecast({ localDate: "2026-10-01", timezone: "UTC", horizonDays: 28 })).resolves.toEqual({ resultState: "feasible_in_scenario" });
    expect(getForecast).toHaveBeenCalledWith({ localDate: "2026-10-01", timezone: "UTC", horizonDays: 28 });
  });
});
