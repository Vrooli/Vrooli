import { renderWithProviders as render } from "../test-utils";
import { describe, it, expect } from "vitest";
import { screen } from "@testing-library/react";
import MessagesPaneStatusLine from "../components/MessagesPaneStatusLine";
import { resolveMessagesPaneStatus } from "../lib/messagesPaneStatus";

/**
 * The pane previously rendered its live-stream notice and its refresh result as
 * two independent banners, which stacked and contradicted each other on screen:
 * "Live updates disconnected — reconnecting" directly above "Up to date".
 * These assertions pin the single-line, priority-ordered contract that replaced
 * them.
 */
describe("resolveMessagesPaneStatus", () => {
  const base = { refreshError: null, liveInterrupted: false, liveInterruptedText: "Live updates disconnected", transient: null };

  it("shows nothing when there is nothing to say", () => {
    expect(resolveMessagesPaneStatus(base)).toBeNull();
  });

  it("never reassures while the stream is interrupted", () => {
    const status = resolveMessagesPaneStatus({ ...base, liveInterrupted: true, transient: "Up to date" });
    expect(status).toEqual({ kind: "disconnected", text: "Live updates disconnected" });
  });

  it("ranks a failed refresh above a dropped stream", () => {
    const status = resolveMessagesPaneStatus({
      ...base,
      refreshError: "Web Console couldn't reach the server.",
      liveInterrupted: true,
      transient: "Up to date",
    });
    expect(status).toEqual({ kind: "error", text: "Web Console couldn't reach the server." });
  });

  it("shows the transient confirmation only when all is well", () => {
    expect(resolveMessagesPaneStatus({ ...base, transient: "3 new messages" }))
      .toEqual({ kind: "success", text: "3 new messages" });
  });
});

describe("MessagesPaneStatusLine", () => {
  it("reserves no space when idle", () => {
    render(<MessagesPaneStatusLine status={null} />);
    expect(screen.queryByTestId("messages-status-line")).toBeNull();
  });

  it("renders exactly one line, tagged with its severity", () => {
    render(<MessagesPaneStatusLine status={{ kind: "error", text: "Couldn't reach the server." }} />);
    const line = screen.getByTestId("messages-status-line");
    expect(line).toHaveAttribute("data-status-kind", "error");
    expect(line).toHaveTextContent("Couldn't reach the server.");
    expect(screen.getAllByTestId("messages-status-line")).toHaveLength(1);
  });
});
