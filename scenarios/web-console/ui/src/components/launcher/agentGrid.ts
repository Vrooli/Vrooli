// DOC: docs/reference/configuration.md#launcher-shortcuts
/**
 * Building the launcher's agent grid.
 *
 * Three facts have to meet here and they come from different places:
 *
 *   membership — which agents exist on the selected machine, and in what
 *                state. Only the target's capability readiness knows this.
 *   order      — which agent the operator wants first. Only their shortcut
 *                profile knows this, and it is the one thing they can change.
 *   command    — what to actually run. The shortcut owns it; the built-in
 *                verb is a fallback for an agent the profile has never seen.
 *
 * Before this module those three were resolved in the view, in that order of
 * precedence reversed: a hardcoded command map won over the operator's own
 * shortcut, and the order came from the probe table's alphabetical sort. So
 * editing a command in Settings changed nothing, and the list opened with
 * "agy" because that sorts first.
 *
 * The rule now is one sentence: membership from the probe, order and command
 * from the profile, built-ins only where the profile is silent.
 *
 * [REQ:P0-006b] Configurable Shortcut Entries
 * [REQ:P0-014a] Launcher Destination And Appearance Disclosure
 */

import type { ShortcutEntry } from "../../consts/shortcuts";
import type { TargetReadinessFact } from "../../api/targets";

/** Prefix that marks a readiness fact as being about a coding agent. */
const CAPABILITY_PREFIX = "capability:";

/**
 * How a card can present. Four states, four affordances — the grid never
 * shows a launch button for something it cannot launch.
 *
 * "ready"          — installed; pressing it starts a session.
 * "missing"        — absent on this machine; pressing Install runs the
 *                    governed installer.
 * "installing"     — an install this session started is still running.
 * "not-applicable" — no build exists for this machine's os/arch. Inert.
 * "unknown"        — the machine has not reported. Launch is still allowed:
 *                    refusing on an unknown is how a working agent becomes
 *                    unreachable because a probe was slow.
 */
export type AgentCardState = "ready" | "missing" | "installing" | "not-applicable" | "unknown";

export interface AgentCard {
  /** Capability id ("codex"), or "" for an operator command with no agent. */
  agentID: string;
  /** Stable key and test id source. Capability id, else the shortcut label. */
  key: string;
  /** What the card is called. Never a slug when the catalogue knows a name. */
  label: string;
  /** The command this card launches. */
  command: string;
  /** The operator's own words for the entry, when they wrote any. */
  description?: string;
  state: AgentCardState;
  /** Installed version, when the machine reported one. */
  version?: string;
  /** Why it is missing or not applicable, in the machine's own words. */
  detail?: string;
  /** True when this card came from capability facts rather than a shortcut. */
  fromCatalogue: boolean;
}

/** The built-in launch verb per agent, used only when no shortcut supplies one. */
export const FALLBACK_COMMANDS: Record<string, string> = {
  claude: "vrooli agent launch --runner claude --arg=--dangerously-skip-permissions",
  codex: "codex --yolo",
  opencode: "opencode",
  grok: "grok",
  agy: "agy",
};

/**
 * Names for agents the catalogue reports without one. The server sends the
 * real name now; this covers a node running an older agent build, so a stale
 * peer degrades to a proper name rather than to a slug.
 */
const FALLBACK_LABELS: Record<string, string> = {
  claude: "Claude",
  codex: "Codex",
  opencode: "OpenCode",
  grok: "Grok",
  agy: "Antigravity",
};

function cardStateFor(fact: TargetReadinessFact): AgentCardState {
  switch (fact.state) {
    case "ready":
      return "ready";
    case "missing":
      return "missing";
    case "not_applicable":
      return "not-applicable";
    default:
      return fact.passed ? "ready" : "unknown";
  }
}

/**
 * The agent id a shortcut launches.
 *
 * The server resolves this and sends it as `agentId`; this reads it back and
 * nothing else. There is deliberately no client-side derivation from command
 * text — that guess is what the server field replaced, and keeping a second
 * copy here would let the two disagree about the same entry.
 */
export function shortcutAgentID(entry: ShortcutEntry): string {
  return entry.agentId?.trim() ?? "";
}

export interface BuildAgentGridInput {
  /** The selected target's readiness facts. */
  readiness?: TargetReadinessFact[];
  /** The effective shortcut list, in the operator's order. */
  shortcuts: readonly ShortcutEntry[];
  /** Capability ids whose install this session started and has not seen finish. */
  installing?: readonly string[];
}

export interface AgentGrid {
  /** Cards for catalogued agents, in the operator's order. */
  agents: AgentCard[];
  /** The operator's own commands, which belong to no agent. */
  commands: AgentCard[];
}

/**
 * Resolve the grid.
 *
 * When the target reports no capability facts at all — an older Bridge agent,
 * or a catalog that has not answered yet — every shortcut falls through to the
 * command list rather than vanishing. A launcher that renders nothing because
 * a probe is missing is worse than one that renders unverified entries.
 */
