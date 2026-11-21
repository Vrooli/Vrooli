import type { DesignDocument, DesignNewLayer, DesignPatchOperation } from "../designs";
import { CONTAINER_TYPES, PALETTE_KEYS } from "./model";
import type { TemplateNode, TemplateNodeType, TemplatePalette } from "../templates/spec";

/**
 * Applying a proposal is a local, synchronous edit.
 *
 * The server already checked the patch against the document it read. This
 * module checks it against the document the person is looking at *now*,
 * because a proposal can arrive after an edit, another tab's save, or a
 * deleted layer. It is the last gate before the command stack, and it never
 * mutates the document it is given.
 */
export class PatchError extends Error {
  readonly reason: string;

  constructor(reason: string) {
    super(reason);
    this.name = "PatchError";
    this.reason = reason;
  }
}

export interface AppliedPatch {
  doc: DesignDocument;
  /** Node ids the patch touched, for highlighting and the diff summary. */
  changed: string[];
}

const TEXT_TYPES = new Set<TemplateNodeType>(["text", "button", "chip"]);
const ADDABLE_TYPES = new Set<TemplateNodeType>([
  "stack",
  "row",
  "grid",
  "surface",
  "text",
  "button",
  "input",
  "card",
  "stat",
  "metric",
  "icon",
  "avatar",
  "chip",
  "meter",
  "divider",
  "spacer",
  "swatch",
]);
const BLOCKED_PROPS = new Set(["id", "children", "src", "href", "role", "accepts"]);
const PLACEMENT_PROPS = new Set(["x", "y", "w", "h"]);
const ID_PATTERN = /^[A-Za-z0-9][A-Za-z0-9._:-]{0,199}$/;
const PROP_PATTERN = /^[a-zA-Z][a-zA-Z0-9]{0,39}$/;
const HEX_COLOUR = /^#[0-9a-fA-F]{3,8}$/;
const NAMED_COLOUR = /^[a-zA-Z]{3,20}$/;
const MAX_OPERATIONS = 8;
const MAX_DEPTH = 12;
const MAX_NODES = 500;
const MAX_TEXT = 2000;

interface Located {
  node: TemplateNode;
  parent?: TemplateNode;
  depth: number;
}

export interface ApplyOptions {
  /** Deterministic ids make the apply path testable. */
  idFactory?: (parentId: string, type: TemplateNodeType, sequence: number) => string;
}

