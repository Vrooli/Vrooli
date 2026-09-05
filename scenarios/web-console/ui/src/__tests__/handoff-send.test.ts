import { beforeEach, describe, expect, it, vi } from "vitest";
import { handoffTargetSections, sendHandoff, targetKey, textForTarget, type HandoffTarget } from "../hooks/useHandoff";
import { useWorkspaceStore, type RoleMeta } from "../stores/useWorkspaceStore";

function role(overrides: Partial<RoleMeta> & { id: string; groupId: string }): RoleMeta {
  return {
    label: "Role",
    command: "agent",
    workingDir: "",
    incomingPrompt: "",
    backend: "",
    targetId: "",
    sessionId: null,
    sortOrder: 0,
    ...overrides,
  };
}

function deps(overrides: Partial<Parameters<typeof sendHandoff>[2]> = {}) {
  return {
    submit: vi.fn(() => ({ status: "sent" as const, offset: 1 })),
    launch: vi.fn(async () => "new-session"),
    queueForSession: vi.fn(),
    attachRole: vi.fn(),
    ...overrides,
  };
}

const sessionTarget = (id: string, prompt = ""): HandoffTarget => ({
  kind: "session",
  sessionId: id,
  label: id,
  incomingPrompt: prompt,
});

describe("sendHandoff", () => {
  it("reports sent when the target's terminal accepts the text", async () => {
    const d = deps();
    const results = await sendHandoff([sessionTarget("s1")], () => "hello", d);
    expect(results).toEqual([{ targetId: "s1", label: "s1", status: "sent" }]);
    expect(d.submit).toHaveBeenCalledWith("hello", "bulk_text", "s1");
  });

  // The single most likely defect in the whole feature: a rejection means no
  // terminal handle exists yet, and treating it as success loses the message.
  it("queues rather than dropping when the target has no mounted terminal", async () => {
    const d = deps({ submit: vi.fn(() => ({ status: "rejected" as const, reason: "disposed" as const })) });
    const results = await sendHandoff([sessionTarget("s1")], () => "hello", d);
    expect(results[0]?.status).toBe("queued");
    expect(d.queueForSession).toHaveBeenCalledWith("s1", "hello");
  });

  it("never reports sent for a queued gate result", async () => {
    const d = deps({ submit: vi.fn(() => ({ status: "queued" as const, reason: "not-ready" as const })) });
    const results = await sendHandoff([sessionTarget("s1")], () => "hello", d);
    expect(results[0]?.status).toBe("queued");
    expect(results[0]?.reason).toBe("not-ready");
  });

  it("sends once per target when several are selected", async () => {
    const d = deps();
    const results = await sendHandoff(
      [sessionTarget("s1"), sessionTarget("s2"), sessionTarget("s3")],
      () => "hello",
      d,
    );
    expect(d.submit).toHaveBeenCalledTimes(3);
    expect(results.map((r) => r.targetId)).toEqual(["s1", "s2", "s3"]);
  });

  it("gives each target its own rendered text", async () => {
    const d = deps();
    const targets = [sessionTarget("s1", "Implement {{payload}}"), sessionTarget("s2", "Critique {{payload}}")];
    await sendHandoff(targets, (target) => textForTarget(target, "plan.md"), d);
    expect(d.submit).toHaveBeenNthCalledWith(1, "Implement plan.md", "bulk_text", "s1");
    expect(d.submit).toHaveBeenNthCalledWith(2, "Critique plan.md", "bulk_text", "s2");
  });

  it("starts a waiting role, attaches it, then queues the text", async () => {
    const d = deps();
    const target: HandoffTarget = {
      kind: "role",
      role: role({ id: "r1", groupId: "g1", label: "Implementer", command: "codex --yolo" }),
      label: "Implementer",
      incomingPrompt: "Implement {{payload}}",
    };
    const results = await sendHandoff([target], (tg) => textForTarget(tg, "plan.md"), d);

    expect(d.launch).toHaveBeenCalledTimes(1);
    expect(d.launch).toHaveBeenCalledWith(expect.objectContaining({ command: "codex --yolo" }));
    expect(d.attachRole).toHaveBeenCalledWith("r1", "new-session");
    expect(d.queueForSession).toHaveBeenCalledWith("new-session", "Implement plan.md");
    expect(results[0]?.status).toBe("queued");
  });

  it("reports failed, and queues nothing, when the role's process never starts", async () => {
    const d = deps({ launch: vi.fn(async () => null) });
    const target: HandoffTarget = {
      kind: "role",
      role: role({ id: "r1", groupId: "g1" }),
      label: "Implementer",
      incomingPrompt: "",
    };
    const results = await sendHandoff([target], () => "hello", d);
    expect(results[0]).toMatchObject({ status: "failed", reason: "start-failed" });
    expect(d.queueForSession).not.toHaveBeenCalled();
  });

  // launchSession returns null for a second concurrent call, so a parallel
  // fan-out to three roles would start one. Sequencing is the fix, and this
  // is the test that would catch its removal.
  it("starts fan-out targets sequentially, never concurrently", async () => {
    let inFlight = 0;
    let maxInFlight = 0;
    const d = deps({
      launch: vi.fn(async () => {
        inFlight += 1;
        maxInFlight = Math.max(maxInFlight, inFlight);
        await new Promise((resolve) => setTimeout(resolve, 1));
        inFlight -= 1;
        return `sess-${maxInFlight}`;
      }),
    });
    const targets: HandoffTarget[] = [1, 2, 3].map((i) => ({
      kind: "role",
      role: role({ id: `r${i}`, groupId: "g1" }),
      label: `Role ${i}`,
      incomingPrompt: "",
    }));
    await sendHandoff(targets, () => "hello", d);
    expect(d.launch).toHaveBeenCalledTimes(3);
    expect(maxInFlight).toBe(1);
  });

  it("refuses to send an empty message rather than delivering nothing", async () => {
    const d = deps();
    const results = await sendHandoff([sessionTarget("s1")], () => "   ", d);
    expect(results[0]).toMatchObject({ status: "failed", reason: "empty" });
    expect(d.submit).not.toHaveBeenCalled();
  });

  it("reuses a running target's session and starts no process", async () => {
    const d = deps();
    await sendHandoff([sessionTarget("s1")], () => "hello", d);
    expect(d.launch).not.toHaveBeenCalled();
  });

  it.each([null, "g1"])("passes a new session's nullable group assignment to launch (%s)", async (groupId) => {
    const d = deps();
    const target: HandoffTarget = { kind: "new-session", groupId, label: "New session", incomingPrompt: "" };
    await sendHandoff([target], () => "hello", d);
    expect(d.launch).toHaveBeenCalledWith({ groupId });
  });
});

