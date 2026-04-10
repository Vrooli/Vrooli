import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

export function safeParseJson(value: string): unknown | undefined {
  try {
    // JSON.parse returns any; treat it as an unknown boundary.
    return JSON.parse(value) as unknown;
  } catch {
    return undefined;
  }
}
