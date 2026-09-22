import { describe, expect, it, vi } from "vitest";

const { getForecast, listForecastSnapshots } = vi.hoisted(() => ({ getForecast: vi.fn(), listForecastSnapshots: vi.fn() }));
vi.mock("@connectrpc/connect", () => ({ createClient: () => ({ getForecast, listForecastSnapshots }) }));

import { fetchForecast } from "./forecasts";

describe("forecasts api", () => {
  it("requests a bounded deterministic outlook", async () => {
    getForecast.mockResolvedValueOnce({ forecast: { resultState: "feasible_in_scenario" } });
    await expect(fetchForecast({ localDate: "2026-10-01", timezone: "UTC", horizonDays: 28 })).resolves.toEqual({ resultState: "feasible_in_scenario" });
    expect(getForecast).toHaveBeenCalledWith({ localDate: "2026-10-01", timezone: "UTC", horizonDays: 28 });
  });

  it("reads bounded outlook history and rejects empty forecast responses", async () => {
    listForecastSnapshots.mockResolvedValueOnce({ snapshots: [{ id: "snapshot-1" }] });
    const { fetchForecastHistory } = await import("./forecasts");
    await expect(fetchForecastHistory(4)).resolves.toEqual([{ id: "snapshot-1" }]);
    expect(listForecastSnapshots).toHaveBeenCalledWith({ limit: 4 });
    getForecast.mockResolvedValueOnce({});
    await expect((await import("./forecasts")).fetchForecast()).rejects.toThrow("no forecast");
  });
});