export function buildAgentGrid({ readiness, shortcuts, installing = [] }: BuildAgentGridInput): AgentGrid {
  const facts = (readiness ?? []).filter((fact) => fact.key.startsWith(CAPABILITY_PREFIX));
  const factByID = new Map(facts.map((fact) => [fact.key.slice(CAPABILITY_PREFIX.length), fact]));
  const installingSet = new Set(installing);

  // A machine that reports no capability facts at all has not told us
  // anything — an older Bridge agent, or a catalog that has not answered yet.
  // Demoting every agent to a plain command there would be a worse lie than
  // showing the cards in an unknown state: the operator loses the grid, the
  // order, and the install affordance because a probe was silent.
  const catalogueSilent = factByID.size === 0;
  const claimsAgent = (agentID: string) => (catalogueSilent ? true : factByID.has(agentID));

  const shortcutByAgent = new Map<string, ShortcutEntry>();
  const order: string[] = [];
  const commands: AgentCard[] = [];

  for (const entry of shortcuts) {
    const agentID = shortcutAgentID(entry);
    // An entry naming an agent this machine does not report is not an agent
    // card here — it is still a command the operator can run, so it keeps a
    // row rather than being dropped for being off-catalogue.
    if (!agentID || !claimsAgent(agentID)) {
      commands.push({
        agentID: "",
        key: `command:${entry.label}`,
        label: entry.label,
        command: entry.command,
        description: entry.description,
        state: "ready",
        fromCatalogue: false,
      });
      continue;
    }
    if (shortcutByAgent.has(agentID)) continue;
    shortcutByAgent.set(agentID, entry);
    order.push(agentID);
  }

  // Membership is the probe's. An agent the profile has never mentioned still
  // gets a card, appended after the ordered ones, so installing a sixth agent
  // can never produce an invisible card.
  for (const id of factByID.keys()) {
    if (!shortcutByAgent.has(id)) order.push(id);
  }

  const agents = order.map((agentID) => {
    const fact = factByID.get(agentID);
    const shortcut = shortcutByAgent.get(agentID);
    const state = installingSet.has(agentID) ? "installing" : fact ? cardStateFor(fact) : "unknown";
    return {
      agentID,
      key: agentID,
      label: fact?.label || shortcut?.label || FALLBACK_LABELS[agentID] || agentID,
      command: shortcut?.command || FALLBACK_COMMANDS[agentID] || agentID,
      description: shortcut?.description,
      state,
      version: fact?.version || undefined,
      detail: fact?.detail || undefined,
      fromCatalogue: true,
    } satisfies AgentCard;
  });

  return { agents, commands };
}

/** True when pressing the card starts a session rather than an install. */
export function cardLaunches(card: AgentCard): boolean {
  return card.state === "ready" || card.state === "unknown";
}

/** True when the card offers a governed install. */
export function cardInstalls(card: AgentCard): boolean {
  return card.state === "missing";
}

/**
 * Reorder a list by moving one index to another, returning a new array.
 * Shared by the pointer drag and the keyboard shortcut so the two can never
 * disagree about what a move means.
 */
export function moveItem<T>(items: readonly T[], from: number, to: number): T[] {
  if (from < 0 || from >= items.length) return [...items];
  const clamped = Math.max(0, Math.min(items.length - 1, to));
  if (clamped === from) return [...items];
  // slice rather than splice: under noUncheckedIndexedAccess a spliced-out
  // element is T | undefined, and re-inserting it needs a cast that would
  // outlive the reason for it.
  const moving = items.slice(from, from + 1);
  const rest = [...items.slice(0, from), ...items.slice(from + 1)];
  return [...rest.slice(0, clamped), ...moving, ...rest.slice(clamped)];
}

/**
 * Project a reordered agent list back onto a shortcut profile.
 *
 * Order lives in the profile, so persisting it means rewriting the profile's
 * entries. Two rules keep that lossless:
 *
 *   - An agent with no entry yet gains one, carrying the command the grid was
 *     already launching. Without this, reordering an agent the profile has
 *     never seen would silently drop its position on the next read.
 *   - The operator's own commands keep their relative order and follow the
 *     agents, so reordering agents never reshuffles or deletes custom entries.
 */
export function applyAgentOrderToShortcuts(
  shortcuts: readonly ShortcutEntry[],
  agents: readonly AgentCard[],
): ShortcutEntry[] {
  const byAgent = new Map<string, ShortcutEntry>();
  const others: ShortcutEntry[] = [];
  const claimed = new Set(agents.map((card) => card.agentID));

  for (const entry of shortcuts) {
    const agentID = shortcutAgentID(entry);
    if (agentID && claimed.has(agentID) && !byAgent.has(agentID)) {
      byAgent.set(agentID, entry);
      continue;
    }
    others.push(entry);
  }

  const ordered = agents.map((card) => {
    const existing = byAgent.get(card.agentID);
    if (existing) return existing;
    return {
      label: card.label,
      command: card.command,
      description: card.description,
      agentId: card.agentID,
    } satisfies ShortcutEntry;
  });

  return [...ordered, ...others];
}
