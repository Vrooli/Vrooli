/**
 * Tests for pipeline utility functions.
 */

import { describe, it, expect } from "vitest";
import {
  mapPipelineStatus,
  resetSessionId,
} from "./pipeline-utils";

// ============================================================================
// Pipeline Status Mapping
// ============================================================================

describe("mapPipelineStatus", () => {
  it("maps pending to building", () => {
    expect(mapPipelineStatus("pending")).toBe("building");
  });

  it("maps running to building", () => {
    expect(mapPipelineStatus("running")).toBe("building");
  });

  it("maps completed to ready", () => {
    expect(mapPipelineStatus("completed")).toBe("ready");
  });

  it("maps failed to failed", () => {
    expect(mapPipelineStatus("failed")).toBe("failed");
  });

  it("maps cancelled to failed", () => {
    expect(mapPipelineStatus("cancelled")).toBe("failed");
  });

  it("maps unknown status to building", () => {
    expect(mapPipelineStatus("unknown")).toBe("building");
    expect(mapPipelineStatus("")).toBe("building");
  });
});

// ============================================================================
// Session ID Reset
// ============================================================================

describe("resetSessionId", () => {
  it("does not throw", () => {
    expect(() => resetSessionId()).not.toThrow();
  });
});
