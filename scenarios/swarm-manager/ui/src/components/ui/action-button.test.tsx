import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Play } from "lucide-react";
import { describe, expect, it, vi } from "vitest";
import { TransitionKind } from "@vrooli/proto-types/swarm-manager/v1/domain/transition_pb";
import { ActionButton, ConsequenceBadge } from "./action-button";

describe("ActionButton", () => {
  it("marks an action that dispatches an agent", () => {
    // The whole point: an operator must be able to tell, before pressing,
    // that this spends agent time.
    render(<ActionButton label="Start review" transitionKind={TransitionKind.WORKFLOW} onClick={vi.fn()} data-testid="btn" />);
    const button = screen.getByTestId("btn");
    expect(button).toHaveAttribute("data-consequence", "agent_workflow");
    expect(button.getAttribute("title")).toMatch(/agent run/i);
  });

  it("distinguishes an interactive session from a batch run", () => {
    render(<ActionButton label="Open session" transitionKind={TransitionKind.SESSION} onClick={vi.fn()} data-testid="btn" />);
    expect(screen.getByTestId("btn")).toHaveAttribute("data-consequence", "agent_session");
  });

  it("does not mark an immediate state change", () => {
    render(<ActionButton label="Accept" actionId="accept_suggestion" effect="state_change" onClick={vi.fn()} data-testid="btn" />);
    const button = screen.getByTestId("btn");
    expect(button).toHaveAttribute("data-consequence", "state_change");
    expect(button.getAttribute("title")).not.toMatch(/agent/i);
  });

  it("signals a destructive action with an ellipsis, promising a confirm step", () => {
    render(<ActionButton label="Archive" actionId="archive" destructive onClick={vi.fn()} data-testid="btn" />);
    const button = screen.getByTestId("btn");
    expect(button).toHaveAttribute("data-consequence", "destructive");
    expect(button).toHaveTextContent("Archive…");
  });

  it("does not append an ellipsis to a non-destructive action", () => {
    render(<ActionButton label="Start review" actionId="review" effect="agent_run" onClick={vi.fn()} data-testid="btn" />);
    expect(screen.getByTestId("btn")).toHaveTextContent("Start review");
    expect(screen.getByTestId("btn")).not.toHaveTextContent("…");
  });

  it("shows the pending label and blocks a second press", async () => {
    const onClick = vi.fn();
    render(<ActionButton label="Start review" actionId="review" effect="agent_run" pending onClick={onClick} data-testid="btn" />);

    expect(screen.getByTestId("btn")).toHaveTextContent("Working…");
    await userEvent.click(screen.getByTestId("btn"));
    expect(onClick).not.toHaveBeenCalled();
  });

  it("fires when idle", async () => {
    const onClick = vi.fn();
    render(<ActionButton label="Go" actionId="accept_plan" onClick={onClick} icon={Play} data-testid="btn" />);
    await userEvent.click(screen.getByTestId("btn"));
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it("lets the caller override the tooltip", () => {
    render(<ActionButton label="Plan goal" actionId="plan_goal" title="Starts from the current snapshot" onClick={vi.fn()} data-testid="btn" />);
    expect(screen.getByTestId("btn")).toHaveAttribute("title", "Starts from the current snapshot");
  });
});

describe("ConsequenceBadge", () => {
  it("appears only for agent-spawning actions", () => {
    const { rerender } = render(<ConsequenceBadge transitionKind={TransitionKind.WORKFLOW} />);
    expect(screen.getByTestId("consequence-badge")).toHaveTextContent("Agent run");

    rerender(<ConsequenceBadge transitionKind={TransitionKind.SESSION} />);
    expect(screen.getByTestId("consequence-badge")).toHaveTextContent("Agent session");

    rerender(<ConsequenceBadge transitionKind={TransitionKind.DETERMINISTIC} />);
    expect(screen.queryByTestId("consequence-badge")).toBeNull();

    rerender(<ConsequenceBadge destructive />);
    expect(screen.queryByTestId("consequence-badge")).toBeNull();
  });
});
