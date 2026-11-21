import { renderWithProviders as render } from "../test-utils";
/**
 * A launched session must land on the tab it launched.
 *
 * The server emits `session.created` onto the SSE stream BEFORE it writes the
 * create response, so this client regularly merges its own brand-new session
 * through the external-session path first — the pane exists before the create
 * call has resolved and before the launch intent has been recorded. The intent
 * used to be read only at the moment a pane was added, so when the pane won the
 * race the intent was stranded: the session started, ran its command, and left
 * the operator looking at whichever tab they were on before. It is a race, so
 * it happened on some launches and not others.
 *
 * These tests drive both arrival orders through the real workspace store.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { screen, fireEvent, cleanup, act } from "@testing-library/react";
import { forwardRef } from "react";
import Workspace from "../components/Workspace";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import type { SessionInfo } from "../api/sessions";

const SESSION_ID = "sess-race-001";

const session = (id: string): SessionInfo => ({
  id,
  shell: "/bin/bash",
  created_at: "2026-01-15T14:30:00Z",
  cols: 80,
  rows: 24,
  backend: "standard",
  survives_restart: false,
  policy: { mode: "never" },
  origin: "ui",
  owner: "",
  display_label: "",
});

const { hookState, launchDeferred } = vi.hoisted(() => ({
  hookState: { panes: [] as Array<{ session: unknown; supportsMessagesView: boolean }> },
  launchDeferred: { resolve: undefined as ((value: unknown) => void) | undefined },
}));

vi.mock("../hooks/useSessionManager", () => ({
  useSessionManager: () => ({
    panes: hookState.panes,
    isHydrated: true,
    isCreating: false,
    createError: null,
    hydrationError: null,
    closeError: null,
    clearError: vi.fn(),
    clearHydrationError: vi.fn(),
    clearCloseError: vi.fn(),
    // Held open so the test can land the pane first, the way the SSE echo does.
    launchSession: vi.fn(() => new Promise((resolve) => { launchDeferred.resolve = resolve; })),
    removePane: vi.fn(),
    undoArchive: vi.fn(),
    deletePanePermanently: vi.fn(),
    mergeExternalSession: vi.fn(),
    endExternalSession: vi.fn(),
    handleExit: vi.fn(),
    submitToActiveTerminal: vi.fn(() => ({ status: "accepted" })),
    subscribeActiveInputSettled: vi.fn(() => () => {}),
    awaitActiveInputOffset: vi.fn(),
    subscribeActivePendingInput: vi.fn(() => () => {}),
    getActivePendingInputSnapshot: vi.fn(() => null),
    discardActivePendingInput: vi.fn(),
    discardAllActivePendingInput: vi.fn(),
    flushActivePendingInputNow: vi.fn(),
    copySelectionOnPane: vi.fn(),
    pasteFromClipboardOnPane: vi.fn(),
    scrollTerminalOnPane: vi.fn(),
    focusActiveTerminal: vi.fn(),
    registerTerminalRef: vi.fn(),
    stopActiveTts: vi.fn(),
    speakTextOnPane: vi.fn(),
    pauseTtsOnPane: vi.fn(),
    resumeTtsOnPane: vi.fn(),
    seekTtsOnPane: vi.fn(),
    setTtsPlaybackRateOnPane: vi.fn(),
    setTtsVolumeOnPane: vi.fn(),
    setTtsMutedOnPane: vi.fn(),
    getTtsStateOnPane: vi.fn(() => null),
  }),
}));

vi.mock("../hooks/useWorkspaceSync", () => ({
  useWorkspaceSync: () => ({
    syncActivePane: vi.fn(),
    syncPaneOrder: vi.fn(),
    syncPaneUpdate: vi.fn(),
    syncCreateGroup: vi.fn(),
    syncUpdateGroup: vi.fn(),
    syncDeleteGroup: vi.fn(),
  }),
}));

vi.mock("../hooks/useTouchControls", () => ({ useTouchControls: () => false }));

vi.mock("../audio-integration", () => ({
  useScenarioVoiceInput: () => ({
    supported: false, backend: "none", voiceState: "idle", error: null, audioLevel: 0,
    voiceActivity: undefined, fallbackNotice: null, partialTranscript: "", voiceMode: "one-shot",
    segments: [], commandSuggestion: null, wakeWordConfigured: false, passiveListeningActive: false,
    staleLiveMicLease: false, rejectedAudio: null, speakerVerificationEnabled: false,
    speakerProfileConfigured: false, isRecording: false, isListening: false, isTranscribing: false,
    isPreparing: false, isPassive: false, isActive: false,
    prepareRecording: vi.fn(), startRecording: vi.fn(), stopRecording: vi.fn(),
    cancelTranscription: vi.fn(), dismissCommandSuggestion: vi.fn(), dismissFallbackNotice: vi.fn(),
    dismissRejection: vi.fn(), retryWithoutFilter: vi.fn(), enterPassiveMode: vi.fn(),
    exitPassiveMode: vi.fn(), releaseMicrophone: vi.fn(),
  }),
  getTTSSummarizeConfig: vi.fn().mockResolvedValue({ level: "moderate" }),
}));

vi.mock("../components/TerminalPane", () => ({
  default: forwardRef<HTMLDivElement, { sessionId: string }>(function MockTerminalPane({ sessionId }, ref) {
    return <div ref={ref} data-testid={`mock-terminal-${sessionId}`} />;
  }),
}));
vi.mock("../components/TerminalHeader", () => ({ default: vi.fn(() => null) }));
vi.mock("../components/SettingsModal", () => ({ default: vi.fn(() => null) }));
vi.mock("../components/WorkspaceMinimap", () => ({ default: vi.fn(() => null) }));
vi.mock("../components/MobileToolbar", () => ({
  default: forwardRef<HTMLDivElement>(function MockMobileToolbar(_props, ref) {
    return <div ref={ref} />;
  }),
}));
vi.mock("../components/FloatingToolbar", () => ({
  default: vi.fn(({ onNewTerminal }: { onNewTerminal: () => void }) => (
    <button data-testid="toolbar-new" onClick={onNewTerminal}>New</button>
  )),
}));

/** Push a session into the session manager's list, as an SSE merge would. */
function landPane(id: string) {
  hookState.panes = [...hookState.panes, { session: session(id), supportsMessagesView: false }];
}

