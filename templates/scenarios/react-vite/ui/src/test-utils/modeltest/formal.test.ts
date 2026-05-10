import { describe, expect, it } from "vitest";

import {
  validateFormalArtifactFresh,
  validateFormalTransitionsReplay,
  validateFormalTracesReplay,
  type FormalArtifact,
} from "./formal";

type Status = "idle" | "busy" | "done";
type Event = "start" | "finish";

const transition = (status: Status, event: Event): Status => {
  if (status === "idle" && event === "start") {
    return "busy";
  }
  if (status === "busy" && event === "finish") {
    return "done";
  }
  throw new Error("illegal transition");
};

const statuses = ["idle", "busy", "done"] as const;
const events = ["start", "finish"] as const;
const modelPath = "ui/src/features/example/model.qnt";
const modelSha256 = "a".repeat(64);

describe("formal modeltest helpers", () => {
  it("accepts a fresh artifact", () => {
    expect(validateFormalArtifactFresh(validArtifact(), { modelPath, modelSha256 })).toEqual([]);
  });

  it("rejects stale hashes and missing checks", () => {
    const artifact = {
      ...validArtifact(),
      source: { ...validArtifact().source, modelSha256: "0".repeat(64) },
      checks: { ...validArtifact().checks, verified: false },
    };

    const errors = validateFormalArtifactFresh(artifact, { modelPath, modelSha256 });
    expect(errors.join("\n")).toContain("formal artifact modelSha256=");
    expect(errors.join("\n")).toContain("formal artifact was not verified");
  });

  it("replays generated transitions", () => {
    const artifact = validArtifact();
    expect(validateFormalTransitionsReplay(artifact, statuses, events, transition)).toEqual([]);
  });

  it("rejects unknown and divergent transitions", () => {
    const artifact = validArtifact();
    const transitions = artifact.transitions.map((transitionRow, index) => {
      if (index === 0) {
        return { ...transitionRow, to: "done" };
      }
      if (index === 1) {
        return { ...transitionRow, event: "ghost" };
      }
      return transitionRow;
    });

    const errors = validateFormalTransitionsReplay({ ...artifact, transitions }, statuses, events, transition);
    expect(errors.join("\n")).toContain("unknown event ghost");
  });

  it("rejects unknown and divergent traces", () => {
    const artifact = validArtifact();
    const traces = [
      {
        name: artifact.traces[0]?.name ?? "",
        initial: artifact.traces[0]?.initial ?? "",
        steps: (artifact.traces[0]?.steps ?? []).map((step, index) => {
          if (index === 0) {
            return { ...step, want: "done" };
          }
          if (index === 1) {
            return { ...step, event: "ghost" };
          }
          return step;
        }),
      },
    ];

    const errors = validateFormalTracesReplay({ ...artifact, traces }, statuses, events, transition);
    expect(errors.join("\n")).toContain("unknown event ghost");
  });
});

const validArtifact = (): FormalArtifact => ({
  schemaVersion: 1,
  flowId: "example.flow",
  source: {
    modelPath,
    modelSha256,
    quintVersion: "0.32.0",
  },
  commands: {
    typecheck: ["quint", "typecheck", modelPath],
    test: ["quint", "test", modelPath],
    verify: ["quint", "verify", modelPath],
    run: ["quint", "run", modelPath],
  },
  states: ["idle", "busy", "done"],
  events: ["start", "finish"],
  transitions: [
    { from: "idle", event: "start", to: "busy", wantError: false },
    { from: "idle", event: "finish", to: "idle", wantError: true },
    { from: "busy", event: "start", to: "busy", wantError: true },
    { from: "busy", event: "finish", to: "done", wantError: false },
    { from: "done", event: "start", to: "done", wantError: true },
    { from: "done", event: "finish", to: "done", wantError: true },
  ],
  traces: [
    {
      name: "generated",
      initial: "idle",
      steps: [
        { event: "start", want: "busy", wantError: false },
        { event: "finish", want: "done", wantError: false },
      ],
    },
  ],
  checks: {
    typechecked: true,
    tested: true,
    verified: true,
    generatedFromModel: true,
  },
});
