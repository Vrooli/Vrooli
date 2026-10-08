import { describe, it, expect } from "vitest";
import { agentTypeFromCommand } from "../hooks/useSessionManager";

describe("agentTypeFromCommand", () => {
  it("classifies each known agent runtime from its launch command", () => {
    expect(agentTypeFromCommand("claude --dangerously-skip-permissions")).toBe("claude");
    expect(agentTypeFromCommand("codex --yolo")).toBe("codex");
    expect(agentTypeFromCommand("opencode")).toBe("opencode");
    expect(agentTypeFromCommand("opencode --session ses_x")).toBe("opencode");
    expect(agentTypeFromCommand("grok")).toBe("grok");
    expect(agentTypeFromCommand("grok --resume abc")).toBe("grok");
  });

  it("classifies the governed project launch wrapper", () => {
    expect(agentTypeFromCommand("vrooli agent launch --runner claude --arg=--dangerously-skip-permissions")).toBe("claude");
    expect(agentTypeFromCommand("vrooli agent launch --runner=codex")).toBe("codex");
    expect(agentTypeFromCommand("vrooli agent launch")).toBe("claude");
  });

  it("classifies launcher-backed shortcuts", () => {
    expect(agentTypeFromCommand("if command -v vrooli-agent-launcher >/dev/null 2>&1; then exec vrooli-agent-launcher --agent codex -- --yolo; fi; exec codex --yolo")).toBe("codex");
    expect(agentTypeFromCommand("vrooli-agent-launcher --agent=opencode --")).toBe("opencode");
    expect(agentTypeFromCommand("vrooli-agent-launcher --agent=claude-code --")).toBe("claude");
    expect(agentTypeFromCommand("vrooli-agent-launcher --agent grok --")).toBe("grok");
  });

  it("classifies every governed runner spelling", () => {
    expect(agentTypeFromCommand("vrooli agent launch --runner=claude-code")).toBe("claude");
    expect(agentTypeFromCommand("vrooli agent launch --runner opencode")).toBe("opencode");
    expect(agentTypeFromCommand("vrooli agent launch --runner grok")).toBe("grok");
  });

  it("is case-insensitive and trims surrounding whitespace", () => {
    expect(agentTypeFromCommand("  OpenCode  ")).toBe("opencode");
    expect(agentTypeFromCommand("GROK --resume x")).toBe("grok");
  });

  it("returns none for unrelated or empty commands", () => {
    expect(agentTypeFromCommand("")).toBe("none");
    expect(agentTypeFromCommand(undefined)).toBe("none");
    expect(agentTypeFromCommand("vim")).toBe("none");
    // A substring match must not misfire (e.g. a script named opencoder).
    expect(agentTypeFromCommand("opencoder")).toBe("none");
  });
});
