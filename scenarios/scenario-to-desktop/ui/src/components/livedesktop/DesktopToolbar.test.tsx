import { createRef } from "react";
import { act } from "@testing-library/react";
import { fireEvent, render, screen } from "@/test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useLiveDesktopStore } from "../../store/liveDesktopStore";
import { DesktopToolbar } from "./DesktopToolbar";

vi.mock("./DesktopControlsMenu", () => ({
  DesktopControlsMenu: () => <span>Desktop controls</span>,
}));
vi.mock("./MetricsBar", () => ({
  MetricsBar: () => <span>Live metrics</span>,
}));

describe("DesktopToolbar", () => {
  const stopSession = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    useLiveDesktopStore.setState({
      activeSession: null,
      connectionStatus: "disconnected",
      scenarioName: null,
      stopSession,
    });
    Object.defineProperty(document, "fullscreenElement", {
      configurable: true,
      value: null,
    });
  });

  it("shows identity and connected-only controls and metrics", () => {
    useLiveDesktopStore.setState({
      connectionStatus: "connected",
      scenarioName: "canvas-lab",
      activeSession: { width: 1440, height: 900 } as never,
    });
    render(
      <DesktopToolbar fullscreenTargetRef={createRef()} onClose={vi.fn()} />,
    );

    expect(screen.getByText("canvas-lab")).toBeInTheDocument();
    expect(screen.getByText("1440×900")).toBeInTheDocument();
    expect(screen.getByText("Desktop controls")).toBeInTheDocument();
    expect(screen.getByText("Live metrics")).toBeInTheDocument();
  });

  it("requests and exits fullscreen as the browser state changes", () => {
    const target = document.createElement("div");
    const requestFullscreen = vi.fn().mockResolvedValue(undefined);
    Object.assign(target, { requestFullscreen });
    const ref = { current: target };
    render(<DesktopToolbar fullscreenTargetRef={ref} onClose={vi.fn()} />);

    fireEvent.click(screen.getByRole("button", { name: "Fullscreen" }));
    expect(requestFullscreen).toHaveBeenCalledOnce();

    const exitFullscreen = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(document, "fullscreenElement", {
      configurable: true,
      value: target,
    });
    Object.assign(document, { exitFullscreen });
    act(() => {
      document.dispatchEvent(new Event("fullscreenchange"));
    });
    fireEvent.click(screen.getByRole("button", { name: "Exit fullscreen" }));
    expect(exitFullscreen).toHaveBeenCalledOnce();
  });

  it("ignores fullscreen requests when the target is not mounted", () => {
    render(
      <DesktopToolbar fullscreenTargetRef={createRef()} onClose={vi.fn()} />,
    );

    expect(() =>
      fireEvent.click(screen.getByRole("button", { name: "Fullscreen" })),
    ).not.toThrow();
  });

  it("stops sessions, closes the drawer, and hides connected controls otherwise", () => {
    const onClose = vi.fn();
    render(
      <DesktopToolbar fullscreenTargetRef={createRef()} onClose={onClose} />,
    );

    expect(screen.queryByText("Desktop controls")).not.toBeInTheDocument();
    expect(screen.queryByText("Live metrics")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Stop session" }));
    fireEvent.click(screen.getByRole("button", { name: "Close" }));
    expect(stopSession).toHaveBeenCalledOnce();
    expect(onClose).toHaveBeenCalledOnce();
  });
});
