import {
  DEFAULT_SLOT_HEIGHT,
  slotAccepts,
  slotRole,
  slotsOf,
  type SlotRole,
  type TemplateNode,
} from "../templates/spec";

/**
 * Auto-placement for the in-context view.
 *
 * Given the component asset and the page's typed slots, this module picks the
 * slot the component belongs in. It is deliberately pure and deterministic:
 * the same asset, spec, viewport and preferred slot always produce the same
 * ranking, so a screenshot, a check and a test all describe one decision.
 *
 * Scores are points out of a fixed ceiling, normalised to 0–100:
 *   role     0–45  a declared or family-level role match
 *   size     0–25  the component's preferred width inside the slot's width
 *   position 0–15  earlier slots above later ones
 *
 * Ties on the total are broken by role specificity, then size, then document
 * order: a slot whose role names the component beats one that merely accepts
 * its kind, even when the latter sits earlier on the page.
 *
 * Height is a floor, not a ceiling: a slot grows to fit its component. Width
 * is the real constraint, so it carries the size score.
 */

export type AssetSlotKind =
  | "hero"
  | "metric"
  | "stat"
  | "number"
  | "chart"
  | "table"
  | "list"
  | "button"
  | "action"
  | "text"
  | "nav"
  | "form"
  | "input"
  | "status"
  | "card";

export interface PlacementAsset {
  displayName?: string;
  slug?: string;
  description?: string;
  category?: string;
  catalogKind?: string;
  slot?: string;
  tags?: string[];
}

export interface SlotBox {
  w: number;
  h: number;
}

export interface AssetPlacementProfile {
  /** The component's primary kind, used for direct role matching. */
  kind: AssetSlotKind;
  /** The kinds this profile can stand in for (a chart is also a metric). */
  kinds: AssetSlotKind[];
  /** Preferred rendered size in CSS pixels at the page's desktop width. */
  size: SlotBox;
}

export type SlotRoleMatch =
  | "exact"
  | "accepted"
  | "family"
  | "role-only"
  | "mismatch"
  | "undeclared";

export interface SlotScore {
  slotId: string;
  label: string;
  role: SlotRole | undefined;
  accepts: string[];
  compatible: boolean;
  /** 0–100, rounded. */
  score: number;
  rolePoints: number;
  sizePoints: number;
  positionPoints: number;
  /** How the page's role relates to the component kind, beyond raw acceptance. */
  roleMatch: SlotRoleMatch;
  index: number;
  box: SlotBox;
}

export interface PlacementDecision {
  profile: AssetPlacementProfile;
  /** The authored slot, when the template declares one. */
  authoredId: string | undefined;
  /** Highest-scoring slot before any manual preference. */
  auto: SlotScore | undefined;
  /** The slot actually used (manual preference wins when it still exists). */
  chosen: SlotScore | undefined;
  /** True when a slot was pinned by hand, even if it is also the best score. */
  manual: boolean;
  /** Every slot, best first. */
  candidates: SlotScore[];
}

const KIND_RULES: Array<{ kind: AssetSlotKind; pattern: RegExp }> = [
  { kind: "chart", pattern: /\b(chart|graph|spark|trend|plot|histogram|series)\b/ },
  { kind: "table", pattern: /\b(table|grid|ledger|matrix)\b/ },
  {
    kind: "metric",
    pattern:
      /\b(metric|stat|kpi|score|readout|health|summary|meter|progress|gauge|counter|usage|credit|balance|account|number)\b/,
  },
  { kind: "list", pattern: /\b(list|feed|queue|inbox|timeline|activity|collection|message|thread|notification)\b/ },
  { kind: "button", pattern: /\b(button|cta|action|toggle|switch)\b/ },
  { kind: "form", pattern: /\b(form|input|field|composer|search|picker|select|upload)\b/ },
  { kind: "nav", pattern: /\b(nav|tab|breadcrumb|menu|sidebar|toolbar)\b/ },
  { kind: "hero", pattern: /\b(hero|headline|banner|masthead|title|display)\b/ },
  { kind: "status", pattern: /\b(status|badge|state)\b/ },
  { kind: "text", pattern: /\b(text|label|chip|caption|paragraph|markdown|typography|mention)\b/ },
  { kind: "card", pattern: /\b(card|panel|surface|shell|container|frame)\b/ },
];

