import { renderWithProviders as render } from "../../test-utils";
import { screen } from "@testing-library/react";
import { forwardRef } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const view = vi.hoisted(() => ({ mode: "terminal" as "terminal" | "messages" }));
vi.mock("../../stores/useConversationStore", () => ({
  useConversationStore: (selector: (state: unknown) => unknown) => selector({
    viewModes: { s1: view.mode }, sessions: { s1: { events: [], cursor: { lastSeenSequence: 0 } } },
  }),
}));
vi.mock("../TerminalPane", () => ({
  default: forwardRef<HTMLDivElement, { sessionId: string; viewMode: string }>(
    ({ sessionId, viewMode }, ref) => (
      <div ref={ref} data-testid="mock-terminal" data-session-id={sessionId} data-view-mode={viewMode}>
        <div data-testid="pane-status-toast">status</div>
        {viewMode === "terminal" && <div data-testid="device-caption-full">following</div>}
      </div>
    ),
  ),
}));
vi.mock("../TerminalHeader", () => ({ default: () => <div data-testid="mock-header" /> }));
vi.mock("../MessagesPane", () => ({ default: () => <div data-testid="mock-messages" /> }));
vi.mock("../ErrorBoundary", () => ({ default: ({ children }: { children: React.ReactNode }) => <>{children}</> }));

import WorkspacePaneShell from "../WorkspacePaneShell";

const paneMeta = {
  sessionId: "s1", name: "Terminal", headerColor: "#123456", themeId: "slate-ocean", fontSize: 14,
  groupId: null, supportsMessagesView: true, manuallyUnread: false,
};

function shell() {
  return <WorkspacePaneShell
    paneMeta={paneMeta} layoutMode="grid" isActive isVisible
    isTtsSpeaking={false} activeSpeakingEventId={null} loadingEventId={null}
    summarizeLevel="moderate" summarizingEventId={null} getSummarizeError={() => null}
    onClearSummarizeError={vi.fn()} onToggleSummarized={vi.fn()} onChangeLevel={vi.fn()}
    selectedVersionForEvent={vi.fn()} playbackState={{
      currentTime: 0, duration: null, isPaused: true, playbackRate: 1, volume: 1, isMuted: false,
      capabilities: { canPause: false, canSeek: false, canAdjustSpeed: false, canAdjustVolume: false },
    }}
    onSetPlaybackRate={vi.fn()} onSetVolume={vi.fn()} onSetMuted={vi.fn()}
    playbackFocusRequest={null} onActivate={vi.fn()} onRequestClose={vi.fn()}
    onHandoff={vi.fn()} onToggleView={vi.fn()} onTerminalExit={vi.fn()}
    onTerminalRef={vi.fn()} onTtsSpeakingChange={vi.fn()} onSpeakingEventChange={vi.fn()}
    onConversationEventReceived={vi.fn()} onNeedsUnlock={vi.fn()} onPlayFromHere={vi.fn()}
    onPlayEvent={vi.fn()}
  />;
}

describe("WorkspacePaneShell stacking and view gates", () => {
  beforeEach(() => { view.mode = "terminal"; });

  it("does not paint terminal chrome over the messages view", () => {
    view.mode = "messages";
    render(shell());
    expect(screen.getByTestId("mock-terminal")).toHaveAttribute("data-view-mode", "messages");
    expect(screen.queryByTestId("device-caption-full")).toBeNull();
    expect(screen.getByTestId("mock-messages")).toBeInTheDocument();
  });

  it("keeps the terminal view mounted while the messages overlay is selected", () => {
    view.mode = "messages";
    const { getByTestId } = render(shell());
    expect(getByTestId("terminal-pane-container").className).toContain("overflow-hidden");
    expect(getByTestId("mock-terminal")).toHaveAttribute("data-view-mode", "messages");
  });

  it("restores the follower frame when the view returns to terminal", () => {
    view.mode = "messages";
    const rendered = render(shell());
    expect(screen.getByTestId("mock-messages")).toBeInTheDocument();

    view.mode = "terminal";
    rendered.rerender(shell());
    expect(screen.queryByTestId("mock-messages")).toBeNull();
    expect(screen.getByTestId("mock-terminal")).toHaveAttribute("data-view-mode", "terminal");
    expect(screen.getByTestId("device-caption-full")).toBeInTheDocument();
  });

  it("keeps the pane status toast inside the pane", () => {
    render(shell());
    const pane = screen.getByTestId("mock-terminal");
    expect(pane).toContainElement(screen.getByTestId("pane-status-toast"));
  });
});
