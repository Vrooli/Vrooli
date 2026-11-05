import { timestampFromMs } from "@bufbuild/protobuf/wkt";
import { screen } from "@testing-library/react";
import { createElement } from "react";
import { expect, test } from "vitest";
import { RunDetailsContent } from "./RunDetailParts.js";
import { makeRun } from "../test-utils/fixtures/runs.js";
import { renderWithProviders } from "../test-utils/index.js";
import { ExecutionMode, RunMode, RunPhase } from "../types.js";

const costTotals = { inputTokens: 0, outputTokens: 0, cacheCreationTokens: 0, cacheReadTokens: 0, totalCostUsd: 0, webSearchRequests: 0, serverToolUseRequests: 0, models: [], serviceTiers: [], events: 0 };
const show = (run: ReturnType<typeof makeRun>, durationMs: number | null) =>
  renderWithProviders(createElement(RunDetailsContent, { run, diff: null, taskTitle: "Task", profileName: "Profile", durationMs, costTotals }));

test("run mode, execution and phase read as words, never as raw enum names", () => {
  show(makeRun({ runMode: RunMode.IN_PLACE, executionMode: ExecutionMode.CODEC_PIPE, phase: RunPhase.EXECUTING }), null);
  const text = document.body.textContent ?? "";
  expect(text).toContain("Mode: In place");
  expect(text).toContain("Execution: Codec pipe");
  expect(text).toContain("Phase: Executing");
  expect(text).not.toMatch(/in_place|codec_pipe/);
});

test("a finished run reports its duration and a running one the time elapsed so far", () => {
  const { unmount } = show(makeRun({ startedAt: timestampFromMs(0), endedAt: timestampFromMs(479_000) }), 479_000);
  expect(document.body.textContent).toContain("Duration: 7 m 59 s");
  unmount();
  show(makeRun({ startedAt: timestampFromMs(0), endedAt: undefined }), 720_000);
  expect(document.body.textContent).toContain("Running: 12 m 0 s");
  expect(screen.queryByText("Duration: ")).toBeNull();
});
