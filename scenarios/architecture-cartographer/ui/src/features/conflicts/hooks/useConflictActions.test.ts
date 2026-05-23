import { renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { useConflictActions } from "./useConflictActions";
import { makeConflict } from "../flow/fixtures";
import { ResolutionStatus } from "@vrooli/proto-types/architecture-cartographer/v1/conflicts/conflicts_pb";

describe("useConflictActions", () => {
  it("falls back to 'detected' when no conflict is provided", () => {
    const { result } = renderHook(() => useConflictActions(undefined));
    expect(result.current.state).toBe("detected");
    expect(result.current.legalEvents).toEqual(
      expect.arrayContaining(["assign", "split", "resolve", "force_resolve"]),
    );
  });

  it("derives the state from the proto ResolutionStatus", () => {
    const conflict = makeConflict({ status: ResolutionStatus.RESOLVED });
    const { result } = renderHook(() => useConflictActions(conflict));
    expect(result.current.state).toBe("resolved");
    expect(result.current.legalEvents).toEqual(expect.arrayContaining(["validate", "reopen"]));
  });

  it("returns no legal events for a terminal conflict", () => {
    const conflict = makeConflict({ status: ResolutionStatus.COMMITTED });
    const { result } = renderHook(() => useConflictActions(conflict));
    expect(result.current.state).toBe("committed");
    expect(result.current.legalEvents).toEqual([]);
  });

  it("returns only 'reopen' for a force-resolved conflict", () => {
    const conflict = makeConflict({ status: ResolutionStatus.FORCE_RESOLVED });
    const { result } = renderHook(() => useConflictActions(conflict));
    expect(result.current.legalEvents).toEqual(["reopen"]);
  });
});
