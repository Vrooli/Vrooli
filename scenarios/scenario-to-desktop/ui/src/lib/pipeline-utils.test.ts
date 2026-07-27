/**
 * Tests for pipeline utility functions.
 */

import { describe, it, expect } from "vitest";
import { mapPipelineStatus, resetSessionId } from "./pipeline-utils";
import { StageStatus } from "@vrooli/proto-types/scenario-to-desktop/v1/shared/common_pb";

// ============================================================================
// Pipeline Status Mapping
// ============================================================================

describe("mapPipelineStatus", () => {
  it("maps pending to building", () => {
    expect(mapPipelineStatus(StageStatus.PENDING)).toBe("building");
  });

  it("maps running to building", () => {
    expect(mapPipelineStatus(StageStatus.RUNNING)).toBe("building");
  });

  it("maps completed to ready", () => {
    expect(mapPipelineStatus(StageStatus.COMPLETED)).toBe("ready");
  });

  it("maps failed to failed", () => {
    expect(mapPipelineStatus(StageStatus.FAILED)).toBe("failed");
  });

  it("maps cancelled to failed", () => {
    expect(mapPipelineStatus(StageStatus.CANCELLED)).toBe("failed");
  });

  it("maps unknown status to building", () => {
    expect(mapPipelineStatus(StageStatus.UNSPECIFIED)).toBe("building");
  });
});

// ============================================================================
// Session ID Reset
// ============================================================================

describe("resetSessionId", () => {
  it("does not throw", () => {
    expect(() => {
      resetSessionId();
    }).not.toThrow();
  });
});
