import { beforeEach, describe, expect, it } from "vitest";
import { useSessionActivityStore } from "./useSessionActivityStore";
import type { SessionActivityView } from "../api/sessionActivity";

function activity(overrides: Partial<SessionActivityView> = {}): SessionActivityView {
  return {
    state: "working",
    source: "screen",
    confidence: 0.85,
    since: "2026-09-11T08:00:00Z",
    harness: "claude",
    ...overrides,
  };
}

describe("useSessionActivityStore", () => {
  beforeEach(() => {
    useSessionActivityStore.setState({ activities: {} });
  });

  it("[REQ:P0-017e] stores the latest activity per session", () => {
    const { apply } = useSessionActivityStore.getState();
    apply("s1", activity());
    apply("s2", activity({ state: "idle" }));
    expect(useSessionActivityStore.getState().activities.s1?.state).toBe("working");
    expect(useSessionActivityStore.getState().activities.s2?.state).toBe("idle");
  });

  it("[REQ:P0-017e] ignores an activity whose state began before the stored one", () => {
    const { apply } = useSessionActivityStore.getState();
    apply("s1", activity({ state: "waiting", since: "2026-09-11T08:00:10Z" }));
    apply("s1", activity({ state: "working", since: "2026-09-11T08:00:05Z" }));
    expect(useSessionActivityStore.getState().activities.s1?.state).toBe("waiting");
  });

  it("[REQ:P0-017e] takes a heartbeat for the same state (same since, newer output)", () => {
    const { apply } = useSessionActivityStore.getState();
    apply("s1", activity({ lastOutputAt: "2026-09-11T08:00:01Z" }));
    apply("s1", activity({ lastOutputAt: "2026-09-11T08:00:16Z" }));
    expect(useSessionActivityStore.getState().activities.s1?.lastOutputAt).toBe("2026-09-11T08:00:16Z");
  });

  it("[REQ:P0-017e] hydrates from the sessions list without overriding newer pushes", () => {
    const { apply, hydrate } = useSessionActivityStore.getState();
    apply("s1", activity({ state: "waiting", since: "2026-09-11T08:00:10Z" }));
    hydrate([
      { id: "s1", activity: activity({ state: "idle", since: "2026-09-11T08:00:00Z" }) },
      { id: "s2", activity: activity({ state: "idle" }) },
      { id: "s3" },
    ]);
    const { activities } = useSessionActivityStore.getState();
    expect(activities.s1?.state).toBe("waiting");
    expect(activities.s2?.state).toBe("idle");
    expect(activities.s3).toBeUndefined();
  });

  it("[REQ:P0-017e] forgets a session", () => {
    const { apply, forget } = useSessionActivityStore.getState();
    apply("s1", activity());
    forget("s1");
    expect(useSessionActivityStore.getState().activities.s1).toBeUndefined();
  });
});
