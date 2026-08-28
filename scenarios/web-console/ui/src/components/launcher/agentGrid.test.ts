import { describe, expect, it } from "vitest";
import { DEFAULT_SHORTCUTS } from "../../consts/shortcuts";
import { cardSupportsAttribution, commandForCard, foldAttributedShortcuts } from "./agentGrid";

describe("foldAttributedShortcuts", () => {
  it("folds each agent's attributed duplicate into its parent card", () => {
    const labels = foldAttributedShortcuts(DEFAULT_SHORTCUTS).map((c) => c.label);
    // Four agents, each folded from a plain and an attributed entry.
    expect(labels).toEqual(expect.arrayContaining(["Claude Code", "Codex", "OpenCode", "Grok"]));
    // No "(attributed)" entry survives as a card of its own.
    expect(labels.filter((label) => label.includes("attributed"))).toHaveLength(0);
    // Eight of the entries collapse to four; anything else is an ordinary
    // shortcut and keeps its own card.
    expect(labels).toHaveLength(DEFAULT_SHORTCUTS.length - 4);
  });

  it("gives every folded agent both variants", () => {
    const agents = ["Claude Code", "Codex", "OpenCode", "Grok"];
    for (const card of foldAttributedShortcuts(DEFAULT_SHORTCUTS)) {
      if (!agents.includes(card.label)) continue;
      expect(cardSupportsAttribution(card)).toBe(true);
    }
  });

  // The sign-in command moved out of the launcher's hardcoded actions block
  // and into the ordinary shortcut list; it must still reach a card.
  it("keeps the Codex sign-in shortcut as its own card", () => {
    const cards = foldAttributedShortcuts(DEFAULT_SHORTCUTS);
    const signIn = cards.find((c) => c.command === "codex login --device-auth");
    expect(signIn).toBeDefined();
    expect(signIn && cardSupportsAttribution(signIn)).toBe(false);
  });

  it("keeps an operator's own shortcut visible even with no attributed sibling", () => {
    const cards = foldAttributedShortcuts([
      { label: "My tool", command: "mytool --go" },
      ...DEFAULT_SHORTCUTS,
    ]);
    const mine = cards.find((c) => c.label === "My tool");
    expect(mine).toBeDefined();
    expect(mine?.command).toBe("mytool --go");
    expect(mine && cardSupportsAttribution(mine)).toBe(false);
  });

  it("preserves the source list's order", () => {
    const cards = foldAttributedShortcuts([
      { label: "Zeta", command: "z" },
      { label: "Alpha", command: "a" },
    ]);
    expect(cards.map((c) => c.label)).toEqual(["Zeta", "Alpha"]);
  });

  it("keeps a card that only has an attributed variant", () => {
    const cards = foldAttributedShortcuts([{ label: "Solo (attributed)", command: "solo --attr" }]);
    expect(cards).toHaveLength(1);
    expect(cards[0]?.attributedCommand).toBe("solo --attr");
    expect(cards[0]?.command).toBeUndefined();
  });
});

describe("commandForCard", () => {
  const card = { label: "Codex", command: "codex --yolo", attributedCommand: "launcher codex" };

  it("returns the plain command when attribution is off", () => {
    expect(commandForCard(card, false)).toBe("codex --yolo");
  });

  it("returns the attributed command when attribution is on", () => {
    expect(commandForCard(card, true)).toBe("launcher codex");
  });

  // A toggle must never leave a card unable to launch.
  it("falls back when the requested variant is missing", () => {
    expect(commandForCard({ label: "A", command: "a" }, true)).toBe("a");
    expect(commandForCard({ label: "B", attributedCommand: "b" }, false)).toBe("b");
  });
});
