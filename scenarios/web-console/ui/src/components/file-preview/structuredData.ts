
export const MAX_STRUCTURE_CHARACTERS = 300_000;
export const MAX_STRUCTURE_NODES = 5000;
export const MAX_STRUCTURE_DEPTH = 60;
export type StructureKind = "object" | "array" | "string" | "number" | "boolean" | "null" | "reference";
export interface StructureNode {
  id: number; parent: number | null; key: string; path: string; kind: StructureKind;
  value: string; start: number; end: number; children: number[];
}
export interface StructureDocument { nodes: StructureNode[]; root: number }
export type StructureResult = { documents: StructureDocument[]; error?: never } | { error: "invalid" | "limit" | "unsupported"; documents?: never };
export function isStructuredPath(path: string) { return /\.(json|ya?ml)$/i.test(path); }

// Re-indent validated JSON tokens without round-tripping numeric values through
// JavaScript numbers. Strings and every scalar token keep their original bytes.
export function formatJsonSource(source: string): string {
  JSON.parse(source);
  let quoted = false, escaped = false, compact = "";
  for (const character of source) {
    if (quoted || !/\s/.test(character)) compact += character;
    if (character === '"' && !escaped) quoted = !quoted;
    escaped = quoted && character === "\\" && !escaped;
  }
  let depth = 0, output = "";
  quoted = false; escaped = false;
  const newline = () => { output += `\n${"  ".repeat(depth)}`; };
  for (let index = 0; index < compact.length; index++) {
    const character = compact.charAt(index);
    if (quoted) output += character;
    else if (character === "{" || character === "[") {
      output += character; depth++;
      if (compact[index + 1] !== "}" && compact[index + 1] !== "]") newline();
    } else if (character === "}" || character === "]") {
      depth--; if (compact[index - 1] !== "{" && compact[index - 1] !== "[") newline(); output += character;
    } else if (character === ",") { output += character; newline(); }
    else output += character === ":" ? ": " : character;
    if (character === '"' && !escaped) quoted = !quoted;
    escaped = quoted && character === "\\" && !escaped;
  }
  return output;
}

export function visibleStructure(document: StructureDocument, expanded: Set<number>, query: string) {
  const matches = new Set<number>(), included = new Set<number>();
  const needle = query.trim().toLocaleLowerCase();
  if (needle) for (const node of document.nodes) {
    if (`${node.key}\n${node.path}\n${node.value}`.toLocaleLowerCase().includes(needle)) {
      matches.add(node.id);
      let ancestor: StructureNode | undefined = node;
      while (ancestor && !included.has(ancestor.id)) { included.add(ancestor.id); ancestor = ancestor.parent === null ? undefined : document.nodes[ancestor.parent]; }
    }
  }
  const rows: { node: StructureNode; depth: number }[] = [];
  const walk = (id: number, depth: number) => {
    const node = document.nodes[id];
    if (!node || (needle && !included.has(id))) return;
    rows.push({ node, depth });
    if (needle || expanded.has(id)) node.children.forEach((child) => { walk(child, depth + 1); });
  };
  walk(document.root, 0);
  return { rows, matches };
}
