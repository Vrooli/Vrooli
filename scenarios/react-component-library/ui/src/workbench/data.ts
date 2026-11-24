import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import type {
  CoverageReport,
  CoverageRow,
} from "@vrooli/proto-types/react-component-library/v1/catalog/catalog_pb";
import {
  AssetKind,
  type Component,
  type ComponentVersion,
} from "@vrooli/proto-types/react-component-library/v1/components/components_pb";

import {
  componentsClient,
  getCatalogAsset,
  getComponentExperience,
  listCatalogAssets,
  listComponentStories,
  type CatalogAsset,
  type ComponentExperience,
  type ComponentStory,
} from "../api/components";
import {
  describeCapabilities,
  getCatalogCoverage,
  listCatalogNextWork,
  type CapabilityDescription,
  type ListNextWorkResponse,
} from "../api/catalog";
import { listComponentTestReports, type ComponentTestReport } from "../api/componentTests";
import { API_BASE, boundedFetch } from "../api/client";
import { getCurrentLocale, i18n } from "../i18n";

export { timestampToDate } from "../lib/timestamp";

export type {
  CatalogAsset,
  Component,
  ComponentExperience,
  ComponentStory,
  ComponentTestReport,
  ComponentVersion,
  CoverageReport,
  CoverageRow,
};

export const wbKeys = {
  catalog: ["wb", "catalog"] as const,
  coverage: ["wb", "coverage"] as const,
  capabilities: ["wb", "capabilities"] as const,
  designScenarios: ["wb", "design-scenarios"] as const,
  designPages: (scenario: string) => ["wb", "design-pages", scenario] as const,
  component: (id: string, version?: string) =>
    version ? (["wb", "component", id, version] as const) : (["wb", "component", id] as const),
};

const CATALOG_LIMIT = 600;

export function useCatalog(): UseQueryResult<CatalogAsset[]> {
  return useQuery({
    queryKey: wbKeys.catalog,
    queryFn: async () => {
      const response = await listCatalogAssets({ limit: CATALOG_LIMIT });
      return response.components;
    },
    staleTime: 30_000,
  });
}

/** A cold gate corpus reports that it is preparing, not that it failed. */
export function isCoverageWarming(error: unknown): boolean {
  const message = error instanceof Error ? error.message : typeof error === "string" ? error : "";
  return /warming|being prepared|preparing/i.test(message);
}

export function useCoverage(): UseQueryResult<CoverageReport> {
  return useQuery({
    queryKey: wbKeys.coverage,
    queryFn: getCatalogCoverage,
    staleTime: 60_000,
    // The first full gate pass takes minutes. Poll while the API says it is
    // preparing so the page recovers on its own instead of asking a person to
    // guess when to refresh.
    retry: (count, error) => count < 2 && !isCoverageWarming(error),
    refetchInterval: (query) => (isCoverageWarming(query.state.error) ? 8_000 : false),
  });
}

export function useCapabilities(): UseQueryResult<CapabilityDescription> {
  return useQuery({
    queryKey: wbKeys.capabilities,
    queryFn: describeCapabilities,
    staleTime: 30_000,
    retry: 0,
  });
}

export function useNextWork(limit = 12): UseQueryResult<ListNextWorkResponse> {
  return useQuery({
    queryKey: ["wb", "next-work", limit] as const,
    queryFn: () => listCatalogNextWork(limit),
    staleTime: 60_000,
    retry: 0,
  });
}

/* ── Design pages (ExperienceService) ─────────────────────────────────────────── */

export interface DesignScenario {
  scenario: string;
  /** The API omits this for scenarios with no declared pages. */
  pageCount?: number;
}

export interface DesignPage {
  page: string;
  title: string;
  route: string;
  status: string;
  contentHash: string;
  regionCount: number;
  registered: boolean;
  routes: string[];
}

async function sketchJSON<T>(method: string, body: Record<string, unknown>): Promise<T> {
  const response = await boundedFetch(
    `${API_BASE}/vrooli.react_component_library.v1.experience.ExperienceService/${method}`,
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    },
  );
  if (!response.ok) {
    const detail = await response.text().catch(() => "");
    throw new Error(
      `${method} failed (${response.status})${detail ? `: ${detail.slice(0, 180)}` : ""}`,
    );
  }
  return (await response.json()) as T;
}

