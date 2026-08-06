import { describe, expect, it } from "vitest";

import type { WorkflowTransition } from "./matrix";
import { replayTraces, type Trace, validateTraces } from "./traces";

type State = "idle" | "busy" | "done";
type Event = "start" | "finish";

const transition: WorkflowTransition<State, Event> = (state, event) => {
  if (state === "idle" && event === "start") {
    return "busy";
  }
  if (state === "busy" && event === "finish") {
    return "done";
  }
  throw new Error("illegal transition");
};

describe("modeltest trace replay", () => {
  it("accepts matching traces", () => {
    const traces = [
      {
        name: "happy_path",
        initial: "idle",
        steps: [
          { name: "start", event: "start", want: "busy" },
          { name: "finish", event: "finish", want: "done" },
        ],
      },
      {
        name: "reject_finish_before_start",
        initial: "idle",
        steps: [{ name: "finish", event: "finish", want: "idle", wantError: true }],
      },
    ] as const satisfies readonly Trace<State, Event>[];

    expect(validateTraces(traces, transition)).toEqual([]);
    expect(() => replayTraces(traces, transition)).not.toThrow();
  });

  it("rejects drift", () => {
    const traces = [
      {
        name: "bad_expectations",
        initial: "idle",
        steps: [
          { name: "start", event: "start", want: "done" },
          { name: "start_again", event: "start", want: "done" },
        ],
      },
    ] as const satisfies readonly Trace<State, Event>[];

    const errors = validateTraces(traces, transition);

    expect(errors.join("\n")).toContain("bad_expectations/start: got state busy, want done");
    expect(errors.join("\n")).toContain("bad_expectations/start_again: unexpected error");
  });
});
