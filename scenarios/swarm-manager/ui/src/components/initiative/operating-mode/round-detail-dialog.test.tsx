import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { RoundDetailDialog } from "./round-detail-dialog";
import { selectors } from "../../../consts/selectors";
import type { OperatingModeRound } from "../../../types/operating-mode";

function makeRound(overrides: Partial<OperatingModeRound> = {}): OperatingModeRound {
  return {
    mode: "holistic-loop",
    round: 7,
    scopeKind: "initiative",
    scopeId: "init-a",
    phase: "investigate",
    runStrategy: "operator_gated_loop",
    agentProfileKey: "swarm-manager/deep-work",
    generatedAt: "2026-04-30T00:00:00Z",
    runId: "run-deadbeef",
    status: "completed",
    items: [{ ref: "execute/some-task", title: "Some task", status: "completed" }],
    handoffs: [
      { summary: "Handoff summary text", nextStep: "Re-run lint" },
    ],
    error: undefined,
    ...overrides,
  };
}

describe("RoundDetailDialog", () => {
  it("renders the round number, phase, and status pill in the title region", () => {
    render(<RoundDetailDialog round={makeRound()} isOpen onClose={() => {}} />);
    const dialog = screen.getByTestId(selectors.initiativeDetails.roundDetailDialog);
    expect(dialog).toHaveTextContent("Round 7 — Investigate");
    expect(dialog).toHaveTextContent(/Completed/i);
  });

  it("shows the agent profile and runId with a Copy button", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText } });
    render(<RoundDetailDialog round={makeRound()} isOpen onClose={() => {}} />);
    expect(screen.getByText("swarm-manager/deep-work")).toBeInTheDocument();
    expect(screen.getByText("run-deadbeef")).toBeInTheDocument();
    await userEvent.click(screen.getByTestId(selectors.initiativeDetails.roundDetailRunIdCopy));
    expect(writeText).toHaveBeenCalledWith("run-deadbeef");
  });

  it("renders an error block when the round failed", () => {
    render(
      <RoundDetailDialog
        round={makeRound({ error: "Build crashed", status: "failed" })}
        isOpen
        onClose={() => {}}
      />,
    );
    expect(screen.getByText("Build crashed")).toBeInTheDocument();
  });

  it("renders the items and handoff sections when present", () => {
    render(<RoundDetailDialog round={makeRound()} isOpen onClose={() => {}} />);
    expect(screen.getByText("execute/some-task")).toBeInTheDocument();
    expect(screen.getByText("Some task")).toBeInTheDocument();
    expect(screen.getByText("Handoff summary text")).toBeInTheDocument();
    expect(screen.getByText(/Next: Re-run lint/)).toBeInTheDocument();
  });

  it("gracefully omits sections when fields are missing", () => {
    const round = makeRound({ items: undefined, handoffs: undefined, error: undefined, runId: undefined });
    render(<RoundDetailDialog round={round} isOpen onClose={() => {}} />);
    // Items header / handoffs header should not appear.
    expect(screen.queryByText(/Items operated on/i)).toBeNull();
    expect(screen.queryByText(/Handoffs/i)).toBeNull();
    // No copy button without runId.
    expect(screen.queryByTestId(selectors.initiativeDetails.roundDetailRunIdCopy)).toBeNull();
  });

  it("invokes onClose when Close is clicked", async () => {
    const onClose = vi.fn();
    render(<RoundDetailDialog round={makeRound()} isOpen onClose={onClose} />);
    await userEvent.click(screen.getByRole("button", { name: "Close" }));
    expect(onClose).toHaveBeenCalledTimes(1);
  });
});
