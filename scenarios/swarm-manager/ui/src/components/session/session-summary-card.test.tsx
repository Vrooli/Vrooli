import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { SessionSummaryCard } from "./session-summary-card";
import { selectors } from "../../consts/selectors";
import type { AgentSession } from "../../types";

function makeSession(overrides: Partial<AgentSession> = {}): AgentSession {
  return {
    id: "sess-1",
    title: "Plan quality work",
    kind: "meta_orchestration",
    status: "running",
    skillId: "skill",
    taskId: "task",
    runId: "run",
    profileKey: "swarm-manager/default",
    createdAt: "2026-05-01T12:00:00Z",
    updatedAt: "2026-05-01T12:10:00Z",
    messages: [],
    proposals: [],
    artifacts: [],
    ...overrides,
  };
}

describe("SessionSummaryCard", () => {
  it("list content: renders the summary without its own button or checkbox", () => {
    render(<SessionSummaryCard session={makeSession()} />);
    expect(screen.getByText("Plan quality work")).toBeInTheDocument();
    expect(screen.getByText("Plan work")).toBeInTheDocument();
    // The CollectionList row owns opening; nested buttons would swallow its click.
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
    expect(screen.queryByRole("checkbox")).not.toBeInTheDocument();
  });

  it("pick mode: renders the context row and toggles on click", async () => {
    const onToggleSelect = vi.fn();
    render(
      <SessionSummaryCard
        session={makeSession()}
        selection={{ selectionMode: true, selected: false, onToggleSelect }}
      />,
    );
    const row = screen.getByTestId(selectors.agentSessions.contextRow);
    expect(row).toHaveAttribute("aria-pressed", "false");
    await userEvent.click(row);
    expect(onToggleSelect).toHaveBeenCalledTimes(1);
  });

  it("pick mode disabled: does not toggle, exposes the reason", async () => {
    const onToggleSelect = vi.fn();
    render(
      <SessionSummaryCard
        session={makeSession()}
        selection={{ selectionMode: true, selected: false, disabled: true, disabledReason: "Cap reached", onToggleSelect }}
      />,
    );
    const row = screen.getByTestId(selectors.agentSessions.contextRow);
    expect(row).toBeDisabled();
    expect(row).toHaveAttribute("title", "Cap reached");
    await userEvent.click(row);
    expect(onToggleSelect).not.toHaveBeenCalled();
  });
});
