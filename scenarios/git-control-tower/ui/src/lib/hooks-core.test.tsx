import { act, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useStageFiles } from "./hooks-core";
import { queryKeys } from "./hooks-query-keys";
import { renderHookWithQueryClient } from "../test-utils";

const mockStageFiles = vi.fn();

vi.mock("./api", () => ({
  stageFiles: (...args: unknown[]) => mockStageFiles(...args),
}));

describe("core repo hooks", () => {
  beforeEach(() => {
    mockStageFiles.mockResolvedValue({ success: true, staged: ["src/App.tsx"], timestamp: "t" });
  });

  it("useStageFiles forwards repo context and invalidates repo status", async () => {
    const { result, queryClient } = renderHookWithQueryClient(() => useStageFiles("repo-1"));
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");

    await act(async () => {
      await result.current.mutateAsync({ paths: ["src/App.tsx"] });
    });

    await waitFor(() => {
      expect(mockStageFiles).toHaveBeenCalledWith({ paths: ["src/App.tsx"] }, "repo-1");
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: queryKeys.repoStatus("repo-1") });
    });
  });
});
