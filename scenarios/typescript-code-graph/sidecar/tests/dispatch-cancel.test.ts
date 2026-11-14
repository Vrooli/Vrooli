import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("../src/extract.js", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../src/extract.js")>();
  return { ...actual, extract: vi.fn(() => ({ graph: { nodes: [], edges: [] }, warnings: [] })) };
});
vi.mock("../src/rewrite.js", () => ({ applyRewrite: vi.fn(() => []) }));

import { extract } from "../src/extract.js";
import { resetStdoutSink, setStdoutSink } from "../src/framing.js";
import { dispatch } from "../src/index.js";
import { _resetLocksForTests, withPathLock } from "../src/lock.js";
import { applyRewrite } from "../src/rewrite.js";

function captureStdout(): string[] {
  const lines: string[] = [];
  setStdoutSink({ write: (chunk: string) => (lines.push(chunk.trim()), true) });
  return lines;
}

/** Holds the per-path lock so later requests queue behind it. */
function holdPath(projectPath: string): () => void {
  let release!: () => void;
  const held = new Promise<void>((resolve) => (release = resolve));
  void withPathLock(projectPath, () => held);
  return release;
}

afterEach(() => {
  resetStdoutSink();
  _resetLocksForTests();
  vi.mocked(extract).mockClear();
  vi.mocked(applyRewrite).mockClear();
});

describe("dispatch cancellation", () => {
  it("never starts an extract cancelled while queued behind the path lock", async () => {
    const out = captureStdout();
    const release = holdPath("/proj");

    const queued = dispatch({ type: "extract", request_id: "abandoned", project_path: "/proj" });
    await dispatch({ type: "cancel", request_id: "abandoned" });
    release();
    await queued;

    expect(extract).not.toHaveBeenCalled();
    expect(out).toEqual([]);
  });

  it("never applies a rewrite cancelled while queued behind the path lock", async () => {
    const out = captureStdout();
    const release = holdPath("/proj");

    const queued = dispatch({ type: "rewrite_apply", request_id: "abandoned", project_path: "/proj", operations: [] });
    await dispatch({ type: "cancel", request_id: "abandoned" });
    release();
    await queued;

    expect(applyRewrite).not.toHaveBeenCalled();
    expect(out).toEqual([]);
  });

  it("still runs queued work that was not cancelled", async () => {
    const out = captureStdout();
    const release = holdPath("/proj");

    const queued = dispatch({ type: "extract", request_id: "wanted", project_path: "/proj" });
    await dispatch({ type: "cancel", request_id: "someone-else" });
    release();
    await queued;

    expect(extract).toHaveBeenCalledTimes(1);
    expect(out.map((l) => JSON.parse(l).request_id)).toEqual(["wanted"]);
  });
});
