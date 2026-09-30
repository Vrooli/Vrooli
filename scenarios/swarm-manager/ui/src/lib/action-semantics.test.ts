/**
 * The classification behind "is this button about to spawn an agent?".
 *
 * The rules that matter are the safety-shaped ones: an unknown action must not
 * be described as harmless, and a destructive action must stay destructive
 * whatever the registry says.
 */

import { describe, expect, it } from "vitest";
import { TransitionKind } from "@vrooli/proto-types/swarm-manager/v1/domain/transition_pb";
import {
  CONSEQUENCE_META,
  consequenceOf,
  consequenceOfEffect,
  consequenceOfTransitionKind,
  spawnsAgent,
  successKindFor,
} from "./action-semantics";

describe("consequenceOfTransitionKind", () => {
  it("maps the server's declared kinds", () => {
    expect(consequenceOfTransitionKind(TransitionKind.WORKFLOW)).toBe("agent_workflow");
    expect(consequenceOfTransitionKind(TransitionKind.SESSION)).toBe("agent_session");
    expect(consequenceOfTransitionKind(TransitionKind.DETERMINISTIC)).toBe("state_change");
  });

  it("returns undefined for unspecified or missing kinds", () => {
    expect(consequenceOfTransitionKind(TransitionKind.UNSPECIFIED)).toBeUndefined();
    expect(consequenceOfTransitionKind(undefined)).toBeUndefined();
  });
});

describe("consequenceOfEffect", () => {
  it("maps the feed's declared effects", () => {
    expect(consequenceOfEffect("agent_run")).toBe("agent_workflow");
    expect(consequenceOfEffect("agent_session")).toBe("agent_session");
    expect(consequenceOfEffect("state_change")).toBe("state_change");
    expect(consequenceOfEffect("none")).toBe("navigation");
    expect(consequenceOfEffect(undefined)).toBeUndefined();
  });
});

describe("consequenceOf", () => {
  it("prefers the feed's declared effect over a resolved transition kind", () => {
    // The feed knows the action; the registry only knows a transition the
    // caller happened to resolve.
    expect(consequenceOf({ effect: "agent_run", transitionKind: TransitionKind.DETERMINISTIC }))
      .toBe("agent_workflow");
  });

  it("falls back to the transition kind for controls with no feed entry", () => {
    // Plan goal / Discover / Classify are wired straight to a transition.
    expect(consequenceOf({ transitionKind: TransitionKind.WORKFLOW })).toBe("agent_workflow");
  });

  it("keeps destructive destructive whatever else is declared", () => {
    expect(consequenceOf({ destructive: true })).toBe("destructive");
    expect(consequenceOf({ destructive: true, effect: "agent_run" })).toBe("destructive");
    expect(consequenceOf({ destructive: true, transitionKind: TransitionKind.WORKFLOW })).toBe("destructive");
  });

  it("classifies agent actions whose transition subject the client never holds", () => {
    // `run`, `retry` and `review` have an empty transition_key by design —
    // plan.execute takes an execution id, work.review a review round. The
    // server's effect is the only way to know an agent is about to run.
    for (const actionId of ["run", "retry", "review"]) {
      expect(consequenceOf({ actionId, effect: "agent_run" }), actionId).toBe("agent_workflow");
    }
  });

  it("does not classify from the action id", () => {
    // The removed fallback list was both redundant and wrong. An id alone
    // must now tell us nothing.
    expect(consequenceOf({ actionId: "review" })).toBe("state_change");
    expect(consequenceOf({ actionId: "archive" })).toBe("state_change");
    expect(consequenceOf({ actionId: "dispatch_followup" })).toBe("state_change");
  });

  it("never calls an unclassifiable action harmless", () => {
    // Under-promising a side effect is safer than implying there is none, so
    // the fallback is state_change rather than navigation.
    expect(consequenceOf({})).toBe("state_change");
    expect(consequenceOf({ actionId: "some_future_action" })).toBe("state_change");
  });
});

describe("spawnsAgent", () => {
  it("is true only for the agent classes", () => {
    expect(spawnsAgent({ effect: "agent_run" })).toBe(true);
    expect(spawnsAgent({ effect: "agent_session" })).toBe(true);
    expect(spawnsAgent({ transitionKind: TransitionKind.WORKFLOW })).toBe(true);
    expect(spawnsAgent({ effect: "state_change" })).toBe(false);
    expect(spawnsAgent({ effect: "none" })).toBe(false);
    expect(spawnsAgent({ destructive: true })).toBe(false);
  });
});

describe("successKindFor", () => {
  it("reports agent work as started, not finished", () => {
    // Dispatching a workflow returns when the run is queued. Saying "done"
    // would be a lie the operator acts on.
    expect(successKindFor({ transitionKind: TransitionKind.WORKFLOW })).toBe("progress");
    expect(successKindFor({ effect: "agent_run" })).toBe("progress");
  });

  it("reports immediate changes as done", () => {
    expect(successKindFor({ transitionKind: TransitionKind.DETERMINISTIC })).toBe("success");
    expect(successKindFor({ effect: "state_change" })).toBe("success");
  });
});

describe("CONSEQUENCE_META", () => {
  it("gives every class a hint and consistent flags", () => {
    for (const [name, meta] of Object.entries(CONSEQUENCE_META)) {
      expect(meta.hint, name).toBeTruthy();
      expect(meta.hint.endsWith("."), `${name} hint should not end with a period`).toBe(false);
    }
    expect(CONSEQUENCE_META.destructive.confirms).toBe(true);
    expect(CONSEQUENCE_META.agent_workflow.confirms).toBe(false);
  });
});