export function useDesignScenarios(): UseQueryResult<DesignScenario[]> {
  return useQuery({
    queryKey: wbKeys.designScenarios,
    queryFn: async () => {
      const response = await sketchJSON<{ scenarios?: DesignScenario[] }>("ListDesignPages", {});
      return (response.scenarios ?? [])
        .filter((entry) => entry.scenario)
        .sort((left, right) => left.scenario.localeCompare(right.scenario));
    },
    staleTime: 120_000,
    retry: 0,
  });
}

export function useDesignPages(scenario: string | null): UseQueryResult<DesignPage[]> {
  return useQuery({
    queryKey: wbKeys.designPages(scenario ?? ""),
    enabled: Boolean(scenario),
    queryFn: async () => {
      const response = await sketchJSON<{ pages?: DesignPage[] }>("ListDesignPages", { scenario });
      return response.pages ?? [];
    },
    staleTime: 60_000,
    retry: 0,
  });
}

export interface ExperienceAdoptionResult {
  workspace?: { id: string; name: string };
  receiptId: string;
  sourceHash: string;
  targetHash: string;
  alreadyAdopted: boolean;
  receiptJson: string;
}

/** Explicitly imports a composition-bearing page into the native workspace owner. */
export function adoptExperiencePage(scenario: string, page: string): Promise<ExperienceAdoptionResult> {
  return sketchJSON<ExperienceAdoptionResult>("AdoptExperiencePage", { scenario, page });
}

/* ── Component bundle ─────────────────────────────────────────────────────── */

/**
 * A declared story, flattened out of a story-contract row. The harness selects
 * stories by their declared id (`primary`, `empty`, …), never by the contract
 * row's UUID. Keeping that distinction here is what makes the fixture iframe
 * resolve instead of 404.
 */
export interface WorkbenchStory {
  key: string;
  id: string;
  name: string;
  role: string;
  args: Record<string, unknown>;
  fieldsJson: string;
  version: string;
  sourcePath: string;
  contractId: string;
}

export interface ComponentBundle {
  asset: CatalogAsset;
  versions: ComponentVersion[];
  stories: WorkbenchStory[];
  experience?: ComponentExperience;
  testReports: ComponentTestReport[];
  partial: string[];
}

/** Fetch the data required by a component binding outside of a React render. */
export async function loadComponentBundle(
  id: string,
  requestedVersion?: string,
): Promise<ComponentBundle> {
  const partial: string[] = [];
  const asset = (await getCatalogAsset(id)).component;
  if (!asset) throw new Error("Component not found");

  const [versions, stories, experience, tests] = await Promise.allSettled([
    componentsClient.listComponentVersions({ componentId: id, limit: 0 }),
    listComponentStories({ componentId: id, version: requestedVersion, limit: 200 }),
    getComponentExperience(id),
    listComponentTestReports({ componentId: id, limit: 3 }),
  ]);

  if (versions.status === "rejected") partial.push("versions");
  const flatStories =
    stories.status === "fulfilled"
      ? parseStories(stories.value.stories, asset.latestVersion || asset.version, requestedVersion)
      : [];
  if (stories.status === "rejected") partial.push("stories");
  else if (flatStories.length === 0 && stories.value.stories.length > 0) partial.push("stories");
  if (experience.status === "rejected") partial.push("experience");
  if (tests.status === "rejected") partial.push("tests");

  return {
    asset,
    versions:
      versions.status === "fulfilled"
        ? [...versions.value.versions].sort((left, right) =>
            right.version.localeCompare(left.version, undefined, { numeric: true }),
          )
        : [],
    stories: flatStories,
    experience: experience.status === "fulfilled" ? experience.value : undefined,
    testReports: tests.status === "fulfilled" ? tests.value : [],
    partial,
  };
}

function parseStories(
  rows: ComponentStory[],
  latestVersion: string,
  requestedVersion?: string,
): WorkbenchStory[] {
  const preferred = rows.filter((row) => row.version === (requestedVersion || latestVersion));
  const source =
    preferred.length > 0
      ? preferred
      : requestedVersion
        ? []
        : [...rows].sort((left, right) => right.version.localeCompare(left.version));
  const seen = new Set<string>();
  const stories: WorkbenchStory[] = [];
  for (const row of source) {
    let parsed: unknown;
    try {
      parsed = JSON.parse(row.storiesJson);
    } catch {
      continue;
    }
    if (!Array.isArray(parsed)) continue;
    for (const entry of parsed) {
      if (!entry || typeof entry !== "object") continue;
      const record = entry as Record<string, unknown>;
      const id = typeof record.id === "string" ? record.id : "";
      if (!id || seen.has(id)) continue;
      seen.add(id);
      stories.push({
        key: `${row.version}:${id}`,
        id,
        name: typeof record.name === "string" && record.name ? record.name : id,
        role: typeof record.role === "string" ? record.role : "story",
        args:
          record.args && typeof record.args === "object" && !Array.isArray(record.args)
            ? (record.args as Record<string, unknown>)
            : {},
        fieldsJson: row.argsJson,
        version: row.version,
        sourcePath: row.sourcePath,
        contractId: row.id,
      });
    }
  }
  return stories;
}

