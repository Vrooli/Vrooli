import { layerLabel, walk, type TemplateNode, type TemplateNodeType } from "../templates/spec";
import { AssetKind } from "@vrooli/proto-types/react-component-library/v1/components/components_pb";
import type { CatalogAsset } from "../data";

export interface ReplacementCandidate {
  type: TemplateNodeType;
  name: string;
  kind: "primitive" | "pattern";
  description: string;
  keywords: readonly string[];
  acceptsChildren: boolean;
}

export interface RankedCatalogCandidate {
  id: string;
  name: string;
  description: string;
  tags: readonly string[];
  score: number;
}

/** Native RCL vocabulary. Catalog components need an explicit renderer binding. */
export const REPLACEMENT_CANDIDATES: readonly ReplacementCandidate[] = [
  {
    type: "button",
    name: "Button",
    kind: "primitive",
    description: "A clear action or call to action.",
    keywords: [
      "action",
      "submit",
      "cta",
      "save",
      "continue",
      "confirm",
      "pay",
      "purchase",
      "buy",
      "add",
      "send",
      "start",
      "finish",
      "apply",
    ],
    acceptsChildren: true,
  },
  {
    type: "input",
    name: "Input",
    kind: "primitive",
    description: "A text entry or search field.",
    keywords: ["field", "form", "search", "email", "password", "text entry"],
    acceptsChildren: false,
  },
  {
    type: "card",
    name: "Card",
    kind: "primitive",
    description: "A contained piece of content.",
    keywords: ["panel", "pricing", "product", "article", "tile", "content"],
    acceptsChildren: true,
  },
  {
    type: "stat",
    name: "Stat",
    kind: "primitive",
    description: "A compact headline value with supporting detail.",
    keywords: ["kpi", "count", "number", "summary", "total"],
    acceptsChildren: false,
  },
  {
    type: "avatar",
    name: "Avatar",
    kind: "primitive",
    description: "A person, account, or team identity.",
    keywords: ["profile", "person", "user", "team", "member", "identity"],
    acceptsChildren: false,
  },
  {
    type: "chip",
    name: "Chip",
    kind: "primitive",
    description: "A short status, tag, or filter label.",
    keywords: ["tag", "badge", "status", "category", "filter"],
    acceptsChildren: false,
  },
  {
    type: "meter",
    name: "Meter",
    kind: "primitive",
    description: "Progress toward a limit or goal.",
    keywords: ["progress", "completion", "quota", "capacity"],
    acceptsChildren: false,
  },
  {
    type: "stack",
    name: "Stack",
    kind: "pattern",
    description: "A vertical group with consistent spacing.",
    keywords: ["section", "vertical", "group", "flow", "column"],
    acceptsChildren: true,
  },
  {
    type: "row",
    name: "Row",
    kind: "pattern",
    description: "A horizontal group of related items.",
    keywords: ["toolbar", "navigation", "actions", "horizontal", "inline"],
    acceptsChildren: true,
  },
  {
    type: "grid",
    name: "Grid",
    kind: "pattern",
    description: "A responsive grid of related content.",
    keywords: ["gallery", "cards", "columns", "list", "tiles"],
    acceptsChildren: true,
  },
  {
    type: "surface",
    name: "Surface",
    kind: "pattern",
    description: "A themed region for nested content.",
    keywords: ["panel", "container", "region", "section", "background"],
    acceptsChildren: true,
  },
  {
    type: "slot",
    name: "Component slot",
    kind: "pattern",
    description: "A named region prepared for a live component.",
    keywords: ["placeholder", "component", "embed", "mount", "region"],
    acceptsChildren: true,
  },
];

const words = (value: string): string[] =>
  value
    .toLowerCase()
    .match(/[a-z0-9]+/g)
    ?.filter((word) => word.length > 1) ?? [];

// Imported layers often carry hundreds of CSS declarations or an inline SVG
// data URL. Those values describe rendering and must not affect semantic
// component matching. Read visible labels and authored semantics across the
// selected subtree instead.
const SEARCHABLE_NODE_PROPS = [
  "text",
  "label",
  "name",
  "title",
  "hint",
  "placeholder",
  "alt",
  "role",
  "variant",
  "kind",
  "category",
] as const;

function semanticText(node: TemplateNode): string {
  const values: string[] = [];
  walk(node, (current) => {
    values.push(layerLabel(current), current.type);
    if (current.role) values.push(current.role);
    for (const key of SEARCHABLE_NODE_PROPS) {
      const value = current.props[key];
      if (typeof value === "string") values.push(value);
    }
  });
  return values.join(" ");
}

