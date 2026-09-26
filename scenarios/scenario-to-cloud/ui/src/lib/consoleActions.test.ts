import { describe, expect, it } from "vitest";
import {
  applyActionFor,
  cancelActionFor,
  classifyNextAction,
  compareRelease,
  describeDenial,
  formatAge,
  isTerminalState,
  nextActionLabel,
  requiredScopeFor,
  shortDigest,
  targetKey,
} from "./consoleActions";
import { apiErrorOf, makeMatrix, makeStanding } from "../test-utils/consoleFixtures";

describe("consoleActions", () => {
  it("derives the target key from the binding, never from the manifest", () => {
    expect(targetKey({ machine_id: "m-1", locator: { host: "h" } })).toBe("machine:m-1");
    expect(targetKey({ locator: { host: "203.0.113.10" } })).toBe("host:203.0.113.10");
    expect(targetKey(undefined)).toBe("unbound");
    expect(targetKey({})).toBe("unbound");
  });

  it("shortens digests and formats ages", () => {
    expect(shortDigest("sha256:abcdef0123456789")).toBe("abcdef012345");
    expect(shortDigest("abc")).toBe("abc");
    expect(shortDigest("")).toBe("");
    const now = Date.parse("2026-09-09T12:00:00Z");
    expect(formatAge("2026-09-09T11:59:30Z", now)).toBe("30s ago");
    expect(formatAge("2026-09-09T11:30:00Z", now)).toBe("30m ago");
    expect(formatAge("2026-09-09T02:00:00Z", now)).toBe("10h ago");
    expect(formatAge("2026-09-01T12:00:00Z", now)).toBe("8d ago");
    expect(formatAge("not a date", now)).toBe("");
    expect(formatAge(undefined, now)).toBe("");
  });

  it("knows the terminal states", () => {
    expect(isTerminalState("succeeded")).toBe(true);
    expect(isTerminalState("failed_recovery")).toBe(true);
    expect(isTerminalState("running")).toBe(false);
    expect(isTerminalState(undefined)).toBe(false);
  });

  it("finds the scope of a route in the authz matrix, including prefix and templated paths", () => {
    const matrix = makeMatrix();
    expect(requiredScopeFor(matrix, "POST", "/api/v1/deployments/dep-1/plan/apply")).toBe("scenario-to-cloud:destructive");
    expect(requiredScopeFor(matrix, "POST", "/api/v1/deployments/dep-1/plan")).toBe("scenario-to-cloud:read");
    expect(requiredScopeFor(matrix, "GET", "/api/v1/operations/op-1")).toBe("scenario-to-cloud:read");
    expect(requiredScopeFor(matrix, "DELETE", "/api/v1/deployments/dep-1")).toBeUndefined();
    expect(requiredScopeFor(null, "GET", "/x")).toBeUndefined();
  });

  // [REQ:STC-P0-039] A denial names the specific missing permission.
  it("describes denials with the missing scope from details or the matrix", () => {
    const matrix = makeMatrix();
    const scoped = describeDenial(apiErrorOf(403, "forbidden_scope", "denied", { details: { required_scope: "scenario-to-cloud:destructive" } }));
    expect(scoped?.missingScope).toBe("scenario-to-cloud:destructive");
    expect(scoped?.explanation).toContain("scenario-to-cloud:destructive");

    const fromMatrix = describeDenial(apiErrorOf(403, "forbidden_scope", "denied"), { matrix, method: "POST", path: "/api/v1/deployments/dep-1/plan/apply" });
    expect(fromMatrix?.missingScope).toBe("scenario-to-cloud:destructive");

    const target = describeDenial(apiErrorOf(403, "forbidden_target", "denied"), { matrix, method: "POST", path: "/api/v1/deployments/dep-1/plan/apply", target: "machine:m-1" });
    expect(target?.title).toBe("Target not granted");
    expect(target?.target).toBe("machine:m-1");
    expect(target?.explanation).toContain("machine:m-1");

    const unauth = describeDenial(apiErrorOf(401, "unauthenticated", "no identity", { next_action: { owner: "operator", kind: "sign_in", reference: "https://example/sign-in", label: "Sign in" } }));
    expect(unauth?.title).toBe("Sign in required");
    expect(unauth?.nextAction?.kind).toBe("sign_in");

    expect(describeDenial(apiErrorOf(403, "forbidden_origin", "origin"))?.title).toBe("Origin refused");
    expect(describeDenial(apiErrorOf(409, "plan_stale", "stale"))).toBeNull();
    expect(describeDenial(new Error("plain"))).toBeNull();
  });

  it("classifies and labels next actions", () => {
    expect(classifyNextAction({ kind: "wait" })).toBe("wait");
    expect(classifyNextAction({ kind: "mystery" })).toBe("other");
    expect(classifyNextAction(null)).toBe("other");
    expect(nextActionLabel({ kind: "wait" })).toBe("Keep waiting for the operation");
    expect(nextActionLabel({ kind: "resume" })).toBe("Resume the operation");
    expect(nextActionLabel({ kind: "reconcile" })).toBe("Reconcile the operation owner");
    expect(nextActionLabel({ kind: "recovery" })).toBe("Open recovery");
    expect(nextActionLabel({ kind: "operation" })).toBe("Inspect the operation");
    expect(nextActionLabel({ kind: "resume_handoff" })).toBe("Finish setup in onboarding");
    expect(nextActionLabel({ kind: "replan" })).toBe("Review the plan again");
    expect(nextActionLabel({ kind: "sign_in" })).toBe("Sign in and retry");
    expect(nextActionLabel({ kind: "custom", label: "Do the thing" })).toBe("Do the thing");
    expect(nextActionLabel({ kind: "custom", reference: "docs/x.md" })).toBe("See docs/x.md");
    expect(nextActionLabel(null)).toBe("No next action provided");
  });

  // [REQ:STC-P0-038] Availability is read from the plan outcome only.
  it("derives the apply action from the plan outcome", () => {
    expect(applyActionFor("apply").available).toBe(true);
    expect(applyActionFor("no_op")).toMatchObject({ available: false, reason: { code: "no_op" } });
    expect(applyActionFor("needs_input")).toMatchObject({ available: false, reason: { code: "needs_input" } });
    expect(applyActionFor(undefined)).toMatchObject({ available: false, reason: { code: "plan_unavailable" } });
  });

  it("derives the cancel action from the standing", () => {
    expect(cancelActionFor(null).reason?.code).toBe("no_operation");
    expect(cancelActionFor(makeStanding({ state: "succeeded", terminal: true })).reason?.code).toBe("terminal");
    expect(cancelActionFor(makeStanding({ cancel_requested: true })).reason?.code).toBe("cancel_requested");
    expect(cancelActionFor(makeStanding({ state: "cancel_requested" })).reason?.code).toBe("cancel_requested");
    expect(cancelActionFor(makeStanding(), "scenario-to-cloud:destructive")).toMatchObject({ available: true, requiredScope: "scenario-to-cloud:destructive" });
  });

  it("compares desired and observed releases without guessing", () => {
    expect(compareRelease("a", "a")).toBe("match");
    expect(compareRelease("a", "b")).toBe("mismatch");
    expect(compareRelease(undefined, "b")).toBe("unknown");
    expect(compareRelease("a", "")).toBe("unknown");
  });
});
