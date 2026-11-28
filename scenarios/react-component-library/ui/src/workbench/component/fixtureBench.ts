import type { CatalogAsset, WorkbenchStory } from "../data";

/**
 * The specimen bench groups stories by the question each one answers.
 *
 * The story contract already names the question (`anatomy`, `axis`, `boundary`).
 * The bench preserves that order and only collapses frames that render the same
 * composition with the same arguments. Collapsed frames stay available under a
 * disclosure so a weaker expectation is never silently deleted.
 */

export interface FixtureGroup {
  id: string;
  label: string;
  role: string;
  axis?: string;
  /** Values the axis story claims in `covers`. */
  values?: string[];
  stories: WorkbenchStory[];
}

export interface FixtureBenchModel {
  anatomy?: WorkbenchStory;
  groups: FixtureGroup[];
  hidden: WorkbenchStory[];
  visibleCount: number;
}

function stableValue(value: unknown): unknown {
  if (Array.isArray(value)) return value.map(stableValue);
  if (!value || typeof value !== "object") return value;
  const record = value as Record<string, unknown>;
  return Object.fromEntries(
    Object.keys(record)
      .sort()
      .map((key) => [key, stableValue(record[key])]),
  );
}

function stableJSON(value: unknown): string {
  return JSON.stringify(stableValue(value));
}

/**
 * Two frames render the same thing when they mount the same specimen export
 * with the same args and environment. A frame without a specimen renders the
 * asset directly, so equal args also mean equal rendering. A specimen and a
 * direct frame are never treated as the same thing.
 */
export function storyVisualKey(story: WorkbenchStory): string {
  const args = stableJSON(story.args);
  const environment = stableJSON(story.environment ?? {});
  if (story.composition || story.specimen) {
    const composition = story.composition
      ? stableJSON(story.composition)
      : `specimen:${story.specimen?.module}#${story.specimen?.export}`;
    return `composition:${composition}:${args}:${environment}`;
  }
  return `direct:${args}:${environment}`;
}

function keeperRank(story: WorkbenchStory): number {
  const pinned = story.mode === "pinned" ? 1 : 0;
  const role = story.role === "anatomy" ? 0 : story.role === "axis" ? 1 : story.role === "boundary" ? 2 : 3;
  return pinned * 10 + role;
}

function collapseDuplicates(stories: WorkbenchStory[]): {
  kept: WorkbenchStory[];
  hidden: WorkbenchStory[];
} {
  const buckets = new Map<string, WorkbenchStory[]>();
  for (const story of stories) {
    const key = storyVisualKey(story);
    const bucket = buckets.get(key);
    if (bucket) bucket.push(story);
    else buckets.set(key, [story]);
  }

  const kept: WorkbenchStory[] = [];
  const hidden: WorkbenchStory[] = [];
  const keptIds = new Set<string>();
  for (const story of stories) {
    const bucket = buckets.get(storyVisualKey(story)) ?? [];
    const ordered = [...bucket].sort(
      (left, right) =>
        keeperRank(left) - keeperRank(right) ||
        stories.indexOf(left) - stories.indexOf(right) ||
        left.id.localeCompare(right.id),
    );
    const keeper = ordered[0] ?? story;
    if (keeper.id === story.id) {
      if (!keptIds.has(story.id)) {
        kept.push(story);
        keptIds.add(story.id);
      }
    } else if (!hidden.some((entry) => entry.id === story.id)) {
      hidden.push(story);
    }
  }
  return { kept, hidden };
}

function axisLabel(axis: string): string {
  const words = axis
    .split(".")
    .pop()
    ?.replace(/[-_]+/g, " ")
    .trim();
  if (!words) return "Axis";
  return words.charAt(0).toUpperCase() + words.slice(1);
}

function normalizeWords(value: string): string {
  return value
    .replace(/([a-z0-9])([A-Z])/g, "$1 $2")
    .replace(/[_./:-]+/g, " ")
    .replace(/\s+/g, " ")
    .trim()
    .toLowerCase();
}

const VALUE_ALIASES: Record<string, string[]> = {
  circle: ["circle", "circular"],
  sm: ["sm", "small"],
  md: ["md", "medium", "default"],
  lg: ["lg", "large"],
  xs: ["xs", "extra small"],
};

function valueAliases(value: string): string[] {
  const normalized = normalizeWords(value);
  return [...new Set([normalized, ...(VALUE_ALIASES[normalized] ?? [])])];
}

function storyText(story: WorkbenchStory): string {
  return normalizeWords(
    [
      story.id,
      story.name,
      story.description ?? "",
      story.specimen?.export ?? "",
    ].join(" "),
  );
}

function matchesValue(story: WorkbenchStory, value: string): boolean {
  const text = ` ${storyText(story)} `;
  return valueAliases(value).some((alias) => alias.length > 1 && text.includes(` ${alias} `));
}

/**
 * An axis story declares the values it covers, but the contract often authors
 * the other values as boundary frames. Bring those frames into the axis row so
 * one dimension reads as one dimension instead of one tile plus a pile of
 * boundaries. The anatomy frame stays the stage baseline and is not duplicated.
 */