/** Stable token-overlap rank. The selected node gets a strong exact-type bias. */
export function rankReplacementCandidates(node: TemplateNode, query = ""): ReplacementCandidate[] {
  if (node.type === "page") return [];
  const hasChildren = Boolean(node.children?.length);
  const semantic = new Set(words(semanticText(node)));
  const queryWords = words(query.trim());
  const button = REPLACEMENT_CANDIDATES.find((candidate) => candidate.type === "button");
  const buttonIntent = new Set(button?.keywords ?? []);
  const buttonIsRelevant =
    node.type === "button" ||
    [...semantic].some((word) => buttonIntent.has(word)) ||
    queryWords.some((word) => "button".includes(word) || buttonIntent.has(word));

  return REPLACEMENT_CANDIDATES.filter((candidate) => !hasChildren || candidate.acceptsChildren)
    .filter((candidate) => candidate.type !== "button" || buttonIsRelevant)
    .map((candidate, order) => {
      const candidateText = `${candidate.name} ${candidate.type} ${candidate.description} ${candidate.keywords.join(" ")}`;
      const candidateWords = new Set(words(candidateText));
      const semanticScore = [...candidateWords].reduce(
        (score, word) => score + (semantic.has(word) ? 2 : 0),
        0,
      );
      const queryScore = queryWords.reduce(
        (score, word) =>
          score +
          ([...candidateWords].some((candidateWord) => candidateWord.includes(word)) ? 1 : 0),
        0,
      );
      const exactTypeBias = candidate.type === node.type ? 8 : 0;
      return {
        candidate,
        score: exactTypeBias + semanticScore + queryScore,
        order,
        candidateText: candidateText.toLowerCase(),
      };
    })
    .filter(
      ({ candidateText }) =>
        queryWords.length === 0 || queryWords.every((word) => candidateText.includes(word)),
    )
    .sort((left, right) => right.score - left.score || left.order - right.order)
    .map(({ candidate }) => candidate);
}

/** Convert a selected native layer while retaining identity and authored content. */
export function asReplacementNode(node: TemplateNode, type: TemplateNodeType): TemplateNode {
  if (node.type === "page") return node;
  const candidate = REPLACEMENT_CANDIDATES.find((entry) => entry.type === type);
  if (!candidate || (node.children?.length && !candidate.acceptsChildren)) return node;
  const props = { ...node.props };
  const content =
    typeof props.text === "string"
      ? props.text
      : typeof props.label === "string"
        ? props.label
        : "";
  if (content && type === "input" && typeof props.placeholder !== "string")
    props.placeholder = content;
  if (type === "slot" && typeof props.minH !== "number") props.minH = 40;
  return { ...node, type, props };
}

/** Rank real published catalog components against a selected custom region. */
export function rankCatalogCandidates(
  selected: TemplateNode,
  candidates: readonly CatalogAsset[],
  query = "",
): RankedCatalogCandidate[] {
  if (selected.type === "page") return [];
  const semantic = new Set(words(semanticText(selected)));
  const queryWords = words(query.trim());
  return candidates
    .filter((asset) => asset.assetKind === AssetKind.COMPONENT)
    .map((asset, order) => {
      const name = asset.displayName || asset.libraryId || asset.id;
      const tags = asset.tags;
      const text = [
        name,
        asset.description,
        asset.libraryId,
        asset.slug,
        asset.slot,
        asset.category,
        asset.catalogDomain,
        asset.catalogKind,
        tags.join(" "),
      ].join(" ");
      const tokens = new Set(words(text));
      const semanticScore = [...tokens].reduce(
        (score, word) => score + (semantic.has(word) ? 2 : 0),
        0,
      );
      const queryScore = queryWords.reduce(
        (score, word) => score + ([...tokens].some((token) => token.includes(word)) ? 1 : 0),
        0,
      );
      return {
        id: asset.id,
        name,
        description: asset.description || tags.join(" · ") || "Published RCL component",
        tags,
        score: semanticScore + queryScore,
        text: text.toLowerCase(),
        order,
      };
    })
    .filter(
      ({ text }) => queryWords.length === 0 || queryWords.every((word) => text.includes(word)),
    )
    .sort((left, right) => right.score - left.score || left.order - right.order)
    .slice(0, queryWords.length === 0 ? 8 : 50)
    .map(({ text: _text, order: _order, ...candidate }) => candidate);
}

/** Preserve the source subtree as a slot fallback while binding a live catalog component. */
export function asCatalogReplacementNode(
  selected: TemplateNode,
  componentId: string,
  storyId: string,
): TemplateNode {
  if (selected.type === "page") return selected;
  const hint = layerLabel(selected);
  return {
    ...selected,
    type: "slot",
    props: {
      ...selected.props,
      minH: typeof selected.props.minH === "number" ? selected.props.minH : 56,
      ...(hint ? { hint } : {}),
      componentId,
      storyId,
    },
  };
}
