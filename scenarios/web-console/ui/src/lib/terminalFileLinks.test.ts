import { describe, expect, it, vi } from "vitest";
import { findTerminalFileLinks, terminalFileLinksForLine } from "./terminalFileLinks";

describe("terminal file links", () => {
  it("recognizes relative paths with optional line and column", () => {
    expect(findTerminalFileLinks("ERROR src/app.ts:42:7 failed", 3)).toMatchObject([
      { text: "src/app.ts:42:7", path: "src/app.ts:42:7", range: { start: { x: 7, y: 3 } } },
    ]);
  });

  it("accepts source extensions without maintaining an extension allowlist", () => {
    expect(findTerminalFileLinks("edit components/Button.tsx now", 1)[0]?.text).toBe("components/Button.tsx");
  });

  it("does not turn ordinary prose into links", () => {
    expect(findTerminalFileLinks("open the application and try again", 1)).toEqual([]);
  });

  it("activates through xterm link contracts", () => {
    const activate = vi.fn();
    const links = terminalFileLinksForLine("see ./README.md", 1, activate);
    links[0]?.activate(new MouseEvent("click"), "./README.md");
    expect(activate).toHaveBeenCalledWith("./README.md");
  });

  it("maps string offsets to terminal cells when a prefix contains wide characters", () => {
    const links = terminalFileLinksForLine("✓ src/App.tsx", 1, vi.fn(), (offset) => offset + 1);
    expect(links[0]?.range).toEqual({ start: { x: 4, y: 1 }, end: { x: 14, y: 1 } });
  });
});
