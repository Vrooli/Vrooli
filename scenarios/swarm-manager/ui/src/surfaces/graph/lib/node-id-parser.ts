/**
 * Node ID Parser
 *
 * Parses graph node IDs into their component parts for API resource path construction.
 * Node IDs follow the pattern: "{entity-prefix}/{identifier}" where identifier
 * may contain slashes (e.g., "execute/my-feature" for backlog items).
 */

import type { EntityType } from "../stores/graph-data-store";

export interface ParsedNodeId {
  entityType: EntityType;
  /** The raw identifier after stripping the entity prefix. */
  identifier: string;
  /** For backlog items: the kind (execute, research, etc.). */
  kind?: string;
  /** For backlog items: the name. For others: same as identifier. */
  name?: string;
}

const PREFIX_MAP: Record<string, EntityType> = {
  "backlog-item": "backlog",
  "scenario": "scenario",
  "execution-record": "execution",
  "agent-run": "agent-run",
  "capture": "capture",
  "initiative": "initiative",
};

/**
 * Parse a node ID into its component parts.
 * Returns null for unrecognized formats.
 *
 * Node ID formats:
 * - backlog-item/{kind}/{name}  → entityType: "backlog", kind, name
 * - scenario/{name}             → entityType: "scenario", name
 * - execution-record/{id}       → entityType: "execution", identifier
 * - agent-run/{runId}           → entityType: "agent-run", identifier
 * - capture/{id}                → entityType: "capture", identifier
 * - initiative/{name}           → entityType: "initiative", name
 *
 * Legacy format (from client-side assembler):
 * - {kind}/{name}               → entityType: "backlog" (when kind matches known backlog kinds)
 * - execution/{id}              → entityType: "execution"
 */
const BACKLOG_KINDS = new Set(["execute", "research", "investigate", "design", "improve", "fix", "test", "document"]);

export function parseNodeId(nodeId: string): ParsedNodeId | null {
  if (!nodeId) return null;

  // Try prefixed format first (server-side projection IDs).
  for (const [prefix, entityType] of Object.entries(PREFIX_MAP)) {
    if (nodeId.startsWith(`${prefix}/`)) {
      const rest = nodeId.slice(prefix.length + 1);
      if (!rest) return null;

      if (entityType === "backlog") {
        // backlog-item/{kind}/{name}
        const slashIdx = rest.indexOf("/");
        if (slashIdx === -1) return null;
        return {
          entityType,
          identifier: rest,
          kind: rest.slice(0, slashIdx),
          name: rest.slice(slashIdx + 1),
        };
      }

      return {
        entityType,
        identifier: rest,
        name: rest,
      };
    }
  }

  // Legacy format: execution/{id}
  if (nodeId.startsWith("execution/")) {
    const id = nodeId.slice("execution/".length);
    if (!id) return null;
    return { entityType: "execution", identifier: id };
  }

  // Legacy format: capture/{id}
  if (nodeId.startsWith("capture/")) {
    const id = nodeId.slice("capture/".length);
    if (!id) return null;
    return { entityType: "capture", identifier: id };
  }

  // Legacy format: agent-run/{id}
  if (nodeId.startsWith("agent-run/")) {
    const id = nodeId.slice("agent-run/".length);
    if (!id) return null;
    return { entityType: "agent-run", identifier: id };
  }

  // Legacy format: scenario/{name}
  if (nodeId.startsWith("scenario/")) {
    const name = nodeId.slice("scenario/".length);
    if (!name) return null;
    return { entityType: "scenario", identifier: name, name };
  }

  // Legacy format: initiative/{name}
  if (nodeId.startsWith("initiative/")) {
    const name = nodeId.slice("initiative/".length);
    if (!name) return null;
    return { entityType: "initiative", identifier: name, name };
  }

  // Legacy backlog format: {kind}/{name} (from client-side assembler)
  const slashIdx = nodeId.indexOf("/");
  if (slashIdx > 0) {
    const maybeKind = nodeId.slice(0, slashIdx);
    if (BACKLOG_KINDS.has(maybeKind)) {
      const name = nodeId.slice(slashIdx + 1);
      if (!name) return null;
      return {
        entityType: "backlog",
        identifier: nodeId,
        kind: maybeKind,
        name,
      };
    }
  }

  return null;
}
