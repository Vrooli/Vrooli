import { describe, it, expect } from "vitest";
import { renderHook } from "@testing-library/react";
import { withExpectedReactHookError } from "../test-utils";
import { BacklogDetailProvider, useBacklogDetail, type BacklogDetailContextValue } from "./BacklogDetailContext";

const mockValue: BacklogDetailContextValue = {
  backlogKind: "idea",
  name: "test-item",
  item: undefined,
  itemActions: null,
  isLocked: false,
  isTerminal: false,
  agentRunIsActive: false,
  latestAgentActivity: null,
  agentRunningLabel: "Agent running\u2026",
};

describe("BacklogDetailContext", () => {
  it("provides values to children", () => {
    const { result } = renderHook(() => useBacklogDetail(), {
      wrapper: ({ children }) => (
        <BacklogDetailProvider value={mockValue}>{children}</BacklogDetailProvider>
      ),
    });

    expect(result.current.backlogKind).toBe("idea");
    expect(result.current.name).toBe("test-item");
  });

  it("throws when used outside provider", async () => {
    await withExpectedReactHookError("useBacklogDetail must be used within a BacklogDetailProvider", () => {
      expect(() => {
        renderHook(() => useBacklogDetail());
      }).toThrow("useBacklogDetail must be used within a BacklogDetailProvider");
    });
  });
});
