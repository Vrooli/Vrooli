import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import type { CoverageReport, CoverageRow } from "@vrooli/proto-types/react-component-library/v1/catalog/catalog_pb";
import { AssetKind, type Component } from "@vrooli/proto-types/react-component-library/v1/components/components_pb";
import type { Version } from "@vrooli/proto-types/react-component-library/v1/versions/versions_pb";

import {
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
  getCatalogHealthOverview,
  listCatalogNextWork,
  type CapabilityDescription,
  type GetHealthOverviewResponse,
  type ListNextWorkResponse,
} from "../api/catalog";
import { listComponentTestReports, type ComponentTestReport } from "../api/componentTests";
import { versionsClient } from "../api/versions";
import { API_BASE, boundedFetch } from "../api/client";

export type { CatalogAsset, Component, ComponentExperience, ComponentStory, ComponentTestReport, CoverageReport, CoverageRow, Version };

export const wbKeys = {
  catalog: ["wb", "catalog"] as const,
  coverage: ["wb", "coverage"] as const,
  health: ["wb", "health"] as const,
  capabilities: ["wb", "capabilities"] as const,
  designScenarios: ["wb", "design-scenarios"] as const,
  designPages: (scenario: string) => ["wb", "design-pages", scenario] as const,
  component: (id: string) => ["wb", "component", id] as const,
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
  const message =
    error instanceof Error ? error.message : typeof error === "string" ? error : "";
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

export function useHealthOverview(): UseQueryResult<GetHealthOverviewResponse> {
  return useQuery({
    queryKey: wbKeys.health,
    queryFn: getCatalogHealthOverview,
    staleTime: 60_000,
    retry: 0,
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

/* ── Design pages (SketchService) ─────────────────────────────────────────── */

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
    `${API_BASE}/vrooli.react_component_library.v1.sketch.SketchService/${method}`,
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    },
  );
  if (!response.ok) {
    const detail = await response.text().catch(() => "");
    throw new Error(`${method} failed (${response.status})${detail ? `: ${detail.slice(0, 180)}` : ""}`);
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
  versions: Version[];
  stories: WorkbenchStory[];
  experience?: ComponentExperience;
  testReports: ComponentTestReport[];
  partial: string[];
}

function parseStories(rows: ComponentStory[], latestVersion: string): WorkbenchStory[] {
  const preferred = rows.filter((row) => row.version === latestVersion);
  const source = preferred.length > 0 ? preferred : [...rows].sort((left, right) => right.version.localeCompare(left.version));
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

export function useComponentBundle(id: string): UseQueryResult<ComponentBundle> {
  return useQuery({
    queryKey: wbKeys.component(id),
    queryFn: async () => {
      const partial: string[] = [];
      const asset = (await getCatalogAsset(id)).component;
      if (!asset) throw new Error("Component not found");

      const [versions, stories, experience, tests] = await Promise.allSettled([
        versionsClient.listVersions({ componentId: id, limit: 0 }),
        listComponentStories({ componentId: id, limit: 200 }),
        getComponentExperience(id),
        listComponentTestReports({ componentId: id, limit: 3 }),
      ]);

      if (versions.status === "rejected") partial.push("versions");
      const flatStories =
        stories.status === "fulfilled" ? parseStories(stories.value.stories, asset.latestVersion || asset.version) : [];
      if (stories.status === "rejected") {
        partial.push("stories");
      } else if (flatStories.length === 0 && stories.value.stories.length > 0) {
        partial.push("stories");
      }
      if (experience.status === "rejected") partial.push("experience");
      if (tests.status === "rejected") partial.push("tests");

      return {
        asset,
        versions: versions.status === "fulfilled" ? versions.value.versions : [],
        stories: flatStories,
        experience: experience.status === "fulfilled" ? experience.value : undefined,
        testReports: tests.status === "fulfilled" ? tests.value : [],
        partial,
      };
    },
    staleTime: 20_000,
    retry: 0,
  });
}

/* ── Findings projection ──────────────────────────────────────────────────── */

export type FindingSeverity = "blocking" | "warning" | "info";

export interface Finding {
  id: string;
  title: string;
  assetId: string;
  assetName: string;
  libraryId: string;
  domain: string;
  gate: string;
  severity: FindingSeverity;
  blocksDownstream: number;
  score: number;
  achieved: string;
  target: string;
  rungName: string;
  newestEvidence: string;
  kind: string;
  failedCount: number;
  secondaryGates: string[];
}

/**
 * Findings are a projection over the coverage report's real gate outcomes. The
 * workbench does not invent a findings store: if the report cannot be produced,
 * Review and Home must say so rather than render an empty queue.
 */
export function findingsFromCoverage(report: CoverageReport | undefined): Finding[] {
  if (!report) return [];
  const findings: Finding[] = [];
  for (const row of report.rows) {
    const gates = row.failedGates;
    if (gates.length === 0) continue;
    const nearest = row.nearestBlockingGate.trim();
    const gate = nearest || gates[0] || "unknown";
    // One finding per asset: the nearest blocking gate is the decision a person
    // can act on. Secondary gate failures are counted on the same card instead
    // of flooding the queue with one row per gate.
    const severity: FindingSeverity =
      nearest && row.blocksDownstream > 0 ? "blocking" : "warning";
    findings.push({
      id: `${row.assetId}:${gate}`,
      title: findingTitle(gate),
      assetId: row.assetId,
      assetName: row.name,
      libraryId: row.name,
      domain: row.domain || "uncategorized",
      gate,
      severity,
      blocksDownstream: row.blocksDownstream,
      score: row.assetScore,
      achieved: row.achieved,
      target: row.target,
      rungName: row.rungName,
      newestEvidence: row.newestEvidence,
      kind: row.kind,
      failedCount: gates.length,
      secondaryGates: gates.filter((entry) => entry !== gate),
    });
  }
  const rank = { blocking: 0, warning: 1, info: 2 };
  return findings.sort(
    (left, right) =>
      rank[left.severity] - rank[right.severity] ||
      right.blocksDownstream - left.blocksDownstream ||
      left.assetName.localeCompare(right.assetName),
  );
}

function findingTitle(gate: string): string {
  const labels: Record<string, string> = {
    types: "Type contract does not compile",
    tokens: "Raw design values in source",
    accessibility: "Accessibility contract failed",
    responsive: "Responsive contract failed",
    visual: "Visual evidence is missing or stale",
    "version-shape": "Version folder shape is invalid",
    "usage-contract": "Usage contract is missing",
    "story-grammar": "Story grammar is invalid",
    "manifest-identity": "Manifest identity does not match",
    "evidence-freshness": "Evidence is stale",
    "graph-reconciled": "Dependency graph is not reconciled",
    "composition-contract": "Composition contract failed",
    i18n: "Locale strings are incomplete",
    rtl: "RTL contract failed",
    "reduced-motion": "Reduced-motion contract failed",
    "console-clean": "Preview console is not clean",
    "style-ownership": "Style ownership boundary was crossed",
    "token-vocabulary": "Token vocabulary is not declared",
  };
  return labels[gate] ?? `${gate} gate failed`;
}

export interface FindingStats {
  open: number;
  blocking: number;
  warning: number;
  affectedAssets: number;
  staleEvidence: number;
}

export function findingStats(findings: Finding[]): FindingStats {
  const assets = new Set(findings.map((finding) => finding.assetId));
  return {
    open: findings.length,
    blocking: findings.filter((finding) => finding.severity === "blocking").length,
    warning: findings.filter((finding) => finding.severity === "warning").length,
    affectedAssets: assets.size,
    staleEvidence: findings.filter((finding) => finding.gate === "evidence-freshness").length,
  };
}

/* ── Activity projection ──────────────────────────────────────────────────── */

export type ActivityKind = "indexed" | "published" | "draft" | "evidence" | "failed";

export interface ActivityEvent {
  id: string;
  kind: ActivityKind;
  title: string;
  assetName: string;
  version: string;
  at: Date;
  actor: string;
  detail: string;
}

/**
 * Activity is derived from the catalog's own timestamps. It is deliberately
 * labelled "Catalog index activity" in the UI: there is no durable event store
 * yet, and a redesign must not pretend one exists.
 */
export function activityFromCatalog(assets: CatalogAsset[]): ActivityEvent[] {
  const events: ActivityEvent[] = [];
  for (const asset of assets) {
    const updated = parseTime(asset.updatedAt);
    if (updated) {
      events.push({
        id: `${asset.id}:updated`,
        kind: asset.draftVersion ? "draft" : "indexed",
        title: asset.draftVersion ? "Draft indexed" : "Source updated",
        assetName: asset.displayName || asset.slug,
        version: asset.draftVersion || asset.latestVersion || asset.version || "unversioned",
        at: updated,
        actor: "catalog",
        detail: asset.catalogId || asset.slot || asset.libraryId,
      });
    }
    const indexed = parseTime(asset.indexedAt);
    if (indexed && (!updated || Math.abs(indexed.getTime() - updated.getTime()) > 60_000)) {
      events.push({
        id: `${asset.id}:indexed`,
        kind: "indexed",
        title: "Source indexed",
        assetName: asset.displayName || asset.slug,
        version: asset.latestVersion || asset.version || "unversioned",
        at: indexed,
        actor: "catalog",
        detail: asset.libraryId,
      });
    }
  }
  return events.sort((left, right) => right.at.getTime() - left.at.getTime());
}

function parseTime(value: unknown): Date | undefined {
  return timestampToDate(value);
}

/**
 * Normalize the timestamp shapes the API can hand us: an ISO string from a
 * story fixture, a protobuf Timestamp object (`seconds`), a Date, or an object
 * exposing `toDate`. Returning undefined keeps "no timestamp" distinct from
 * "1970".
 */
export function timestampToDate(value: unknown): Date | undefined {
  if (!value) return undefined;
  if (value instanceof Date) return Number.isNaN(value.getTime()) ? undefined : value;
  if (typeof value === "string") {
    const date = new Date(value);
    return Number.isNaN(date.getTime()) ? undefined : date;
  }
  if (typeof value === "number") {
    const date = new Date(value);
    return Number.isNaN(date.getTime()) ? undefined : date;
  }
  if (typeof value === "object") {
    const record = value as Record<string, unknown>;
    if (typeof record.toDate === "function") {
      const date = (record.toDate as () => Date).call(value);
      return Number.isNaN(date.getTime()) ? undefined : date;
    }
    if (typeof record.seconds === "number" || typeof record.seconds === "bigint") {
      const seconds = Number(record.seconds);
      if (!Number.isFinite(seconds)) return undefined;
      const date = new Date(seconds * 1000);
      return Number.isNaN(date.getTime()) ? undefined : date;
    }
  }
  return undefined;
}

/* ── Formatting ───────────────────────────────────────────────────────────── */

export function formatNumber(value: number): string {
  return new Intl.NumberFormat("en-US").format(value);
}

export function formatPercent(value: number, digits = 1): string {
  return `${(value * 100).toFixed(digits)}%`;
}

export function formatRelative(date: Date | undefined): string {
  if (!date) return "unknown";
  if (Number.isNaN(date.getTime())) return "unknown";
  const seconds = Math.round((date.getTime() - Date.now()) / 1000);
  // API timestamps can be a few seconds ahead of the browser clock. "in 6
  // seconds" reads as a bug; treat the near future as now.
  if (seconds > -60 && seconds < 60) return "just now";
  const units: Array<[Intl.RelativeTimeFormatUnit, number]> = [
    ["year", 31_536_000],
    ["month", 2_592_000],
    ["week", 604_800],
    ["day", 86_400],
    ["hour", 3_600],
    ["minute", 60],
  ];
  const formatter = new Intl.RelativeTimeFormat("en", { numeric: "auto" });
  for (const [unit, size] of units) {
    if (Math.abs(seconds) >= size) return formatter.format(Math.round(seconds / size), unit);
  }
  return formatter.format(seconds, "second");
}

export function formatTime(date: Date): string {
  return new Intl.DateTimeFormat("en", { hour: "2-digit", minute: "2-digit", hour12: false }).format(date);
}

export function formatDay(date: Date): string {
  const today = new Date();
  const sameDay = date.toDateString() === today.toDateString();
  if (sameDay) return "Today";
  const yesterday = new Date(today);
  yesterday.setDate(today.getDate() - 1);
  if (date.toDateString() === yesterday.toDateString()) return "Yesterday";
  return new Intl.DateTimeFormat("en", { month: "short", day: "numeric" }).format(date);
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

export function assetEffectiveAdoptions(asset: CatalogAsset): number {
  return asset.metrics?.effectiveAdoptionCount ?? 0;
}

export function isRenderable(asset: CatalogAsset): boolean {
  return assetGroup(asset) !== "hooks" && asset.assetKind !== AssetKind.HOOK;
}
