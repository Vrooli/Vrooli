import { describe, expect, it } from "vitest";
import { buildRunIndex, listRuns, runHue, shortRunId } from "./runAttribution";

describe("run attribution", () => {
  it("indexes agent runs by path and groups them deterministically", () => {
    const index = buildRunIndex([
      { relativePath: "api/a.go", status: "M", agentManagerRunId: "run-alpha", sandboxOwner: "agent-a" },
      { relativePath: "ui/a.tsx", status: "A", sandboxId: "sandbox-beta", sandboxOwner: "agent-b" },
      { relativePath: "README.md", status: "M" },
    ]);
    expect(index.get("api/a.go")?.runId).toBe("run-alpha");
    expect(index.get("ui/a.tsx")?.runId).toBe("sandbox-beta");
    expect(listRuns(index).map((run) => [run.runId, run.fileCount])).toEqual([
      ["run-alpha", 1],
      ["sandbox-beta", 1],
    ]);
  });

  it("keeps hues stable and uses a compact chip label", () => {
    expect(runHue("run-alpha")).toBe(runHue("run-alpha"));
    expect(runHue("run-alpha")).not.toBe(runHue("run-beta"));
    expect(shortRunId("123456789")).toBe("12345678");
  });

  it("enriches approved files with provenance timestamps", () => {
    const index = buildRunIndex(
      [{ relativePath: "a.ts", status: "M", sandboxId: "sandbox-a" }],
      [{ runId: "agent-run-a", sandboxOwner: "agent-a", latestAppliedAt: "2026-08-30T12:00:00Z", files: [{ relativePath: "a.ts", appliedAt: "2026-08-30T11:59:00Z" }] }],
    );
    expect(index.get("a.ts")).toMatchObject({ runId: "agent-run-a", owner: "agent-a", appliedAt: "2026-08-30T11:59:00Z" });
  });
});
