import { layerLabel, type TemplateNode } from "../templates/spec";
import { RCL_PRIMITIVES } from "./rcl";

/**
 * Extraction candidates: repeated custom structures in one frame.
 *
 * The question is "which repeated custom structure is worth turning into a
 * shared component?", and the answer must be reproducible from the document
 * alone. Nothing here guesses about intent, naming or semantics: two subtrees
 * match when their *shape* matches.
 *
 * Shape = `type` + sorted prop **keys** (never values) + child shapes, in
 * document order. Copy, ids and placement are content, so they are ignored:
 * two pricing cards that differ only in their headline and padding still group.
 */
export interface ExtractionCandidate {
  /** Stable structural signature (no ids, no copy). */
  signature: string;
  /** Human label for the example, e.g. "card · title + body". */
  label: string;
  nodeCount: number;
  occurrences: number;
  exampleIds: string[];
  /** (occurrences - 1) * nodeCount — the rough duplication cost. */
  score: number;
}

/**
 * Prop keys that carry content or placement rather than shape. `id` lives on
 * the node, not in `props`, but it is listed here so a future `id` prop cannot
 * leak into a signature.
 */
const CONTENT_PROPS: ReadonlySet<string> = new Set([
  "id",
  "text",
  "value",
  "label",
  "hint",
  "placeholder",
  "alt",
  "src",
  "x",
  "y",
  "w",
  "h",
]);

/**
 * Roots that never become candidates: `page` and `slot` are structural, and
 * an RCL primitive is already a shared asset. Only the *root* decides — a
 * custom `row` repeated inside two cards is still extractable.
 */
const EXCLUDED_ROOTS: ReadonlySet<string> = new Set<string>([
  ...Object.keys(RCL_PRIMITIVES),
  "page",
  "slot",
]);

const MIN_NODE_COUNT = 3;
const MIN_OCCURRENCES = 2;
const MAX_CANDIDATES = 5;

/** How many child descriptors the human label shows before it elides. */
const DESCENDANTS_TO_SHOW = 3;
const LABEL_PART_LENGTH = 32;

interface Shape {
  signature: string;
  nodeCount: number;
}

interface ShapeGroup extends Shape {
  ids: string[];
  example: TemplateNode;
}

/**
 * Signature for one subtree. Props contribute their keys only, sorted so that
 * declaration order cannot change the signature; children keep document order
 * because a card with the title above the body is not the card with the body
 * above the title.
 */
function shapeOf(
  current: TemplateNode,
  groups: Map<string, ShapeGroup>,
  parents: Map<string, string>,
): Shape {
  const children = (current.children ?? []).map((child) => {
    parents.set(child.id, current.id);
    return shapeOf(child, groups, parents);
  });
  const propKeys = Object.keys(current.props)
    .filter((key) => !CONTENT_PROPS.has(key))
    .sort();
  const signature = `${current.type}[${propKeys.join(",")}]{${children
    .map((child) => child.signature)
    .join("|")}}`;
  const nodeCount = 1 + children.reduce((total, child) => total + child.nodeCount, 0);

  if (!EXCLUDED_ROOTS.has(current.type)) {
    const group = groups.get(signature);
    if (group) group.ids.push(current.id);
    else groups.set(signature, { signature, nodeCount, ids: [current.id], example: current });
  }

  return { signature, nodeCount };
}

/** `surface · Pricing + Renewals` — the example's own words, not a guess. */
function labelOf(root: TemplateNode): string {
  const children = root.children ?? [];
  // An imported/authored label ("Group", "Section · Why now") is more useful
  // than the layout type ("stack") and matches what the layer tree shows.
  const authored = typeof root.props.label === "string" ? root.props.label.trim() : "";
  // The authored label can already contain the first heading ("Group · Signals");
  // keep only the role so the child description below does not repeat it.
  const prefix = authored ? (authored.split(" · ")[0] ?? authored) : root.type;
  if (children.length === 0) return prefix;
  const parts = children.slice(0, DESCENDANTS_TO_SHOW).map(describe);
  const more = children.length > DESCENDANTS_TO_SHOW ? " +…" : "";
  return `${prefix} · ${parts.join(" + ")}${more}`;
}

function describe(current: TemplateNode): string {
  const label = layerLabel(current).trim();
  const separator = label.lastIndexOf("·");
  const tail = (separator === -1 ? label : label.slice(separator + 1)).trim();
  const text = tail.length > 0 ? tail : current.type;
  return text.length > LABEL_PART_LENGTH ? `${text.slice(0, LABEL_PART_LENGTH - 1)}…` : text;
}

/** True when `id` is one of `accepted` or a descendant of one. */
function sitsInside(id: string, accepted: ReadonlySet<string>, parents: Map<string, string>): boolean {
  let cursor: string | undefined = id;
  while (cursor !== undefined) {
    if (accepted.has(cursor)) return true;
    cursor = parents.get(cursor);
  }
  return false;
}

/**
 * Ranked extraction candidates for one frame.
 *
 * Noise rule: a smaller structure nested inside a larger repeated structure
 * adds no new information when it repeats exactly as often — the larger one
 * already tells the story. Candidates are therefore considered largest first,
 * and a candidate is dropped when every one of its occurrences sits at or
 * inside an occurrence of an already accepted candidate. (Comparing root ids
 * alone cannot express nesting: the nested copies have different ids.) An
 * inner structure that also repeats somewhere outside the larger one keeps its
 * own entry, because it is telling a second story.
 */
export function extractionCandidates(spec: TemplateNode): ExtractionCandidate[] {
  const groups = new Map<string, ShapeGroup>();
  const parents = new Map<string, string>();
  shapeOf(spec, groups, parents);

  const ranked = [...groups.values()]
    .filter((group) => group.nodeCount >= MIN_NODE_COUNT && group.ids.length >= MIN_OCCURRENCES)
    .map<ExtractionCandidate>((group) => ({
      signature: group.signature,
      label: labelOf(group.example),
      nodeCount: group.nodeCount,
      occurrences: group.ids.length,
      exampleIds: [...group.ids],
      score: (group.ids.length - 1) * group.nodeCount,
    }))
    .sort(
      (left, right) =>
        right.nodeCount - left.nodeCount || left.label.localeCompare(right.label),
    );

  const accepted: ExtractionCandidate[] = [];
  const acceptedIds = new Map<ExtractionCandidate, Set<string>>();
  for (const candidate of ranked) {
    const subsumed = accepted.some((larger) => {
      const ids = acceptedIds.get(larger);
      if (!ids) return false;
      return candidate.exampleIds.every((id) => sitsInside(id, ids, parents));
    });
    if (subsumed) continue;
    accepted.push(candidate);
    acceptedIds.set(candidate, new Set(candidate.exampleIds));
  }

  return accepted
    .sort(
      (left, right) =>
        right.score - left.score ||
        right.nodeCount - left.nodeCount ||
        left.label.localeCompare(right.label),
    )
    .slice(0, MAX_CANDIDATES);
}
