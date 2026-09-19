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

const described = (
  id: string,
  label: string,
  meta: NonNullable<HandoffTarget["meta"]>,
): HandoffTarget => ({ kind: "session", sessionId: id, label, incomingPrompt: "", meta });

const manySections = (count: number): HandoffTargetSection[] => [
  {
    kind: "other",
    labelKey: "handoff.sections.other",
    targets: Array.from({ length: count }, (_, i) => session(`s${String(i)}`, `session ${String(i)}`)),
  },
  {
    kind: "new",
    labelKey: "handoff.sections.new",
    targets: [{ kind: "new-session", groupId: null, label: "New session", incomingPrompt: "" }],
  },
];

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
  // The complaint this whole pass answers: four terminals launched the same
  // way all render "/bin/bash", and the row carried nothing else at all.
  it("describes a session by more than its name", () => {
    render(
      <HandoffComposer
        open
        onClose={onClose}
        sourceLabel="planner"
        payload=""
        targets={[{
          kind: "other",
          labelKey: "handoff.sections.other",
          targets: [described("s2", "/bin/bash", {
            color: "#38d9c0",
            groupName: "Backend",
            activityLabel: "2m",
            activityAt: 20,
            unreadCount: 3,
          })],
        }]}
        onSend={vi.fn()}
      />,
    );
    const row = screen.getByTestId("handoff-target-s2");
    expect(row).toHaveTextContent("2m");
    expect(row).toHaveTextContent("Backend");
    expect(within(row).getByLabelText("handoff.unreadAria")).toHaveTextContent("3");
  });

  // Inside "this group" every row shares the one group, so repeating its name
  // on each line is noise rather than information.
  it("names the group only where it distinguishes one row from another", () => {
    const meta = { groupName: "Backend", activityLabel: "2m", activityAt: 20 };
    render(
      <HandoffComposer
        open
        onClose={onClose}
        sourceLabel="planner"
        payload=""
        targets={[
          { kind: "group", labelKey: "handoff.sections.group", targets: [described("s2", "one", meta)] },
          { kind: "other", labelKey: "handoff.sections.other", targets: [described("s3", "two", meta)] },
        ]}
        onSend={vi.fn()}
      />,
    );
    expect(screen.getByTestId("handoff-target-s2")).not.toHaveTextContent("Backend");
    expect(screen.getByTestId("handoff-target-s3")).toHaveTextContent("Backend");
  });

  // The row IS the shared selection control. Nesting it inside a second
  // <label> was invalid markup and reserved a whole empty 44px copy column,
  // which is what made every row stand 84px tall over one line of text.
  it("builds a target row from one label, not a label inside a label", () => {
    render(
      <HandoffComposer open onClose={onClose} sourceLabel="planner" payload="" targets={sections(session("s2", "builder"))} onSend={vi.fn()} />,
    );
    const row = screen.getByTestId("handoff-target-s2");
    expect(row.querySelectorAll("label")).toHaveLength(1);
    expect(row.querySelector("[data-rcl-selection-row]")).not.toBeNull();
    // Still one real checkbox, reflecting the sole target's preselection —
    // the markup fix must not cost the control the row is built around.
    expect(within(row).getByRole("checkbox")).toBeChecked();
  });

  it("holds the filter back until the list is too long to scan", () => {
    const { unmount } = render(
      <HandoffComposer open onClose={onClose} sourceLabel="planner" payload="" targets={manySections(3)} onSend={vi.fn()} />,
    );
    expect(screen.queryByTestId("handoff-filter")).toBeNull();
    unmount();
    render(
      <HandoffComposer open onClose={onClose} sourceLabel="planner" payload="" targets={manySections(9)} onSend={vi.fn()} />,
    );
    expect(screen.getByTestId("handoff-filter")).toBeInTheDocument();
  });

  // "Somewhere new" is the escape hatch FROM a list too long to read, so a
  // query nothing matched must not take it away.
  it("narrows the list without ever filtering away somewhere new", () => {
    render(
      <HandoffComposer open onClose={onClose} sourceLabel="planner" payload="" targets={manySections(9)} onSend={vi.fn()} />,
    );
    fireEvent.change(screen.getByTestId("handoff-filter"), { target: { value: "session 4" } });
    expect(screen.getByTestId("handoff-target-s4")).toBeInTheDocument();
    expect(screen.queryByTestId("handoff-target-s5")).toBeNull();
    expect(screen.getByTestId("handoff-section-new")).toBeInTheDocument();

    fireEvent.change(screen.getByTestId("handoff-filter"), { target: { value: "nothing matches" } });
    expect(screen.getByTestId("handoff-no-matches")).toBeInTheDocument();
    expect(screen.getByTestId("handoff-section-new")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("handoff-filter-clear"));
    expect(screen.getByTestId("handoff-target-s5")).toBeInTheDocument();
  });

  // A real session name wrapped the interpolated title onto two lines and
  // pushed the list off the screen. The source is context, not the heading.
  it("keeps the source out of the heading and states it once below", () => {
    render(
      <HandoffComposer
        open
        onClose={onClose}
        sourceLabel="Session group templates and prompt management"
        payload=""
        targets={sections(session("s2", "builder"))}
        onSend={vi.fn()}
      />,
    );
    expect(screen.getByRole("heading")).toHaveTextContent("handoff.handOff");
    expect(screen.getByRole("heading")).not.toHaveTextContent("Session group templates");
    const band = screen.getByTestId("handoff-composer.subheader");
    expect(within(band).getByTestId("handoff-source")).toBeInTheDocument();
  });

  it("seats the snippet control on the message it rewrites", async () => {
    render(
      <HandoffComposer open onClose={onClose} sourceLabel="planner" payload="plan.md" targets={sections(session("s2", "builder"))} onSend={vi.fn()} />,
    );
    fireEvent.click(screen.getByTestId("handoff-message-source"));
    fireEvent.click(screen.getByTestId("snippet-row-payload-snippet"));
    await waitFor(() => { expect(screen.getByTestId("handoff-message")).toHaveValue("Snippet says: plan.md") });
    // Once chosen, the control names the snippet rather than staying a
    // generic "message source" row three sections away from the text.
    expect(screen.getByTestId("handoff-message-source")).toHaveTextContent("Payload wrapper");
  });

  // `textFor` prefers an operator edit over the snippet, so choosing one
  // after typing used to change nothing at all and read as a dead control.
  it("applies a snippet chosen after the message was already typed into", async () => {
    render(
      <HandoffComposer open onClose={onClose} sourceLabel="planner" payload="plan.md" targets={sections(session("s2", "builder"))} onSend={vi.fn()} />,
    );
    fireEvent.change(screen.getByTestId("handoff-message"), { target: { value: "typed first" } });
    fireEvent.click(screen.getByTestId("handoff-message-source"));
    fireEvent.click(screen.getByTestId("snippet-row-payload-snippet"));
    await waitFor(() => { expect(screen.getByTestId("handoff-message")).toHaveValue("Snippet says: plan.md") });
  });

  it("restores the target's own prompt when the snippet is cleared", async () => {
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
    await waitFor(() => { expect(screen.getByTestId("handoff-message")).toHaveValue("Snippet says: plan.md") });
    fireEvent.click(screen.getByTestId("handoff-message-source-clear"));
    expect(screen.getByTestId("handoff-message")).toHaveValue("Role says: plan.md");
    expect(screen.queryByTestId("handoff-message-source-clear")).toBeNull();
  });

  // Fanning out to three sessions at once should not be something you
  // discover from the results panel after pressing the button.
  it("says how many targets Send will reach", () => {
    render(
      <HandoffComposer
        open
        onClose={onClose}
        sourceLabel="planner"
        payload=""
        targets={sections(session("s2", "one"), session("s3", "two"))}
        onSend={vi.fn()}
      />,
    );
    expect(screen.getByTestId("handoff-send")).toHaveTextContent("handoff.send");
    fireEvent.click(within(screen.getByTestId("handoff-target-s2")).getByRole("checkbox"));
    fireEvent.click(within(screen.getByTestId("handoff-target-s3")).getByRole("checkbox"));
    expect(screen.getByTestId("handoff-send")).toHaveTextContent("handoff.sendTo");
  });
});
