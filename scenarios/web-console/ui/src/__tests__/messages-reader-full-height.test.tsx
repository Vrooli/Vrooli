import type { ReactNode } from "react";
import { renderWithProviders as render } from "../test-utils";
import { describe, expect, it, vi } from "vitest";
import { act, fireEvent, screen, within } from "@testing-library/react";
import { MessagesReader } from "../components/messages/MessagesReader";
import type { MessageActionContext } from "../components/messages/messageActions";
import { strings } from "../consts/strings";
import type { ConversationEvent } from "../api/conversation";

// The drawer's own gesture is the library's to test; here a stand-in renders
// its slots and hands the test the drawer's expanded-state callback.
const drawer = vi.hoisted(() => ({ onExpandedChange: undefined as ((expanded: boolean) => void) | undefined }));

vi.mock("@vrooli/react-component-library/FullPageDrawer/1", () => ({
  FullPageDrawer: (props: {
    title: ReactNode;
    ariaLabel?: string;
    headerExtra?: ReactNode;
    headerActions?: ReactNode;
    subheader?: ReactNode;
    footer?: ReactNode;
    children: ReactNode;
    onExpandedChange?: (expanded: boolean) => void;
  }) => {
    drawer.onExpandedChange = props.onExpandedChange;
    return (
      <section data-testid="messages-reader" role="dialog" aria-label={props.ariaLabel}>
        <header data-testid="reader-header">
          <h2>{props.title}</h2>
          {props.headerExtra}
          {props.headerActions}
        </header>
        {props.subheader ? <div data-testid="reader-subheader">{props.subheader}</div> : null}
        {props.children}
        {props.footer}
      </section>
    );
  },
}));

vi.mock("../components/markdown", () => ({
  MarkdownRenderer: ({ content }: { content: string }) => <div>{content}</div>,
}));

const event: ConversationEvent = {
  id: "long", sessionId: "sess-1", sequence: 2, source: "claude_hook", role: "assistant",
  text: "alpha one. beta two. alpha three.", speechParagraphs: ["alpha one."], summarized: false,
  createdAt: new Date().toISOString(), deliveryState: "received", ttsState: "idle", consumptionState: "seen",
};

const actionContext: MessageActionContext = {
  event, sessionId: "sess-1", readOnly: false, copied: false, isPlaintext: false, isAudioLoading: false,
  isTtsSpeaking: false, activeSpeakingEventId: null, summarizeLevel: "moderate", selectedVersion: "active",
  summarizingEventId: null, getSummarizeError: () => null, onClearSummarizeError: vi.fn(), onToggleSummarized: vi.fn(),
  onChangeLevel: vi.fn(), onCopy: vi.fn(), onPlayFromHere: vi.fn(), onToggleRenderMode: vi.fn(),
};

describe("messages reader at full height", () => {
  it("[REQ:P0-017d] spends the room on the find field: it becomes the header, the title and its band go, and a query in progress keeps its text and caret", () => {
    Element.prototype.scrollIntoView = vi.fn();
    render(
      <MessagesReader
        actionContext={actionContext}
        fontSize={16}
        onFontSizeChange={vi.fn()}
        coarsePointer
        onClose={vi.fn()}
        onLinkClick={vi.fn()}
        onFileReferenceClick={vi.fn()}
        onMermaidOpen={vi.fn()}
      />,
    );
    const speaker = strings.messagesPane.speaker.claude;

    // Settled: the speaker titles the header and the field has its own band.
    expect(screen.getByTestId("reader-header")).toHaveTextContent(speaker);
    const settledField = within(screen.getByTestId("reader-subheader")).getByTestId("reader-find-input");
    settledField.focus();
    fireEvent.change(settledField, { target: { value: "alpha" } });

    act(() => { drawer.onExpandedChange?.(true); });
    const header = screen.getByTestId("reader-header");
    const field = within(header).getByTestId("reader-find-input");
    expect(field).toHaveValue("alpha");
    expect(document.activeElement).toBe(field);
    expect(header).not.toHaveTextContent(speaker);
    expect(header).not.toHaveTextContent("#2");
    expect(screen.queryByTestId("reader-subheader")).toBeNull();
    // The dialog is still named for the speaker.
    expect(screen.getByTestId("messages-reader")).toHaveAttribute("aria-label", speaker);

    act(() => { drawer.onExpandedChange?.(false); });
    expect(screen.getByTestId("reader-header")).toHaveTextContent(speaker);
    expect(within(screen.getByTestId("reader-subheader")).getByTestId("reader-find-input")).toHaveValue("alpha");
  });
});
