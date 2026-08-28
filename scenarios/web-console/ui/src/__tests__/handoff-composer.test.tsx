import { renderWithProviders as render } from "../test-utils";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, fireEvent, waitFor, within } from "@testing-library/react";

import HandoffComposer from "../components/handoff/HandoffComposer";
import type { HandoffTarget } from "../hooks/useHandoff";
import type { RoleMeta } from "../stores/useWorkspaceStore";

// [REQ:P0-014d] Handoff Between Sessions In A Group

const role = (id: string, label: string, incomingPrompt = ""): RoleMeta => ({
  id,
  groupId: "g1",
  label,
  command: "codex --yolo",
  workingDir: "",
  incomingPrompt,
  backend: "",
  targetId: "",
  sessionId: null,
  sortOrder: 0,
});

const session = (id: string, label: string, incomingPrompt = ""): HandoffTarget => ({
  kind: "session",
  sessionId: id,
  label,
  incomingPrompt,
});

const waiting = (id: string, label: string, incomingPrompt = ""): HandoffTarget => ({
  kind: "role",
  role: role(id, label, incomingPrompt),
  label,
  incomingPrompt,
});

describe("HandoffComposer", () => {
  const onClose = vi.fn();

  beforeEach(() => { vi.clearAllMocks(); });

  it("says so when the group holds nobody to hand off to", () => {
    render(
      <HandoffComposer open onClose={onClose} sourceLabel="planner" payload="" targets={[]} onSend={vi.fn()} />,
    );
    expect(screen.getByTestId("handoff-no-targets")).toBeInTheDocument();
    expect(screen.getByTestId("handoff-send")).toBeDisabled();
  });

  it("shows the payload it is carrying", () => {
    render(
      <HandoffComposer
        open
        onClose={onClose}
        sourceLabel="planner"
        payload="/home/me/plan.md"
        targets={[session("s2", "builder")]}
        onSend={vi.fn()}
      />,
    );
    expect(screen.getByTestId("handoff-composer")).toHaveTextContent("/home/me/plan.md");
  });

  // Nothing is dispatched without the operator seeing the exact text.
  it("renders the target's prompt with the payload substituted, and lets it be edited", () => {
    render(
      <HandoffComposer
        open
        onClose={onClose}
        sourceLabel="planner"
        payload="plan.md"
        targets={[session("s2", "builder", "Implement the plan at {{payload}}")]}
        onSend={vi.fn()}
      />,
    );
    const field = screen.getByTestId("handoff-message");
    expect(field).toHaveValue("Implement the plan at plan.md");
    fireEvent.change(field, { target: { value: "Actually, review it first" } });
    expect(field).toHaveValue("Actually, review it first");
  });

  it("preselects the only target so the common case costs no click", () => {
    render(
      <HandoffComposer
        open
        onClose={onClose}
        sourceLabel="planner"
        payload=""
        targets={[session("s2", "builder")]}
        onSend={vi.fn()}
      />,
    );
    expect(screen.getByTestId("handoff-send")).not.toBeDisabled();
  });

  it("sends the edited text to the selected target", async () => {
    const onSend = vi.fn().mockResolvedValue([{ targetId: "s2", label: "builder", status: "sent" }]);
    render(
      <HandoffComposer
        open
        onClose={onClose}
        sourceLabel="planner"
        payload="plan.md"
        targets={[session("s2", "builder")]}
        onSend={onSend}
      />,
    );
    fireEvent.click(screen.getByTestId("handoff-send"));
    await waitFor(() => { expect(onSend).toHaveBeenCalled(); });

    const [targets, textFor] = onSend.mock.calls[0] as [HandoffTarget[], (t: HandoffTarget) => string];
    expect(targets).toHaveLength(1);
    expect(textFor(targets[0]!)).toBe("plan.md");
    // Everything delivered, so the surface gets out of the way.
    await waitFor(() => { expect(onClose).toHaveBeenCalled(); });
  });

  it("accepts more than one target and sends to each", async () => {
    const onSend = vi.fn().mockResolvedValue([
      { targetId: "s2", label: "builder", status: "sent" },
      { targetId: "s3", label: "critic", status: "sent" },
    ]);
    render(
      <HandoffComposer
        open
        onClose={onClose}
        sourceLabel="planner"
        payload="plan.md"
        targets={[session("s2", "builder"), session("s3", "critic")]}
        onSend={onSend}
      />,
    );
    fireEvent.click(within(screen.getByTestId("handoff-target-s2")).getByRole("checkbox"));
    fireEvent.click(within(screen.getByTestId("handoff-target-s3")).getByRole("checkbox"));
    fireEvent.click(screen.getByTestId("handoff-send"));

    await waitFor(() => { expect(onSend).toHaveBeenCalled(); });
    const [targets] = onSend.mock.calls[0] as [HandoffTarget[]];
    expect(targets.map((t) => t.label)).toEqual(["builder", "critic"]);
  });

  it("gives each target its own field when their prompts differ", () => {
    render(
      <HandoffComposer
        open
        onClose={onClose}
        sourceLabel="planner"
        payload="plan.md"
        targets={[
          session("s2", "builder", "Implement {{payload}}"),
          session("s3", "critic", "Critique {{payload}}"),
        ]}
        initialSelection={["s2", "s3"]}
        onSend={vi.fn()}
      />,
    );
    expect(screen.getByTestId("handoff-message-s2")).toHaveValue("Implement plan.md");
    expect(screen.getByTestId("handoff-message-s3")).toHaveValue("Critique plan.md");
  });

  // A waiting target has to start before it can receive, and the surface says
  // so rather than making the delay look like a hang.
  it("marks a waiting role as starting first", () => {
    render(
      <HandoffComposer
        open
        onClose={onClose}
        sourceLabel="planner"
        payload=""
        targets={[waiting("r1", "Implementer")]}
        onSend={vi.fn()}
      />,
    );
    expect(screen.getByTestId("handoff-target-r1")).toHaveTextContent("handoff.startsFirst");
  });

  // The most important assertion here: queued is reported as queued.
  it("reports a queued target without claiming it was delivered", async () => {
    const onSend = vi.fn().mockResolvedValue([
      { targetId: "s2", label: "builder", status: "queued", reason: "not-ready" },
    ]);
    render(
      <HandoffComposer
        open
        onClose={onClose}
        sourceLabel="planner"
        payload="plan.md"
        targets={[session("s2", "builder")]}
        onSend={onSend}
      />,
    );
    fireEvent.click(screen.getByTestId("handoff-send"));

    await waitFor(() => { expect(screen.getByTestId("handoff-result-s2")).toBeInTheDocument(); });
    expect(screen.getByTestId("handoff-result-s2")).toHaveAttribute("data-status", "queued");
    // The composer stays open and keeps the text so the operator can retry.
    expect(onClose).not.toHaveBeenCalled();
    expect(screen.getByTestId("handoff-message")).toHaveValue("plan.md");
  });

  it("reports a failed target and keeps the text", async () => {
    const onSend = vi.fn().mockResolvedValue([
      { targetId: "r1", label: "Implementer", status: "failed", reason: "start-failed" },
    ]);
    render(
      <HandoffComposer
        open
        onClose={onClose}
        sourceLabel="planner"
        payload="plan.md"
        targets={[waiting("r1", "Implementer")]}
        onSend={onSend}
      />,
    );
    fireEvent.click(screen.getByTestId("handoff-send"));

    await waitFor(() => { expect(screen.getByTestId("handoff-result-r1")).toBeInTheDocument(); });
    expect(screen.getByTestId("handoff-result-r1")).toHaveAttribute("data-status", "failed");
    expect(onClose).not.toHaveBeenCalled();
  });

  it("reports a mixed outcome per target", async () => {
    const onSend = vi.fn().mockResolvedValue([
      { targetId: "s2", label: "builder", status: "sent" },
      { targetId: "r1", label: "Implementer", status: "queued", reason: "not-ready" },
    ]);
    render(
      <HandoffComposer
        open
        onClose={onClose}
        sourceLabel="planner"
        payload="plan.md"
        targets={[session("s2", "builder"), waiting("r1", "Implementer")]}
        initialSelection={["s2", "r1"]}
        onSend={onSend}
      />,
    );
    fireEvent.click(screen.getByTestId("handoff-send"));

    await waitFor(() => { expect(screen.getByTestId("handoff-results")).toBeInTheDocument(); });
    expect(screen.getByTestId("handoff-result-s2")).toHaveAttribute("data-status", "sent");
    expect(screen.getByTestId("handoff-result-r1")).toHaveAttribute("data-status", "queued");
    expect(onClose).not.toHaveBeenCalled();
  });

  it("honours a caller's preselection, as a suggestion chip supplies", () => {
    render(
      <HandoffComposer
        open
        onClose={onClose}
        sourceLabel="planner"
        payload="plan.md"
        targets={[session("s2", "builder"), waiting("r1", "Implementer")]}
        initialSelection={["r1"]}
        onSend={vi.fn()}
      />,
    );
    expect(within(screen.getByTestId("handoff-target-r1")).getByRole("checkbox")).toBeChecked();
    expect(within(screen.getByTestId("handoff-target-s2")).getByRole("checkbox")).not.toBeChecked();
  });
});
