import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create } from "@bufbuild/protobuf";
import { DiffResultSchema, type DiffResult } from "@vrooli/proto-types/image-tools/v1/diff/diff_pb";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { CompareView } from "./CompareView";
import type { CompareClient } from "./useCompare";

const diffResult = (): DiffResult =>
  create(DiffResultSchema, {
    jobId: "job-diff",
    verdict: "different",
    dimensionsMatch: false,
    baseWidth: 100,
    baseHeight: 80,
    compareWidth: 120,
    compareHeight: 80,
    changedPixels: 1234n,
    totalPixels: 8000n,
    changedFraction: 0.154,
    mae: 12.5,
    rmse: 20.1,
    psnr: 28.3,
    phashDistance: 7,
    phashSimilarity: 0.89,
    ssim: 0.74,
    heatmapRef: "out/heat.png",
    warnings: ["images differ in size"],
  });

const fakeClient = (): CompareClient => ({
  compare: vi.fn().mockResolvedValue(diffResult()),
  blobUrl: (key: string) => `/api/v1/blobs/${key}`,
});

beforeEach(() => {
  vi.stubGlobal("URL", {
    ...URL,
    createObjectURL: vi.fn(() => "blob:fake"),
    revokeObjectURL: vi.fn(),
  });
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
  vi.clearAllMocks();
});

describe("CompareView a11y", () => {
  it("has no a11y violations in the empty state", async () => {
    const { container } = renderWithProviders(<CompareView client={fakeClient()} />);
    await expectNoA11yViolations(container);
  });

  it("has no a11y violations once a comparison is rendered", async () => {
    const { container } = renderWithProviders(<CompareView client={fakeClient()} />);
    await userEvent.upload(
      screen.getByTestId<HTMLInputElement>(selectors.compare.baseFileInput),
      new File(["base"], "base.png", { type: "image/png" }),
    );
    await userEvent.upload(
      screen.getByTestId<HTMLInputElement>(selectors.compare.compareFileInput),
      new File(["compare"], "compare.png", { type: "image/png" }),
    );
    await userEvent.click(screen.getByTestId(selectors.compare.runButton));
    await waitFor(() => expect(screen.getByTestId(selectors.compare.result)).toBeInTheDocument());
    await expectNoA11yViolations(container);
  });
});