/**
 * The workspace starts on an existing, focused session.
 *
 * That matters: with a single pane the "no active pane" fallback would focus it
 * whatever the intent said, and the test would pass without the fix. Launching
 * a SECOND session is the only shape where activation has to come from the
 * intent, which is also the shape the operator hits every time.
 */
const EXISTING_ID = "sess-existing";

describe("launch intents survive the create/SSE race", () => {
  beforeEach(() => {
    hookState.panes = [{ session: session(EXISTING_ID), supportsMessagesView: false }];
    launchDeferred.resolve = undefined;
    useWorkspaceStore.setState({
      panes: [{
        sessionId: EXISTING_ID,
        name: EXISTING_ID,
        headerColor: "transparent",
        themeId: "default",
        fontSize: 14,
        groupId: null,
        supportsMessagesView: false,
        manuallyUnread: false,
      }],
      activePane: EXISTING_ID,
      groups: [],
      roles: [],
    });
  });
  afterEach(() => { cleanup(); });

  it("activates the launched pane when the pane arrives before the create resolves", async () => {
    const { rerender } = render(<Workspace />);
    expect(useWorkspaceStore.getState().activePane).toBe(EXISTING_ID);

    fireEvent.click(screen.getByTestId("toolbar-new"));

    // The SSE echo wins: the pane exists while the create call is still open.
    await act(async () => {
      landPane(SESSION_ID);
      rerender(<Workspace />);
    });
    expect(useWorkspaceStore.getState().panes.map((p) => p.sessionId)).toEqual([EXISTING_ID, SESSION_ID]);

    // Now the create resolves and records the intent — with nothing left to add.
    await act(async () => {
      launchDeferred.resolve?.(session(SESSION_ID));
      await Promise.resolve();
    });

    expect(useWorkspaceStore.getState().activePane).toBe(SESSION_ID);
  });

  it("activates the launched pane when the create resolves before the pane arrives", async () => {
    const { rerender } = render(<Workspace />);

    fireEvent.click(screen.getByTestId("toolbar-new"));

    await act(async () => {
      launchDeferred.resolve?.(session(SESSION_ID));
      await Promise.resolve();
    });
    await act(async () => {
      landPane(SESSION_ID);
      rerender(<Workspace />);
    });

    expect(useWorkspaceStore.getState().activePane).toBe(SESSION_ID);
  });
});
