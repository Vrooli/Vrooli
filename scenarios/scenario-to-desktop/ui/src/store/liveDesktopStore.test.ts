import { beforeEach, describe, expect, it, vi } from "vitest";
import { act } from "@testing-library/react";
import { create } from "@bufbuild/protobuf";
import {
  DesktopSessionSchema,
  DesktopSessionState,
} from "@vrooli/proto-types/scenario-to-desktop/v1/domain/evidence_pb";
import { Platform } from "@vrooli/proto-types/scenario-to-desktop/v1/shared/common_pb";

const desktopApi = vi.hoisted(() => ({
  startDesktopSession: vi.fn(),
  stopDesktopSession: vi.fn(),
  heartbeatSession: vi.fn(),
  getDesktopSession: vi.fn(),
  executeDesktopControl: vi.fn(),
  controlResultString: vi.fn(),
}));

vi.mock("../lib/api/livedesktop", () => desktopApi);

import { useCapturesStore } from "./capturesStore";
import { useLiveDesktopStore } from "./liveDesktopStore";

const session = (sessionId: string, state = DesktopSessionState.RUNNING) =>
  create(DesktopSessionSchema, {
    sessionId,
    scenarioName: "calculator",
    state,
  });

function resetStore() {
  useLiveDesktopStore.setState({
    activeSession: null,
    connectionStatus: "disconnected",
    error: null,
    isOpen: false,
    scenarioName: null,
    appPath: null,
  });
}

beforeEach(() => {
  resetStore();
  vi.clearAllMocks();
  desktopApi.startDesktopSession.mockResolvedValue(session("desktop-1"));
  desktopApi.stopDesktopSession.mockResolvedValue(undefined);
  desktopApi.getDesktopSession.mockResolvedValue(session("desktop-1"));
  desktopApi.heartbeatSession.mockResolvedValue(undefined);
  vi.spyOn(useCapturesStore.getState(), "fetchSummary").mockResolvedValue();
});

describe("liveDesktopStore", () => {
  it("opens a local artifact and starts its live desktop session", async () => {
    act(() => { useLiveDesktopStore.getState().open("calculator", "/tmp/app"); });
    await act(async () =>
      useLiveDesktopStore.getState().startSession({
        scenarioName: "calculator",
        artifactPath: "/tmp/app",
        platform: Platform.LINUX,
      }),
    );

    expect(desktopApi.startDesktopSession).toHaveBeenCalledWith({
      scenarioName: "calculator",
      artifactPath: "/tmp/app",
      platform: Platform.LINUX,
    });
    expect(useLiveDesktopStore.getState()).toMatchObject({
      isOpen: true,
      scenarioName: "calculator",
      appPath: "/tmp/app",
      connectionStatus: "connecting",
    });
    expect(useLiveDesktopStore.getState().activeSession?.sessionId).toBe("desktop-1");
  });

  it("surfaces a failed session response without retaining a stale desktop", async () => {
    desktopApi.startDesktopSession.mockResolvedValue(
      session("desktop-1", DesktopSessionState.ERROR),
    );

    await act(async () =>
      useLiveDesktopStore.getState().startSession({
        scenarioName: "calculator",
        platform: Platform.LINUX,
      }),
    );

    expect(useLiveDesktopStore.getState()).toMatchObject({
      activeSession: null,
      connectionStatus: "error",
      error: "Session failed to start",
    });
  });

  it("stops an active session and refreshes the scenario capture summary on close", async () => {
    useLiveDesktopStore.setState({
      isOpen: true,
      scenarioName: "calculator",
      activeSession: session("desktop-1"),
      connectionStatus: "connected",
    });

    act(() => { useLiveDesktopStore.getState().close(); });
    await vi.waitFor(() => {
      expect(desktopApi.stopDesktopSession).toHaveBeenCalledWith("desktop-1");
    });

    expect(useCapturesStore.getState().fetchSummary).toHaveBeenCalledWith(
      "calculator",
    );
    expect(useLiveDesktopStore.getState()).toMatchObject({
      isOpen: false,
      activeSession: null,
      connectionStatus: "disconnected",
      scenarioName: null,
    });
  });

  it("resets after an explicit stop even when backend cleanup is best effort", async () => {
    desktopApi.stopDesktopSession.mockRejectedValue(new Error("network lost"));
    useLiveDesktopStore.setState({
      activeSession: session("desktop-1"),
      connectionStatus: "connected",
      error: "old error",
    });

    await act(async () => useLiveDesktopStore.getState().stopSession());

    expect(useLiveDesktopStore.getState()).toMatchObject({
      activeSession: null,
      connectionStatus: "disconnected",
      error: null,
    });
  });
});
