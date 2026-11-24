import { useEffect, useRef, useState } from "react";
import type { FixturePosition } from "./FixtureCanvas";

function storageKey(id: string): string {
  return `rcl.workbench.fixtures.${id}.v1`;
}

function isPlainRecord(value: unknown): value is Record<string, unknown> {
  if (value === null || typeof value !== "object" || Array.isArray(value)) return false;
  const prototype = Object.getPrototypeOf(value) as object | null;
  return prototype === Object.prototype || prototype === null;
}

function readPositions(id: string): Record<string, FixturePosition> {
  try {
    const raw = window.localStorage.getItem(storageKey(id));
    if (!raw) return {};
    const parsed: unknown = JSON.parse(raw);
    if (!isPlainRecord(parsed)) return {};

    const entries = Object.entries(parsed);
    if (
      entries.some(
        ([, position]) =>
          !isPlainRecord(position) ||
          typeof position.x !== "number" ||
          !Number.isFinite(position.x) ||
          typeof position.y !== "number" ||
          !Number.isFinite(position.y),
      )
    ) {
      return {};
    }
    return parsed as Record<string, FixturePosition>;
  } catch {
    return {};
  }
}

export function useFixturePositions(id: string) {
  const [positions, setPositions] = useState<Record<string, FixturePosition>>(() =>
    readPositions(id),
  );
  const saveTimer = useRef<number>();

  useEffect(() => {
    setPositions(readPositions(id));
    return () => window.clearTimeout(saveTimer.current);
  }, [id]);

  const persistPositions = (next: Record<string, FixturePosition>) => {
    setPositions(next);
    window.clearTimeout(saveTimer.current);
    saveTimer.current = window.setTimeout(() => {
      try {
        window.localStorage.setItem(storageKey(id), JSON.stringify(next));
      } catch {
        // Keep the current arrangement as an in-memory draft.
      }
    }, 200);
  };

  return { positions, persistPositions };
}
