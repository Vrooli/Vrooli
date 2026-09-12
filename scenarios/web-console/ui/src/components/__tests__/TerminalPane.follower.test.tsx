import { renderWithProviders as render } from "../../test-utils";
import { screen } from "@testing-library/react";
import { useEffect } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const session = vi.hoisted(() => ({
  submitInput: vi.fn(() => ({ status: "sent", offset: 1 })),
  sendControl: vi.fn(() => true),
  setMouseMode: vi.fn(() => true),
  mouseMode: null,
  scrollBy: vi.fn(),
  serverSize: { cols: 80, rows: 24 },
  followerMode: "leader" as "leader" | "follower" | "self-echo",
  leaderDevice: "",
  leaderClass: "",
  leaderKbOpen: false,
  takeLease: vi.fn(),
  setKeyboardOpen: vi.fn(),
  subscribeInputSettled: vi.fn(() => () => {}),
  awaitOffset: vi.fn(() => () => {}),
  subscribePendingInput: vi.fn(() => () => {}),
  getPendingInputSnapshot: vi.fn(() => []),
  discardPendingInput: vi.fn(),
  discardAllPendingInput: vi.fn(),
  flushPendingInputNow: vi.fn(),
  sendConversationAck: vi.fn(),
}));

const followerFrame = {
  rect: { x: 0, y: 0, width: 300, height: 500, fontSize: 12, scale: 1 },
  screenRect: { x: 20, y: 20, width: 260, height: 460, fontSize: 12, scale: 1 },
  apertureRect: { x: 16, y: 16, width: 268, height: 468, radius: 12 },
  tier: "full" as const,
  archetype: "phone" as const,
  cols: 80,
  rows: 24,
  kbOpen: false,
  keyboardShare: 0,
  captionOffset: 8,
};

vi.mock("react-i18next", () => ({ useTranslation: () => ({ t: (key: string) => key }) }));
vi.mock("../../hooks/terminal/useTerminalSession", () => ({
  useTerminalSession: (options: { onFollowerModeChange: (mode: "leader" | "follower" | "self-echo") => void }) => {
    useEffect(() => { options.onFollowerModeChange(session.followerMode); }, [options]);
    return session;
  },
}));
vi.mock("../../hooks/terminal/useXtermLifecycle", () => ({
  useXtermLifecycle: () => ({
    containerRef: { current: null },
    terminalHostRef: { current: null },
    fitRef: { current: { fit: vi.fn() } },
    terminal: { cols: 80, rows: 24, options: {}, focus: vi.fn() },
    paneSize: { width: 800, height: 600 },
    scrollAwareFit: vi.fn(),
  }),
}));
vi.mock("../../hooks/terminal/useFollowerPresentation", () => ({
  useFollowerPresentation: ({ followerMode }: { followerMode: string }) =>
    followerMode === "follower" ? followerFrame : null,
}));
vi.mock("../../hooks/terminal/useFollowerViewportLayout", () => ({ useFollowerViewportLayout: vi.fn() }));
vi.mock("../../hooks/terminal/usePaneAttachments", () => ({
  usePaneAttachments: () => ({
    fileInputRef: { current: null }, dragOver: false, uploading: false, uploadError: null,
    handleCtxUploadImage: vi.fn(), handleFileInputChange: vi.fn(), handlePaste: vi.fn(),
    handleDragOver: vi.fn(), handleDragLeave: vi.fn(), handleDrop: vi.fn(),
  }),
}));
vi.mock("../../hooks/useWorkspaceSync", () => ({ useWorkspaceSync: () => ({ syncPaneUpdate: vi.fn() }) }));
vi.mock("../../hooks/terminal/usePaneSpeech", () => ({
  usePaneSpeech: () => ({ supported: false, playback: { speak: vi.fn() } }),
}));
vi.mock("../../hooks/terminal/usePaneSelection", () => ({
  usePaneSelection: () => ({
    copySelection: vi.fn(), pasteFromClipboard: vi.fn(), closeContextMenu: vi.fn(),
    contextMenu: null, hasSelection: false, inputError: null, handleCopy: vi.fn(),
    handlePaste: vi.fn(), selectAll: vi.fn(), clear: vi.fn(), handleCtxSpeak: vi.fn(),
  }),
}));
vi.mock("../../hooks/terminal/useTerminalBackgroundDetector", () => ({ useTerminalBackgroundDetector: vi.fn() }));
vi.mock("../../hooks/useKeyboardListeners", () => ({ useTerminalVoiceShortcut: vi.fn() }));
vi.mock("../../stores/useWorkspaceStore", () => {
  const state = {
    paneStatuses: {}, panes: [], wheelScrollSensitivity: 1, activePane: "s1", displayMode: "grid",
    adaptiveChrome: false, keyboardOpen: false, voiceShortcut: "", renamePaneById: vi.fn(),
    setPaneStatus: vi.fn(), setDeviceFontSize: vi.fn(), setPendingInputBuffer: vi.fn(),
    consumePendingInputBuffer: vi.fn(() => []),
  };
  const useWorkspaceStore = (selector: (value: typeof state) => unknown) => selector(state);
  useWorkspaceStore.getState = () => state;
  return { useWorkspaceStore, useEffectiveFontSize: () => 14 };
});
vi.mock("../PaneSelectionLayer", () => ({ PaneSelectionLayer: ({ children }: { children: React.ReactNode }) => <>{children}</> }));
vi.mock("../terminal/DeviceFrame", () => ({ DeviceFrame: () => <div data-testid="device-frame" /> }));

import TerminalPane from "../TerminalPane";

describe("TerminalPane follower presentation", () => {
  beforeEach(() => { session.followerMode = "leader"; });

  it("does not render a device frame for a self-echo session", () => {
    session.followerMode = "self-echo";
    render(<TerminalPane sessionId="s1" />);
    expect(screen.getByTestId("terminal-pane").className).toContain("isolate");
    expect(screen.queryByTestId("device-frame")).toBeNull();
  });

  it("renders the follower device frame only after the session becomes a follower", () => {
    session.followerMode = "follower";
    render(<TerminalPane sessionId="s1" />);
    expect(screen.getByTestId("device-frame")).toBeInTheDocument();
  });
});