function attachAxisVariants(
  group: FixtureGroup,
  axisStory: WorkbenchStory,
  kept: WorkbenchStory[],
  anatomyId: string | undefined,
  assigned: Set<string>,
): void {
  const axis = axisStory.axis ?? "";
  const values = (axisStory.covers?.[axis] ?? [])
    .filter((value): value is string => typeof value === "string")
    .map((value) => value.trim())
    .filter(Boolean);
  group.values = values;
  if (values.length === 0) return;

  for (const story of kept) {
    if (story.id === anatomyId || assigned.has(story.id)) continue;
    if (values.some((value) => matchesValue(story, value))) {
      group.stories.push(story);
      assigned.add(story.id);
    }
  }
  group.stories.sort(
    (left, right) =>
      axisValueIndex(left, values) - axisValueIndex(right, values) ||
      kept.indexOf(left) - kept.indexOf(right),
  );
}

function axisValueIndex(story: WorkbenchStory, values: string[]): number {
  const index = values.findIndex((value) => matchesValue(story, value));
  return index === -1 ? values.length : index;
}

export function buildFixtureBench(stories: WorkbenchStory[]): FixtureBenchModel {
  const { kept, hidden } = collapseDuplicates(stories);
  const anatomy = kept.find((story) => story.role === "anatomy");
  const groups: FixtureGroup[] = [];
  const axisGroups: FixtureGroup[] = [];
  const byAxis = new Map<string, FixtureGroup>();
  const assignedToAxis = new Set<string>();

  for (const story of kept) {
    if (story.role !== "axis") continue;
    const axis = story.axis || "axis";
    const existing = byAxis.get(axis);
    if (existing) {
      existing.stories.push(story);
      continue;
    }
    const group: FixtureGroup = {
      id: `axis:${axis}`,
      label: axisLabel(axis),
      role: "axis",
      axis,
      stories: [story],
    };
    byAxis.set(axis, group);
    axisGroups.push(group);
    assignedToAxis.add(story.id);
  }

  for (const group of axisGroups) {
    const axisStory = group.stories[0];
    if (!axisStory) continue;
    attachAxisVariants(group, axisStory, kept, anatomy?.id, assignedToAxis);
  }
  groups.push(...axisGroups);

  const boundaries: WorkbenchStory[] = [];
  const others = new Map<string, FixtureGroup>();
  for (const story of kept) {
    if (story.id === anatomy?.id || assignedToAxis.has(story.id)) continue;
    if (story.role === "boundary") {
      boundaries.push(story);
      continue;
    }
    if (story.role === "axis") continue;
    const role = story.role || "story";
    const existing = others.get(role);
    if (existing) {
      existing.stories.push(story);
    } else {
      const group: FixtureGroup = {
        id: `role:${role}`,
        label: role === "story" ? "Other frames" : `${role.charAt(0).toUpperCase()}${role.slice(1)} frames`,
        role,
        stories: [story],
      };
      others.set(role, group);
      groups.push(group);
    }
  }

  if (boundaries.length > 0) {
    groups.push({
      id: "boundaries",
      label: "Boundaries",
      role: "boundary",
      stories: boundaries,
    });
  }

  return {
    anatomy,
    groups,
    hidden,
    visibleCount: kept.length,
  };
}

function pixelValue(value: string | undefined): number | undefined {
  const match = value?.trim().match(/^([1-9]\d{0,3})px$/);
  if (!match) return undefined;
  const parsed = Number.parseInt(match[1] ?? "", 10);
  return Number.isFinite(parsed) ? parsed : undefined;
}

/**
 * The width the harness lays the story out at. An intrinsic control still needs
 * room for its press target and tooltip; a block component needs a realistic
 * column width. Authored geometry wins when it exists.
 */
export function fixtureViewportWidth(story: WorkbenchStory, asset: CatalogAsset): number {
  const authored = pixelValue(story.geometry?.width);
  if (authored && authored >= 240 && authored <= 1920) return authored;
  const kind = `${asset.assetKind} ${asset.slot} ${asset.catalogId}`.toLowerCase();
  if (kind.includes("page") || kind.includes("navigation")) return 1180;
  const label = `${asset.displayName} ${asset.slot} ${asset.catalogId}`.toLowerCase();
  if (/(icon|button|chip|badge|avatar|switch|checkbox|radio|toggle|input|select|slider)/.test(label)) {
    return 200;
  }
  return 360;
}

/** The height the harness is given before its painted bounds are measured. */
export function fixtureViewportHeight(story: WorkbenchStory): number {
  const authored = pixelValue(story.geometry?.height);
  if (authored && authored >= 240 && authored <= 2160) return authored;
  return 900;
}

export interface FixturePaint {
  x: number;
  y: number;
  width: number;
  height: number;
}

/** A deterministic placeholder until the harness reports its painted bounds. */
export function initialPaint(story: WorkbenchStory, asset: CatalogAsset): FixturePaint {
  const width = fixtureViewportWidth(story, asset);
  const height = pixelValue(story.geometry?.height) ?? 120;
  return { x: 0, y: 0, width, height };
}