describe("handoffTargetSections", () => {
  beforeEach(() => {
    useWorkspaceStore.setState({ panes: [], groups: [], roles: [] });
  });

  it("returns useful sections but no group section for an ungrouped session", () => {
    const sections = handoffTargetSections("s1", null);
    expect(sections.map(({ kind }) => kind)).toEqual(["other", "new"]);
    expect(sections.find(({ kind }) => kind === "new")?.targets).toEqual([
      { kind: "new-session", groupId: null, label: "New session", incomingPrompt: "" },
    ]);
  });

  it("lists the other panes in the group and excludes the source", () => {
    useWorkspaceStore.setState({
      panes: [
        { sessionId: "s1", name: "planner", groupId: "g1" },
        { sessionId: "s2", name: "builder", groupId: "g1" },
        { sessionId: "s3", name: "elsewhere", groupId: "g2" },
      ] as never,
    });
    const targets = handoffTargetSections("s1", "g1").find(({ kind }) => kind === "group")?.targets ?? [];
    expect(targets.map(targetKey)).toEqual(["s2"]);
  });

  it("includes waiting roles as targets", () => {
    useWorkspaceStore.setState({
      panes: [{ sessionId: "s1", name: "planner", groupId: "g1" }] as never,
      roles: [role({ id: "r1", groupId: "g1", label: "Implementer", incomingPrompt: "Do {{payload}}" })],
    });
    const targets = handoffTargetSections("s1", "g1").find(({ kind }) => kind === "group")?.targets ?? [];
    expect(targets).toHaveLength(1);
    expect(targets[0]?.kind).toBe("role");
    expect(targets[0]?.incomingPrompt).toBe("Do {{payload}}");
  });

  // A running role has a pane. Listing it from both sides would show the same
  // agent twice and send it the message twice.
  it("lists a running role once, carrying its own incoming prompt", () => {
    useWorkspaceStore.setState({
      panes: [
        { sessionId: "s1", name: "planner", groupId: "g1" },
        { sessionId: "s2", name: "terminal", groupId: "g1" },
      ] as never,
      roles: [role({ id: "r1", groupId: "g1", label: "Implementer", sessionId: "s2", incomingPrompt: "Do {{payload}}" })],
    });
    const targets = handoffTargetSections("s1", "g1").find(({ kind }) => kind === "group")?.targets ?? [];
    expect(targets).toHaveLength(1);
    expect(targets[0]?.label).toBe("Implementer");
    expect(targets[0]?.incomingPrompt).toBe("Do {{payload}}");
  });

  it("puts members of other groups in the other section", () => {
    useWorkspaceStore.setState({
      panes: [
        { sessionId: "s1", name: "a", groupId: "g1" },
        { sessionId: "s2", name: "elsewhere", groupId: "g2" },
      ] as never,
    });
    const sections = handoffTargetSections("s1", "g1");
    expect(sections.find(({ kind }) => kind === "group")).toBeUndefined();
    expect(sections.find(({ kind }) => kind === "other")?.targets.map(targetKey)).toEqual(["s2"]);
  });

  it("keeps group content and order while excluding those panes from other", () => {
    useWorkspaceStore.setState({
      panes: [
        { sessionId: "s1", name: "source", groupId: "g1" },
        { sessionId: "s2", name: "group pane", groupId: "g1" },
        { sessionId: "s3", name: "other pane", groupId: null },
      ] as never,
      roles: [role({ id: "r1", groupId: "g1", label: "Waiting role" })],
    });
    const sections = handoffTargetSections("s1", "g1");
    expect(sections.find(({ kind }) => kind === "group")?.targets.map(targetKey)).toEqual(["s2", "r1"]);
    expect(sections.find(({ kind }) => kind === "other")?.targets.map(targetKey)).toEqual(["s3"]);
  });
});