export function applyPatch(
  doc: DesignDocument,
  operations: DesignPatchOperation[],
  options: ApplyOptions = {},
): AppliedPatch {
  if (operations.length === 0 || operations.length > MAX_OPERATIONS) {
    throw new PatchError(`a patch needs 1 to ${MAX_OPERATIONS} operations`);
  }
  const next: DesignDocument = {
    ...doc,
    theme: { ...doc.theme },
    frames: doc.frames.map((frame) => ({ ...frame, spec: cloneNode(frame.spec) })),
  };
  const changed = new Set<string>();
  let sequence = 0;
  const idFactory =
    options.idFactory ?? ((parentId, type, index) => `${parentId}.${type}-${Date.now().toString(36)}-${index}`);

  const totalNodes = () => {
    let count = 0;
    for (const frame of next.frames) {
      walkNode(frame.spec, () => {
        count += 1;
      });
    }
    return count;
  };

  for (const [position, operation] of operations.entries()) {
    const problem = (reason: string) =>
      new PatchError(`operation ${position + 1} (${operation.kind}): ${reason}`);
    // `fail` is the terse path; a `throw problem(...)` is used where the
    // following line needs TypeScript's narrowing, which does not survive a
    // call through a function variable.
    const fail = (reason: string): never => {
      throw problem(reason);
    };
    const index = indexNodes(next);
    switch (operation.kind) {
      case "setText": {
        const located = locate(index, operation.nodeId, fail);
        const text = operation.text;
        if (typeof text !== "string") throw problem("text is required");
        if (!TEXT_TYPES.has(located.node.type)) fail(`layer type ${located.node.type} does not render text`);
        assertCopy(text, fail);
        located.node.props = { ...located.node.props, text };
        changed.add(located.node.id);
        break;
      }
      case "setProp": {
        const located = locate(index, operation.nodeId, fail);
        const key = operation.key;
        if (!key) throw problem("key is required");
        const value = operation.value;
        if (value === undefined) throw problem("value is required");
        assertProp(located.node, key, value, fail);
        located.node.props = {
          ...located.node.props,
          [key]: value,
        };
        changed.add(located.node.id);
        break;
      }
      case "setTheme": {
        const key = operation.key;
        if (!key || !PALETTE_KEYS.includes(key as keyof TemplatePalette)) {
          fail(`theme key ${JSON.stringify(key)} is not a palette token`);
        }
        const value = operation.value;
        if (typeof value !== "string" || !(HEX_COLOUR.test(value) || NAMED_COLOUR.test(value))) {
          fail("theme value must be a hex colour or a colour name");
        }
        next.theme = { ...next.theme, [key as keyof TemplatePalette]: value as string };
        changed.add(`theme.${key as string}`);
        break;
      }
      case "addLayer": {
        const parent = locate(index, operation.parentId, fail);
        if (!CONTAINER_TYPES.has(parent.node.type)) fail(`layer type ${parent.node.type} cannot own children`);
        const newNode = operation.node;
        if (!newNode) throw problem("node is required");
        const built = buildLayer(newNode, parent.node.id, idFactory, () => {
          sequence += 1;
          return sequence;
        }, fail);
        if (totalNodes() + countNodes(built) > MAX_NODES) fail(`the design would exceed ${MAX_NODES} layers`);
        if (parent.depth + 1 + layerDepth(newNode) > MAX_DEPTH) {
          fail(`the layer would nest deeper than ${MAX_DEPTH} levels`);
        }
        parent.node.children = [...(parent.node.children ?? []), built];
        changed.add(built.id);
        break;
      }
      case "removeLayer": {
        const located = locate(index, operation.nodeId, fail);
        const parent = located.parent;
        if (!parent) throw problem("the page root cannot be removed");
        if (located.node.type === "slot") fail("component slots are structural and cannot be removed");
        parent.children = (parent.children ?? []).filter((child) => child.id !== located.node.id);
        changed.add(located.node.id);
        break;
      }
      case "moveLayer": {
        const located = locate(index, operation.nodeId, fail);
        const origin = located.parent;
        if (!origin) throw problem("the page root cannot be moved");
        const destination = locate(index, operation.parentId, fail);
        if (!CONTAINER_TYPES.has(destination.node.type)) fail(`layer type ${destination.node.type} cannot own children`);
        if (destination.node.id === located.node.id || isDescendant(index, destination.node.id, located.node.id)) {
          fail("a layer cannot move into itself");
        }
        origin.children = (origin.children ?? []).filter((child) => child.id !== located.node.id);
        const list = destination.node.children ?? [];
        const at = operation.index ?? list.length;
        if (!Number.isInteger(at) || at < 0 || at > list.length) fail(`index ${String(operation.index)} is outside the destination`);
        const copy = [...list];
        copy.splice(at, 0, located.node);
        destination.node.children = copy;
        changed.add(located.node.id);
        break;
      }
      default:
        fail(`unknown kind ${String((operation as { kind?: unknown }).kind)}`);
    }
  }
  return { doc: next, changed: [...changed] };
}

function locate(
  index: Map<string, Located>,
  id: string | undefined,
  fail: (reason: string) => never,
): Located {
  if (!id || !ID_PATTERN.test(id)) fail(`layer id ${JSON.stringify(id)} is not a valid id`);
  const located = index.get(id);
  if (!located) fail(`layer ${id} does not exist`);
  return located;
}

