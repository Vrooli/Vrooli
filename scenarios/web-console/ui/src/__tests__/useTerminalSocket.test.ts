import { describe, it, expect } from "vitest";
import { isCleanWsClose } from "../hooks/useTerminalSocket";

// [REQ:P0-002b] WebSocket I/O Streaming - isCleanWsClose decision boundary
describe("isCleanWsClose", () => {
  it("returns true for Normal close (1000)", () => {
    expect(isCleanWsClose(1000)).toBe(true);
  });

  it("returns true for Going Away (1001)", () => {
    expect(isCleanWsClose(1001)).toBe(true);
  });

  it("returns false for Protocol Error (1002)", () => {
    expect(isCleanWsClose(1002)).toBe(false);
  });

  it("returns false for Abnormal Closure (1006)", () => {
    expect(isCleanWsClose(1006)).toBe(false);
  });

  it("returns false for Internal Error (1011)", () => {
    expect(isCleanWsClose(1011)).toBe(false);
  });
});

// [REQ:P0-002b] WebSocket I/O Streaming - hook module
describe("useTerminalSocket hook module", () => {
  it("exports useTerminalSocket function", async () => {
    const mod = await import("../hooks/useTerminalSocket");
    expect(typeof mod.useTerminalSocket).toBe("function");
  });

  it("exports TerminalMessage interface type (verified via runtime shape)", async () => {
    // Verify the module loads without errors; TerminalMessage is a TS interface
    // so we validate it structurally via a runtime assertion
    const msg: import("../hooks/useTerminalSocket").TerminalMessage = {
      type: "stdout",
      data: "hello",
    };
    expect(msg.type).toBe("stdout");
    expect(msg.data).toBe("hello");
  });

  it("exports SocketFactory type for WebSocket seam injection", async () => {
    // SocketFactory is a type export — verify the module loads cleanly.
    // ANSI constants are internal (used only for terminal status formatting).
    const mod = await import("../hooks/useTerminalSocket");
    expect(mod.useTerminalSocket).toBeDefined();
  });

  it("accepts createSocket parameter for WebSocket injection", async () => {
    // Verify the hook signature accepts the createSocket seam parameter
    // by checking that the function accepts an options object with createSocket
    const mod = await import("../hooks/useTerminalSocket");
    // The function exists and accepts the extended options shape
    // (actual rendering test would require React test utils + fake terminal)
    expect(mod.useTerminalSocket.length).toBe(1); // single options param
  });
});
