import { matchProseFilePaths } from "../../../lib/fileReferences";

// remarkProsePaths — turns strongly path-shaped tokens in plain prose into
// link nodes so bare paths (no backticks, no link syntax) get the same
// click-to-preview affordance as authored references. The emitted links carry
// `data-prose-path` so the renderer can style them quieter than real links.
//
// Only text nodes are transformed; code, inline code, existing links, images,
// and raw HTML subtrees are skipped so authored markup is never re-decorated.

interface MdastNode {
  type: string;
  value?: string;
  url?: string;
  children?: MdastNode[];
  data?: { hProperties?: Record<string, string> };
}

const SKIPPED_PARENTS = new Set(["code", "inlineCode", "link", "linkReference", "image", "imageReference", "definition", "html"]);

function transformTextNode(node: MdastNode): MdastNode[] | null {
  const text = node.value ?? "";
  const matches = matchProseFilePaths(text);
  if (matches.length === 0) return null;

  const replacement: MdastNode[] = [];
  let cursor = 0;
  for (const match of matches) {
    if (match.start > cursor) {
      replacement.push({ type: "text", value: text.slice(cursor, match.start) });
    }
    replacement.push({
      type: "link",
      url: match.path,
      data: { hProperties: { "data-prose-path": "true" } },
      children: [{ type: "text", value: match.path }],
    });
    cursor = match.end;
  }
  if (cursor < text.length) {
    replacement.push({ type: "text", value: text.slice(cursor) });
  }
  return replacement;
}

function walk(node: MdastNode): void {
  if (!node.children || SKIPPED_PARENTS.has(node.type)) return;
  for (let i = 0; i < node.children.length; i++) {
    const child = node.children[i];
    if (!child) continue;
    if (child.type === "text") {
      const replacement = transformTextNode(child);
      if (replacement) {
        node.children.splice(i, 1, ...replacement);
        i += replacement.length - 1;
      }
      continue;
    }
    walk(child);
  }
}

export function remarkProsePaths() {
  return (tree: unknown) => {
    walk(tree as MdastNode);
  };
}
