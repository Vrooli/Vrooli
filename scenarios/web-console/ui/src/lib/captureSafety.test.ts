import { describe, expect, it } from "vitest";

import { assessCapture, governedRewrite } from "./captureSafety";

describe("assessCapture", () => {
  it("calls a launcher-routed command governed and names the form", () => {
    expect(assessCapture("vrooli agent launch --runner codex --arg=--yolo", "codex")).toEqual({
      verdict: "governed",
      via: "vrooli agent launch",
    });
    expect(assessCapture("vrooli-agent-launcher --agent grok --", "grok")).toEqual({
      verdict: "governed",
      via: "vrooli-agent-launcher",
    });
  });

  // The exact shape that broke capture in production: a bare agent command
  // whose session home depends on the PATH shim resolving first.
  it("warns on a bare agent command whose capture depends on PATH", () => {
    expect(assessCapture("codex --yolo", "codex").verdict).toBe("path-dependent");
    expect(assessCapture("grok", "grok").verdict).toBe("path-dependent");
  });

  // Claude is identified out of band and OpenCode is read through its own
  // server, so neither depends on how it was started. Warning about them
  // would be a false alarm on the two agents that always work.
  it("does not warn about agents whose capture is start-independent", () => {
    expect(assessCapture("claude --dangerously-skip-permissions", "claude").verdict).toBe("independent");
    expect(assessCapture("opencode", "opencode").verdict).toBe("independent");
  });

  it("says nothing about a command that launches no agent", () => {
    expect(assessCapture("make deploy ENV=staging", undefined).verdict).toBe("not-an-agent");
    expect(assessCapture("codex login --device-auth", undefined).verdict).toBe("not-an-agent");
  });

  it("treats a blank command as empty rather than broken", () => {
    expect(assessCapture("", "codex").verdict).toBe("empty");
    expect(assessCapture("   ", "codex").verdict).toBe("empty");
  });
});

describe("governedRewrite", () => {
  it("rewrites a bare agent command into its governed form, keeping the arguments", () => {
    expect(governedRewrite("codex --yolo", "codex")).toBe("vrooli agent launch --runner codex --arg=--yolo");
    expect(governedRewrite("grok --resume abc", "grok")).toBe("vrooli agent launch --runner grok --arg=--resume --arg=abc");
  });

  // Offering a rewrite for something already governed, start-independent, or
  // not an agent at all would put a fix button on a command with no problem.
  it("offers nothing when there is nothing to fix", () => {
    expect(governedRewrite("vrooli agent launch --runner codex", "codex")).toBeNull();
    expect(governedRewrite("claude", "claude")).toBeNull();
    expect(governedRewrite("make deploy", undefined)).toBeNull();
    expect(governedRewrite("", "codex")).toBeNull();
  });
});
