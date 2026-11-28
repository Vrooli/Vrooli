import { afterEach, describe, expect, it, vi } from "vitest";
import { act } from "@testing-library/react";
import { createPipelineStatus } from "../test-utils/mocks";
import { usePipelineStore } from "./pipelineStore";
import { createDeferred } from "./pipelineStore.testUtils";

const apiMocks = vi.hoisted(() => ({
  getActivePipeline: vi.fn(),
  getPipelineStatus: vi.fn(),
}));

vi.mock("../lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../lib/api")>();
  return {
    ...actual,
    getActivePipeline: apiMocks.getActivePipeline,
    getPipelineStatus: apiMocks.getPipelineStatus,
  };
});

describe("pipeline store scenario switching", () => {
  afterEach(() => {
    act(() => usePipelineStore.getState().reset());
    vi.clearAllMocks();
  });

  it("loads the latest rapid selection and ignores late responses from earlier selections", async () => {
    const loads = {
      A: createDeferred<{
        pipeline: { pipelineId: string } | null;
        created: boolean;
      }>(),
      B: createDeferred<{
        pipeline: { pipelineId: string } | null;
        created: boolean;
      }>(),
      C: createDeferred<{
        pipeline: { pipelineId: string } | null;
        created: boolean;
      }>(),
    };
    apiMocks.getActivePipeline.mockImplementation(
      (scenario: "A" | "B" | "C") => loads[scenario].promise,
    );
    apiMocks.getPipelineStatus.mockImplementation((pipelineId: string) =>
      Promise.resolve(createPipelineStatus({ pipelineId })),
    );

    try {
      act(() => {
        usePipelineStore.getState().setScenario("A");
        usePipelineStore.getState().setScenario("B");
        usePipelineStore.getState().setScenario("C");
      });

      expect(
        apiMocks.getActivePipeline.mock.calls.map(([scenario]) => scenario),
      ).toEqual(["A", "B", "C"]);

      loads.A.resolve({
        pipeline: { pipelineId: "pipeline-A" },
        created: false,
      });
      loads.B.resolve({
        pipeline: { pipelineId: "pipeline-B" },
        created: false,
      });
      await Promise.resolve();
      await Promise.resolve();

      expect(usePipelineStore.getState()).toMatchObject({
        scenarioName: "C",
        isLoadingActivePipeline: true,
        pipelineId: null,
        pipelineStatus: null,
      });

      loads.C.resolve({
        pipeline: { pipelineId: "pipeline-C" },
        created: false,
      });

      await vi.waitFor(() => {
        expect(usePipelineStore.getState()).toMatchObject({
          scenarioName: "C",
          isLoadingActivePipeline: false,
          pipelineId: "pipeline-C",
          pipelineStatus: expect.objectContaining({ pipelineId: "pipeline-C" }),
        });
      });
    } finally {
      // Settle any requests left pending when an assertion fails on the baseline.
      loads.A.resolve({
        pipeline: { pipelineId: "pipeline-A" },
        created: false,
      });
      loads.B.resolve({
        pipeline: { pipelineId: "pipeline-B" },
        created: false,
      });
      loads.C.resolve({ pipeline: null, created: false });
      await Promise.resolve();
    }
  });
});
