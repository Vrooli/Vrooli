import { renderWithProviders as render } from "../../test-utils";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, screen, within } from "@testing-library/react";
import { i18n } from "../../i18n";
import { SentHistorySheet } from "./SentHistorySheet";
import type { SentHistoryEntry } from "../../hooks/useCommandHistory";

const NOW = Date.parse("2026-09-11T12:00:00Z");

const entries: SentHistoryEntry[] = [
  { text: "git status", at: NOW - 3 * 60 * 60 * 1000 },
  { text: "run the tests", at: NOW - 4 * 60 * 1000 },
];

function renderSheet(overrides: Partial<Parameters<typeof SentHistorySheet>[0]> = {}) {
  const props = {
    open: true,
    entries,
    onClose: vi.fn(),
    onInsert: vi.fn(),
    onSend: vi.fn(),
    onClear: vi.fn(),
    ...overrides,
  };
  render(<SentHistorySheet {...props} />);
  return props;
}

describe("SentHistorySheet", () => {
  beforeEach(async () => {
    // Relative times are part of what the user reads here, so render English.
    await i18n.changeLanguage("en");
    vi.useFakeTimers({ now: NOW, toFake: ["Date"] });
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("[REQ:P0-017f] lists sends newest first with how long ago", () => {
    renderSheet();
    const rows = screen.getAllByTestId("sent-history-row");
    expect(rows.map((row) => within(row).getByTestId("sent-history-text").textContent)).toEqual(["run the tests", "git status"]);
    expect(rows[0]).toHaveTextContent("4 m ago");
    expect(rows[1]).toHaveTextContent("3 h ago");
  });

  it("[REQ:P0-017f] narrows the list by substring", () => {
    renderSheet();
    fireEvent.change(screen.getByTestId("sent-history-filter"), { target: { value: "GIT" } });
    const rows = screen.getAllByTestId("sent-history-row");
    expect(rows).toHaveLength(1);
    expect(rows[0]).toHaveTextContent("git status");
  });

  it("[REQ:P0-017f] Insert hands the text to the draft and closes", () => {
    const props = renderSheet();
    const first = screen.getAllByTestId("sent-history-row")[0] as HTMLElement;
    fireEvent.click(within(first).getByTestId("sent-history-insert"));
    expect(props.onInsert).toHaveBeenCalledWith("run the tests");
    expect(props.onClose).toHaveBeenCalled();
  });

  it("[REQ:P0-017f] Resend sends the text the way Send does and closes", () => {
    const props = renderSheet();
    const second = screen.getAllByTestId("sent-history-row")[1] as HTMLElement;
    fireEvent.click(within(second).getByTestId("sent-history-resend"));
    expect(props.onSend).toHaveBeenCalledWith("git status");
    expect(props.onClose).toHaveBeenCalled();
  });

  it("[REQ:P0-017f] clears the device's history", () => {
    const props = renderSheet();
    fireEvent.click(screen.getByTestId("sent-history-clear"));
    expect(props.onClear).toHaveBeenCalledTimes(1);
  });

  it("[REQ:P0-017f] says so when nothing has been sent", () => {
    renderSheet({ entries: [] });
    expect(screen.getByTestId("sent-history-empty")).toHaveTextContent("Nothing sent from this device yet");
    expect(screen.queryByTestId("sent-history-row")).toBeNull();
  });
});
