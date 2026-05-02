import assert from "node:assert/strict";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createElement } from "react";
import { beforeEach, test, vi } from "vitest";
import type { useQuery } from "@tanstack/react-query";
import { ExportButton } from "../../src/features/stats/components/controls/ExportButton.js";
import type {
  ModelBreakdownResponse,
  ProfileBreakdownResponse,
  RunnerBreakdownResponse,
  SummaryResponse,
  TimePreset,
  ToolUsageResponse,
} from "../../src/features/stats/api/types.js";
import { useModelBreakdown } from "../../src/features/stats/hooks/useModelBreakdown.js";
import { useProfileBreakdown } from "../../src/features/stats/hooks/useProfileBreakdown.js";
import { useRunnerPerformance } from "../../src/features/stats/hooks/useRunnerPerformance.js";
import { useStatsSummary } from "../../src/features/stats/hooks/useStatsSummary.js";
import { useTimeWindow } from "../../src/features/stats/hooks/useTimeWindow.js";
import { useToolUsage } from "../../src/features/stats/hooks/useToolUsage.js";
import { renderWithProviders } from "../../src/test-utils/index.js";
import {
  makeModelBreakdownResponse,
  makeProfileBreakdownResponse,
  makeRunnerBreakdownResponse,
  makeSummaryResponse,
  makeToolUsageResponse,
} from "../testutil/stats.js";

vi.mock("../../src/features/stats/hooks/useStatsSummary.js", () => ({
  useStatsSummary: vi.fn(),
}));

vi.mock("../../src/features/stats/hooks/useRunnerPerformance.js", () => ({
  useRunnerPerformance: vi.fn(),
}));

vi.mock("../../src/features/stats/hooks/useProfileBreakdown.js", () => ({
  useProfileBreakdown: vi.fn(),
}));

vi.mock("../../src/features/stats/hooks/useModelBreakdown.js", () => ({
  useModelBreakdown: vi.fn(),
}));

vi.mock("../../src/features/stats/hooks/useToolUsage.js", () => ({
  useToolUsage: vi.fn(),
}));

vi.mock("../../src/features/stats/hooks/useTimeWindow.js", () => ({
  getPresetLabel: (preset: TimePreset) => `Window ${preset}`,
  useTimeWindow: vi.fn(),
}));

type QueryResult<T> = ReturnType<typeof useQuery<T, Error>>;

const presetOptions: readonly TimePreset[] = ["6h", "12h", "24h", "7d", "30d"];

function queryResult<T>(overrides: Partial<QueryResult<T>>): QueryResult<T> {
  return {
    data: undefined,
    isLoading: false,
    error: null,
    ...overrides,
  } as QueryResult<T>;
}

function readBlobText(blob: Blob): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onerror = () => reject(reader.error);
    reader.onload = () => resolve(String(reader.result));
    reader.readAsText(blob);
  });
}

beforeEach(() => {
  vi.mocked(useTimeWindow).mockReturnValue({
    preset: "7d",
    setPreset: vi.fn(),
    filter: { preset: "7d" },
    presetOptions,
  });
  vi.mocked(useStatsSummary).mockReturnValue(queryResult<SummaryResponse>({
    data: makeSummaryResponse(),
  }));
  vi.mocked(useRunnerPerformance).mockReturnValue(queryResult<RunnerBreakdownResponse>({
    data: makeRunnerBreakdownResponse(),
  }));
  vi.mocked(useProfileBreakdown).mockReturnValue(queryResult<ProfileBreakdownResponse>({
    data: makeProfileBreakdownResponse(),
  }));
  vi.mocked(useModelBreakdown).mockReturnValue(queryResult<ModelBreakdownResponse>({
    data: makeModelBreakdownResponse(),
  }));
  vi.mocked(useToolUsage).mockReturnValue(queryResult<ToolUsageResponse>({
    data: makeToolUsageResponse(),
  }));
});

test("ExportButton is disabled until summary data is available", () => {
  vi.mocked(useStatsSummary).mockReturnValue(queryResult<SummaryResponse>({}));

  renderWithProviders(createElement(ExportButton));

  assert.equal(screen.getByRole("button", { name: /export/i }).hasAttribute("disabled"), true);
});

test("ExportButton writes a CSV download with summary and breakdown sections", async () => {
  const user = userEvent.setup();
  let exportedBlob: Blob | undefined;
  const clickSpy = vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => undefined);
  const createObjectURL = vi.fn((blob: Blob | MediaSource) => {
    exportedBlob = blob as Blob;
    return "blob:stats-export";
  });
  const revokeObjectURL = vi.fn();
  Object.defineProperty(URL, "createObjectURL", {
    configurable: true,
    value: createObjectURL,
  });
  Object.defineProperty(URL, "revokeObjectURL", {
    configurable: true,
    value: revokeObjectURL,
  });

  renderWithProviders(createElement(ExportButton));

  await user.click(screen.getByRole("button", { name: /export/i }));

  assert.equal(clickSpy.mock.calls.length, 1);
  assert.equal(createObjectURL.mock.calls.length, 1);
  assert.deepEqual(revokeObjectURL.mock.calls, [["blob:stats-export"]]);
  assert.ok(exportedBlob);

  const csv = await readBlobText(exportedBlob);
  assert.match(csv, /Agent Manager Stats Export - Window 7d/);
  assert.match(csv, /=== Summary ===/);
  assert.match(csv, /Success Rate,87\.5%/);
  assert.match(csv, /=== Runner Breakdown ===/);
  assert.match(csv, /codex,18,16,2,88\.9%,8\.25,90000/);
  assert.match(csv, /=== Profile Breakdown ===/);
  assert.match(csv, /"Maintenance Agent",profile-maintenance,14,12,2,85\.7%,7\.50/);
  assert.match(csv, /=== Model Breakdown ===/);
  assert.match(csv, /claude-3-opus,12,9,75\.0%,3\.25,42000/);
  assert.match(csv, /=== Tool Usage ===/);
  assert.match(csv, /Edit,20,18,2,90\.0%/);
});
