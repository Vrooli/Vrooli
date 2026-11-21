import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { RunStatus } from "../../api/runs";

vi.mock("../../api/health", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../../api/health")>()),
  fetchHealth: vi.fn(),
}));
vi.mock("../../hooks/useTargetStatus", () => ({
  useTargetStatus: vi.fn(),
}));

import * as healthApi from "../../api/health";
import { useTargetStatus } from "../../hooks/useTargetStatus";
import { summarizePosture } from "./posture";
import { PostureBanner } from "./PostureBanner";

const now = new Date("2026-11-18T12:00:00Z");
const old = timestampFromDate(new Date("2026-08-01T12:00:00Z"));
const recent = timestampFromDate(new Date("2026-11-18T11:00:00Z"));

const status = (overrides: Partial<{ targetId: string; overdue: boolean; lastVerifiedAt?: unknown }>) =>
  ({
    targetId: "t1",
    overdue: false,
    lastRunStatus: RunStatus.COMPLETED,
    lastVerifiedAt: recent,
    ...overrides,
  }) as never;

afterEach(() => cleanup());

describe("summarizePosture", () => {
  it("counts the union of overdue and unverified targets, not the sum", () => {
    const summary = summarizePosture(
      [status({ targetId: "t1", overdue: true, lastVerifiedAt: old }), status({ targetId: "t2", overdue: true, lastVerifiedAt: old })],
      now,
    );
    expect(summary).toMatchObject({ attention: 2, overdue: 2, unverified: 2, allClear: false });
  });

  it("counts a target that is only overdue once", () => {
    const summary = summarizePosture([status({ overdue: true })], now);
    expect(summary).toMatchObject({ attention: 1, overdue: 1, unverified: 0 });
  });

  it("is all-clear only with targets and no attention needed", () => {
    expect(summarizePosture([status({})], now).allClear).toBe(true);
    expect(summarizePosture([], now).allClear).toBe(false);
  });
});

describe("PostureBanner", () => {
  it("renders the attention headline when something needs attention", async () => {
    vi.mocked(healthApi.fetchHealth).mockResolvedValue({
      status: "degraded",
      service: "data-backup-manager",
    } as never);
    vi.mocked(useTargetStatus).mockReturnValue({
      data: [status({ overdue: true, lastVerifiedAt: old })],
      isLoading: false,
    } as never);

    renderWithProviders(<PostureBanner />);

    const headline = await screen.findByTestId(selectors.overview.postureHeadline);
    expect(headline).toHaveTextContent("posture.attention");
  });
});
