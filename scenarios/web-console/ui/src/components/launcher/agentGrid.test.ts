import { describe, expect, it } from "vitest";

import { DEFAULT_SHORTCUTS, type ShortcutEntry } from "../../consts/shortcuts";
import type { TargetReadinessFact } from "../../api/targets";
import {
  applyAgentOrderToShortcuts,
  buildAgentGrid,
  cardInstalls,
  cardLaunches,
  moveItem,
} from "./agentGrid";

const fact = (id: string, over: Partial<TargetReadinessFact> = {}): TargetReadinessFact => ({
  key: `capability:${id}`,
  label: { claude: "Claude Code", codex: "Codex", opencode: "OpenCode", grok: "Grok", agy: "Antigravity" }[id] ?? id,
  passed: true,
  detail: "",
  state: "ready",
  ...over,
});

const READY_LOCAL: TargetReadinessFact[] = [
  { key: "local_process", label: "Web Console process", passed: true, detail: "" },
  fact("agy", { version: "1.1.15" }),
  fact("claude", { version: "2.1.4" }),
  fact("codex", { version: "0.149.1" }),
  fact("grok", { version: "1.0.9" }),
  fact("opencode", { version: "0.4.2" }),
];

describe("buildAgentGrid", () => {
  // The regression this whole module exists for: the probe table is sorted by
  // id, so an unordered grid opens with "agy". The operator's profile order is
  // the only order they can change, so it is the one that must win.
  it("orders agent cards by the operator's profile, not the probe's alphabet", () => {
    const { agents } = buildAgentGrid({ readiness: READY_LOCAL, shortcuts: DEFAULT_SHORTCUTS });
    expect(agents.map((card) => card.agentID)).toEqual(["claude", "codex", "opencode", "grok", "agy"]);
  });

  // Names come from the catalogue, which carries the real ones. Rendering the
  // slug is what produced a launcher offering "codex" and "agy".
  it("labels cards with the catalogue's names", () => {
    const { agents } = buildAgentGrid({ readiness: READY_LOCAL, shortcuts: DEFAULT_SHORTCUTS });
    expect(agents.map((card) => card.label)).toEqual(["Claude Code", "Codex", "OpenCode", "Grok", "Antigravity"]);
  });

  // The editor's command has to be the command that runs, or the editor's
  // capture warning is describing something the launcher will not do.
  it("launches the operator's command rather than a built-in verb", () => {
    const shortcuts: ShortcutEntry[] = [
      { label: "Codex", command: "vrooli agent launch --runner codex --arg=--yolo", agentId: "codex" },
    ];
    const { agents } = buildAgentGrid({ readiness: READY_LOCAL, shortcuts });
    const codex = agents.find((card) => card.agentID === "codex");
    expect(codex?.command).toBe("vrooli agent launch --runner codex --arg=--yolo");
  });

  it("falls back to the built-in verb for an agent the profile never mentions", () => {
    const { agents } = buildAgentGrid({ readiness: READY_LOCAL, shortcuts: [] });
    expect(agents.find((card) => card.agentID === "grok")?.command).toBe("grok");
  });

  // Installing a sixth agent must never produce an invisible card.
  it("appends a catalogued agent the profile has never seen", () => {
    const shortcuts: ShortcutEntry[] = [{ label: "Codex", command: "codex --yolo", agentId: "codex" }];
    const { agents } = buildAgentGrid({ readiness: READY_LOCAL, shortcuts });
    expect(agents[0]?.agentID).toBe("codex");
    expect(agents.map((card) => card.agentID)).toEqual(expect.arrayContaining(["claude", "grok", "opencode", "agy"]));
    expect(agents).toHaveLength(5);
  });

  it("keeps an operator's own command out of the agent grid but on the screen", () => {
    const { agents, commands } = buildAgentGrid({ readiness: READY_LOCAL, shortcuts: DEFAULT_SHORTCUTS });
    expect(agents.some((card) => card.label === "Codex sign-in")).toBe(false);
    expect(commands.map((card) => card.label)).toContain("Codex sign-in");
  });

  it("maps every readiness state onto its own card state", () => {
    const readiness = [
      fact("claude", { state: "ready", version: "2.1.4" }),
      fact("codex", { state: "missing", passed: false, detail: "codex is not installed" }),
      fact("agy", { state: "not_applicable", passed: false, detail: "No darwin/arm64 build published" }),
      fact("grok", { state: "unknown", passed: false }),
    ];
    const { agents } = buildAgentGrid({ readiness, shortcuts: [] });
    const state = (id: string) => agents.find((card) => card.agentID === id)?.state;
    expect(state("claude")).toBe("ready");
    expect(state("codex")).toBe("missing");
    expect(state("agy")).toBe("not-applicable");
    expect(state("grok")).toBe("unknown");
  });

  it("reports an in-flight install ahead of the machine's last-known state", () => {
    const readiness = [fact("codex", { state: "missing", passed: false })];
    const { agents } = buildAgentGrid({ readiness, shortcuts: [], installing: ["codex"] });
    expect(agents[0]?.state).toBe("installing");
  });

  // A node whose Bridge agent predates capability reporting must not lose its
  // agent grid, its order, or its install affordance because a probe was
  // silent. Unknown is the honest state; demoting to plain commands is not.
  it("still renders agent cards when the machine reports no capabilities", () => {
    const { agents, commands } = buildAgentGrid({ readiness: [], shortcuts: DEFAULT_SHORTCUTS });
    expect(agents.map((card) => card.agentID)).toEqual(["claude", "codex", "opencode", "grok"]);
    expect(agents.every((card) => card.state === "unknown")).toBe(true);
    expect(commands.map((card) => card.label)).toEqual(["Codex sign-in"]);
  });

  it("uses the shortcut's label when the catalogue supplies none", () => {
    const readiness = [fact("codex", { label: "" })];
    const shortcuts: ShortcutEntry[] = [{ label: "My Codex", command: "codex", agentId: "codex" }];
    expect(buildAgentGrid({ readiness, shortcuts }).agents[0]?.label).toBe("My Codex");
  });

  it("never renders a slug when neither the catalogue nor the profile names the agent", () => {
    const { agents } = buildAgentGrid({ readiness: [fact("grok", { label: "" })], shortcuts: [] });
    expect(agents[0]?.label).toBe("Grok");
  });
});

