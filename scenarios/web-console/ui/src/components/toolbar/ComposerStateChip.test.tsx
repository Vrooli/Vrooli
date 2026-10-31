import { renderWithProviders as render } from "../../test-utils";
import { beforeEach, describe, expect, it } from "vitest";
import { screen } from "@testing-library/react";
import { ComposerStateChip } from "./ComposerStateChip";
import { strings } from "../../consts/strings";
import { useSessionActivityStore } from "../../stores/useSessionActivityStore";
import type { SessionActivityView } from "../../api/sessionActivity";

function activity(overrides: Partial<SessionActivityView>): SessionActivityView {
  return { state: "idle", source: "screen", confidence: 0.85, since: "2026-09-11T08:00:00Z", harness: "claude", ...overrides };
}

describe("ComposerStateChip", () => {
  beforeEach(() => {
    useSessionActivityStore.setState({ activities: {} });
  });

  it.each([
    ["idle", strings.composerState.idle],
    ["working", strings.composerState.working],
    ["waiting", strings.composerState.waiting],
  ] as const)("[REQ:P0-017f] names the %s state", (state, label) => {
    useSessionActivityStore.setState({ activities: { s1: activity({ state }) } });
    render(<ComposerStateChip sessionId="s1" />);
    const chip = screen.getByTestId("composer-state-chip");
    expect(chip).toHaveAttribute("data-state", state);
    expect(chip).toHaveTextContent(label);
  });

  it("[REQ:P0-017f] reads unknown for no activity, an unknown state, or a low-confidence reading", () => {
    const { rerender } = render(<ComposerStateChip sessionId="s1" />);
    expect(screen.getByTestId("composer-state-chip")).toHaveAttribute("data-state", "unknown");

    useSessionActivityStore.setState({ activities: { s1: activity({ state: "working", confidence: 0.4 }) } });
    rerender(<ComposerStateChip sessionId="s1" />);
    expect(screen.getByTestId("composer-state-chip")).toHaveAttribute("data-state", "unknown");
    expect(screen.getByTestId("composer-state-chip")).toHaveTextContent(strings.composerState.unknown);
  });

  it("[REQ:P0-017f] renders nothing without a session", () => {
    render(<ComposerStateChip sessionId={null} />);
    expect(screen.queryByTestId("composer-state-chip")).toBeNull();
  });
});
