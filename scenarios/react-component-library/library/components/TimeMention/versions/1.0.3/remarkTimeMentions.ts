import {
  detectTimeMentions,
  type TimeMentionContext,
  type TimeMentionData,
  type TimeMentionRole,
} from "./timeMentionGrammar";

/**
 * A remark plugin that marks time mentions in prose.
 *
 * Each mention becomes a `<time>` element carrying its detected data in
 * `data-rcl-time-mention`; a renderer maps `time` to `MarkdownTimeMention` to
 * turn it into an interactive mention. Code, links, images and raw HTML are
 * never touched: their text is literal, and people copy it.
 */

interface MarkdownNode {
  type: string;
  value?: string;
  children?: MarkdownNode[];
  data?: Record<string, unknown>;
}

export interface TimeMentionPayload {
  text: string;
  data: TimeMentionData;
  role: TimeMentionRole | null;
}

export const TIME_MENTION_ATTRIBUTE = "data-rcl-time-mention";

const LITERAL_PARENTS = new Set([
  "link",
  "linkReference",
  "image",
  "imageReference",
  "inlineCode",
  "code",
  "html",
  "definition",
]);

function splitText(value: string, context: TimeMentionContext = {}): MarkdownNode[] {
  const mentions = detectTimeMentions(value, context);
  if (mentions.length === 0) return [{ type: "text", value }];
  const nodes: MarkdownNode[] = [];
  let cursor = 0;
  for (const mention of mentions) {
    if (mention.start > cursor)
      nodes.push({ type: "text", value: value.slice(cursor, mention.start) });
    const payload: TimeMentionPayload = {
      text: mention.text,
      data: mention.data,
      role: mention.role,
    };
    nodes.push({
      type: "timeMention",
      value: mention.text,
      data: {
        hName: "time",
        hProperties: { dataRclTimeMention: JSON.stringify(payload) },
        hChildren: [{ type: "text", value: mention.text }],
      },
    });
    cursor = mention.end;
  }
  if (cursor < value.length) nodes.push({ type: "text", value: value.slice(cursor) });
  return nodes;
}

// Blocks whose inline content reads as one run of prose. A mention's role is
// read from the whole block, not just the text node it sits in.
const PROSE_BLOCKS = new Set(["paragraph", "heading", "tableCell"]);

function plainText(node: MarkdownNode): string {
  if (node.type === "text" || node.type === "inlineCode") return node.value ?? "";
  return (node.children ?? []).map(plainText).join("");
}

function transformProse(block: MarkdownNode) {
  const flat = plainText(block);
  let cursor = 0;
  const visit = (node: MarkdownNode) => {
    if (!node.children) return;
    node.children = node.children.flatMap((child) => {
      if (child.type === "text" && child.value) {
        const start = cursor;
        cursor += child.value.length;
        return splitText(child.value, {
          before: flat.slice(Math.max(0, start - 48), start),
          after: flat.slice(cursor, cursor + 32),
        });
      }
      if (child.type === "inlineCode" || LITERAL_PARENTS.has(child.type)) {
        cursor += plainText(child).length;
        return [child];
      }
      visit(child);
      return [child];
    });
  };
  visit(block);
}

function transform(node: MarkdownNode) {
  if (!node.children || LITERAL_PARENTS.has(node.type)) return;
  if (PROSE_BLOCKS.has(node.type)) {
    transformProse(node);
    return;
  }
  node.children = node.children.flatMap((child) => {
    if (child.type === "text" && child.value) return splitText(child.value);
    transform(child);
    return [child];
  });
}

export function remarkTimeMentions() {
  return (tree: MarkdownNode) => {
    transform(tree);
  };
}

/** Reads the payload written by `remarkTimeMentions`, or null when absent or malformed. */
export function readTimeMentionPayload(value: unknown): TimeMentionPayload | null {
  if (typeof value !== "string" || !value) return null;
  try {
    const parsed = JSON.parse(value) as Partial<TimeMentionPayload>;
    return typeof parsed.text === "string" && parsed.data && typeof parsed.data.kind === "string"
      ? { text: parsed.text, data: parsed.data, role: parsed.role ?? null }
      : null;
  } catch {
    return null;
  }
}
