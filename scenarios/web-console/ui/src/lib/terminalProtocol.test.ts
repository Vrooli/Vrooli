import { describe, expect, it } from "vitest";
import { initialTerminalProtocolState, initialTerminalProtocolStateFor, reduceTerminalMessage } from "./terminalProtocol";

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

  // size_info and presence carry the same leader-presentation fields. They are
  // applied by one function so the two cases cannot drift apart.
  it.each(["size_info", "presence"] as const)("applies leader presentation from %s", (type) => {
    const next = reduceTerminalMessage(initialTerminalProtocolState, {
      type,
      cols: 46,
      rows: 13,
      holdsLease: false,
      leaderDevice: "iPhone",
      deviceClass: "phone",
      kbOpen: true,
      viewerCount: 2,
    });
    expect(next).toMatchObject({
      holdsLease: false,
      leaderDevice: "iPhone",
      leaderClass: "phone",
      leaderKbOpen: true,
      viewerCount: 2,
    });
  });

  it("leaves leader presentation untouched when a message omits it", () => {
    const declared = reduceTerminalMessage(initialTerminalProtocolState, {
      type: "size_info", cols: 46, rows: 26, leaderDevice: "iPhone", deviceClass: "phone", kbOpen: true, viewerCount: 2,
    });
    // An older leader sends no deviceClass at all; that must not silently
    // clear what a previous message established.
    const next = reduceTerminalMessage(declared, { type: "presence", viewerCount: 2 });
    expect(next.leaderClass).toBe("phone");
    expect(next.leaderKbOpen).toBe(true);
  });

	it("clears the keyboard state when the leader closes it", () => {
    const open = reduceTerminalMessage(initialTerminalProtocolState, { type: "presence", kbOpen: true, viewerCount: 2 });
    expect(reduceTerminalMessage(open, { type: "presence", kbOpen: false, viewerCount: 2 }).leaderKbOpen).toBe(false);
	});

	it("reduces a self-leader presence frame to self-echo", () => {
		const state = initialTerminalProtocolStateFor("device-1");
		const next = reduceTerminalMessage(state, { type: "presence", leader: "device-1", holdsLease: false, viewerCount: 2 });
		expect(next.followerMode).toBe("self-echo");
	});

	it("reduces a foreign leader presence frame to follower", () => {
		const state = initialTerminalProtocolStateFor("device-1");
		const next = reduceTerminalMessage(state, { type: "presence", leader: "device-2", holdsLease: false, viewerCount: 2 });
		expect(next.followerMode).toBe("follower");
	});

	it("keeps a lease holder as leader regardless of reported leader identity", () => {
		const state = initialTerminalProtocolStateFor("device-1");
		const next = reduceTerminalMessage(state, { type: "presence", leader: "device-2", holdsLease: true, viewerCount: 2 });
		expect(next.followerMode).toBe("leader");
	});
});
