import { useState } from "react";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../test-utils";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import { strings } from "../consts/strings";
import VoiceMicButton from "./VoiceMicButton";
import VoiceModeMenu from "./VoiceModeMenu";

beforeEach(() => { useWorkspaceStore.setState({ agentVoiceConfigured: false, agentVoiceMode: "auto" }); });
function Harness({ start, prepare }: { start: ReturnType<typeof vi.fn>; prepare: ReturnType<typeof vi.fn> }) {
  const [anchor, setAnchor] = useState<HTMLElement | null>(null);
  const configured = useWorkspaceStore((state) => state.agentVoiceConfigured);
  return <>
    <VoiceMicButton supported isPreparing={false} isRecording={false} isTranscribing={false} error={null} testId="mic" onPrepare={prepare} onStart={() => { start(useWorkspaceStore.getState().agentVoiceMode); }} onStop={vi.fn()} onRequestVoiceMode={configured ? undefined : setAnchor} />
    <VoiceModeMenu anchor={anchor} nativeAvailable onClose={() => { setAnchor(null); }} onChoose={(mode) => { useWorkspaceStore.getState().setAgentVoiceMode(mode); setAnchor(null); start(mode); }} />
  </>;
}
describe("first microphone use", () => {
  it("has no persistent status row, and asks for a mode before capturing", async () => {
    const start = vi.fn(), prepare = vi.fn();
    renderWithProviders(<Harness start={start} prepare={prepare} />);
    expect(screen.queryByTestId("voice-input-mode-menu")).toBeNull();
    expect(screen.queryByText(strings.agentVoice.firstUseHint)).toBeNull();
    fireEvent.pointerDown(screen.getByTestId("mic")); fireEvent.click(screen.getByTestId("mic"));
    await waitFor(() => { expect(screen.getByTestId("voice-input-mode-menu")).toBeVisible(); });
    expect(screen.getByText(strings.agentVoice.firstUseHint)).toBeVisible();
    expect(start).not.toHaveBeenCalled(); expect(prepare).not.toHaveBeenCalled();
    fireEvent.click(screen.getByTestId("voice-input-mode-agent"));
    expect(start).toHaveBeenCalledWith("agent");
    await waitFor(() => { expect(screen.queryByTestId("voice-input-mode-menu")).toBeNull(); });
    start.mockClear();
    fireEvent.click(screen.getByTestId("mic"));
    expect(start).toHaveBeenCalledWith("agent");
    expect(screen.queryByTestId("voice-input-mode-menu")).toBeNull();
  });
  it("does not remember a choice when the menu is dismissed", async () => {
    renderWithProviders(<Harness start={vi.fn()} prepare={vi.fn()} />);
    fireEvent.click(screen.getByTestId("mic"));
    const menu = await screen.findByTestId("voice-input-mode-menu");
    fireEvent.keyDown(menu, { key: "Escape" });
    await waitFor(() => { expect(screen.queryByTestId("voice-input-mode-menu")).toBeNull(); });
    expect(useWorkspaceStore.getState().agentVoiceConfigured).toBe(false);
  });
  it("shows an unavailable agent option without hiding dictation", async () => {
    const anchor = document.createElement("button"); document.body.append(anchor);
    const choose = vi.fn();
    const view = renderWithProviders(<VoiceModeMenu anchor={anchor} nativeAvailable={false} onChoose={choose} onClose={vi.fn()} />);
    expect(await screen.findByTestId("voice-input-mode-agent")).toBeDisabled();
    fireEvent.click(screen.getByTestId("voice-input-mode-dictation"));
    expect(choose).toHaveBeenCalledWith("dictation");
    view.unmount(); anchor.remove();
  });
  it("shows checking instead of unavailable while observing native startup", async () => {
    const anchor = document.createElement("button"); document.body.append(anchor);
    const view = renderWithProviders(<VoiceModeMenu anchor={anchor} nativeAvailable={false} nativeChecking onChoose={vi.fn()} onClose={vi.fn()} />);
    const option = await screen.findByTestId("voice-input-mode-agent");
    expect(option).toBeDisabled();
    expect(option).toHaveAttribute("aria-busy", "true");
    expect(option).toHaveTextContent(strings.agentVoice.checkingMode);
    expect(option).not.toHaveTextContent(strings.agentVoice.unavailableMode);
    view.unmount(); anchor.remove();
  });
  it.each(["resize", "scroll"])("follows an anchor moved by visual viewport %s without a window resize", async (eventName) => {
    const viewport = new EventTarget();
    Object.assign(viewport, { offsetTop: 0, offsetLeft: 0, width: 390, height: 844 });
    const original = Object.getOwnPropertyDescriptor(window, "visualViewport");
    Object.defineProperty(window, "visualViewport", { configurable: true, value: viewport });
    const anchor = document.createElement("button"); document.body.append(anchor);
    let top = 400;
    vi.spyOn(anchor, "getBoundingClientRect").mockImplementation(() => DOMRect.fromRect({x:200,y:top,width:100,height:40}));
    const view = renderWithProviders(<VoiceModeMenu anchor={anchor} nativeAvailable onChoose={vi.fn()} onClose={vi.fn()} />);
    try {
      const menu = await screen.findByTestId("voice-input-mode-menu");
      // Geometry is synthetic; the event and rendered popover are real.
      vi.spyOn(menu, "getBoundingClientRect").mockImplementation(() => DOMRect.fromRect({x:Number.parseFloat(menu.style.getPropertyValue("--rcl-popover-left")) || 0,y:Number.parseFloat(menu.style.getPropertyValue("--rcl-popover-top")) || 0,width:280,height:200}));
      act(() => { viewport.dispatchEvent(new Event(eventName)); });
      await waitFor(() => { expect(menu.style.getPropertyValue("--rcl-popover-top")).toBe("192px"); });
      top = 600;
      act(() => { viewport.dispatchEvent(new Event(eventName)); });
      await waitFor(() => { expect(menu.style.getPropertyValue("--rcl-popover-top")).toBe("392px"); });
    } finally {
      view.unmount(); anchor.remove();
      if (original) Object.defineProperty(window, "visualViewport", original);
      else Reflect.deleteProperty(window, "visualViewport");
    }
  });
  it("treats a Settings preference change as the saved first-use choice", () => {
    renderWithProviders(<Harness start={vi.fn()} prepare={vi.fn()} />);
    act(() => { useWorkspaceStore.getState().setAgentVoiceMode("dictation"); });
    fireEvent.click(screen.getByTestId("mic"));
    expect(screen.queryByTestId("voice-input-mode-menu")).toBeNull();
    expect(useWorkspaceStore.getState().agentVoiceConfigured).toBe(true);
  });
});