describe("card affordances", () => {
  it("lets a ready or unknown card launch and nothing else", () => {
    const base = { agentID: "codex", key: "codex", label: "Codex", command: "codex", fromCatalogue: true };
    expect(cardLaunches({ ...base, state: "ready" })).toBe(true);
    expect(cardLaunches({ ...base, state: "unknown" })).toBe(true);
    expect(cardLaunches({ ...base, state: "missing" })).toBe(false);
    expect(cardLaunches({ ...base, state: "installing" })).toBe(false);
    expect(cardLaunches({ ...base, state: "not-applicable" })).toBe(false);
  });

  // Only "missing" offers an install. Offering one for not-applicable would
  // put a button on a card whose installer has already refused.
  it("offers an install only for a missing agent", () => {
    const base = { agentID: "codex", key: "codex", label: "Codex", command: "codex", fromCatalogue: true };
    expect(cardInstalls({ ...base, state: "missing" })).toBe(true);
    expect(cardInstalls({ ...base, state: "not-applicable" })).toBe(false);
    expect(cardInstalls({ ...base, state: "ready" })).toBe(false);
  });
});

describe("moveItem", () => {
  it("moves an item forward and backward", () => {
    expect(moveItem(["a", "b", "c", "d"], 0, 2)).toEqual(["b", "c", "a", "d"]);
    expect(moveItem(["a", "b", "c", "d"], 3, 1)).toEqual(["a", "d", "b", "c"]);
  });

  it("is a no-op for an out-of-range source or an unchanged position", () => {
    expect(moveItem(["a", "b"], 5, 0)).toEqual(["a", "b"]);
    expect(moveItem(["a", "b"], -1, 0)).toEqual(["a", "b"]);
    expect(moveItem(["a", "b"], 1, 1)).toEqual(["a", "b"]);
  });

  it("clamps a target beyond the end rather than dropping the item", () => {
    expect(moveItem(["a", "b", "c"], 0, 99)).toEqual(["b", "c", "a"]);
  });

  it("does not mutate its input", () => {
    const input = ["a", "b", "c"];
    moveItem(input, 0, 2);
    expect(input).toEqual(["a", "b", "c"]);
  });
});

describe("applyAgentOrderToShortcuts", () => {
  const cards = (ids: string[]) =>
    ids.map((id) => ({
      agentID: id,
      key: id,
      label: id.toUpperCase(),
      command: `${id} --go`,
      state: "ready" as const,
      fromCatalogue: true,
    }));

  it("rewrites the profile in the new agent order", () => {
    const shortcuts: ShortcutEntry[] = [
      { label: "Claude Code", command: "claude", agentId: "claude" },
      { label: "Codex", command: "codex --yolo", agentId: "codex" },
    ];
    const next = applyAgentOrderToShortcuts(shortcuts, cards(["codex", "claude"]));
    expect(next.map((entry) => entry.agentId)).toEqual(["codex", "claude"]);
    // The operator's own command text survives the move untouched.
    expect(next[0]?.command).toBe("codex --yolo");
  });

  // Without this, reordering an agent the profile has never seen would lose
  // its position the moment the list was read back.
  it("materializes an entry for an agent the profile had no row for", () => {
    const shortcuts: ShortcutEntry[] = [{ label: "Codex", command: "codex --yolo", agentId: "codex" }];
    const next = applyAgentOrderToShortcuts(shortcuts, cards(["grok", "codex"]));
    expect(next.map((entry) => entry.agentId)).toEqual(["grok", "codex"]);
    expect(next[0]).toMatchObject({ command: "grok --go", agentId: "grok" });
  });

  it("keeps operator commands, in their own order, after the agents", () => {
    const shortcuts: ShortcutEntry[] = [
      { label: "Deploy", command: "make deploy" },
      { label: "Codex", command: "codex --yolo", agentId: "codex" },
      { label: "Logs", command: "make logs" },
    ];
    const next = applyAgentOrderToShortcuts(shortcuts, cards(["codex"]));
    expect(next.map((entry) => entry.label)).toEqual(["Codex", "Deploy", "Logs"]);
  });

  it("drops nothing: every input entry survives the projection", () => {
    const shortcuts: ShortcutEntry[] = [
      { label: "Claude Code", command: "claude", agentId: "claude" },
      { label: "Deploy", command: "make deploy" },
      { label: "Codex", command: "codex --yolo", agentId: "codex" },
    ];
    const next = applyAgentOrderToShortcuts(shortcuts, cards(["codex", "claude"]));
    expect(next).toHaveLength(3);
    for (const entry of shortcuts) {
      expect(next.some((item) => item.command === entry.command)).toBe(true);
    }
  });
});