function indexNodes(doc: DesignDocument): Map<string, Located> {
  const index = new Map<string, Located>();
  const visit = (node: TemplateNode, parent: TemplateNode | undefined, depth: number) => {
    index.set(node.id, { node, parent, depth });
    for (const child of node.children ?? []) visit(child, node, depth + 1);
  };
  for (const frame of doc.frames) visit(frame.spec, undefined, 1);
  return index;
}

function assertCopy(text: string, fail: (reason: string) => never): void {
  if (Array.from(text).length > MAX_TEXT) fail(`copy exceeds ${MAX_TEXT} characters`);
  for (const character of text) {
    if (character === "\n" || character === "\t") continue;
    const code = character.codePointAt(0) ?? 0;
    if (code < 0x20 || code === 0x7f) fail("copy contains a control character");
  }
}

function assertProp(
  node: TemplateNode,
  key: string | undefined,
  value: string | number | boolean | undefined,
  fail: (reason: string) => never,
): void {
  if (!key || !PROP_PATTERN.test(key)) fail(`prop key ${JSON.stringify(key)} is not allowed`);
  if (BLOCKED_PROPS.has(key) || key.startsWith("on") || key.startsWith("_")) {
    fail(`prop ${JSON.stringify(key)} cannot be set by a proposal`);
  }
  if (PLACEMENT_PROPS.has(key) && node.type !== "slot") {
    fail(`placement prop ${JSON.stringify(key)} is only valid on a slot`);
  }
  if (typeof value === "string") {
    assertCopy(value, fail);
    return;
  }
  if (typeof value === "number") {
    if (!Number.isFinite(value) || Math.abs(value) > 100_000) fail(`prop ${JSON.stringify(key)} is out of range`);
    return;
  }
  if (typeof value !== "boolean") fail(`prop ${JSON.stringify(key)} must be a string, number or boolean`);
}

function buildLayer(
  source: DesignNewLayer,
  parentId: string,
  idFactory: (parentId: string, type: TemplateNodeType, sequence: number) => string,
  nextSequence: () => number,
  fail: (reason: string) => never,
): TemplateNode {
  if (!ADDABLE_TYPES.has(source.type)) fail(`layer type ${source.type} cannot be added by a proposal`);
  if (Object.keys(source.props).length > 16) fail("a new layer may carry at most 16 props");
  for (const [key, value] of Object.entries(source.props)) {
    assertProp({ id: "", type: source.type, props: {} }, key, value, fail);
  }
  if ((source.children?.length ?? 0) > 6) fail("a new layer may carry at most 6 children");
  const id = idFactory(parentId, source.type, nextSequence());
  const node: TemplateNode = {
    id,
    type: source.type,
    props: { ...source.props },
  };
  if (source.children && source.children.length > 0) {
    node.children = source.children.map((child) => {
      if ((child.children?.length ?? 0) > 0) fail("a new layer may only nest one level");
      return buildLayer(child, id, idFactory, nextSequence, fail);
    });
  }
  return node;
}

function layerDepth(source: DesignNewLayer): number {
  return source.children && source.children.length > 0 ? 2 : 1;
}

function countNodes(node: TemplateNode): number {
  return 1 + (node.children ?? []).reduce((total, child) => total + countNodes(child), 0);
}

function isDescendant(index: Map<string, Located>, candidateId: string, ancestorId: string): boolean {
  let cursor = index.get(candidateId)?.parent;
  while (cursor) {
    if (cursor.id === ancestorId) return true;
    cursor = index.get(cursor.id)?.parent;
  }
  return false;
}

function cloneNode(node: TemplateNode): TemplateNode {
  return {
    ...node,
    props: { ...node.props },
    children: node.children?.map(cloneNode),
  };
}

function walkNode(node: TemplateNode, visit: (current: TemplateNode) => void): void {
  visit(node);
  for (const child of node.children ?? []) walkNode(child, visit);
}
