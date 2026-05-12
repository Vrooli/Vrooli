/**
 * preferences — typed client for `/api/v1/settings`.
 *
 * The settings row is single-tenant ('local' principal) and read by the
 * shell at boot, the theme provider at mount, and the inventory page when
 * applying default filters. Writes are optimistic: the caller updates the
 * UI first and rolls back via the returned promise.
 *
 * localStorage is only a first-paint cache — never the source of truth.
 */
import { buildApiUrl } from "@vrooli/api-base";

import { API_BASE, decodeApiError } from "../api/client";

export type ThemePref = "light" | "dark" | "system";
export type FontScale = "sm" | "md" | "lg";
export type Density = "comfortable" | "compact";

export interface InventoryFilters {
  search: string;
  language: "ts" | "go" | "all";
  status: Array<"passed" | "failed" | "error" | "none">;
  sort: { key: "flowId" | "language" | "status" | "finishedAt"; dir: "asc" | "desc" };
}

export interface UserSettings {
  theme: ThemePref;
  fontScale: FontScale;
  reducedMotion: boolean;
  rtl: boolean;
  defaultRoot: string;
  density: Density;
  sidebarWidth: number;
  inventoryFilters: InventoryFilters;
  updatedAt?: string;
}

export const DEFAULT_SETTINGS: UserSettings = {
  theme: "system",
  fontScale: "md",
  reducedMotion: false,
  rtl: false,
  defaultRoot: ".",
  density: "comfortable",
  sidebarWidth: 320,
  inventoryFilters: {
    search: "",
    language: "all",
    status: [],
    sort: { key: "flowId", dir: "asc" },
  },
};

const CACHE_KEY = "flow-verifier.settings.cache.v1";

export function readCache(): UserSettings | undefined {
  if (typeof window === "undefined") return undefined;
  try {
    const raw = window.localStorage.getItem(CACHE_KEY);
    if (!raw) return undefined;
    const parsed = JSON.parse(raw) as Partial<UserSettings>;
    return { ...DEFAULT_SETTINGS, ...parsed };
  } catch {
    return undefined;
  }
}

export function writeCache(s: UserSettings): void {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(CACHE_KEY, JSON.stringify(s));
  } catch {
    /* ignore */
  }
}

export async function fetchSettings(): Promise<UserSettings> {
  const res = await fetch(buildApiUrl("/api/v1/settings", { baseUrl: API_BASE }), {
    method: "GET",
    cache: "no-store",
  });
  if (!res.ok) throw await decodeApiError(res);
  const body = (await res.json()) as Partial<UserSettings>;
  const merged = { ...DEFAULT_SETTINGS, ...body };
  writeCache(merged);
  return merged;
}

export async function putSettings(
  patch: Partial<UserSettings>,
): Promise<UserSettings> {
  const res = await fetch(buildApiUrl("/api/v1/settings", { baseUrl: API_BASE }), {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
    body: JSON.stringify(patch),
  });
  if (!res.ok) throw await decodeApiError(res);
  const body = (await res.json()) as Partial<UserSettings>;
  const merged = { ...DEFAULT_SETTINGS, ...body };
  writeCache(merged);
  return merged;
}
