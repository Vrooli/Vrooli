import { renderWithProviders as render } from "../test-utils";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, fireEvent, waitFor } from "@testing-library/react";
import MessagesPaneState from "../components/MessagesPaneState";
import { strings } from "../consts/strings";
import { UNKNOWN_CAPTURE, type MessageCaptureStatus } from "../api/messageCapture";

/**
 * The Messages pane used to answer every one of these situations with the same
 * sentence: "No conversation events yet for this session." These assertions
 * pin the distinction, because a regression here is silent — the pane still
 * renders, it just lies about why it is empty.
 */

const capture = (overrides: Partial<MessageCaptureStatus>): MessageCaptureStatus => ({
  ...UNKNOWN_CAPTURE,
  ...overrides,
});

describe("MessagesPaneState", () => {
  const onRetry = vi.fn();

  beforeEach(() => {
    onRetry.mockReset();
  });

  it("renders message-shaped skeletons while loading, never an empty message", () => {
    render(<MessagesPaneState view={{ kind: "loading" }} onRetry={onRetry} />);

    expect(screen.getByTestId("messages-state-loading")).toBeInTheDocument();
    expect(screen.queryByTestId("messages-state-empty")).toBeNull();
    expect(screen.queryByTestId("messages-state-unavailable")).toBeNull();
  });

  it("offers a retry for a retryable failure and states the reason", () => {
    render(
      <MessagesPaneState
        view={{ kind: "failed", error: { message: "Web Console couldn't reach the server.", code: "unavailable", retryable: true } }}
        onRetry={onRetry}
      />,
    );

    expect(screen.getByTestId("messages-state-failed")).toBeInTheDocument();
    expect(screen.getByText("Web Console couldn't reach the server.")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("messages-state-failed-action"));
    expect(onRetry).toHaveBeenCalledTimes(1);
  });

  it("does not offer a retry for a failure retrying cannot fix", () => {
    render(
      <MessagesPaneState
        view={{ kind: "failed", error: { message: "This session no longer exists.", code: "not_found", retryable: false } }}
        onRetry={onRetry}
      />,
    );

    expect(screen.getByTestId("messages-state-failed")).toBeInTheDocument();
    expect(screen.queryByTestId("messages-state-failed-action")).toBeNull();
  });

  it("names the cause and hides the operator detail behind a disclosure when capture is unavailable", async () => {
    render(
      <MessagesPaneState
        view={{
          kind: "unavailable",
          capture: capture({
            state: "unavailable",
            reasonCode: "hook_not_registered",
            summary: "Messages aren't being captured — Web Console isn't connected to Claude Code in this project.",
            detail: "Claude Stop hook is not registered in project settings (hook_missing).",
            remediation: "Run 'web-console hooks register' to reconnect message capture.",
          }),
        }}
        onRetry={onRetry}
      />,
    );

    expect(screen.getByTestId("messages-state-unavailable")).toBeInTheDocument();
    expect(screen.getByText(/isn't connected to Claude Code/)).toBeInTheDocument();
    // The path/hook specifics are for an operator, not the primary sentence.
    expect(screen.queryByTestId("messages-state-unavailable-details")).toBeNull();

    fireEvent.click(screen.getByTestId("messages-state-unavailable-details-toggle"));
    await waitFor(() => expect(screen.getByTestId("messages-state-unavailable-details")).toBeInTheDocument());
    expect(screen.getByTestId("messages-state-unavailable-details")).toHaveTextContent("hook_missing");
    expect(screen.getByTestId("messages-state-unavailable-details")).toHaveTextContent("hooks register");
  });

  it("explains a plain terminal without implying a fault", () => {
    render(
      <MessagesPaneState
        view={{ kind: "not-applicable", capture: capture({ state: "not_applicable", reasonCode: "no_agent", summary: "This is a plain terminal, so there are no messages to show." }) }}
        onRetry={onRetry}
      />,
    );

    expect(screen.getByTestId("messages-state-not-applicable")).toBeInTheDocument();
    expect(screen.queryByTestId("messages-state-not-applicable-action")).toBeNull();
  });

  it("offers no action for a genuinely new conversation", () => {
    render(
      <MessagesPaneState
        view={{ kind: "empty", capture: capture({ state: "capturing" }) }}
        onRetry={onRetry}
      />,
    );

    expect(screen.getByTestId("messages-state-empty")).toBeInTheDocument();
    expect(screen.getByText(strings.messagesPane.state.emptyTitle)).toBeInTheDocument();
    expect(screen.queryByTestId("messages-state-empty-action")).toBeNull();
  });

  it("says it is still waiting when capture has not identified the session yet", () => {
    render(
      <MessagesPaneState
        view={{ kind: "empty", capture: capture({ state: "pending", reasonCode: "awaiting_first_turn" }) }}
        onRetry={onRetry}
      />,
    );

    expect(screen.getByText(strings.messagesPane.state.pendingTitle)).toBeInTheDocument();
  });
});
