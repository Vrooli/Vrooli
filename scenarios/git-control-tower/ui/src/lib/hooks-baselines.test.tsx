import { act, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useCompareOnDemand } from "./hooks-baselines";
import { renderHookWithQueryClient } from "../test-utils";

const diffBaseline = vi.fn();

vi.mock("./api-baselines", () => ({
  diffBaseline: (...a: unknown[]) => diffBaseline(...a),
}));

describe("useCompareOnDemand", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    diffBaseline.mockResolvedValue({ verdict: "clean", phases: [] });
  });

  it("does not diff until start() and clears on exit()", async () => {
    const { result } = renderHookWithQueryClient(() =>
      useCompareOnDemand("s", { baselineName: "b" }),
    );

    // Idle: no server diff.
    expect(result.current.comparing).toBe(false);
    expect(diffBaseline).not.toHaveBeenCalled();

    act(() => result.current.start());
    expect(result.current.comparing).toBe(true);

    await waitFor(() => expect(diffBaseline).toHaveBeenCalledTimes(1));
    await waitFor(() => expect(result.current.diff).toBeDefined());

    act(() => result.current.exit());
    expect(result.current.comparing).toBe(false);
  });

  it("never diffs without a resolved baseline", () => {
    const { result } = renderHookWithQueryClient(() =>
      useCompareOnDemand("s", { baselineName: "" }),
    );
    act(() => result.current.start());
    expect(diffBaseline).not.toHaveBeenCalled();
    expect(result.current.baselineName).toBe("");
  });
});
