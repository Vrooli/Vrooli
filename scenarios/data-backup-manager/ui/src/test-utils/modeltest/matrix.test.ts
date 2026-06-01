import { describe, expect, it } from "vitest";

import { assertTransitionMatrix, type MatrixRow, validateTransitionMatrix } from "./matrix";

const states = ["idle", "busy", "done"] as const;
const events = ["start", "finish"] as const;

type State = (typeof states)[number];
type Event = (typeof events)[number];

const transition = (state: State, event: Event): State => {
  if (state === "idle" && event === "start") {
    return "busy";
  }
  if (state === "busy" && event === "finish") {
    return "done";
  }
  throw new Error("illegal transition");
};

const rows = [
  { name: "idle_start", from: "idle", event: "start", to: "busy" },
  { name: "idle_finish", from: "idle", event: "finish", to: "idle", wantError: true },
  { name: "busy_start", from: "busy", event: "start", to: "busy", wantError: true },
  { name: "busy_finish", from: "busy", event: "finish", to: "done" },
  { name: "done_start", from: "done", event: "start", to: "done", wantError: true },
  { name: "done_finish", from: "done", event: "finish", to: "done", wantError: true },
] as const satisfies readonly MatrixRow<State, Event>[];

describe("modeltest transition matrices", () => {
  it("accepts a complete matrix", () => {
    expect(validateTransitionMatrix(states, events, rows, transition)).toEqual([]);
    expect(() => assertTransitionMatrix(states, events, rows, transition)).not.toThrow();
  });

  it("rejects structural drift", () => {
    const badRows = [
      rows[0],
      ...rows.slice(2),
      { name: "unknown", from: "ghost", event: "start", to: "busy" },
      { name: "duplicate", from: "idle", event: "start", to: "busy" },
    ];

    const errors = Reflect.apply(validateTransitionMatrix, undefined, [
      states,
      events,
      badRows,
      transition,
    ]);

    expect(errors.join("\n")).toContain("unknown: unknown from state ghost");
    expect(errors.join("\n")).toContain("duplicate: duplicate pair idle/start");
    expect(errors.join("\n")).toContain("missing pair idle/finish");
  });

  it("rejects behavior drift", () => {
    const badRows = [
      { ...rows[0], to: "done" },
      { ...rows[1], wantError: false },
      ...rows.slice(2),
    ] as const satisfies readonly MatrixRow<State, Event>[];

    const errors = validateTransitionMatrix(states, events, badRows, transition);

    expect(errors.join("\n")).toContain("idle_start: got state busy, want done");
    expect(errors.join("\n")).toContain("idle_finish: unexpected error");
  });
});
