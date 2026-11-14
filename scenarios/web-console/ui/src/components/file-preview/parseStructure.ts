import { isAlias, isMap, isNode, isScalar, isSeq, parseAllDocuments } from "yaml";
import { MAX_STRUCTURE_CHARACTERS, MAX_STRUCTURE_DEPTH, MAX_STRUCTURE_NODES, type StructureDocument, type StructureKind, type StructureNode, type StructureResult } from "./structuredData";

// Inspect syntax nodes rather than constructing JavaScript objects. This retains
// numeric spelling, never expands aliases, and cannot activate custom YAML tags.
export function parseStructure(source: string, json: boolean): StructureResult {
  if (source.length > MAX_STRUCTURE_CHARACTERS) return { error: "limit" };
  try {
    if (json) JSON.parse(source); // Strict JSON syntax; never use the rounded result.
    const documents = parseAllDocuments(source, { intAsBigInt: true, uniqueKeys: true, strict: true });
    if (!documents.length || documents.some((doc) => doc.errors.length)) return { error: "invalid" };
    if (documents.some((doc) => doc.warnings.length)) return { error: "unsupported" };
    let count = 0;
    const output: StructureDocument[] = [];
    for (const document of documents) {
      const nodes: StructureNode[] = [];
      const visit = (input: unknown, key: string, path: string, parent: number | null, depth: number): number => {
        if (input !== null && input !== undefined && !isNode(input)) throw new TypeError();
        const node = isNode(input) ? input : null;
        if (++count > MAX_STRUCTURE_NODES || depth > MAX_STRUCTURE_DEPTH) throw new RangeError();
        if (node?.tag && !/^tag:yaml\.org,2002:(map|seq|str|int|float|bool|null)$/.test(node.tag)) throw new TypeError();
        if (isScalar(node) && node.value !== null && !["string", "number", "bigint", "boolean"].includes(typeof node.value)) throw new TypeError();
        const id = nodes.length;
        const start = node?.range?.[0] ?? 0, end = node?.range?.[1] ?? start;
        const kind: StructureKind = isMap(node) ? "object" : isSeq(node) ? "array" : isAlias(node) ? "reference" : !isScalar(node) || node.value === null ? "null" : typeof node.value === "string" ? "string" : typeof node.value === "boolean" ? "boolean" : "number";
        const value = isScalar(node) && typeof node.value === "string" ? node.value : source.slice(start, end) || "null";
        const result: StructureNode = { id, parent, key, path, kind, value: kind === "object" || kind === "array" ? "" : value, start, end, children: [] };
        nodes.push(result);
        if (isMap(node)) {
          const keys = new Set<string>();
          for (const pair of node.items) {
            if (!isScalar(pair.key)) throw new TypeError();
            const name = typeof pair.key.value === "string" ? pair.key.value : pair.key.source ?? String(pair.key.value);
            if (keys.has(name)) throw new TypeError(); // Distinct YAML keys must not share a displayed path.
            keys.add(name);
            const childPath = /^[A-Za-z_$][\w$]*$/.test(name) ? `${path}.${name}` : `${path}[${JSON.stringify(name)}]`;
            result.children.push(visit(pair.value, name, childPath, id, depth + 1));
          }
        } else if (isSeq(node)) {
          node.items.forEach((child, index) => { result.children.push(visit(child, String(index), `${path}[${String(index)}]`, id, depth + 1)); });
        }
        return id;
      };
      const root = visit(document.contents, "$", "$", null, 0);
      output.push({ nodes, root });
    }
    return { documents: output };
  } catch (error) { return { error: error instanceof RangeError ? "limit" : error instanceof TypeError ? "unsupported" : "invalid" }; }
}
