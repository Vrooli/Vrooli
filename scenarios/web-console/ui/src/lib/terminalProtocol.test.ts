import { describe, expect, it } from "vitest";
import { initialTerminalProtocolState, reduceTerminalMessage } from "./terminalProtocol";

describe("terminal protocol reducer", () => {
  it("tracks cursor-bearing replay without DOM or transport effects", () => {
    const replaying = reduceTerminalMessage(initialTerminalProtocolState, { type: "stdout", data: "x", output_cursor: 4 });
    const live = reduceTerminalMessage(replaying, { type: "history_end", output_cursor: 4 });
    expect(live).toMatchObject({ inSnapshot: false, outputCursor: 4 });
  });

  it("makes a full resync explicit", () => {
    const live = reduceTerminalMessage(initialTerminalProtocolState, { type: "history_end" });
    expect(reduceTerminalMessage(live, { type: "resync" }).inSnapshot).toBe(true);
  });

  it("applies presence independently of terminal dimensions", () => {
    const next = reduceTerminalMessage(initialTerminalProtocolState, {
      type: "presence",
      holdsLease: false,
      leaderDevice: "phone",
      viewerCount: 2,
    });
    expect(next).toMatchObject({ holdsLease: false, leaderDevice: "phone", viewerCount: 2, serverSize: null });
  });
});
