import { renderWithProviders as render } from "../test-utils";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, fireEvent, waitFor, within } from "@testing-library/react";

import HandoffComposer from "../components/handoff/HandoffComposer";
import type { HandoffTarget, HandoffTargetSection } from "../hooks/useHandoff";
import type { RoleMeta } from "../stores/useWorkspaceStore";

const { snippetTouch } = vi.hoisted(() => ({ snippetTouch: vi.fn().mockResolvedValue(undefined) }));
vi.mock("../hooks/useSnippets", () => ({
  useSnippets: () => ({
    snippets: [
      { id: "payload-snippet", name: "Payload wrapper", body: "Snippet says: {{payload}}", color: "#22c55e", pinned: false, sort_order: 0, use_count: 0, last_used_at: null, created_at: "", updated_at: "" },
      { id: "name-snippet", name: "Needs a name", body: "Hello {{name}}", color: "#3b82f6", pinned: false, sort_order: 1, use_count: 0, last_used_at: null, created_at: "", updated_at: "" },
    ],
    status: "ready",
    error: null,
    touch: snippetTouch,
    save: vi.fn(),
    remove: vi.fn(),
    reload: vi.fn(),
  }),
}));

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

const sections = (...targets: HandoffTarget[]): HandoffTargetSection[] => (
  targets.length === 0 ? [] : [{ kind: "group", labelKey: "handoff.sections.group", targets }]
);

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
        targets={sections(session("s2", "builder"))}
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
        targets={sections(session("s2", "builder", "Implement the plan at {{payload}}"))}
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
        targets={sections(session("s2", "builder"))}
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
        targets={sections(session("s2", "builder"))}
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
        targets={sections(session("s2", "builder"), session("s3", "critic"))}
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
        targets={sections(
          session("s2", "builder", "Implement {{payload}}"),
          session("s3", "critic", "Critique {{payload}}"),
        )}
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
        targets={sections(waiting("r1", "Implementer"))}
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
        targets={sections(session("s2", "builder"))}
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
        targets={sections(waiting("r1", "Implementer"))}
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
        targets={sections(session("s2", "builder"), waiting("r1", "Implementer"))}
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
        targets={sections(session("s2", "builder"), waiting("r1", "Implementer"))}
        initialSelection={["r1"]}
        onSend={vi.fn()}
      />,
    );
    expect(within(screen.getByTestId("handoff-target-r1")).getByRole("checkbox")).toBeChecked();
    expect(within(screen.getByTestId("handoff-target-s2")).getByRole("checkbox")).not.toBeChecked();
  });

  it("renders ordered non-empty target sections with stable target ids", () => {
    render(
      <HandoffComposer
        open
        onClose={onClose}
        sourceLabel="planner"
        payload="plan.md"
        targets={[
          { kind: "group", labelKey: "handoff.sections.group", targets: [session("s2", "builder")] },
          { kind: "other", labelKey: "handoff.sections.other", targets: [session("s3", "critic")] },
          { kind: "new", labelKey: "handoff.sections.new", targets: [{ kind: "new-session", groupId: "g1", label: "New session", incomingPrompt: "" }] },
        ]}
        onSend={vi.fn()}
      />,
    );
    expect(screen.getByTestId("handoff-section-group")).toHaveTextContent("handoff.sections.group");
    expect(screen.getByTestId("handoff-section-other")).toHaveTextContent("handoff.sections.other");
    expect(screen.getByTestId("handoff-section-new")).toHaveTextContent("handoff.sections.new");
    expect(screen.getByTestId("handoff-target-s2")).toBeInTheDocument();
    expect(screen.getByTestId("handoff-target-s3")).toBeInTheDocument();
    expect(screen.getByTestId("handoff-target-new-g1")).toBeInTheDocument();
  });

  it("uses edit, selected snippet, role prompt, then payload precedence", async () => {
    render(
      <HandoffComposer
        open
        onClose={onClose}
        sourceLabel="planner"
        payload="plan.md"
        targets={sections(session("s2", "builder", "Role says: {{payload}}"))}
        onSend={vi.fn()}
      />,
    );
    expect(screen.getByTestId("handoff-message")).toHaveValue("Role says: plan.md");
    fireEvent.click(screen.getByTestId("handoff-message-source"));
    fireEvent.click(screen.getByTestId("snippet-row-payload-snippet"));
    await waitFor(() => expect(screen.getByTestId("handoff-message")).toHaveValue("Snippet says: plan.md"));
    fireEvent.change(screen.getByTestId("handoff-message"), { target: { value: "Operator edit" } });
    expect(screen.getByTestId("handoff-message")).toHaveValue("Operator edit");
  });

  it("keeps send disabled until unresolved snippet variables are completed", async () => {
    render(
      <HandoffComposer open onClose={onClose} sourceLabel="planner" payload="" targets={sections(session("s2", "builder"))} onSend={vi.fn()} />,
    );
    fireEvent.click(screen.getByTestId("handoff-message-source"));
    fireEvent.click(screen.getByTestId("snippet-row-name-snippet"));
    expect(screen.getByTestId("handoff-send")).toBeDisabled();
    fireEvent.change(screen.getByTestId("snippet-variable-input-name"), { target: { value: "Ada" } });
    fireEvent.click(screen.getByTestId("snippet-variable-insert"));
    await waitFor(() => expect(screen.getByTestId("handoff-message")).toHaveValue("Hello Ada"));
    expect(screen.getByTestId("handoff-send")).not.toBeDisabled();
  });
});
