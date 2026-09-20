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
