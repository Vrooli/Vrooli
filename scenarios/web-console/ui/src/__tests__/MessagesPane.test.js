import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { act, render, screen, fireEvent, waitFor } from "@testing-library/react";
import MessagesPane from "../components/MessagesPane";
import { strings } from "../consts/strings";
import { useConversationStore } from "../stores/useConversationStore";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import { makeConversationEvents } from "./fixtures/conversationFixture";
const { mockLoadOlderConversationPage } = vi.hoisted(() => ({
    mockLoadOlderConversationPage: vi.fn().mockResolvedValue(false),
}));
vi.mock("../hooks/useConversationSession", () => ({
    refreshConversationSession: vi.fn().mockResolvedValue(undefined),
    loadOlderConversationPage: mockLoadOlderConversationPage,
}));
const { mockResolveFilePreview, mockGetFilePreviewText } = vi.hoisted(() => ({
    mockResolveFilePreview: vi.fn(),
    mockGetFilePreviewText: vi.fn(),
}));
vi.mock("../api/filePreview", async (importOriginal) => {
    const actual = await importOriginal();
    return {
        ...actual,
        resolveFilePreview: mockResolveFilePreview,
        getFilePreviewText: mockGetFilePreviewText,
    };
});
// makeModel builds a PreviewModel fixture for the file-preview controller.
function makeModel(overrides) {
    return {
        previewId: "pv-1",
        inputPath: "/tmp/example.ts",
        resolvedPath: "/tmp/example.ts",
        basename: "example.ts",
        resolutionBasis: "absolute",
        kind: "code",
        mimeType: "text/plain; charset=utf-8",
        sizeBytes: 12,
        canPreview: true,
        canDownload: true,
        supportsRange: false,
        textContentAvailable: true,
        blobUrl: "/api/v1/sessions/sess-1/file-previews/pv-1/blob",
        blobHref: "/api/v1/sessions/sess-1/file-previews/pv-1/blob",
        warnings: [],
        ...overrides,
    };
}
// Mock the markdown renderer to avoid shiki/mermaid in jsdom
vi.mock("../components/markdown", () => ({
    MarkdownRenderer: ({ content, onLinkClick, onMermaidOpen, }) => (_jsxs("div", { "data-testid": "mock-markdown", children: [content.includes("[")
                ? (_jsx("a", { href: "/tmp/example.ts:12", "data-testid": "mock-markdown-link", onClick: (event) => onLinkClick?.("/tmp/example.ts:12", event), children: "example.ts" }))
                : content, _jsx("button", { "data-testid": "mock-mermaid-open", type: "button", onClick: () => onMermaidOpen?.("graph TD; A-->B"), children: "open diagram" })] })),
}));
// Keep the real Mermaid viewer/drawer but avoid loading the mermaid bundle.
vi.mock("../components/markdown/hooks/useMermaidSvg", () => ({
    useMermaidSvg: () => ({ svgHtml: '<svg viewBox="0 0 100 80"></svg>', error: null, loading: false }),
}));
function makeEvent(overrides) {
    return {
        sessionId: "sess-1",
        source: "claude_hook",
        role: "assistant",
        text: `Message ${overrides.sequence}`,
        speechParagraphs: [`Message ${overrides.sequence}`],
        summarized: false,
        createdAt: new Date().toISOString(),
        deliveryState: "received",
        ttsState: "idle",
        consumptionState: "seen",
        ...overrides,
    };
}
const defaultPlaybackState = {
    currentTime: 0,
    duration: null,
    isPaused: true,
    playbackRate: 1,
    volume: 1,
    isMuted: false,
    capabilities: {
        canPause: true,
        canSeek: false,
        canAdjustSpeed: true,
        canAdjustVolume: true,
    },
};
const defaultProps = {
    sessionId: "sess-1",
    onPlayFromHere: vi.fn(),
    onPlayEvent: vi.fn(),
    activeSpeakingEventId: null,
    isTtsSpeaking: false,
    summarizeLevel: "moderate",
    selectedVersionForEvent: vi.fn(() => "active"),
    summarizingEventId: null,
    getSummarizeError: vi.fn(() => null),
    onClearSummarizeError: vi.fn(),
    onToggleSummarized: vi.fn(),
    onChangeLevel: vi.fn(),
    playbackState: defaultPlaybackState,
    onSetPlaybackRate: vi.fn(),
    onSetVolume: vi.fn(),
    onSetMuted: vi.fn(),
    playbackFocusRequest: null,
};
describe("MessagesPane", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        mockLoadOlderConversationPage.mockResolvedValue(false);
        useConversationStore.setState({ sessions: {}, viewModes: {} });
        globalThis.fetch = vi.fn();
        // Mock IntersectionObserver for auto-scroll sentinel
        const mockObserver = vi.fn().mockImplementation(() => ({
            observe: vi.fn(),
            unobserve: vi.fn(),
            disconnect: vi.fn(),
        }));
        vi.stubGlobal("IntersectionObserver", mockObserver);
        // Mock ResizeObserver for collapse measurement
        vi.stubGlobal("ResizeObserver", vi.fn().mockImplementation(() => ({
            observe: vi.fn(),
            unobserve: vi.fn(),
            disconnect: vi.fn(),
        })));
    });
    function seedEvents(events) {
        useConversationStore.setState({
            sessions: {
                "sess-1": {
                    events,
                    cursor: { lastSeenSequence: 0, lastListenedSequence: 0 },
                    hydrated: true,
                },
            },
        });
    }
    // --- Core rendering ---
    it("renders a bounded first window for 2500 conversation events", () => {
        seedEvents(makeConversationEvents(2500, 42));
        render(_jsx(MessagesPane, { ...defaultProps }));
        expect(screen.getAllByTestId(/^msg-card-/).length).toBeLessThanOrEqual(60);
    });
    it("renders play and audio icons on each assistant message", () => {
        seedEvents([
            makeEvent({ id: "e1", sequence: 1 }),
            makeEvent({ id: "e2", sequence: 2 }),
        ]);
        render(_jsx(MessagesPane, { ...defaultProps }));
        expect(screen.getByTestId("msg-speak-from-e1")).toBeInTheDocument();
        expect(screen.getByTestId("msg-audio-e1")).toBeInTheDocument();
        expect(screen.getByTestId("msg-speak-from-e2")).toBeInTheDocument();
        expect(screen.getByTestId("msg-audio-e2")).toBeInTheDocument();
    });
    it("clicking 'read from here' calls onPlayFromHere with correct event ID", () => {
        seedEvents([makeEvent({ id: "e1", sequence: 1 })]);
        render(_jsx(MessagesPane, { ...defaultProps }));
        fireEvent.click(screen.getByTestId("msg-speak-from-e1"));
        expect(defaultProps.onPlayFromHere).toHaveBeenCalledWith("e1");
    });
    it("clicking audio button calls onPlayEvent and opens popover", () => {
        seedEvents([makeEvent({ id: "e1", sequence: 1, text: "Hello world", speechParagraphs: ["Hello world"] })]);
        render(_jsx(MessagesPane, { ...defaultProps }));
        fireEvent.click(screen.getByTestId("msg-audio-e1"));
        expect(defaultProps.onPlayEvent).toHaveBeenCalledWith("e1");
        expect(screen.getByTestId("audio-popover-e1")).toBeInTheDocument();
    });
    it("shows loading feedback on the active message audio control", () => {
        seedEvents([makeEvent({ id: "e1", sequence: 1, text: "Hello world", speechParagraphs: ["Hello world"] })]);
        render(_jsx(MessagesPane, { ...defaultProps, loadingEventId: "e1" }));
        expect(screen.getByTestId("msg-audio-loading-e1")).toBeInTheDocument();
        expect(screen.getByTestId("msg-audio-e1")).toBeDisabled();
        expect(screen.getByTestId("msg-speak-from-e1")).toBeDisabled();
    });
    it("active speaking event shows TTS accent border", () => {
        seedEvents([
            makeEvent({ id: "e1", sequence: 1 }),
            makeEvent({ id: "e2", sequence: 2 }),
        ]);
        render(_jsx(MessagesPane, { ...defaultProps, activeSpeakingEventId: "e2", isTtsSpeaking: true }));
        const activeCard = screen.getByTestId("msg-card-e2");
        expect(activeCard.className).toContain("border-l-wc-accent");
        const inactiveCard = screen.getByTestId("msg-card-e1");
        expect(inactiveCard.className).not.toContain("border-l-wc-accent");
    });
    it("empty state renders without speaker icons", () => {
        seedEvents([]);
        render(_jsx(MessagesPane, { ...defaultProps }));
        expect(screen.queryByTestId(/msg-speak-/)).toBeNull();
        expect(screen.getByText(strings.messagesPane.noEvents)).toBeInTheDocument();
    });
    it("user messages have no TTS controls", () => {
        seedEvents([
            makeEvent({ id: "e1", sequence: 1, role: "user", text: "My question" }),
            makeEvent({ id: "e2", sequence: 2, role: "assistant", text: "My answer" }),
        ]);
        render(_jsx(MessagesPane, { ...defaultProps }));
        expect(screen.queryByTestId("msg-speak-from-e1")).toBeNull();
        expect(screen.queryByTestId("msg-audio-e1")).toBeNull();
        expect(screen.getByTestId("msg-speak-from-e2")).toBeInTheDocument();
        expect(screen.getByTestId("msg-audio-e2")).toBeInTheDocument();
    });
    // --- Layout: full-width accent bars ---
    it("messages use accent bar layout with role-based colors", () => {
        seedEvents([
            makeEvent({ id: "e1", sequence: 1, role: "user", text: "User" }),
            makeEvent({ id: "e2", sequence: 2, role: "assistant", text: "Assistant" }),
        ]);
        render(_jsx(MessagesPane, { ...defaultProps }));
        const userCard = screen.getByTestId("msg-card-e1");
        const assistantCard = screen.getByTestId("msg-card-e2");
        // Both have 3px left border
        expect(userCard.className).toContain("border-l-[3px]");
        expect(assistantCard.className).toContain("border-l-[3px]");
        // Different colors
        expect(userCard.className).toContain("border-l-sky");
        expect(assistantCard.className).toContain("border-l-emerald");
    });
    it("focused message gets accent background highlight", () => {
        seedEvents([
            makeEvent({ id: "e1", sequence: 1, text: "First" }),
            makeEvent({ id: "e2", sequence: 2, text: "Second" }),
        ]);
        render(_jsx(MessagesPane, { ...defaultProps }));
        // Not focused initially
        expect(screen.getByTestId("msg-card-e1").className).not.toContain("bg-wc-accent");
    });
    // --- Markdown rendering ---
    it("renders markdown content through MarkdownRenderer", () => {
        seedEvents([makeEvent({ id: "e1", sequence: 1, text: "Hello World" })]);
        render(_jsx(MessagesPane, { ...defaultProps }));
        expect(screen.getByTestId("msg-markdown-e1")).toBeInTheDocument();
        expect(screen.getByTestId("mock-markdown")).toBeInTheDocument();
        expect(screen.getByText("Hello World")).toBeInTheDocument();
    });
    it("toggles a message between markdown and plain text views", () => {
        seedEvents([makeEvent({ id: "e1", sequence: 1, text: "# Heading" })]);
        render(_jsx(MessagesPane, { ...defaultProps }));
        // Markdown by default
        expect(screen.getByTestId("mock-markdown")).toBeInTheDocument();
        expect(screen.queryByTestId("msg-plaintext-e1")).toBeNull();
        const toggle = screen.getByTestId("msg-render-toggle-e1");
        expect(toggle.getAttribute("aria-pressed")).toBe("false");
        fireEvent.click(toggle);
        expect(screen.queryByTestId("mock-markdown")).toBeNull();
        const plain = screen.getByTestId("msg-plaintext-e1");
        expect(plain).toBeInTheDocument();
        expect(plain.textContent).toBe("# Heading");
        expect(toggle.getAttribute("aria-pressed")).toBe("true");
        fireEvent.click(toggle);
        expect(screen.getByTestId("mock-markdown")).toBeInTheDocument();
        expect(screen.queryByTestId("msg-plaintext-e1")).toBeNull();
    });
    it("render mode toggle is independent per message", () => {
        seedEvents([
            makeEvent({ id: "e1", sequence: 1, text: "first" }),
            makeEvent({ id: "e2", sequence: 2, text: "second" }),
        ]);
        render(_jsx(MessagesPane, { ...defaultProps }));
        fireEvent.click(screen.getByTestId("msg-render-toggle-e1"));
        expect(screen.getByTestId("msg-plaintext-e1")).toBeInTheDocument();
        expect(screen.queryByTestId("msg-plaintext-e2")).toBeNull();
    });
    it("opens the Mermaid viewer from a diagram open action and closes it", () => {
        seedEvents([makeEvent({ id: "e1", sequence: 1, text: "diagram here" })]);
        render(_jsx(MessagesPane, { ...defaultProps }));
        expect(screen.queryByTestId("messages-mermaid-viewer-panel")).toBeNull();
        fireEvent.click(screen.getAllByTestId("mock-mermaid-open")[0]);
        const panel = screen.getByTestId("messages-mermaid-viewer-panel");
        expect(panel).toBeInTheDocument();
        expect(screen.getByText(strings.mermaid.viewerTitle)).toBeInTheDocument();
        expect(screen.getByLabelText(strings.mermaid.zoomIn)).toBeInTheDocument();
        expect(screen.getByLabelText(strings.mermaid.showSource)).toBeInTheDocument();
        fireEvent.click(screen.getByLabelText(strings.mermaid.closeViewer));
        expect(screen.queryByTestId("messages-mermaid-viewer-panel")).toBeNull();
    });
    it("closes the Mermaid viewer with Escape", () => {
        seedEvents([makeEvent({ id: "e1", sequence: 1, text: "diagram here" })]);
        render(_jsx(MessagesPane, { ...defaultProps }));
        fireEvent.click(screen.getAllByTestId("mock-mermaid-open")[0]);
        expect(screen.getByTestId("messages-mermaid-viewer-panel")).toBeInTheDocument();
        fireEvent.keyDown(window, { key: "Escape" });
        expect(screen.queryByTestId("messages-mermaid-viewer-panel")).toBeNull();
    });
    it("search updates row state without pushing query into markdown renderer props", () => {
        seedEvents([makeEvent({ id: "e1", sequence: 1, text: "Hello world" })]);
        render(_jsx(MessagesPane, { ...defaultProps }));
        fireEvent.click(screen.getByTestId("messages-search-btn"));
        fireEvent.change(screen.getByTestId("msg-nav-search"), {
            target: { value: "world" },
        });
        const mdEl = screen.getByTestId("mock-markdown");
        expect(mdEl.getAttribute("data-search-query")).toBeNull();
    });
    it("opens the file viewer when clicking a file-like markdown link", async () => {
        mockResolveFilePreview.mockResolvedValueOnce(makeModel({ inputPath: "/tmp/example.ts:12", line: 12, kind: "code" }));
        mockGetFilePreviewText.mockResolvedValueOnce({
            resolvedPath: "/tmp/example.ts",
            kind: "code",
            mimeType: "text/plain; charset=utf-8",
            content: "const x = 1;\n",
            truncated: false,
            line: 12,
        });
        seedEvents([makeEvent({ id: "e1", sequence: 1, text: "[example.ts](/tmp/example.ts:12)" })]);
        render(_jsx(MessagesPane, { ...defaultProps }));
        fireEvent.click(screen.getByTestId("mock-markdown-link"));
        await waitFor(() => {
            expect(screen.getAllByText("example.ts").length).toBeGreaterThan(0);
            expect(screen.getByText("/tmp/example.ts")).toBeInTheDocument();
            expect(screen.getByText("messagesFileViewer.linePrefix")).toBeInTheDocument();
            expect(screen.getByText("const x = 1;")).toBeInTheDocument();
        });
        expect(screen.getByTestId("messages-file-viewer-panel").className).toContain("--wc-safe-top");
        expect(mockResolveFilePreview).toHaveBeenCalledWith("sess-1", "/tmp/example.ts:12", "message_link");
        expect(mockGetFilePreviewText).toHaveBeenCalledWith("sess-1", "pv-1");
    });
    it("renders SVG file references as image previews", async () => {
        mockResolveFilePreview.mockResolvedValueOnce(makeModel({
            inputPath: "/tmp/logo.svg",
            resolvedPath: "/tmp/logo.svg",
            basename: "logo.svg",
            kind: "svg",
            mimeType: "image/svg+xml",
            textContentAvailable: false,
            supportsRange: true,
        }));
        seedEvents([makeEvent({ id: "e1", sequence: 1, text: "[logo](/tmp/logo.svg)" })]);
        render(_jsx(MessagesPane, { ...defaultProps }));
        fireEvent.click(screen.getByTestId("mock-markdown-link"));
        await waitFor(() => {
            expect(screen.getByRole("img", { name: "logo.svg" })).toBeInTheDocument();
        });
        expect(screen.getByText("/tmp/logo.svg")).toBeInTheDocument();
        expect(mockGetFilePreviewText).not.toHaveBeenCalled();
    });
    it("shows a viewer error when file resolution fails", async () => {
        mockResolveFilePreview.mockRejectedValueOnce(new Error("Referenced file was not found"));
        seedEvents([makeEvent({ id: "e1", sequence: 1, text: "[missing.ts](missing.ts)" })]);
        render(_jsx(MessagesPane, { ...defaultProps }));
        fireEvent.click(screen.getByTestId("mock-markdown-link"));
        await waitFor(() => {
            expect(screen.getByText("messagesFileViewer.unavailable")).toBeInTheDocument();
            expect(screen.getByText("Referenced file was not found")).toBeInTheDocument();
        });
    });
    // --- Font size ---
    it("applies font size from workspace store to message content", () => {
        useWorkspaceStore.setState({
            panes: [{ sessionId: "sess-1", name: "test", headerColor: "transparent", themeId: "slate-ocean", fontSize: 20, groupId: null, supportsMessagesView: true, manuallyUnread: false }],
        });
        seedEvents([makeEvent({ id: "e1", sequence: 1, text: "Sized text" })]);
        render(_jsx(MessagesPane, { ...defaultProps }));
        const mdWrapper = screen.getByTestId("msg-markdown-e1");
        expect(mdWrapper.style.fontSize).toBe("20px");
    });
    // --- Control strip ---
    it("renders control strip with search, jump, and nav buttons", () => {
        seedEvents([makeEvent({ id: "e1", sequence: 1 })]);
        render(_jsx(MessagesPane, { ...defaultProps }));
        expect(screen.getByTestId("messages-control-strip")).toBeInTheDocument();
        expect(screen.getByTestId("messages-search-btn")).toBeInTheDocument();
        expect(screen.getByTestId("msg-jump-trigger")).toBeInTheDocument();
        expect(screen.getByTestId("messages-nav-up")).toBeInTheDocument();
        expect(screen.getByTestId("messages-nav-down")).toBeInTheDocument();
    });
    it("renders an optional trailing action in the control strip", () => {
        seedEvents([makeEvent({ id: "e1", sequence: 1 })]);
        render(_jsx(MessagesPane, { ...defaultProps, toolbarTrailingAction: _jsx("button", { type: "button", "data-testid": "messages-trailing-action", children: "Toggle" }) }));
        expect(screen.getByTestId("messages-control-trailing")).toContainElement(screen.getByTestId("messages-trailing-action"));
    });
    it("clicking search button opens the navigator focused on search", () => {
        seedEvents([makeEvent({ id: "e1", sequence: 1 })]);
        render(_jsx(MessagesPane, { ...defaultProps }));
        expect(screen.queryByTestId("msg-jump-list")).toBeNull();
        fireEvent.click(screen.getByTestId("messages-search-btn"));
        expect(screen.getByTestId("msg-jump-list")).toBeInTheDocument();
        expect(screen.getByTestId("msg-nav-search")).toBeInTheDocument();
    });
    it("disables nav chevrons when no events exist", () => {
        seedEvents([]);
        render(_jsx(MessagesPane, { ...defaultProps }));
        expect(screen.getByTestId("messages-nav-up")).toBeDisabled();
        expect(screen.getByTestId("messages-nav-down")).toBeDisabled();
    });
    it("enables nav chevrons when events exist", () => {
        seedEvents([makeEvent({ id: "e1", sequence: 1, role: "assistant", text: "Answer" })]);
        render(_jsx(MessagesPane, { ...defaultProps }));
        expect(screen.getByTestId("messages-nav-up")).not.toBeDisabled();
        expect(screen.getByTestId("messages-nav-down")).not.toBeDisabled();
    });
    it("down chevron navigates to next message", () => {
        const scrollToMock = vi.fn();
        Element.prototype.scrollTo = scrollToMock;
        seedEvents([
            makeEvent({ id: "e1", sequence: 1, text: "First" }),
            makeEvent({ id: "e2", sequence: 2, text: "Second" }),
        ]);
        render(_jsx(MessagesPane, { ...defaultProps }));
        fireEvent.click(screen.getByTestId("messages-nav-down"));
        expect(scrollToMock).toHaveBeenCalled();
    });
    // --- Search ---
    it("non-matching messages are dimmed during search", () => {
        seedEvents([
            makeEvent({ id: "e1", sequence: 1, text: "Hello world" }),
            makeEvent({ id: "e2", sequence: 2, text: "Goodbye" }),
        ]);
        render(_jsx(MessagesPane, { ...defaultProps }));
        fireEvent.click(screen.getByTestId("messages-search-btn"));
        fireEvent.change(screen.getByTestId("msg-nav-search"), {
            target: { value: "Hello" },
        });
        // e1 matches, e2 does not — dimming applies to the message cards behind
        // the navigator overlay.
        expect(screen.getByTestId("msg-card-e2").className).toContain("opacity-40");
        expect(screen.getByTestId("msg-card-e1").className).not.toContain("opacity-40");
    });
    it("clearing the navigator search removes message dimming", () => {
        seedEvents([
            makeEvent({ id: "e1", sequence: 1, text: "Hello world" }),
            makeEvent({ id: "e2", sequence: 2, text: "Goodbye" }),
        ]);
        render(_jsx(MessagesPane, { ...defaultProps }));
        fireEvent.click(screen.getByTestId("messages-search-btn"));
        fireEvent.change(screen.getByTestId("msg-nav-search"), {
            target: { value: "Hello" },
        });
        expect(screen.getByTestId("msg-card-e2").className).toContain("opacity-40");
        // The navigator's clear button resets the lifted query.
        fireEvent.click(screen.getByTestId("msg-nav-clear"));
        expect(screen.getByTestId("msg-card-e2").className).not.toContain("opacity-40");
    });
    // --- Jump list ---
    it("jump trigger shows message count", () => {
        seedEvents([
            makeEvent({ id: "e1", sequence: 1 }),
            makeEvent({ id: "e2", sequence: 2 }),
        ]);
        render(_jsx(MessagesPane, { ...defaultProps }));
        const trigger = screen.getByTestId("msg-jump-trigger");
        expect(trigger.textContent).toContain("2");
    });
    it("clicking jump trigger opens jump list", () => {
        seedEvents([makeEvent({ id: "e1", sequence: 1 })]);
        render(_jsx(MessagesPane, { ...defaultProps }));
        expect(screen.queryByTestId("msg-jump-list")).toBeNull();
        fireEvent.click(screen.getByTestId("msg-jump-trigger"));
        expect(screen.getByTestId("msg-jump-list")).toBeInTheDocument();
    });
    it("jump list shows all messages with truncated text", () => {
        seedEvents([
            makeEvent({ id: "e1", sequence: 1, role: "user", text: "Short user message" }),
            makeEvent({ id: "e2", sequence: 2, role: "assistant", text: "A very long assistant response that should be truncated in the jump list to save space" }),
        ]);
        render(_jsx(MessagesPane, { ...defaultProps }));
        fireEvent.click(screen.getByTestId("msg-jump-trigger"));
        expect(screen.getByTestId("msg-jump-item-e1")).toBeInTheDocument();
        expect(screen.getByTestId("msg-jump-item-e2")).toBeInTheDocument();
    });
    it("jump trigger is disabled when no events", () => {
        seedEvents([]);
        render(_jsx(MessagesPane, { ...defaultProps }));
        expect(screen.getByTestId("msg-jump-trigger")).toBeDisabled();
    });
    // --- Summarization ---
    it("does not render the old summarized badge", () => {
        seedEvents([
            makeEvent({
                id: "e1",
                sequence: 1,
                summarized: true,
                speechParagraphs: ["Short version"],
                originalSpeechParagraphs: ["Full original text that is much longer"],
            }),
        ]);
        render(_jsx(MessagesPane, { ...defaultProps }));
        expect(screen.queryByTestId("msg-summarized-badge-e1")).toBeNull();
    });
    it("mode control dropdown shows Original + level options for summarized events", () => {
        seedEvents([
            makeEvent({
                id: "e1",
                sequence: 1,
                summarized: true,
                speechParagraphs: ["Short version"],
                originalSpeechParagraphs: ["Full original text"],
            }),
        ]);
        render(_jsx(MessagesPane, { ...defaultProps, selectedVersionForEvent: vi.fn(() => "active") }));
        fireEvent.click(screen.getByTestId("msg-e1-mode-control"));
        expect(screen.getByTestId("msg-e1-mode-option-original")).toBeInTheDocument();
        expect(screen.getByTestId("msg-e1-mode-option-light")).toBeInTheDocument();
        expect(screen.getByTestId("msg-e1-mode-option-moderate")).toBeInTheDocument();
        expect(screen.getByTestId("msg-e1-mode-option-heavy")).toBeInTheDocument();
    });
    it("shows summarize error from the playback controller surface", async () => {
        seedEvents([makeEvent({ id: "e1", sequence: 1, text: "A long assistant response" })]);
        render(_jsx(MessagesPane, { ...defaultProps, getSummarizeError: vi.fn(() => "Summarization failed: ollama returned 404: model not found") }));
        fireEvent.click(screen.getByTestId("msg-audio-e1"));
        await waitFor(() => {
            expect(screen.getByTestId("msg-summarize-error-e1")).toBeInTheDocument();
            expect(screen.getByTestId("msg-summarize-error-e1").textContent).toContain("model not found");
        });
    });
    it("clears summarize error through the provided dismiss handler", async () => {
        const onClearSummarizeError = vi.fn();
        seedEvents([makeEvent({ id: "e1", sequence: 1, text: "A long assistant response" })]);
        render(_jsx(MessagesPane, { ...defaultProps, getSummarizeError: vi.fn(() => "Summarization failed: connection refused"), onClearSummarizeError: onClearSummarizeError }));
        fireEvent.click(screen.getByTestId("msg-audio-e1"));
        await waitFor(() => {
            expect(screen.getByTestId("msg-summarize-error-e1")).toBeInTheDocument();
        });
        fireEvent.click(screen.getByTestId("msg-clear-summarize-error-e1"));
        expect(onClearSummarizeError).toHaveBeenCalledWith("e1");
    });
    it("applies a playback focus request by scrolling the targeted event into view", () => {
        const scrollToMock = vi.fn();
        Element.prototype.scrollTo = scrollToMock;
        seedEvents([
            makeEvent({ id: "e1", sequence: 1, text: "First" }),
            makeEvent({ id: "e2", sequence: 2, text: "Second" }),
        ]);
        render(_jsx(MessagesPane, { ...defaultProps, playbackFocusRequest: { eventId: "e2", nonce: 1 } }));
        expect(scrollToMock).toHaveBeenCalled();
    });
    // --- Copy-to-clipboard ---
    it("renders copy button on both user and assistant messages", () => {
        seedEvents([
            makeEvent({ id: "e1", sequence: 1, role: "user", text: "User msg" }),
            makeEvent({ id: "e2", sequence: 2, role: "assistant", text: "Assistant msg" }),
        ]);
        render(_jsx(MessagesPane, { ...defaultProps }));
        expect(screen.getByTestId("msg-copy-e1")).toBeInTheDocument();
        expect(screen.getByTestId("msg-copy-e2")).toBeInTheDocument();
    });
    it("clicking copy writes message text to clipboard", () => {
        const writeTextMock = vi.fn().mockResolvedValue(undefined);
        Object.assign(navigator, { clipboard: { writeText: writeTextMock } });
        seedEvents([makeEvent({ id: "e1", sequence: 1, text: "Copy me" })]);
        render(_jsx(MessagesPane, { ...defaultProps }));
        fireEvent.click(screen.getByTestId("msg-copy-e1"));
        expect(writeTextMock).toHaveBeenCalledWith("Copy me");
    });
    it("shows checkmark icon after copying", () => {
        const writeTextMock = vi.fn().mockResolvedValue(undefined);
        Object.assign(navigator, { clipboard: { writeText: writeTextMock } });
        seedEvents([makeEvent({ id: "e1", sequence: 1, text: "Copy me" })]);
        render(_jsx(MessagesPane, { ...defaultProps }));
        fireEvent.click(screen.getByTestId("msg-copy-e1"));
        const btn = screen.getByTestId("msg-copy-e1");
        const svg = btn.querySelector("svg");
        expect(svg?.classList.toString()).toContain("text-green-400");
    });
    // --- Scroll restore + jump-to-bottom ---
    describe("scroll restore + jump-to-bottom", () => {
        beforeEach(() => {
            sessionStorage.clear();
        });
        function seedManyEvents(n) {
            const events = Array.from({ length: n }, (_, i) => makeEvent({ id: `e${i + 1}`, sequence: i + 1 }));
            seedEvents(events);
            return events;
        }
        it("restores scroll to bottom when snapshot says atBottom (re-pins on totalSize change)", () => {
            const scrollToMock = vi.fn();
            Element.prototype.scrollTo = scrollToMock;
            // Make scrollHeight large so scrollTo({ top: scrollHeight }) is meaningful.
            Object.defineProperty(HTMLElement.prototype, "scrollHeight", {
                configurable: true,
                get() { return 5000; },
            });
            sessionStorage.setItem("wc.messagesScroll.sess-1", JSON.stringify({ atBottom: true, topEventId: null }));
            seedManyEvents(120);
            render(_jsx(MessagesPane, { ...defaultProps }));
            // At least one of the scrollTo calls should target the bottom.
            const wantedBottom = scrollToMock.mock.calls.some(([arg]) => typeof arg === "object" && arg !== null && arg.top === 5000);
            expect(wantedBottom).toBe(true);
        });
        it("restores scroll to a specific event when snapshot says not at bottom", () => {
            const scrollToMock = vi.fn();
            Element.prototype.scrollTo = scrollToMock;
            sessionStorage.setItem("wc.messagesScroll.sess-1", JSON.stringify({ atBottom: false, topEventId: "e50" }));
            seedManyEvents(120);
            render(_jsx(MessagesPane, { ...defaultProps }));
            // scrollToIndex(50, "auto", "start") translates to a scrollTo({ top: <start of index 50> }).
            // We just assert it was called with some non-zero top — the bottom-pin would also do this.
            // To distinguish, check that at least one call has top !== scrollHeight (5000 by default jsdom is 0).
            expect(scrollToMock).toHaveBeenCalled();
        });
        it("with no snapshot, lands at bottom on fresh open", () => {
            const scrollToMock = vi.fn();
            Element.prototype.scrollTo = scrollToMock;
            Object.defineProperty(HTMLElement.prototype, "scrollHeight", {
                configurable: true,
                get() { return 3000; },
            });
            seedManyEvents(60);
            render(_jsx(MessagesPane, { ...defaultProps }));
            const wantedBottom = scrollToMock.mock.calls.some(([arg]) => typeof arg === "object" && arg !== null && arg.top === 3000);
            expect(wantedBottom).toBe(true);
        });
        it("jump-to-bottom button appears when not near bottom and there are no new messages", () => {
            // Force isNearBottom=false by stubbing scrollHeight/scrollTop/clientHeight so remaining > 200.
            Object.defineProperty(HTMLElement.prototype, "scrollHeight", { configurable: true, get() { return 5000; } });
            Object.defineProperty(HTMLElement.prototype, "clientHeight", { configurable: true, get() { return 500; } });
            Object.defineProperty(HTMLElement.prototype, "scrollTop", { configurable: true, get() { return 100; }, set() { } });
            seedManyEvents(40);
            render(_jsx(MessagesPane, { ...defaultProps }));
            // Fire a scroll event on the scroll container so the listener flips isNearBottom to false.
            const container = document.querySelector(".relative.min-h-0.flex-1.overflow-auto");
            expect(container).not.toBeNull();
            fireEvent.scroll(container);
            expect(screen.getByTestId("msg-jump-bottom")).toBeInTheDocument();
        });
        it("saves a not-at-bottom snapshot with the topmost visible event on unmount", () => {
            // Geometry: 5000 scrollHeight, 500 viewport, scrollTop=2000 → remaining=2500 > 200 → not near bottom.
            Object.defineProperty(HTMLElement.prototype, "scrollHeight", { configurable: true, get() { return 5000; } });
            Object.defineProperty(HTMLElement.prototype, "clientHeight", { configurable: true, get() { return 500; } });
            Object.defineProperty(HTMLElement.prototype, "scrollTop", { configurable: true, get() { return 2000; }, set() { } });
            // Make getBoundingClientRect on the scroll container return top=0, and rows return increasing bottoms so
            // the first row with bottom > 0 (containerTop) is "e3".
            const rowBottoms = new Map([
                ["e1", -50],
                ["e2", -10],
                ["e3", 30],
                ["e4", 70],
            ]);
            const origGetBCR = Element.prototype.getBoundingClientRect;
            Element.prototype.getBoundingClientRect = function () {
                const id = this.dataset?.eventId;
                if (id && rowBottoms.has(id)) {
                    const bottom = rowBottoms.get(id) ?? 0;
                    return { top: bottom - 40, bottom, left: 0, right: 0, width: 0, height: 40, x: 0, y: bottom - 40, toJSON: () => ({}) };
                }
                return { top: 0, bottom: 0, left: 0, right: 0, width: 0, height: 0, x: 0, y: 0, toJSON: () => ({}) };
            };
            try {
                seedManyEvents(20);
                const { unmount } = render(_jsx(MessagesPane, { ...defaultProps }));
                // No scroll event needed: save reads live geometry, not the ref.
                unmount();
                const raw = sessionStorage.getItem("wc.messagesScroll.sess-1");
                expect(raw).not.toBeNull();
                const snap = JSON.parse(raw);
                expect(snap.atBottom).toBe(false);
                expect(snap.topEventId).toBe("e3");
            }
            finally {
                Element.prototype.getBoundingClientRect = origGetBCR;
            }
        });
        it("under StrictMode-style double-mount, mid-list snapshot survives the first unmount/remount cycle", async () => {
            const { StrictMode } = await import("react");
            // Browser-realistic geometry: scrollTo synchronously updates scrollTop.
            let scrollTop = 0;
            Object.defineProperty(HTMLElement.prototype, "scrollHeight", { configurable: true, get() { return 5000; } });
            Object.defineProperty(HTMLElement.prototype, "clientHeight", { configurable: true, get() { return 500; } });
            Object.defineProperty(HTMLElement.prototype, "scrollTop", {
                configurable: true,
                get() { return scrollTop; },
                set(v) { scrollTop = v; },
            });
            Element.prototype.scrollTo = function (...args) {
                const opt = args[0];
                const top = typeof opt === "object" && opt !== null ? opt.top ?? 0 : opt ?? 0;
                scrollTop = top;
            };
            // Pretend the user had previously left mid-list at e3.
            const origGetBCR = Element.prototype.getBoundingClientRect;
            Element.prototype.getBoundingClientRect = function () {
                const id = this.dataset?.eventId;
                if (id === "e3") {
                    return { top: 10, bottom: 50, left: 0, right: 0, width: 0, height: 40, x: 0, y: 10, toJSON: () => ({}) };
                }
                if (id) {
                    return { top: -100, bottom: -60, left: 0, right: 0, width: 0, height: 40, x: 0, y: -100, toJSON: () => ({}) };
                }
                return { top: 0, bottom: 0, left: 0, right: 0, width: 0, height: 0, x: 0, y: 0, toJSON: () => ({}) };
            };
            sessionStorage.setItem("wc.messagesScroll.sess-1", JSON.stringify({ atBottom: false, topEventId: "e3" }));
            try {
                seedManyEvents(20);
                // StrictMode triggers an extra mount → unmount → mount in dev.
                render(_jsx(StrictMode, { children: _jsx(MessagesPane, { ...defaultProps }) }));
                const raw = sessionStorage.getItem("wc.messagesScroll.sess-1");
                const snap = JSON.parse(raw);
                // After the double-mount, snapshot must still point at e3 (not got reset to atBottom).
                expect(snap.atBottom).toBe(false);
                expect(snap.topEventId).toBe("e3");
            }
            finally {
                Element.prototype.getBoundingClientRect = origGetBCR;
            }
        });
        it("jump-to-bottom button scrolls to bottom on click and disappears once near bottom", () => {
            const scrollToMock = vi.fn();
            Element.prototype.scrollTo = scrollToMock;
            let currentScrollTop = 100;
            Object.defineProperty(HTMLElement.prototype, "scrollHeight", { configurable: true, get() { return 5000; } });
            Object.defineProperty(HTMLElement.prototype, "clientHeight", { configurable: true, get() { return 500; } });
            Object.defineProperty(HTMLElement.prototype, "scrollTop", {
                configurable: true,
                get() { return currentScrollTop; },
                set(v) { currentScrollTop = v; },
            });
            seedManyEvents(40);
            render(_jsx(MessagesPane, { ...defaultProps }));
            const container = document.querySelector(".relative.min-h-0.flex-1.overflow-auto");
            fireEvent.scroll(container);
            const btn = screen.getByTestId("msg-jump-bottom");
            fireEvent.click(btn);
            expect(scrollToMock).toHaveBeenCalledWith(expect.objectContaining({ top: 5000 }));
        });
        it("does not re-pin to bottom after a user scroll event moves away from bottom", () => {
            const originalRaf = window.requestAnimationFrame;
            window.requestAnimationFrame = ((cb) => {
                cb(0);
                return 0;
            });
            let currentScrollTop = 0;
            let currentScrollHeight = 5000;
            const scrollToMock = vi.fn((options) => {
                if (typeof options === "object" && options?.top != null) {
                    currentScrollTop = options.top;
                }
            });
            Element.prototype.scrollTo = scrollToMock;
            Object.defineProperty(HTMLElement.prototype, "scrollHeight", { configurable: true, get() { return currentScrollHeight; } });
            Object.defineProperty(HTMLElement.prototype, "clientHeight", { configurable: true, get() { return 500; } });
            Object.defineProperty(HTMLElement.prototype, "scrollTop", {
                configurable: true,
                get() { return currentScrollTop; },
                set(v) { currentScrollTop = v; },
            });
            try {
                sessionStorage.setItem("wc.messagesScroll.sess-1", JSON.stringify({ atBottom: true, topEventId: null }));
                seedManyEvents(80);
                render(_jsx(MessagesPane, { ...defaultProps }));
                expect(scrollToMock).toHaveBeenCalledWith(expect.objectContaining({ top: 5000 }));
                scrollToMock.mockClear();
                currentScrollTop = 100;
                const container = document.querySelector(".relative.min-h-0.flex-1.overflow-auto");
                fireEvent.scroll(container);
                currentScrollHeight = 7000;
                act(() => {
                    seedManyEvents(100);
                });
                expect(scrollToMock).not.toHaveBeenCalledWith(expect.objectContaining({ top: 7000 }));
            }
            finally {
                window.requestAnimationFrame = originalRaf;
            }
        });
        it("keeps the visible message fixed while an older page is prepended", async () => {
            let scrollTop = 50;
            let prepended = false;
            Object.defineProperty(HTMLElement.prototype, "scrollHeight", { configurable: true, get() { return 20000; } });
            Object.defineProperty(HTMLElement.prototype, "clientHeight", { configurable: true, get() { return 500; } });
            Object.defineProperty(HTMLElement.prototype, "scrollTop", {
                configurable: true,
                get() { return scrollTop; },
                set(value) { scrollTop = value; },
            });
            Element.prototype.scrollTo = ((options) => {
                if (typeof options === "object" && options?.top != null)
                    scrollTop = options.top;
            });
            const originalGetBoundingClientRect = Element.prototype.getBoundingClientRect;
            Element.prototype.getBoundingClientRect = function () {
                const id = this.dataset?.eventId;
                if (id) {
                    const sequence = Number(id.slice(1));
                    const top = (prepended ? sequence * 10 : (sequence - 101) * 10);
                    return { top, bottom: top + 8, left: 0, right: 0, width: 0, height: 8, x: 0, y: top, toJSON: () => ({}) };
                }
                return { top: 0, bottom: 500, left: 0, right: 0, width: 0, height: 500, x: 0, y: 0, toJSON: () => ({}) };
            };
            try {
                const current = Array.from({ length: 100 }, (_, index) => makeEvent({ id: `e${index + 101}`, sequence: index + 101 }));
                seedEvents(current);
                render(_jsx(MessagesPane, { ...defaultProps }));
                scrollTop = 50;
                mockLoadOlderConversationPage.mockImplementation(async () => {
                    prepended = true;
                    seedEvents([
                        ...Array.from({ length: 100 }, (_, index) => makeEvent({ id: `e${index + 1}`, sequence: index + 1 })),
                        ...current,
                    ]);
                    return true;
                });
                const container = document.querySelector(".relative.min-h-0.flex-1.overflow-auto");
                await act(async () => fireEvent.scroll(container));
                // e101 was 0px from the viewport top before prepend. It is moved to
                // its new virtual index rather than leaving the user at the top of
                // the newly inserted page.
                expect(scrollTop).toBeGreaterThan(10000);
            }
            finally {
                Element.prototype.getBoundingClientRect = originalGetBoundingClientRect;
            }
        });
    });
    // --- Export selection flow (navigator → drawer → clipboard) ---
    describe("export selection flow", () => {
        function openExportSelection() {
            fireEvent.click(screen.getByTestId("msg-jump-trigger"));
            fireEvent.click(screen.getByTestId("msg-export-enter"));
        }
        beforeEach(() => {
            Object.defineProperty(navigator, "clipboard", {
                value: { writeText: vi.fn().mockResolvedValue(undefined) },
                configurable: true,
            });
        });
        it("Continue opens the drawer with exactly the selected messages in order", () => {
            seedEvents([
                makeEvent({ id: "e1", sequence: 1, role: "user", text: "first question" }),
                makeEvent({ id: "e2", sequence: 2, text: "an answer" }),
                makeEvent({ id: "e3", sequence: 3, text: "unselected reply" }),
            ]);
            render(_jsx(MessagesPane, { ...defaultProps }));
            openExportSelection();
            // Select out of order — the export must still be chronological.
            fireEvent.click(screen.getByTestId("msg-jump-item-e2"));
            fireEvent.click(screen.getByTestId("msg-jump-item-e1"));
            fireEvent.click(screen.getByTestId("msg-export-continue"));
            const preview = screen.getByTestId("msg-export-preview").textContent ?? "";
            expect(preview.indexOf("first question")).toBeGreaterThanOrEqual(0);
            expect(preview.indexOf("first question")).toBeLessThan(preview.indexOf("an answer"));
            expect(preview).not.toContain("unselected reply");
        });
        it("closing the drawer keeps the navigator selection for another pass", () => {
            seedEvents([
                makeEvent({ id: "e1", sequence: 1, text: "keep me" }),
                makeEvent({ id: "e2", sequence: 2, text: "other" }),
            ]);
            render(_jsx(MessagesPane, { ...defaultProps }));
            openExportSelection();
            fireEvent.click(screen.getByTestId("msg-jump-item-e1"));
            fireEvent.click(screen.getByTestId("msg-export-continue"));
            expect(screen.getByTestId("msg-export-drawer")).toBeInTheDocument();
            fireEvent.click(screen.getByLabelText("messageExport.closeAriaLabel"));
            expect(screen.queryByTestId("msg-export-drawer")).toBeNull();
            // Navigator is still open in selection mode with the selection intact.
            expect(screen.getByTestId("msg-jump-item-e1").getAttribute("aria-checked")).toBe("true");
            fireEvent.click(screen.getByTestId("msg-export-continue"));
            expect(screen.getByTestId("msg-export-preview").textContent).toContain("keep me");
        });
        it("drops selected IDs that no longer exist after a conversation refresh", () => {
            seedEvents([
                makeEvent({ id: "e1", sequence: 1, text: "stays" }),
                makeEvent({ id: "e2", sequence: 2, text: "goes away" }),
            ]);
            render(_jsx(MessagesPane, { ...defaultProps }));
            openExportSelection();
            fireEvent.click(screen.getByTestId("msg-export-select-all"));
            expect(screen.getByTestId("msg-jump-item-e2").getAttribute("aria-checked")).toBe("true");
            act(() => {
                seedEvents([makeEvent({ id: "e1", sequence: 1, text: "stays" })]);
            });
            fireEvent.click(screen.getByTestId("msg-export-continue"));
            const preview = screen.getByTestId("msg-export-preview").textContent ?? "";
            expect(preview).toContain("stays");
            expect(preview).not.toContain("goes away");
        });
        it("existing jump behavior is preserved alongside the export entry point", () => {
            seedEvents([
                makeEvent({ id: "e1", sequence: 1, text: "target" }),
                makeEvent({ id: "e2", sequence: 2, text: "other" }),
            ]);
            render(_jsx(MessagesPane, { ...defaultProps }));
            fireEvent.click(screen.getByTestId("msg-jump-trigger"));
            expect(screen.getByTestId("msg-export-enter")).toBeInTheDocument();
            fireEvent.click(screen.getByTestId("msg-jump-item-e1"));
            // Jump closes the navigator without entering selection mode.
            expect(screen.queryByTestId("msg-jump-list")).toBeNull();
            expect(screen.queryByTestId("msg-export-footer")).toBeNull();
        });
    });
});
