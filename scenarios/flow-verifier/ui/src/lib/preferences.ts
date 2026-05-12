/**
 * preferences — typed client for the SettingsService Connect RPC.
 *
 * The settings row is single-tenant ('local' principal) and read by the
 * shell at boot, the theme provider at mount, and the inventory page when
 * applying default filters. Writes are optimistic: the caller updates the
 * UI first and rolls back via the returned promise.
 *
 * localStorage is only a first-paint cache — never the source of truth.
 */
import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { createClient } from "@connectrpc/connect";

import { transport } from "../api/client";

import {
  SettingsService,
  Theme as ProtoTheme,
  FontScale as ProtoFontScale,
  Density as ProtoDensity,
  type Settings as ProtoSettings,
} from "@vrooli/proto-types/flow-verifier/v1/settings/settings_pb";

const settingsClient = createClient(SettingsService, transport);

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
  const resp = await settingsClient.getSettings({});
  const merged = protoToUserSettings(resp.settings);
  writeCache(merged);
  return merged;
}

export async function putSettings(patch: Partial<UserSettings>): Promise<UserSettings> {
  const { settings, paths } = userSettingsToProtoPatch(patch);
  const resp = await settingsClient.updateSettings({
    settings,
    updateMask: create(FieldMaskSchema, { paths }),
  });
  const merged = protoToUserSettings(resp.settings);
  writeCache(merged);
  return merged;
}

function protoToUserSettings(s: ProtoSettings | undefined): UserSettings {
  if (!s) return DEFAULT_SETTINGS;
  return {
    theme: protoTheme(s.theme),
    fontScale: protoFontScale(s.fontScale),
    reducedMotion: s.reducedMotion,
    rtl: s.rtl,
    defaultRoot: s.defaultRoot || DEFAULT_SETTINGS.defaultRoot,
    density: protoDensity(s.density),
    sidebarWidth: s.sidebarWidth || DEFAULT_SETTINGS.sidebarWidth,
    inventoryFilters: s.inventoryFilters
      ? {
          search: s.inventoryFilters.search,
          language: pickLanguage(s.inventoryFilters.language),
          status: s.inventoryFilters.status as InventoryFilters["status"],
          sort: s.inventoryFilters.sort
            ? {
                key: pickSortKey(s.inventoryFilters.sort.key),
                dir: pickSortDir(s.inventoryFilters.sort.dir),
              }
            : DEFAULT_SETTINGS.inventoryFilters.sort,
        }
      : DEFAULT_SETTINGS.inventoryFilters,
    updatedAt: s.updatedAt || undefined,
  };
}

function userSettingsToProtoPatch(
  patch: Partial<UserSettings>,
): { settings: ProtoSettings; paths: string[] } {
  const paths: string[] = [];
  const settings: ProtoSettings = {
    $typeName: "vrooli.flow_verifier.v1.settings.Settings",
    principalId: "",
    theme: ProtoTheme.UNSPECIFIED,
    fontScale: ProtoFontScale.UNSPECIFIED,
    reducedMotion: false,
    rtl: false,
    defaultRoot: "",
    density: ProtoDensity.UNSPECIFIED,
    sidebarWidth: 0,
    inventoryFilters: undefined,
    updatedAt: "",
  };
  if (patch.theme !== undefined) {
    settings.theme = themeToProto(patch.theme);
    paths.push("theme");
  }
  if (patch.fontScale !== undefined) {
    settings.fontScale = fontScaleToProto(patch.fontScale);
    paths.push("font_scale");
  }
  if (patch.reducedMotion !== undefined) {
    settings.reducedMotion = patch.reducedMotion;
    paths.push("reduced_motion");
  }
  if (patch.rtl !== undefined) {
    settings.rtl = patch.rtl;
    paths.push("rtl");
  }
  if (patch.defaultRoot !== undefined) {
    settings.defaultRoot = patch.defaultRoot;
    paths.push("default_root");
  }
  if (patch.density !== undefined) {
    settings.density = densityToProto(patch.density);
    paths.push("density");
  }
  if (patch.sidebarWidth !== undefined) {
    settings.sidebarWidth = patch.sidebarWidth;
    paths.push("sidebar_width");
  }
  if (patch.inventoryFilters !== undefined) {
    settings.inventoryFilters = {
      $typeName: "vrooli.flow_verifier.v1.settings.InventoryFilters",
      search: patch.inventoryFilters.search,
      language: patch.inventoryFilters.language,
      status: patch.inventoryFilters.status,
      sort: {
        $typeName: "vrooli.flow_verifier.v1.settings.InventorySortOrder",
        key: patch.inventoryFilters.sort.key,
        dir: patch.inventoryFilters.sort.dir,
      },
    };
    paths.push("inventory_filters");
  }
  return { settings, paths };
}

function protoTheme(t: ProtoTheme): ThemePref {
  switch (t) {
    case ProtoTheme.LIGHT:
      return "light";
    case ProtoTheme.DARK:
      return "dark";
    case ProtoTheme.SYSTEM:
      return "system";
  }
  return DEFAULT_SETTINGS.theme;
}
function themeToProto(t: ThemePref): ProtoTheme {
  switch (t) {
    case "light":
      return ProtoTheme.LIGHT;
    case "dark":
      return ProtoTheme.DARK;
    case "system":
      return ProtoTheme.SYSTEM;
  }
}
function protoFontScale(f: ProtoFontScale): FontScale {
  switch (f) {
    case ProtoFontScale.SM:
      return "sm";
    case ProtoFontScale.MD:
      return "md";
    case ProtoFontScale.LG:
      return "lg";
  }
  return DEFAULT_SETTINGS.fontScale;
}
function fontScaleToProto(f: FontScale): ProtoFontScale {
  switch (f) {
    case "sm":
      return ProtoFontScale.SM;
    case "md":
      return ProtoFontScale.MD;
    case "lg":
      return ProtoFontScale.LG;
  }
}
function protoDensity(d: ProtoDensity): Density {
  switch (d) {
    case ProtoDensity.COMFORTABLE:
      return "comfortable";
    case ProtoDensity.COMPACT:
      return "compact";
  }
  return DEFAULT_SETTINGS.density;
}
function densityToProto(d: Density): ProtoDensity {
  switch (d) {
    case "comfortable":
      return ProtoDensity.COMFORTABLE;
    case "compact":
      return ProtoDensity.COMPACT;
  }
}

function pickLanguage(v: string): InventoryFilters["language"] {
  if (v === "ts" || v === "go" || v === "all") return v;
  return DEFAULT_SETTINGS.inventoryFilters.language;
}
function pickSortKey(v: string): InventoryFilters["sort"]["key"] {
  if (v === "flowId" || v === "language" || v === "status" || v === "finishedAt") return v;
  return DEFAULT_SETTINGS.inventoryFilters.sort.key;
}
function pickSortDir(v: string): InventoryFilters["sort"]["dir"] {
  if (v === "asc" || v === "desc") return v;
  return DEFAULT_SETTINGS.inventoryFilters.sort.dir;
}
