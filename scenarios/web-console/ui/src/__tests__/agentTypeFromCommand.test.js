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