const KIND_FAMILY: Record<AssetSlotKind, AssetSlotKind[]> = {
  hero: ["hero", "text"],
  metric: ["metric", "stat", "number"],
  chart: ["chart", "metric"],
  table: ["table", "list", "chart"],
  list: ["list", "table"],
  button: ["button", "action"],
  text: ["text"],
  nav: ["nav", "button"],
  form: ["form", "input", "action"],
  status: ["status", "metric"],
  card: ["card", "list", "metric"],
  stat: ["stat", "metric", "number"],
  number: ["number", "stat", "metric"],
  action: ["action", "button"],
  input: ["input", "form"],
};

const PREFERRED_SIZE: Record<AssetSlotKind, SlotBox> = {
  hero: { w: 520, h: 160 },
  metric: { w: 248, h: 124 },
  stat: { w: 200, h: 96 },
  number: { w: 120, h: 64 },
  chart: { w: 560, h: 280 },
  table: { w: 560, h: 320 },
  list: { w: 320, h: 220 },
  button: { w: 140, h: 44 },
  action: { w: 140, h: 44 },
  text: { w: 280, h: 40 },
  nav: { w: 360, h: 44 },
  form: { w: 320, h: 48 },
  input: { w: 320, h: 48 },
  status: { w: 160, h: 32 },
  card: { w: 320, h: 180 },
};

/**
 * A role's natural tenant. This is the fallback when a template declares a
 * role but forgets to list the kind in `accepts`; an authored `accepts` list
 * is still the gate for compatibility, never a suggestion.
 */
const ROLE_FAMILY: Record<SlotRole, string[]> = {
  hero: ["hero", "text", "metric", "card", "status"],
  metric: ["metric", "stat", "number", "chart", "status"],
  chart: ["chart", "metric", "table"],
  list: ["list", "table", "metric", "card"],
  action: ["button", "action", "form", "status"],
  nav: ["nav", "button"],
  text: ["text", "hero"],
};

const MAX_POINTS = 45 + 25 + 15;

/** "MetricCard" and "metric-card" must both expose the word "metric". */
function tokenize(value: string | undefined): string {
  if (!value) return "";
  return value
    .replace(/([a-z0-9])([A-Z])/g, "$1 $2")
    .replace(/([A-Z]+)([A-Z][a-z])/g, "$1 $2")
    .replace(/[_./:-]+/g, " ");
}

export function assetPlacementProfile(asset: PlacementAsset): AssetPlacementProfile {
  const haystack = [
    asset.displayName,
    asset.slug,
    asset.description,
    asset.category,
    asset.catalogKind,
    asset.slot,
    ...(asset.tags ?? []),
  ]
    .filter((part): part is string => typeof part === "string" && part.length > 0)
    .map((part) => tokenize(part))
    .join(" ")
    .toLowerCase();

  const matched = KIND_RULES.find((rule) => rule.pattern.test(haystack));
  const kind: AssetSlotKind = matched?.kind ?? "card";
  const kinds = [...new Set([kind, ...KIND_FAMILY[kind]])];
  return { kind, kinds, size: PREFERRED_SIZE[kind] };
}

function horizontalPad(pad: unknown): number {
  if (typeof pad === "number") return pad;
  if (typeof pad !== "string") return 0;
  const parts = pad.trim().split(/\s+/).map((part) => Number.parseInt(part, 10));
  const horizontal = Number.isFinite(parts[1]) ? parts[1] : parts[0];
  return Number.isFinite(horizontal) ? (horizontal as number) : 0;
}

function numberProp(value: unknown, fallback: number): number {
  return typeof value === "number" && Number.isFinite(value) ? value : fallback;
}

/**
 * Estimates the layout box of a slot without touching the DOM.
 *
 * The walker carries the available width down the authored composition and
 * splits it at rows and grids. Text-sized children in a row can be narrower
 * than their share; that would only shrink a real slot, so the estimate is an
 * upper bound on width and never invents room the page does not have.
 */
export function estimateSlotBox(
  spec: TemplateNode,
  slotId: string,
  viewportWidth: number,
): SlotBox | undefined {
  const walk = (current: TemplateNode, available: number): SlotBox | undefined => {
    const pad = horizontalPad(current.props.pad);
    const inner = Math.max(0, available - pad * 2);
    const children = current.children ?? [];

    if (current.type === "slot") {
      if (current.id !== slotId) return undefined;
      return {
        w: Math.max(0, Math.round(available)),
        h: Math.max(24, numberProp(current.props.minH, DEFAULT_SLOT_HEIGHT)),
      };
    }

    if (current.type === "row") {
      const gap = numberProp(current.props.gap, 10);
      const share = children.length > 0 ? (inner - gap * (children.length - 1)) / children.length : inner;
      for (const child of children) {
        const found = walk(child, Math.max(0, share));
        if (found) return found;
      }
      return undefined;
    }

    if (current.type === "grid") {
      const gap = numberProp(current.props.gap, 12);
      const cols = Math.max(1, numberProp(current.props.cols, 1));
      const share = (inner - gap * (cols - 1)) / cols;
      for (const child of children) {
        const found = walk(child, Math.max(0, share));
        if (found) return found;
      }
      return undefined;
    }

    for (const child of children) {
      const found = walk(child, inner);
      if (found) return found;
    }
    return undefined;
  };

  return walk(spec, Math.max(120, viewportWidth));
}

