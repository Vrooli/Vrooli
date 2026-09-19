import { describe, expect, it, vi } from "vitest";
import { create } from "@bufbuild/protobuf";
import {
  EvidenceTargetSchema,
  EvidenceTarget_Kind,
} from "@vrooli/proto-types/scenario-to-desktop/v1/domain/evidence_pb";
import { Platform } from "@vrooli/proto-types/scenario-to-desktop/v1/shared/common_pb";

const client = vi.hoisted(() => ({
  startDesktopSession: vi.fn(),
  stopDesktopSession: vi.fn(),
  heartbeatDesktopSession: vi.fn(),
  getDesktopSession: vi.fn(),
  listDesktopSessions: vi.fn(),
  launchDesktopArtifact: vi.fn(),
  findDesktopArtifact: vi.fn(),
  controlDesktop: vi.fn(),
}));
vi.mock("./connect", () => ({ evidenceConnectClient: client }));
vi.mock("./client", () => ({
  buildUrl: (path: string) => `https://api.test${path}`,
}));

import {
  buildVncWsUrl,
  executeDesktopControl,
  findArtifact,
  heartbeatSession,
  launchAppOnDesktop,
  listDesktopSessions,
  controlResultString,
  startDesktopSession,
  stopDesktopSession,
} from "./livedesktop";

describe("live desktop Connect client", () => {
  it("uses generated requests for local or bridge-targeted desktop sessions", async () => {
    const session = { sessionId: "session-1", scenarioName: "calculator" };
    client.startDesktopSession.mockResolvedValue(session);
    client.getDesktopSession.mockResolvedValue(session);
    client.findDesktopArtifact.mockResolvedValue({
      artifactPath: "/tmp/calculator.AppImage",
    });
    client.controlDesktop.mockResolvedValue({
      result: { fields: { title: "Calculator" } },
    });

    const bridgeTarget = create(EvidenceTargetSchema, {
      kind: EvidenceTarget_Kind.BRIDGE_NODE,
      bridgeNodeId: "bridge-host",
    });
    await expect(
      startDesktopSession({
        scenarioName: "calculator",
        platform: Platform.LINUX,
        target: bridgeTarget,
      }),
    ).resolves.toEqual(session);
    await expect(findArtifact("session-1")).resolves.toBe(
      "/tmp/calculator.AppImage",
    );
    await launchAppOnDesktop("session-1", "/tmp/app");
    await stopDesktopSession("session-1");
    await expect(
      executeDesktopControl("session-1", { action: "click", params: { x: 2 } }),
    ).resolves.toMatchObject({ result: { fields: { title: "Calculator" } } });

    expect(client.startDesktopSession).toHaveBeenCalledWith({
      scenarioName: "calculator",
      platform: Platform.LINUX,
      target: bridgeTarget,
    });
    expect(client.findDesktopArtifact).toHaveBeenCalledWith({
      scenarioName: "calculator",
    });
    expect(client.launchDesktopArtifact).toHaveBeenCalledWith({
      sessionId: "session-1",
      artifactPath: "/tmp/app",
    });
    expect(client.stopDesktopSession).toHaveBeenCalledWith({
      sessionId: "session-1",
    });
    expect(client.controlDesktop).toHaveBeenCalledWith({
      sessionId: "session-1",
      action: "click",
      params: { x: 2 },
    });
  });

  it("builds an encoded VNC websocket URL for the browser client", () => {
    expect(buildVncWsUrl("session/a b")).toBe(
      "wss://api.test/livedesktop/sessions/session%2Fa%20b/ws",
    );
  });

  it("keeps session maintenance and result decoding on the generated client", async () => {
    client.heartbeatDesktopSession.mockResolvedValue({});
    client.listDesktopSessions.mockResolvedValue({
      sessions: [{ sessionId: "session-2" }],
    });

    await heartbeatSession("session-2");
    await expect(listDesktopSessions()).resolves.toEqual([
      { sessionId: "session-2" },
    ]);
    expect(client.heartbeatDesktopSession).toHaveBeenCalledWith({
      sessionId: "session-2",
    });
    expect(client.listDesktopSessions).toHaveBeenCalledWith({});

    expect(
      controlResultString(
        { result: { fields: { summary: "desktop ready" } } } as never,
        "summary",
      ),
    ).toBe("desktop ready");
    expect(
      controlResultString(
        { result: { fields: { count: 1 } } } as never,
        "count",
      ),
    ).toBeUndefined();
    expect(controlResultString({} as never, "summary")).toBeUndefined();
  });
});
