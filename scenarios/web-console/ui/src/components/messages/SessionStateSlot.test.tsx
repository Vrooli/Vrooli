import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils";
import { SessionStateSlot } from "./SessionStateSlot";

describe("SessionStateSlot agent termination recovery", () => {
  it("offers terminal inspection and never offers a command to the exited agent", () => {
    const onOpenTerminal = vi.fn();
    const onAnswer = vi.fn();

    renderWithProviders(
      <SessionStateSlot
        slot={{
          kind: "interruption",
          harness: "codex",
          source: "screen",
          since: "2026-11-03T12:00:00Z",
          prompt: {
            kind: "agent_terminated",
            text: "The terminal reported 'Killed'. The cause is unknown.",
            options: [],
            answerable: false,
          },
        }}
        onOpenTerminal={onOpenTerminal}
        onAnswer={onAnswer}
      />,
    );

    fireEvent.click(screen.getByTestId("state-slot-open-terminal"));
    expect(onOpenTerminal).toHaveBeenCalledTimes(1);
    expect(screen.getByText(/cause is unknown/)).toBeInTheDocument();
    expect(screen.queryByTestId("state-slot-option-continue")).toBeNull();
    expect(onAnswer).not.toHaveBeenCalled();
  });
});