/** Comfortable, human-readable name for a slot in a picker. */
export function slotLabel(node: TemplateNode): string {
  const labelled = node.props.label;
  if (typeof labelled === "string" && labelled.trim()) return labelled.trim();
  const hint = node.props.hint;
  if (typeof hint === "string" && hint.trim()) return hint.trim();
  const tail = node.id.split(".").pop()?.replace(/[-_]/g, " ");
  return tail && tail !== "slot" ? `Slot · ${tail}` : "Component slot";
}

function scoreRole(
  role: SlotRole | undefined,
  accepts: string[],
  profile: AssetPlacementProfile,
): { points: number; match: SlotRoleMatch } {
  const acceptedByKind =
    accepts.length === 0 || accepts.some((kind) => profile.kinds.includes(kind as AssetSlotKind));
  if (!acceptedByKind) return { points: 6, match: "mismatch" };
  const family = Boolean(role && ROLE_FAMILY[role].includes(profile.kind));
  if (role && role === profile.kind) return { points: 45, match: "exact" };
  if (accepts.some((kind) => profile.kinds.includes(kind as AssetSlotKind))) {
    // Explicit acceptance is stronger than an inferred family, but a role that
    // naturally hosts the kind (action/button, list/card) is still a fit.
    return { points: 34, match: family ? "family" : "accepted" };
  }
  if (family) return { points: 22, match: "family" };
  return { points: 10, match: "role-only" };
}

export function scoreSlot(input: {
  node: TemplateNode;
  index: number;
  profile: AssetPlacementProfile;
  box: SlotBox;
}): SlotScore {
  const { node, index, profile, box } = input;
  const role = slotRole(node);
  const accepts = slotAccepts(node);
  const compatible =
    accepts.length === 0 || accepts.some((kind) => profile.kinds.includes(kind as AssetSlotKind));

  const roleScore = scoreRole(role, accepts, profile);
  const rolePoints = roleScore.points;
  const widthFit = Math.min(1, profile.size.w === 0 ? 0 : box.w / profile.size.w);
  const sizePoints = Math.round(25 * widthFit);
  const positionPoints = Math.max(0, 15 - index * 3);
  const raw = rolePoints + sizePoints + positionPoints;

  return {
    slotId: node.id,
    label: slotLabel(node),
    role,
    accepts,
    compatible,
    score: Math.round((raw / MAX_POINTS) * 100),
    rolePoints,
    sizePoints,
    positionPoints,
    roleMatch: roleScore.match,
    index,
    box,
  };
}

export function placeComponent(input: {
  spec: TemplateNode;
  asset: PlacementAsset;
  viewportWidth: number;
  /** A manual slot choice. Falls back to auto when the slot no longer exists. */
  preferredSlotId?: string | null;
}): PlacementDecision {
  const { spec, asset, viewportWidth, preferredSlotId } = input;
  const profile = assetPlacementProfile(asset);
  const slots = slotsOf(spec);

  const candidates = slots
    .map((node, index) =>
      scoreSlot({
        node,
        index,
        profile,
        box: estimateSlotBox(spec, node.id, viewportWidth) ?? { w: 0, h: 40 },
      }),
    )
    // A tie on total affinity is broken by specificity, not by document order:
    // a slot whose role names the component outranks one that merely accepts
    // it. At 390 the authored KPI slot is estimated narrow (container queries
    // collapse the grid, which the pure estimator cannot see), so it tied the
    // header action; role specificity puts the metric back in the KPI card.
    .sort(
      (left, right) =>
        right.score - left.score ||
        right.rolePoints - left.rolePoints ||
        right.sizePoints - left.sizePoints ||
        left.index - right.index ||
        left.slotId.localeCompare(right.slotId),
    );

  const authoredId = slots[0]?.id;
  const auto = candidates[0];
  const manualMatch = preferredSlotId
    ? candidates.find((candidate) => candidate.slotId === preferredSlotId)
    : undefined;

  return {
    profile,
    authoredId,
    auto,
    chosen: manualMatch ?? auto,
    manual: Boolean(manualMatch),
    candidates,
  };
}
