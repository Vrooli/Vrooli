import { useEffect, useState } from "react";

export type ObservatoryAppearance = "day" | "night";

export const OBSERVATORY_DAY_START_KEY = "planner.auto-day-start";
export const OBSERVATORY_NIGHT_START_KEY = "planner.auto-night-start";
export const OBSERVATORY_PREFERENCES_EVENT = "planner.appearance-preferences-changed";
export const OBSERVATORY_SCENERY_EVENT = "planner.scenery-preferences-changed";

export type ObservatorySceneryPreferences = {
  artFree: boolean;
  reducedScenery: boolean;
  subduedNight: boolean;
};

export function readSceneryPreferences(storage: Storage | undefined = typeof window === "undefined" ? undefined : window.localStorage): ObservatorySceneryPreferences {
  return {
    artFree: storage?.getItem("planner.art-free") === "true",
    reducedScenery: storage?.getItem("planner.reduced-scenery") === "true",
    subduedNight: storage?.getItem("planner.subdued-night") === "true",
  };
}

export const DEFAULT_TRANSITION_HOURS = { dayStart: "07:00", nightStart: "19:00" } as const;

export function parseClock(value: string, fallback: number): number {
  const match = /^(\d{2}):(\d{2})$/.exec(value);
  if (!match) return fallback;
  const hour = Number(match[1]);
  const minute = Number(match[2]);
  if (hour > 23 || minute > 59) return fallback;
  return hour * 60 + minute;
}

export function readTransitionMinutes(storage: Storage | undefined = typeof window === "undefined" ? undefined : window.localStorage) {
  return {
    dayStart: parseClock(storage?.getItem(OBSERVATORY_DAY_START_KEY) ?? DEFAULT_TRANSITION_HOURS.dayStart, 7 * 60),
    nightStart: parseClock(storage?.getItem(OBSERVATORY_NIGHT_START_KEY) ?? DEFAULT_TRANSITION_HOURS.nightStart, 19 * 60),
  };
}

export function appearanceAt(date: Date, dayStart: number, nightStart: number): ObservatoryAppearance {
  const minutes = date.getHours() * 60 + date.getMinutes();
  if (dayStart === nightStart) return "day";
  if (dayStart < nightStart) return minutes >= dayStart && minutes < nightStart ? "day" : "night";
  return minutes >= dayStart || minutes < nightStart ? "day" : "night";
}

export const SCENE_DAY_URL = "/public/scenes/day-panorama.webp";
export const SCENE_NIGHT_URL = "/public/scenes/night-panorama.webp";

export type SampledScene = { color?: string; aspect?: number };

/**
 * Reads the actual scenery asset and returns (a) the average colour of its top
 * "sky handoff" strip and (b) its intrinsic aspect ratio. The Today sky uses
 * these to derive its own gradient and band height from the image itself, so a
 * regenerated or slightly-different panorama can never leave a mismatched seam.
 */
export function sampleSceneAsset(url: string): Promise<SampledScene> {
  return new Promise((resolve) => {
    if (typeof document === "undefined" || typeof Image === "undefined") {
      resolve({});
      return;
    }
    const img = new Image();
    img.crossOrigin = "anonymous";
    img.onload = () => {
      const aspect = img.naturalHeight > 0 ? img.naturalWidth / img.naturalHeight : undefined;
      try {
        const cw = 32;
        const ch = 8;
        const canvas = document.createElement("canvas");
        canvas.width = cw;
        canvas.height = ch;
        const ctx = canvas.getContext("2d");
        if (!ctx) {
          resolve({ aspect });
          return;
        }
        // Sample only the top ~6% strip (the flat sky the panorama hands off to us).
        const stripHeight = Math.max(1, Math.round(img.naturalHeight * 0.06));
        ctx.drawImage(img, 0, 0, img.naturalWidth, stripHeight, 0, 0, cw, ch);
        const { data } = ctx.getImageData(0, 0, cw, ch);
        let r = 0;
        let g = 0;
        let b = 0;
        let n = 0;
        for (let i = 0; i < data.length; i += 4) {
          r += data[i] ?? 0;
          g += data[i + 1] ?? 0;
          b += data[i + 2] ?? 0;
          n += 1;
        }
        if (!n) {
          resolve({ aspect });
          return;
        }
        resolve({ color: `rgb(${Math.round(r / n)} ${Math.round(g / n)} ${Math.round(b / n)})`, aspect });
      } catch {
        // Cross-origin taint or decode failure: fall back to the palette, keep the aspect if we got it.
        resolve({ aspect });
      }
    };
    img.onerror = () => resolve({});
    img.src = url;
  });
}

export type SampledSceneColors = { day?: string; night?: string; aspect?: number };

/** Derives the seam colours (day + night) and band aspect from the live scenery assets. */
export function useSampledSceneColors(): SampledSceneColors {
  const [colors, setColors] = useState<SampledSceneColors>({});
  useEffect(() => {
    let active = true;
    Promise.all([sampleSceneAsset(SCENE_DAY_URL), sampleSceneAsset(SCENE_NIGHT_URL)]).then(([day, night]) => {
      if (!active) return;
      setColors({ day: day.color, night: night.color, aspect: day.aspect ?? night.aspect });
    });
    return () => {
      active = false;
    };
  }, []);
  return colors;
}

export function useObservatoryAutoAppearance(): ObservatoryAppearance {
  const [now, setNow] = useState(() => new Date());
  const [transition, setTransition] = useState(() => readTransitionMinutes());
  const desired = appearanceAt(now, transition.dayStart, transition.nightStart);

  useEffect(() => {
    const timer = window.setInterval(() => setNow(new Date()), 60_000);
    return () => window.clearInterval(timer);
  }, []);

  useEffect(() => {
    const refresh = () => setTransition(readTransitionMinutes());
    window.addEventListener("storage", refresh);
    window.addEventListener(OBSERVATORY_PREFERENCES_EVENT, refresh);
    return () => {
      window.removeEventListener("storage", refresh);
      window.removeEventListener(OBSERVATORY_PREFERENCES_EVENT, refresh);
    };
  }, []);

  return desired;
}