export function useComponentBundle(id: string, version?: string): UseQueryResult<ComponentBundle> {
  return useQuery({
    queryKey: wbKeys.component(id, version),
    queryFn: () => loadComponentBundle(id, version),
    staleTime: 20_000,
    retry: 0,
  });
}

/**
 * Normalize the timestamp shapes the API can hand us: an ISO string from a
 * story fixture, a protobuf Timestamp object (`seconds`), a Date, or an object
 * exposing `toDate`. Returning undefined keeps "no timestamp" distinct from
 * "1970".
 */
/* ── Formatting ───────────────────────────────────────────────────────────── */

export function formatNumber(value: number): string {
  return new Intl.NumberFormat(getCurrentLocale()).format(value);
}

export function formatPercent(value: number, digits = 1): string {
  return `${(value * 100).toFixed(digits)}%`;
}

export function formatRelative(date: Date | undefined): string {
  if (!date || Number.isNaN(date.getTime())) {
    // Non-component copy: use the singleton when it is live, else the literal.
    return i18n.isInitialized
      ? i18n.t("common.relativeUnknown", { defaultValue: "unknown" })
      : "unknown";
  }
  const seconds = Math.round((date.getTime() - Date.now()) / 1000);
  // API timestamps can be a few seconds ahead of the browser clock. "in 6
  // seconds" reads as a bug; treat the near future as now.
  if (seconds > -60 && seconds < 60) {
    return i18n.isInitialized ? i18n.t("common.justNow", { defaultValue: "just now" }) : "just now";
  }
  const units: Array<[Intl.RelativeTimeFormatUnit, number]> = [
    ["year", 31_536_000],
    ["month", 2_592_000],
    ["week", 604_800],
    ["day", 86_400],
    ["hour", 3_600],
    ["minute", 60],
  ];
  const formatter = new Intl.RelativeTimeFormat(getCurrentLocale(), { numeric: "auto" });
  for (const [unit, size] of units) {
    if (Math.abs(seconds) >= size) return formatter.format(Math.round(seconds / size), unit);
  }
  return formatter.format(seconds, "second");
}

export function formatTime(date: Date): string {
  return new Intl.DateTimeFormat(getCurrentLocale(), {
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  }).format(date);
}

export function formatDay(date: Date): string {
  const locale = getCurrentLocale();
  const startOfDay = (value: Date) =>
    Date.UTC(value.getFullYear(), value.getMonth(), value.getDate());
  const days = Math.round((startOfDay(date) - startOfDay(new Date())) / 86_400_000);
  if (days === 0 || days === -1) {
    // `numeric: "auto"` yields the locale's "today"/"yesterday" instead of
    // shipping English words from this module.
    return new Intl.RelativeTimeFormat(locale, { numeric: "auto" }).format(days, "day");
  }
  return new Intl.DateTimeFormat(locale, { month: "short", day: "numeric" }).format(date);
}

/* ── Catalog helpers ──────────────────────────────────────────────────────── */

export type AssetGroup = "components" | "hooks" | "support";

export function assetGroup(asset: CatalogAsset): AssetGroup {
  const kind = asset.catalogKind || "";
  if (kind === "runtime-hook") return "hooks";
  if (["foundation", "runtime-service", "adapter", "generator", "fixture", ""].includes(kind)) {
    if (asset.assetKind === AssetKind.HOOK) return "hooks";
    return "support";
  }
  return "components";
}

export function assetDomain(asset: CatalogAsset): string {
  return asset.catalogDomain || asset.slot || asset.category || "uncategorized";
}

export function assetVersion(asset: CatalogAsset): string {
  return asset.latestVersion || asset.version || asset.draftVersion || "unversioned";
}

export function assetHasDraft(asset: CatalogAsset): boolean {
  return Boolean(asset.draftVersion && asset.draftVersion !== asset.latestVersion);
}

export function assetAdoptions(asset: CatalogAsset): number {
  return asset.metrics?.directAdoptionCount ?? 0;
}
