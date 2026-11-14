import { renderHook, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { appearanceAt, parseClock, readSceneryPreferences, readTransitionMinutes, useSampledSceneColors } from "./observatoryAppearance";

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe("observatory appearance transitions", () => {
  it("uses configured local-time boundaries", () => {
    expect(appearanceAt(new Date(2026, 8, 19, 6, 59), 7 * 60, 19 * 60)).toBe("night");
    expect(appearanceAt(new Date(2026, 8, 19, 7, 0), 7 * 60, 19 * 60)).toBe("day");
    expect(appearanceAt(new Date(2026, 8, 19, 19, 0), 7 * 60, 19 * 60)).toBe("night");
  });

  it("supports a daylight window that crosses midnight", () => {
    expect(appearanceAt(new Date(2026, 8, 19, 23, 0), 18 * 60, 6 * 60)).toBe("day");
    expect(appearanceAt(new Date(2026, 8, 19, 12, 0), 18 * 60, 6 * 60)).toBe("night");
  });

  it("falls back safely for malformed settings", () => {
    expect(parseClock("bad", 420)).toBe(420);
    expect(parseClock("25:00", 420)).toBe(420);
    expect(parseClock("07:30", 420)).toBe(450);
    expect(parseClock("07:99", 420)).toBe(420);
  });

  it("reads persisted transition settings without coupling the resolver to the UI", () => {
    const storage = window.localStorage;
    storage.setItem("planner.auto-day-start", "06:45");
    storage.setItem("planner.auto-night-start", "20:15");
    expect(readTransitionMinutes(storage)).toEqual({ dayStart: 405, nightStart: 1215 });
    storage.clear();
    expect(readTransitionMinutes(storage)).toEqual({ dayStart: 420, nightStart: 1140 });
  });

  it("reads scenery controls as a durable scene contract", () => {
    const storage = window.localStorage;
    storage.setItem("planner.art-free", "true");
    storage.setItem("planner.reduced-scenery", "false");
    storage.setItem("planner.subdued-night", "true");
    expect(readSceneryPreferences(storage)).toEqual({ artFree: true, reducedScenery: false, subduedNight: true });
    storage.clear();
    expect(readSceneryPreferences(storage)).toEqual({ artFree: false, reducedScenery: false, subduedNight: false });
  });

  it("shares successful panorama samples across concurrent consumers and retries failures", async () => {
    let failedDayLoads = 1;
    let missingContexts = 1;
    let imageLoads = 0;
    let activeColor: [number, number, number] = [0, 0, 0];
    const dayURL = "/public/scenes/day-panorama.webp";

    class FakeImage {
      crossOrigin: string | null = null;
      naturalWidth = 0;
      naturalHeight = 0;
      onload: ((event: Event) => void) | null = null;
      onerror: ((event: Event) => void) | null = null;
      private source = "";

      get src() { return this.source; }
      set src(url: string) {
        this.source = url;
        imageLoads += 1;
        this.naturalWidth = url === dayURL ? 3200 : 2800;
        this.naturalHeight = url === dayURL ? 1600 : 2000;
        queueMicrotask(() => {
          if (url === dayURL && failedDayLoads > 0) {
            failedDayLoads -= 1;
            this.onerror?.(new Event("error"));
            return;
          }
          this.onload?.(new Event("load"));
        });
      }
    }

    let drawCalls = 0;
    let readCalls = 0;
    const context = {
      drawImage: vi.fn((image: FakeImage) => {
        drawCalls += 1;
        activeColor = image.src === dayURL ? [40, 50, 60] : [10, 20, 30];
      }),
      getImageData: vi.fn(() => {
        readCalls += 1;
        const data = new Uint8ClampedArray(32 * 8 * 4);
        for (let index = 0; index < data.length; index += 4) {
          data[index] = activeColor[0];
          data[index + 1] = activeColor[1];
          data[index + 2] = activeColor[2];
          data[index + 3] = 255;
        }
        return { data };
      }),
    } as unknown as CanvasRenderingContext2D;

    vi.stubGlobal("Image", FakeImage);
    vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockImplementation(() => {
      if (missingContexts > 0) {
        missingContexts -= 1;
        return null;
      }
      return context;
    });

    const firstFailedSample = await import("./observatoryAppearance").then(({ sampleSceneAsset }) => sampleSceneAsset(dayURL));
    expect(firstFailedSample).toEqual({});
    const incompleteSample = await import("./observatoryAppearance").then(({ sampleSceneAsset }) => sampleSceneAsset(dayURL));
    expect(incompleteSample).toEqual({ aspect: 2 });

    const first = renderHook(() => useSampledSceneColors());
    const second = renderHook(() => useSampledSceneColors());
    await waitFor(() => expect(first.result.current.day).toBe("rgb(40 50 60)"));
    await waitFor(() => expect(second.result.current.night).toBe("rgb(10 20 30)"));
    expect(first.result.current).toEqual({ day: "rgb(40 50 60)", night: "rgb(10 20 30)", aspect: 2 });
    expect(second.result.current).toEqual(first.result.current);
    expect(imageLoads).toBe(4); // failed image, incomplete canvas, then one shared day/night load
    expect(drawCalls).toBe(2);
    expect(readCalls).toBe(2);

    first.unmount();
    second.unmount();
    const remounted = renderHook(() => useSampledSceneColors());
    await waitFor(() => expect(remounted.result.current.day).toBe("rgb(40 50 60)"));
    expect(remounted.result.current).toEqual({ day: "rgb(40 50 60)", night: "rgb(10 20 30)", aspect: 2 });
    expect(imageLoads).toBe(4);
    expect(drawCalls).toBe(2);
    expect(readCalls).toBe(2);
    remounted.unmount();

  });
});
